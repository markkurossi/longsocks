//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"log"
	"os"

	"github.com/markkurossi/longsocks"
	"github.com/markkurossi/longsocks/control"
)

func processCommands(args []string) error {
	switch args[0] {
	case "init":
		if len(args) != 2 {
			return fmt.Errorf("init: invalid arguments: %v", args)
		}
		return cmdInit(args[1])

	case "remote-init":
		if len(args) != 4 {
			return fmt.Errorf("remote-init: invalid arguments: %v", args)
		}
		return cmdRemoteInit(args[1], args[2], args[3])

	default:
		return fmt.Errorf("unknown command: %v", args[0])
	}
}

func cmdInit(commonName string) error {
	loadConfig()

	priv, err := longsocks.CreateKeypair()
	if err != nil {
		return err
	}
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
		return err
	}
	defer conn.Close()

	csr, err := makeCSR(priv, commonName)
	if err != nil {
		return err
	}

	return initInstance(priv, conn, control.HostInit{
		CSR: csr,
	})
}

func cmdRemoteInit(server, cookieStr, digestStr string) error {
	cookie, err := longsocks.ParseHex([]string{cookieStr})
	if err != nil {
		return err
	}
	if len(cookie) < 4 {
		return fmt.Errorf("truncated cookie")
	}
	digest, err := longsocks.ParseHex([]string{digestStr})
	if err != nil {
		return err
	}
	if len(digest) < 4 {
		return fmt.Errorf("truncated certificate digest")
	}

	priv, err := longsocks.CreateKeypair()
	if err != nil {
		return err
	}
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte,
			verifiedChains [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return fmt.Errorf("no server certificate")
			}
			mac := hmac.New(sha256.New, cookie)
			mac.Write(rawCerts[0])
			hmac := mac.Sum(nil)

			l := len(digest)
			if l > len(hmac) {
				l = len(hmac)
			}

			if !bytes.Equal(digest[:l], hmac[:l]) {
				return fmt.Errorf("certificate digest mismatch")
			}

			return nil
		},
		// ServerName ensures hostname verification succeeds if
		// connecting via IP address.
		// ServerName: config.Longsocksd.Hostname,
	}
	conn, err := tls.Dial("tcp", server, tlsConfig)
	if err != nil {
		return err
	}
	defer conn.Close()

	csr, err := makeCSR(priv, cookieStr)
	if err != nil {
		return err
	}

	return initInstance(priv, conn, control.HostRemoteInit{
		CSR: csr,
	})
}

func makeCSR(priv interface{}, commonName string) ([]byte, error) {
	csr, err := x509.CreateCertificateRequest(rand.Reader,
		&x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName: commonName,
			},
		}, priv)
	if err != nil {
		return nil, err
	}
	return csr, nil
}

func initInstance(priv *ecdsa.PrivateKey, conn *tls.Conn,
	req interface{}) error {

	msgType, resp, err := control.RPC(conn, req)
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
		return saveInit(priv, msg.Cert)

	case control.MsgHostRemoteInitResp:
		var msg control.HostRemoteInitResp
		_, err = longsocks.UnmarshalFrom(resp, &msg)
		if err != nil {
			return err
		}
		config, err = longsocks.Parse(msg.Config)
		if err != nil {
			return err
		}
		err = os.MkdirAll(longsocks.EtcDir, 0755)
		if err != nil {
			return err
		}
		cfgfile := longsocks.ConfigFile(dialdConfig)
		log.Printf("saving config file %v", cfgfile)
		err = os.WriteFile(cfgfile, msg.Config, 0666)
		if err != nil {
			return err
		}
		return saveInit(priv, msg.Cert)

	default:
		return fmt.Errorf("invalid response: %v", msgType)
	}
}

func saveInit(priv *ecdsa.PrivateKey, cert []byte) error {
	log.Printf("saving certificate %v", config.Diald.CertificateFile)
	err := longsocks.SaveCertificate(config.Diald.CertificateFile, cert)
	if err != nil {
		return err
	}
	log.Printf("saving private key %v", config.Diald.PrivateKeyFile)
	return longsocks.SavePrivateKey(config.Diald.PrivateKeyFile, priv)
}
