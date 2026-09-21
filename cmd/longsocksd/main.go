//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"encoding/binary"
	"log"
	"os"
	"sync"

	"github.com/markkurossi/longsocks"
)

var (
	bo       = binary.BigEndian
	config   *longsocks.Config
	identity *longsocks.Identity
)

func main() {
	var err error

	config, err = longsocks.LoadConfig("longsocksd.toml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	identity, err = longsocks.LoadIdentity(config)
	if os.IsNotExist(err) {
		log.Printf("creating identity %v", config.Longsocksd.CertificateFile)
		identity, err = longsocks.CreateIdentity(config)
		if err != nil {
			log.Fatalf("failed to create identity: %v", err)
		}
	} else {
		log.Printf("loaded identity %v", config.Longsocksd.CertificateFile)
	}

	var wg sync.WaitGroup

	ipc, err := NewIPCListener(longsocks.IPCListener)
	if err != nil {
		log.Fatal(err)
	}
	wg.Go(ipc.Run)

	ctrl, err := NewControlListener(identity)
	if err != nil {
		log.Fatal(err)
	}
	wg.Go(ctrl.Run)

	wg.Wait()
}
