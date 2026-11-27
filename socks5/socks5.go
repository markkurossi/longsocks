// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

// Package socks5 implements the SOCKS protocol version 5 (RFC 1928).
package socks5

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

// Method defines the authentication methods.
type Method byte

// SOCKS5 authentication methods.
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

// Command defines the request commands.
type Command byte

// SOCKS5 request commands.
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

// Handler is called to handle the client command.
type Handler func(conn net.Conn, cmd Command, ip net.IP, hostname string,
	port int) error

// Server implements SOCKS5 server.
type Server struct {
	listener net.Listener
	handler  Handler
}

// NewServer creates a new SOCKS5 server listening connections on
// addr. The handler is called to handle all client requests.
func NewServer(addr string, handler Handler) (*Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &Server{
		listener: listener,
		handler:  handler,
	}, nil
}

// Run listens and handles client connections.
func (server *Server) Run() {
	for {
		conn, err := server.listener.Accept()
		if err != nil {
			log.Printf("failed to accept SOCKS connection: %v", err)
			continue
		}
		go server.handleConn(conn)
	}
}

func (server *Server) handleConn(conn net.Conn) {
	err := server.handle(conn)
	if err != nil {
		log.Printf("SOCKS connection from %v failed: %v",
			conn.RemoteAddr(), err)
		conn.Close()
	}
}

var (
	HttpGET     = []byte("GET ")
	HttpVersion = []byte(" HTTP/")
)

func (server *Server) handle(conn net.Conn) error {
	var hdr [1 + 1 + 255]byte

	n, err := conn.Read(hdr[:])
	if err != nil {
		return err
	}
	if n == 0 {
		return io.EOF
	}
	version := hdr[0]
	if version != 0x05 {
		// Check HTTP requests.
		if bytes.HasPrefix(hdr[:n], HttpGET) ||
			bytes.Contains(hdr[:n], HttpVersion) {
			return fmt.Errorf("HTTP request: %+q", hdr[:n])
		}

		return fmt.Errorf("unsupported SOCKS version %v", version)
	}
	if n < 3 || n < int(2+hdr[1]) {
		return fmt.Errorf("truncated SOCKS5 header: len=%v", n)
	}

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

	port := int(binary.BigEndian.Uint16(hdr[5+l-2:]))

	return server.handler(conn, cmd, ip, hostname, port)
}

// Reply defines server replies.
type Reply byte

// Server reply messages.
const (
	ReplySucceeded Reply = iota
	ReplySOCKSServerFailure
	ReplyConnectionNotAllowed
	ReplyNetworkUnreachable
	ReplyHostUnreachable
	ReplyConnectionRefused
	ReplyTTLExpired
	ReplyCommandNotSupported
	ReplyAddressTypeNotSupported
)

var replies = map[Reply]string{
	ReplySucceeded:               "succeeded",
	ReplySOCKSServerFailure:      "general SOCKS server failure",
	ReplyConnectionNotAllowed:    "connection not allowed by ruleset",
	ReplyNetworkUnreachable:      "Network unreachable",
	ReplyHostUnreachable:         "Host unreachable",
	ReplyConnectionRefused:       "Connection refused",
	ReplyTTLExpired:              "TTL expired",
	ReplyCommandNotSupported:     "Command not supported",
	ReplyAddressTypeNotSupported: "Address type not supported",
}

func (reply Reply) String() string {
	name, ok := replies[reply]
	if ok {
		return name
	}
	return fmt.Sprintf("{Reply %v}", int(reply))
}

// Response creates a server reply and writes it to conn.
func Response(conn io.Writer, reply Reply, ip net.IP, addr string,
	port int) error {

	// The SOCKS request information is sent by the client as soon as it has
	// established a connection to the SOCKS server, and completed the
	// authentication negotiations.  The server evaluates the request, and
	// returns a reply formed as follows:
	//
	//      +----+-----+-------+------+----------+----------+
	//      |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
	//      +----+-----+-------+------+----------+----------+
	//      | 1  |  1  | X'00' |  1   | Variable |    2     |
	//      +----+-----+-------+------+----------+----------+
	//
	//   Where:
	//
	//        o  VER    protocol version: X'05'
	//        o  REP    Reply field:
	//           o  X'00' succeeded
	//           o  X'01' general SOCKS server failure
	//           o  X'02' connection not allowed by ruleset
	//           o  X'03' Network unreachable
	//           o  X'04' Host unreachable
	//           o  X'05' Connection refused
	//           o  X'06' TTL expired
	//           o  X'07' Command not supported
	//           o  X'08' Address type not supported
	//           o  X'09' to X'FF' unassigned
	//        o  RSV    RESERVED
	//        o  ATYP   address type of following address
	//           o  IP V4 address: X'01'
	//           o  DOMAINNAME: X'03'
	//           o  IP V6 address: X'04'
	//        o  BND.ADDR       server bound address
	//        o  BND.PORT       server bound port in network octet order
	//
	// Fields marked RESERVED (RSV) must be set to X'00'.

	// Maximum reply length is 6 + domainname length + 255 bytes.
	buf := make([]byte, 6+1+255)

	buf[0] = 0x05
	buf[1] = byte(reply)

	var portOfs int

	if len(addr) > 0 {
		buf[3] = 0x03
		ab := []byte(addr)
		if len(ab) > 255 {
			ab = ab[:255]
		}
		buf[4] = byte(len(ab))
		copy(buf[5:], ab)
		portOfs = 5 + len(ab)
	} else if len(ip) == 16 {
		buf[3] = 0x04
		copy(buf[4:], ip)
		portOfs = 4 + 16

	} else if len(ip) == 4 {
		buf[3] = 0x01
		copy(buf[4:], ip)
		portOfs = 4 + 4
	} else {
		buf[3] = 0x01
		portOfs = 4 + 4
	}

	binary.BigEndian.PutUint16(buf[portOfs:], uint16(port))

	_, err := conn.Write(buf[:portOfs+2])
	if err != nil {
		return err
	}
	return nil
}
