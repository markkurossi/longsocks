// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"encoding/hex"
	"log"
	"net"
)

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
	defer conn.Close()

	buf := make([]byte, 1024)

	for {
		n, err := conn.Read(buf)
		if err != nil {
			log.Print(err)
			return
		}
		log.Printf("%s", hex.Dump(buf[:n]))
	}
}
