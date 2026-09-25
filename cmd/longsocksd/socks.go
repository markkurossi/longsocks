// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"fmt"
	"log"
	"net"
	"strings"
)

type Method byte

const (
	MethodNoAuthenticationRequired Method = iota
	MethodGSSAPI
	MethodUsernamePassword

	MethodNoAcceptableMethods Method = 0xff
)

func (m Method) String() string {
	switch m {
	case MethodNoAuthenticationRequired:
		return "NO_AUTHENTICATION_REQUIRED"
	case MethodGSSAPI:
		return "GSSAPI"
	case MethodUsernamePassword:
		return "USERNAME/PASSWORD"
	case MethodNoAcceptableMethods:
		return "NO_ACCEPTABLE_METHODS"
	default:
		if 0x03 <= m && m <= 0x7f {
			return fmt.Sprintf("IANA_ASSIGNED_%02X", byte(m))
		}
		return fmt.Sprintf("PRIVATE_METHOD_%02X", byte(m))
	}
}

type Command byte

const (
	CmdConnect Command = iota + 1
	CmdBind
	CmdUDPAssociate
)

var commands = map[Command]string{
	CmdConnect:      "CONNECT",
	CmdBind:         "BIND",
	CmdUDPAssociate: "UDP_ASSOCIATE",
}

func (cmd Command) String() string {
	name, ok := commands[cmd]
	if ok {
		return name
	}
	return fmt.Sprintf("{Command %02X}", int(cmd))
}

type Socks struct {
	listener net.Listener
}

func NewSocksListener() (*Socks, error) {
	listener, err := net.Listen("tcp", config.Longsocksd.Socks.Listen)
	if err != nil {
		return nil, err
	}

	return &Socks{
		listener: listener,
	}, nil
}

func (socks *Socks) Run() {
	for {
		conn, err := socks.listener.Accept()
		if err != nil {
			log.Printf("failed to accept SOCKS connection: %v", err)
			continue
		}
		go socks.handler(conn)
	}
}

func (socks *Socks) handler(conn net.Conn) {
	err := socks.handle(conn)
	if err != nil {
		log.Printf("handler failed: %v", err)
		conn.Close()
	}
}

func (socks *Socks) handle(conn net.Conn) error {
	var hdr [1 + 1 + 255]byte

	for {
		n, err := conn.Read(hdr[:])
		if err != nil {
			return err
		}
		if n < 3 || n < int(2+hdr[1]) {
			return fmt.Errorf("truncated header: len=%v", n)
		}
		version := hdr[0]

		var methods []string
		selected := MethodNoAcceptableMethods

		for _, b := range hdr[2:n] {
			m := Method(b)
			if m == MethodNoAuthenticationRequired {
				selected = m
			}
			methods = append(methods, Method(b).String())
		}

		log.Printf("SOCKS%v: %v", version, strings.Join(methods, ","))

		hdr[0] = 0x05
		hdr[1] = byte(selected)
		_, err = conn.Write(hdr[:2])
		if err != nil {
			return err
		}
		log.Printf("selected %v", selected)
		if selected == MethodNoAcceptableMethods {
			return fmt.Errorf("no acceptable methods")
		}

		_, err = conn.Read(hdr[:5])
		if err != nil {
			return err
		}
		// Check address type to resolve the message length.
		var l int
		switch hdr[3] {
		case 0x01: // IPv4
			l = 4
		case 0x03: // FQDN
			l = 1 + int(hdr[4])
		case 0x04: // IPv6
			l = 16
		default:
			log.Printf("invalid address type: %02X", hdr[3])
		}
		// We have read the first address byte.
		l = l - 1 + 2
		_, err = conn.Read(hdr[5 : 5+l])
		if err != nil {
			return err
		}

		cmd := Command(hdr[1])

		var ip net.IP
		var hostname string

		switch hdr[3] {
		case 0x01:
			ip = hdr[4:8]
		case 0x03:
			hostname = string(hdr[5 : 5+hdr[4]])
		case 0x04:
			ip = hdr[4:20]
		}

		port := bo.Uint16(hdr[5+l-2:])

		log.Printf("%v: ip=%v, hostname=%v, port=%d", cmd, ip, hostname, port)

		host, err := LookupHost(ip, hostname)
		if err != nil {
			log.Printf("host lookup failed: %v", err)
			return err
		}

		log.Printf("host: %v", host)

		host.Ch <- ConnReq{
			IP:       ip,
			Hostname: hostname,
			Port:     port,
			Conn:     conn,
		}

		return nil
	}
}
