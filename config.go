//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

// Package longsocks implements a SOCKS5 reverse proxy that provides
// access to private machines through long-lived outbound tunnels.
package longsocks

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Longsocksd struct {
		Hostname        string
		Port            int
		CertificateFile string `toml:"certificate_file"`
		PrivateKeyFile  string `toml:"private_key_file"`
		Certificate     struct {
			Country      []string `toml:"country"`
			Organization []string `toml:"organization"`
			CommonName   string   `toml:"common_name"`
		} `toml:"certificate"`
	}
	Socks struct {
		Listen string
	}
	Http struct {
		Listen string
	}
	Diald struct {
		AllowedPorts []int `toml:"allowed_ports"`
	}
}

func Parse(data []byte) (*Config, error) {
	var cfg Config

	decoder := toml.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	err := decoder.Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	return &cfg, nil
}

func LoadConfig() (*Config, error) {
	f, err := os.Open(ConfigFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return Parse(data)
}
