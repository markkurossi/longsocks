// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/markkurossi/longsocks"
	"github.com/markkurossi/longsocks/control"
)

type Control struct {
	caCertPool *x509.CertPool
	listener   net.Listener
}

func NewControlListener(id *longsocks.Identity) (*Control, error) {
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(id.Cert)

	tlsCert := id.TLSCertificate()

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		ClientCAs:    caCertPool,
		ClientAuth:   tls.VerifyClientCertIfGiven,
		MinVersion:   tls.VersionTLS12,
	}

	addr := fmt.Sprintf(":%v", config.Longsocksd.Port)

	listener, err := tls.Listen("tcp", addr, tlsConfig)
	if err != nil {
		return nil, err
	}

	return &Control{
		caCertPool: caCertPool,
		listener:   listener,
	}, nil
}

func (ctrl *Control) Close() error {
	return ctrl.listener.Close()
}

func (ctrl *Control) Run() {
	for {
		conn, err := ctrl.listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go ctrl.handler(conn)
	}
}

func (ctrl *Control) handler(conn net.Conn) {
	defer conn.Close()

	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		log.Println("Not a TLS connection")
		return
	}

	if err := tlsConn.Handshake(); err != nil {
		log.Printf("Handshake failed: %v", err)
		return
	}

	state := tlsConn.ConnectionState()

	// Check if this is client init with random UUID.
	if len(state.PeerCertificates) == 0 {
		log.Printf("client init with UUID")
		err := ctrl.tokenHandler(conn)
		if err != nil {
			// XXX send error message here.
			log.Printf("tokenHandler failed: %v", err)
		}
		return
	}
	if len(state.VerifiedChains) == 0 {
		log.Printf("mTLS with invalid client certificate")
		return
	}

	clientCert := state.VerifiedChains[0][0]

	fmt.Printf("Client certificate valid: Subject=%s, DNSNames=%v\n",
		clientCert.Subject, clientCert.DNSNames)

	// Start processing commands from the authenticated control
	// connection.

	// Read and write data over the verified connection...
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return
	}
	log.Printf("Received payload: %s", string(buf[:n]))

	fmt.Fprintln(conn, "Hello, secure world via Custom CA!")
}

func (ctrl *Control) tokenHandler(conn net.Conn) error {
	// Set deadline for unauthenticated connections.
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	var hdr [5]byte

	_, err := conn.Read(hdr[:])
	if err != nil {
		return err
	}

	msgType := control.MsgType(hdr[0])
	l := bo.Uint32(hdr[1:])

	log.Printf("%v: %d bytes", msgType, l)

	// Allow only 16kB for unauthenticated connections.
	if l > 16*1024 {
		return fmt.Errorf("request length too big: %v", l)
	}

	data := make([]byte, l)
	_, err = conn.Read(data)
	if err != nil {
		return err
	}
	switch msgType {
	case control.MsgHostInit:
		var msg control.HostInit
		_, err = longsocks.UnmarshalFrom(data, &msg)
		if err != nil {
			return err
		}
		csr, err := x509.ParseCertificateRequest(msg.CSR)
		if err != nil {
			return err
		}
		err = csr.CheckSignature()
		if err != nil {
			return err
		}
		fmt.Printf("CSR: UUID=%s\n", csr.Subject.CommonName)
		host, err := Registration(csr.Subject.CommonName)
		if err != nil {
			return err
		}
		fmt.Printf("hostname: %v\n", host.Name)
		cert, err := identity.CreateHostCertificate(config,
			[]string{host.Name}, csr)
		if err != nil {
			return err
		}
		fmt.Printf("cert: issuer=%v, subject=%v\n", cert.Issuer, cert.Subject)
		data, err = longsocks.Marshal(control.HostInitResp{
			Cert: cert.Raw,
		})
		if err != nil {
			return err
		}
		msgType = control.MsgHostInitResp
	}

	hdr[0] = byte(msgType)
	bo.PutUint32(hdr[1:], uint32(len(data)))
	_, err = conn.Write(hdr[:])
	if err != nil {
		return err
	}
	_, err = conn.Write(data)

	return err
}
