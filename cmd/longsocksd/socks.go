//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"log"
	"net"

	"github.com/markkurossi/longsocks/socks5"
)

// NewSocksServer creates a new SOCKS5 server.
func NewSocksServer() (*socks5.Server, error) {
	return socks5.NewServer(config.Longsocksd.Socks.Listen, socksHandler)
}

func socksHandler(conn net.Conn, cmd socks5.Command, ip net.IP, hostname string,
	port int) error {

	log.Printf("%v: ip=%v, hostname=%v, port=%d", cmd, ip, hostname, port)

	host, err := LookupHost(ip, hostname)
	if err != nil {
		return err
	}

	log.Printf("host: %v", host)

	host.ch <- ConnReq{
		IP:       ip,
		Hostname: hostname,
		Port:     port,
		Conn:     conn,
	}
	return nil
}
