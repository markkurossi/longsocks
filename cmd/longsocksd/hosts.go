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
	"sync"

	"github.com/markkurossi/go-libs/uuid"
	"github.com/markkurossi/longsocks/control"
)

var (
	hosts         = make(map[string]*Host)
	registrations = make(map[string]*Host)
	appConnReqs   = make(map[string]net.Conn)
	m             sync.Mutex
)

type Host struct {
	Names []string
	ID    uuid.UUID
	Ch    chan ConnReq
	conn  net.Conn
}

type ConnReq struct {
	IP       net.IP
	Hostname string
	Port     uint16
	Conn     net.Conn
}

func (cr ConnReq) String() string {
	if len(cr.Hostname) > 0 {
		return fmt.Sprintf("ConnReq:%v", cr.Hostname)
	}
	return fmt.Sprintf("ConnReq:%v", cr.IP)
}

func (host *Host) String() string {
	return fmt.Sprintf("%v", host.Names)
}

func NewRegistration(hostname string) (*Host, error) {
	id, err := uuid.New()
	if err != nil {
		return nil, err
	}

	host := &Host{
		Names: []string{strings.ToLower(hostname)},
		ID:    id,
	}
	idstr := id.String()

	m.Lock()
	defer m.Unlock()

	registrations[idstr] = host

	return host, nil
}

func Registration(id string) (*Host, error) {
	id = strings.ToUpper(id)

	m.Lock()
	defer m.Unlock()

	host, ok := registrations[id]
	if !ok {
		return nil, fmt.Errorf("unknown registration: %v", id)
	}
	host.ID = uuid.Nil
	delete(registrations, id)

	return host, nil
}

func NewHost(names []string, conn net.Conn) {
	host := &Host{
		Names: names,
		Ch:    make(chan ConnReq),
		conn:  conn,
	}

	m.Lock()
	for _, name := range names {
		hosts[name] = host
	}
	m.Unlock()

	host.control()

	m.Lock()
	defer m.Unlock()

	for _, name := range names {
		h, ok := hosts[name]
		if !ok || h != host {
			continue
		}
		delete(hosts, name)
	}
}

func LookupHost(ip net.IP, hostname string) (*Host, error) {
	m.Lock()
	defer m.Unlock()

	if len(hostname) > 0 {
		host, ok := hosts[hostname]
		if !ok {
			return nil, fmt.Errorf("unknown host: %v", hostname)
		}
		return host, nil
	}

	return nil, fmt.Errorf("unknown host: %v", ip)
}

func NewAppConnReq(conn net.Conn) (string, error) {
	id, err := uuid.New()
	if err != nil {
		return "", err
	}
	idstr := id.String()

	m.Lock()
	defer m.Unlock()

	appConnReqs[idstr] = conn

	return idstr, nil
}

func NewAppConn(conn net.Conn, cookie string) {

	m.Lock()
	appConn, ok := appConnReqs[cookie]
	m.Unlock()

	if !ok {
		log.Printf("unknown appcon %v", cookie)
		return
	}
	delete(appConnReqs, cookie)

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

	var reply [10]byte

	reply[0] = 0x05
	reply[3] = 0x01

	_, err := appConn.Write(reply[:])
	if err != nil {
		log.Printf("SOCK5 reply error: %v", err)
		appConn.Close()
		conn.Close()
		return
	}

	control.Relay(appConn, conn)
	log.Printf("appcon %v closed", cookie)
}
