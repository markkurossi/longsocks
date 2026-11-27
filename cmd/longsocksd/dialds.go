//
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
	"time"

	"github.com/markkurossi/go-libs/uuid"
	"github.com/markkurossi/longsocks/control"
	"github.com/markkurossi/longsocks/socks5"
	"github.com/markkurossi/longsocks/sysinfo"
)

var (
	dialds        = make(map[string]*Diald)
	registrations = make(map[string]*Diald)
	appConnReqs   = make(map[string]net.Conn)
	m             sync.Mutex
)

type Diald struct {
	Names          []string
	ID             string
	AllowedPorts   map[int]bool
	AgentInfo      sysinfo.AgentInfo
	HostInfo       sysinfo.HostInfo
	MemInfo        sysinfo.MemInfo
	SwapInfo       sysinfo.SwapInfo
	ConnectedSince time.Time
	LastSeen       time.Time
	Healthy        bool
	Connections    int
	BytesIn        uint64
	BytesOut       uint64
	ch             chan ConnReq
	conn           net.Conn
}

type ConnReq struct {
	IP       net.IP
	Hostname string
	Port     int
	Conn     net.Conn
}

func (cr ConnReq) String() string {
	if len(cr.Hostname) > 0 {
		return fmt.Sprintf("ConnReq:%v", cr.Hostname)
	}
	return fmt.Sprintf("ConnReq:%v", cr.IP)
}

func (diald *Diald) String() string {
	return fmt.Sprintf("%v", diald.Names[0])
}

func (diald *Diald) Seen() {
	diald.LastSeen = time.Now()
}

func (diald *Diald) SetHealthy(healthy bool) {
	diald.Healthy = healthy
}

func NewRegistration(hostname string) (*Diald, error) {
	id, err := uuid.New()
	if err != nil {
		return nil, err
	}

	host := &Diald{
		Names: []string{hostname},
		ID:    id.String(),
	}
	idstr := id.String()

	m.Lock()
	defer m.Unlock()

	registrations[idstr] = host

	return host, nil
}

func Registration(id string) (*Diald, error) {
	m.Lock()
	defer m.Unlock()

	diald, ok := registrations[id]
	if !ok {
		return nil, fmt.Errorf("unknown registration: %v", id)
	}
	diald.ID = ""
	delete(registrations, id)

	return diald, nil
}

func CookieRegistration(cookie []byte) (*Diald, error) {
	prefix := fmt.Sprintf("%X", cookie)

	m.Lock()
	defer m.Unlock()

	for id, diald := range registrations {
		if strings.HasPrefix(id, prefix) {
			delete(registrations, diald.ID)
			diald.ID = ""
			return diald, nil
		}
	}

	return nil, fmt.Errorf("unknown cookie: %v", prefix)
}

func NewHost(names []string, conn net.Conn, allowedPorts []int,
	agentInfo sysinfo.AgentInfo, hostInfo sysinfo.HostInfo,
	memInfo sysinfo.MemInfo, swapInfo sysinfo.SwapInfo) {

	now := time.Now()

	diald := &Diald{
		Names:          names,
		ID:             names[0],
		AllowedPorts:   make(map[int]bool),
		AgentInfo:      agentInfo,
		HostInfo:       hostInfo,
		MemInfo:        memInfo,
		SwapInfo:       swapInfo,
		ConnectedSince: now,
		LastSeen:       now,
		ch:             make(chan ConnReq),
		conn:           conn,
	}

	for _, port := range allowedPorts {
		diald.AllowedPorts[port] = true
	}

	m.Lock()
	for _, name := range names {
		dialds[name] = diald
	}
	m.Unlock()

	diald.control()

	m.Lock()
	defer m.Unlock()

	for _, name := range names {
		d, ok := dialds[name]
		if !ok || d != diald {
			continue
		}
		delete(dialds, name)
	}
}

func LookupHost(ip net.IP, hostname string) (*Diald, error) {
	m.Lock()
	defer m.Unlock()

	if len(hostname) > 0 {
		diald, ok := dialds[hostname]
		if !ok {
			return nil, fmt.Errorf("hostname %v is not registered", hostname)
		}
		return diald, nil
	}

	return nil, fmt.Errorf("address %v is not registered", ip)
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

	err := socks5.Response(appConn, socks5.ReplySucceeded, nil, "", 0)
	if err != nil {
		log.Printf("SOCK5 reply error: %v", err)
		appConn.Close()
		conn.Close()
		return
	}

	control.Relay(appConn, conn)
	log.Printf("appcon %v closed", cookie)
}
