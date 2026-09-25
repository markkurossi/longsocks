// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
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
			log.Printf("failed to accept control connection: %v", err)
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

	log.Printf("client for DNS names %v", clientCert.DNSNames)

	// Start processing commands from the authenticated control
	// connection.

	msgType, data, err := control.Recv(conn)
	if err != nil {
		log.Printf("failed to read message: %v", err)
		return
	}
	switch msgType {
	case control.MsgCtrlCh:
		var msg control.CtrlCh
		_, err = longsocks.UnmarshalFrom(data, &msg)
		if err != nil {
			log.Printf("invalid %v message: %v", msgType, err)
			return
		}
		NewHost(clientCert.DNSNames, conn)
		log.Printf("hosts: %v", hosts)

	case control.MsgAppCh:
		var msg control.AppCh
		_, err = longsocks.UnmarshalFrom(data, &msg)
		if err != nil {
			log.Printf("invalid %v message: %v", msgType, err)
			return
		}
		log.Printf("appcon %v", msg.Cookie)
		NewAppConn(conn, msg.Cookie)

	default:
		log.Printf("msg %v not implemented yet:\n%s", msgType, hex.Dump(data))
	}
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
		log.Printf("CSR: UUID=%s", csr.Subject.CommonName)
		host, err := Registration(csr.Subject.CommonName)
		if err != nil {
			return err
		}
		log.Printf("hostname: %v", host.Names)
		cert, err := identity.CreateHostCertificate(config, host.Names, csr)
		if err != nil {
			return err
		}
		log.Printf("cert: issuer=%v, subject=%v", cert.Issuer, cert.Subject)
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

func (host *Host) control() {
	log.Printf("control handler for host %v", host)
	err := host.eventLoop()
	log.Printf("%v: connection terminated: %v", host, err)
}

func (host *Host) eventLoop() error {
	for {
		select {
		case cr := <-host.Ch:
			cookie, err := NewAppConnReq(cr.Conn)
			if err != nil {
				log.Printf("failed to create connection request: %v", err)
				cr.Conn.Close()
				continue
			}

			_, _, err = control.RPC(host.conn, control.ConnReq{
				IP:       []byte(cr.IP),
				Hostname: cr.Hostname,
				Port:     cr.Port,
				Cookie:   cookie,
			})
			if err != nil {
				log.Printf("failed to create appcon: %v", err)
				cr.Conn.Close()
			}

		case <-time.After(5 * time.Second):
			reqTime := time.Now().UnixMicro()
			t, data, err := control.RPC(host.conn, control.Ping{
				Time: uint64(reqTime),
			})
			if err != nil {
				return err
			}
			respTime := time.Now().UnixMicro()
			expectedTime := reqTime + (respTime-reqTime)/2

			switch t {
			case control.MsgPong:
				var msg control.Pong
				_, err := longsocks.UnmarshalFrom(data, &msg)
				if err != nil {
					return err
				}
				d := int64(msg.Time) - expectedTime
				log.Printf("keepalive: delta=%v", time.Duration(d*1000))

			default:
				log.Printf("%v:\n%s", t, hex.Dump(data))
				return fmt.Errorf("%v unsupported", t)
			}
		}
	}
}
