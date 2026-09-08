//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package longsocks

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math"
	"math/big"
	"os"
	"time"
)

type Identity struct {
	Priv *ecdsa.PrivateKey
	Cert *x509.Certificate
}

func CreateIdentity(cfg *Config) (*Identity, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	serial, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return nil, err
	}

	subject := pkix.Name{
		Country:      cfg.Longsocksd.Certificate.Country,
		Organization: cfg.Longsocksd.Certificate.Organization,
		CommonName:   cfg.Longsocksd.Certificate.CommonName,
	}
	now := time.Now()

	caTmpl := &x509.Certificate{
		SignatureAlgorithm: x509.ECDSAWithSHA512,
		SerialNumber:       serial,
		Subject:            subject,
		NotBefore:          now,
		NotAfter:           now.Add(time.Hour * 24 * 365 * 20),
		KeyUsage: x509.KeyUsageDigitalSignature |
			x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}
	der, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl,
		&priv.PublicKey, priv)
	if err != nil {
		return nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}

	err = SavePrivateKey(cfg.Longsocksd.PrivateKeyFile, priv)
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(cfg.Longsocksd.CertificateFile, cert.Raw, 0666)
	if err != nil {
		return nil, err
	}

	return &Identity{
		Priv: priv,
		Cert: cert,
	}, nil
}

// SavePrivateKey saves the private key to the specified file.
func SavePrivateKey(name string, privateKey *ecdsa.PrivateKey) error {
	x509Encoded, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return err
	}
	data := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509Encoded,
	})
	return os.WriteFile(name, data, 0600)
}

func LoadIdentity(cfg *Config) (*Identity, error) {
	pemEncoded, err := os.ReadFile(cfg.Longsocksd.PrivateKeyFile)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(pemEncoded)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM data")
	}
	priv, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	der, err := os.ReadFile(cfg.Longsocksd.CertificateFile)
	if err != nil {
		return nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}

	return &Identity{
		Priv: priv,
		Cert: cert,
	}, nil
}
