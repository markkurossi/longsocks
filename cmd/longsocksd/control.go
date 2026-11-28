//
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
	"net"
	"time"

	"github.com/markkurossi/longsocks"
	"github.com/markkurossi/longsocks/control"
	"github.com/markkurossi/longsocks/log"
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
			log.Errorf("failed to accept control connection: %v", err)
			continue
		}
		go ctrl.handler(conn)
	}
}

func (ctrl *Control) handler(conn net.Conn) {
	defer conn.Close()

	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		log.Errorf("Not a TLS connection")
		return
	}

	if err := tlsConn.Handshake(); err != nil {
		log.Errorf("Handshake failed: %v", err)
		return
	}

	state := tlsConn.ConnectionState()

	// Check if this is client init with random UUID.
	if len(state.PeerCertificates) == 0 {
		log.Infof("client init with UUID")
		err := ctrl.tokenHandler(conn)
		if err != nil {
			log.Errorf("tokenHandler failed: %v", err)
			ctrl.sendError(conn, err)
		}
		return
	}
	if len(state.VerifiedChains) == 0 {
		log.Errorf("mTLS with invalid client certificate")
		return
	}

	clientCert := state.VerifiedChains[0][0]

	log.Infof("client for DNS names %v", clientCert.DNSNames)

	// Start processing commands from the authenticated control
	// connection.

	msgType, data, err := control.Recv(conn)
	if err != nil {
		log.Errorf("failed to read message: %v", err)
		return
	}

	switch msgType {
	case control.MsgCtrlCh:
		var msg control.CtrlCh
		_, err = longsocks.UnmarshalFrom(data, &msg)
		if err != nil {
			log.Errorf("invalid %v message: %v", msgType, err)
			return
		}
		NewHost(clientCert.DNSNames, conn, msg.AllowedPorts, msg.AgentInfo,
			msg.HostInfo, msg.MemInfo, msg.SwapInfo)

	case control.MsgAppCh:
		var msg control.AppCh
		_, err = longsocks.UnmarshalFrom(data, &msg)
		if err != nil {
			log.Errorf("invalid %v message: %v", msgType, err)
			return
		}
		log.Infof("appcon %v", msg.Cookie)
		NewAppConn(conn, msg.Cookie)

	default:
		log.Errorf("msg %v not implemented yet:\n%s", msgType, hex.Dump(data))
	}
}

func (ctrl *Control) sendError(conn net.Conn, err error) error {
	data, err := longsocks.Marshal(control.Error{
		Error: err.Error(),
	})
	if err != nil {
		return err
	}

	var hdr [5]byte
	hdr[0] = byte(control.MsgError)
	bo.PutUint32(hdr[1:], uint32(len(data)))
	_, err = conn.Write(hdr[:])
	if err != nil {
		return err
	}
	_, err = conn.Write(data)
	return err
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

	log.Tracef("%v: %d bytes", msgType, l)

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
		log.Debugf("CSR: UUID=%s", csr.Subject.CommonName)
		host, err := Registration(csr.Subject.CommonName)
		if err != nil {
			return err
		}
		log.Debugf("hostname: %v", host.Names)
		cert, err := identity.CreateHostCertificate(config, host.Names, csr)
		if err != nil {
			return err
		}
		log.Debugf("cert: issuer=%v, subject=%v", cert.Issuer, cert.Subject)
		data, err = longsocks.Marshal(control.HostInitResp{
			Cert: cert.Raw,
		})
		if err != nil {
			return err
		}
		msgType = control.MsgHostInitResp

	case control.MsgHostRemoteInit:
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
		log.Debugf("CSR: UUID=%s", csr.Subject.CommonName)
		cookie, err := longsocks.ParseHex([]string{csr.Subject.CommonName})
		if err != nil {
			return err
		}
		if len(cookie) < 4 {
			return fmt.Errorf("cookie too short")
		}
		host, err := CookieRegistration(cookie)
		if err != nil {
			return err
		}
		log.Debugf("hostname: %v", host.Names)
		cert, err := identity.CreateHostCertificate(config, host.Names, csr)
		if err != nil {
			return err
		}
		log.Debugf("cert: issuer=%v, subject=%v", cert.Issuer, cert.Subject)

		config, err := makeDialdConfig(host)
		if err != nil {
			return err
		}
		data, err = longsocks.Marshal(control.HostRemoteInitResp{
			Config: config,
			Cert:   cert.Raw,
		})
		if err != nil {
			return err
		}
		msgType = control.MsgHostRemoteInitResp

	default:
		return fmt.Errorf("invalid message: %v", msgType)
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

func (diald *Diald) control() {
	log.Infof("control handler for host %v", diald)
	err := diald.eventLoop()
	log.Infof("%v: connection terminated: %v", diald, err)
}

func (diald *Diald) eventLoop() error {
	for {
		select {
		case cr := <-diald.ch:
			cookie, err := NewAppConnReq(cr.Conn)
			if err != nil {
				log.Errorf("failed to create connection request: %v", err)
				cr.Conn.Close()
				continue
			}

			_, _, err = control.RPC(diald.conn, control.ConnReq{
				IP:       []byte(cr.IP),
				Hostname: cr.Hostname,
				Port:     uint16(cr.Port),
				Cookie:   cookie,
			})
			diald.SetHealthy(err == nil)
			if err != nil {
				log.Errorf("failed to create appcon: %v", err)
				cr.Conn.Close()
			} else {
				diald.Seen()
			}

		case <-time.After(30 * time.Second):
			reqTime := time.Now().UnixMicro()
			t, data, err := control.RPC(diald.conn, control.Ping{
				Time: uint64(reqTime),
			})
			diald.SetHealthy(err == nil)
			if err != nil {
				return err
			}
			respTime := time.Now().UnixMicro()
			rtt := respTime - reqTime
			expectedTime := reqTime + rtt/2

			diald.Seen()

			switch t {
			case control.MsgPong:
				var msg control.Pong
				_, err := longsocks.UnmarshalFrom(data, &msg)
				if err != nil {
					diald.SetHealthy(false)
					log.Errorf("failed to decode pong: %v\n%s",
						err, hex.Dump(data))
				}

				d := int64(msg.Time) - expectedTime

				netRTT := rtt - int64(msg.Elapsed)

				log.Debugf("%v keepalive: rtt=%v, netRTT=%v, delta=%v",
					diald, time.Duration(rtt*1000), time.Duration(netRTT*1000),
					time.Duration(d*1000))

				diald.MemInfo = msg.MemInfo
				diald.SwapInfo = msg.SwapInfo

				log.Debugf("%v mem: %v/%v, swap: %v/%v",
					diald,
					msg.MemInfo.Available, msg.MemInfo.Total,
					msg.SwapInfo.Free, msg.SwapInfo.Total)

			default:
				log.Errorf("%v:\n%s", t, hex.Dump(data))
				return fmt.Errorf("%v unsupported", t)
			}
		}
	}
}
