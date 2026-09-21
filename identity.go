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
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math"
	"math/big"
	"net"
	"os"
	"time"
)

type Identity struct {
	Priv *ecdsa.PrivateKey
	Cert *x509.Certificate
}

func (id *Identity) CertPEM() []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: id.Cert.Raw,
	})
}

func (id *Identity) TLSCertificate() tls.Certificate {
	return tls.Certificate{
		Certificate: [][]byte{id.Cert.Raw},
		PrivateKey:  id.Priv,
		Leaf:        id.Cert,
	}
}

func (id *Identity) CreateHostCertificate(cfg *Config, dnsNames []string,
	csr *x509.CertificateRequest) (*x509.Certificate, error) {

	serial, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	subject := pkix.Name{
		Country:      cfg.Longsocksd.Certificate.Country,
		Organization: cfg.Longsocksd.Certificate.Organization,
		CommonName:   cfg.Longsocksd.Certificate.HostCommonName,
	}
	now := time.Now()

	tmpl := &x509.Certificate{
		SignatureAlgorithm: x509.ECDSAWithSHA512,
		SerialNumber:       serial,
		Subject:            subject,
		NotBefore:          now,
		NotAfter:           now.Add(time.Hour * 24 * 365),
		KeyUsage:           x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
		},
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, id.Cert,
		csr.PublicKey, id.Priv)
	if err != nil {
		return nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}

	return cert, nil
}

func CreateKeypair() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

func CreateIdentity(cfg *Config) (*Identity, error) {
	priv, err := CreateKeypair()
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

		DNSNames: []string{
			cfg.Longsocksd.Hostname,
			"localhost",
			"*.localhost",
			"*.local",
		},
		IPAddresses: []net.IP{
			net.ParseIP("127.0.0.1"),
			net.ParseIP("::1"),
		},
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

func SaveCertificate(name string, der []byte) error {
	return os.WriteFile(name, der, 0666)
}

func LoadIdentity(privFile, certFile string) (*Identity, error) {
	pemEncoded, err := os.ReadFile(privFile)
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

	der, err := os.ReadFile(certFile)
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
