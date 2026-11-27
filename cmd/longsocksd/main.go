//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"encoding/binary"
	"flag"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/markkurossi/longsocks"
)

var (
	bo       = binary.BigEndian
	config   *longsocks.Config
	identity *longsocks.Identity

	version   = "dev"
	gitCommit = "unknown"
)

func main() {
	log.SetFlags(0)

	cfgDir := flag.String("config", longsocks.EtcDir, "config file directory")
	stateDir := flag.String("state-dir", longsocks.VarLibDir,
		"directory for persistent state")
	socket := flag.String("socket", longsocks.IPCListener,
		"control socket path")
	flag.Parse()

	longsocks.Dirs.Etc = *cfgDir

	var newStateDir bool
	if *stateDir != longsocks.VarLibDir {
		newStateDir = true
		longsocks.Dirs.VarLib = *stateDir
	}

	var err error

	config, err = longsocks.LoadConfig("longsocksd.toml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	if newStateDir {
		config.Longsocksd.PrivateKeyFile =
			filepath.Join(*stateDir, longsocks.PrivateKeyName)
		config.Longsocksd.CertificateFile =
			filepath.Join(*stateDir, longsocks.CertificateName)
	}

	identity, err = longsocks.LoadIdentity(config.Longsocksd.PrivateKeyFile,
		config.Longsocksd.CertificateFile)
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

	ipc, err := NewIPCServer(*socket)
	if err != nil {
		log.Fatal(err)
	}
	wg.Go(ipc.Run)

	ctrl, err := NewControlListener(identity)
	if err != nil {
		log.Fatal(err)
	}
	wg.Go(ctrl.Run)

	socks, err := NewSocksServer()
	if err != nil {
		log.Fatal(err)
	}
	wg.Go(socks.Run)

	if len(config.Longsocksd.HTTP.Listen) > 0 {
		httpd, err := NewHTTPD()
		if err != nil {
			log.Fatal(err)
		}
		wg.Go(httpd.Run)
	}

	wg.Wait()
}
