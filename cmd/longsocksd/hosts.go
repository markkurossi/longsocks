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
)

var (
	hosts         = make(map[string]*Host)
	registrations = make(map[string]*Host)
	m             sync.Mutex
)

type Host struct {
	Names []string
	ID    uuid.UUID
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
	idstr := strings.ToUpper(id.String())

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
	}

	m.Lock()
	defer m.Unlock()

	for _, name := range names {
		hosts[name] = host
	}

	go host.control()
}

func (host *Host) control() {
	log.Printf("control handler for host %v", host)
}
