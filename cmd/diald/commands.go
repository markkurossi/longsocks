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
	"encoding/hex"
	"fmt"
	"log"

	"github.com/markkurossi/longsocks"
	"github.com/markkurossi/longsocks/control"
)

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

		msgType, resp, err := control.RPC(conn, control.HostInit{
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
