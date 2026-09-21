//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/markkurossi/longsocks"
	"github.com/markkurossi/longsocks/control"
)

var (
	bo       = binary.BigEndian
	config   *longsocks.Config
	identity *longsocks.Identity
)

func main() {
	var err error

	flag.Parse()

	config, err = longsocks.LoadConfig("diald.toml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if len(flag.Args()) == 0 {
		err = run()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("diald terminated")
	} else {
		err = processCommands(flag.Args())
		if err != nil {
			log.Fatal(err)
		}
	}
}

func tx(conn net.Conn, req interface{}) (control.MsgType, []byte, error) {
	data, err := longsocks.Marshal(req)
	if err != nil {
		return control.MsgError, nil, err
	}

	var msgType control.MsgType
	switch req.(type) {
	case control.HostInit:
		msgType = control.MsgHostInit
	default:
		return control.MsgError, nil, fmt.Errorf("unknown request [%T]", req)
	}

	var hdr [5]byte
	hdr[0] = byte(msgType)
	bo.PutUint32(hdr[1:], uint32(len(data)))

	_, err = conn.Write(hdr[:])
	if err != nil {
		return control.MsgError, nil, err
	}
	_, err = conn.Write(data)
	if err != nil {
		return control.MsgError, nil, err
	}

	_, err = conn.Read(hdr[:])
	if err != nil {
		return control.MsgError, nil, err
	}
	fmt.Printf("tx: hdr:\n%s", hex.Dump(hdr[:]))
	l := bo.Uint32(hdr[1:])
	data = make([]byte, l)
	_, err = conn.Read(data)

	return control.MsgType(hdr[0]), data, err
}

func run() error {
	var err error

	log.Printf("loading identity")
	identity, err = longsocks.LoadIdentity(config.Diald.PrivateKeyFile,
		config.Diald.CertificateFile)
	if err != nil {
		return err
	}
	rootCAs, err := getCACertificates()
	if err != nil {
		return err
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{identity.TLSCertificate()},
		RootCAs:      rootCAs,
		MinVersion:   tls.VersionTLS12,
		ServerName:   config.Longsocksd.Hostname,
	}

	dialer := &tls.Dialer{
		Config: tlsConfig,
	}
	addr := fmt.Sprintf("%v:%v", config.Longsocksd.Hostname,
		config.Longsocksd.Port)
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	log.Printf("diald started")
	return nil
}

func getCACertificates() (*x509.CertPool, error) {
	certPool := x509.NewCertPool()

	if !certPool.AppendCertsFromPEM([]byte(config.Diald.CA)) {
		return nil, fmt.Errorf("invalid CA certificate")
	}

	return certPool, nil
}
