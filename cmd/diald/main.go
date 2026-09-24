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
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/markkurossi/longsocks"
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
		log.Fatal(err)
		log.Printf("diald terminated")
	} else {
		err = processCommands(flag.Args())
		if err != nil {
			log.Fatal(err)
		}
	}
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

	delay := 1 * time.Second

	for {
		log.Printf("dialing %v", addr)
		conn, err := dialer.Dial("tcp", addr)
		if err != nil {
			log.Printf("dial failed: %v", err)
			delay *= 2
			if delay > 64*time.Second {
				delay = 64 * time.Second
			}
		} else {
			log.Printf("diald connected")
			err = processControl(conn)
			if err != nil {
				log.Printf("diald connection terminated: %v", err)
			} else {
				log.Printf("diald disconnected")
			}
			delay = 1 * time.Second
		}

		sleep := time.Duration(rand.Float64() * float64(delay))

		log.Printf("retrying in %v (backoff %v)", sleep, delay)
		time.Sleep(sleep)
	}
}

func getCACertificates() (*x509.CertPool, error) {
	certPool := x509.NewCertPool()

	if !certPool.AppendCertsFromPEM([]byte(config.Diald.CA)) {
		return nil, fmt.Errorf("invalid CA certificate")
	}

	return certPool, nil
}
