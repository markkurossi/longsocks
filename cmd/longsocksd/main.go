//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"log"
	"os"

	"github.com/markkurossi/longsocks"
)

func main() {
	cfg, err := longsocks.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	identity, err := longsocks.LoadIdentity(cfg)
	if os.IsNotExist(err) {
		log.Printf("creating identity %v", cfg.Longsocksd.CertificateFile)
		identity, err = longsocks.CreateIdentity(cfg)
		if err != nil {
			log.Fatalf("failed to create identity: %v", err)
		}
	} else {
		log.Printf("loaded identity %v", cfg.Longsocksd.CertificateFile)
	}
	_ = identity
}
