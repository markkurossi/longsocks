//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
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
	bo     = binary.BigEndian
	config *longsocks.Config
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

func processCommands(args []string) error {
	switch args[0] {
	case "init":
		if len(args) != 2 {
			return fmt.Errorf("init: invalid arguments: %v", args)
		}
		priv, err := longsocks.CreateKeypair()
		if err != nil {
			return err
		}
		csr, err := x509.CreateCertificateRequest(rand.Reader,
			&x509.CertificateRequest{
				Subject: pkix.Name{
					CommonName: args[1],
				},
			}, priv)
		if err != nil {
			return err
		}
		log.Printf("CSR:\n%s", hex.Dump(csr))

		rootCAs, err := getCACertificates()
		if err != nil {
			return err
		}
		tlsConfig := &tls.Config{
			RootCAs:    rootCAs,
			MinVersion: tls.VersionTLS12,
			// ServerName ensures hostname verification succeeds if
			// connecting via IP address.
			// ServerName: config.Longsocksd.Hostname,
		}
		addr := fmt.Sprintf("%v:%v", config.Longsocksd.Hostname,
			config.Longsocksd.Port)
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			log.Fatalf("failed to connect to longsocksd %v: %v", addr, err)
		}
		defer conn.Close()

		msgType, resp, err := tx(conn, control.HostInit{
			CSR: csr,
		})
		if err != nil {
			return err
		}
		switch msgType {
		case control.MsgHostInitResp:
			var msg control.HostInitResp
			_, err = longsocks.UnmarshalFrom(resp, &msg)
			if err != nil {
				return err
			}
			err = longsocks.SavePrivateKey(config.Diald.PrivateKeyFile, priv)
			if err != nil {
				return err
			}
			err = longsocks.SaveCertificate(config.Diald.CertificateFile,
				msg.Cert)
			if err != nil {
				return err
			}

		default:
			return fmt.Errorf("invalid response: %v", msgType)
		}

	default:
		return fmt.Errorf("unknown command: %v", args[0])
	}
	return nil
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
	log.Printf("loading identity")
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
