// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"fmt"
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
	Name string
	ID   uuid.UUID
}

func NewRegistration(hostname string) (*Host, error) {
	id, err := uuid.New()
	if err != nil {
		return nil, err
	}

	host := &Host{
		Name: strings.ToLower(hostname),
		ID:   id,
	}
	idstr := strings.ToUpper(id.String())

	m.Lock()
	defer m.Unlock()

	registrations[idstr] = host

	hosts[host.Name] = host

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
