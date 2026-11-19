// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"text/template"

	"github.com/markkurossi/longsocks"
)

// IPC defines the local control IPC server.
type IPC struct {
	listener net.Listener
}

// NewIPCServer creates a new IPC server
func NewIPCServer(path string) (*IPC, error) {
	os.RemoveAll(path)
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}

	return &IPC{
		listener: listener,
	}, nil
}

// Run processes the control IPC connections.
func (ipc *IPC) Run() {
	for {
		conn, err := ipc.listener.Accept()
		if err != nil {
			log.Printf("IPC accept: %v", err)
			continue
		}

		c := &IPCConnection{
			conn: conn,
		}

		go c.msgLoop()
	}
}

// IPCConnection implements a control IPC connection.
type IPCConnection struct {
	conn net.Conn
}

func (c *IPCConnection) msgLoop() {
	err := c.processMessages()
	if err != nil {
		log.Printf("IPC: %v", err)
	}
	c.conn.Close()
}

func (c *IPCConnection) processMessages() error {
	var buf [4]byte

	var data []byte

	for {
		_, err := c.conn.Read(buf[:4])
		if err != nil {
			if err != io.EOF {
				return err
			}
			return nil
		}
		l := int(bo.Uint32(buf[:4]))
		if l > len(data) {
			data = make([]byte, l)
		}
		_, err = c.conn.Read(data[:l])
		if err != nil {
			return err
		}

		var args []string
		n, err := longsocks.UnmarshalFrom(data[:l], &args)
		if err != nil {
			return err
		}
		if n != l {
			return fmt.Errorf("IPC stream out of sync: %v vs. %v", n, l)
		}
		log.Printf("args: %v", args)

		var responseValues []string

		result, err := processCommand(args)
		if err != nil {
			responseValues = []string{"ERROR", err.Error()}
		} else {
			responseValues = append(responseValues, "OK")
			responseValues = append(responseValues, result...)
		}

		response, err := longsocks.Marshal(responseValues)
		if err != nil {
			return err
		}
		longsocks.BO.PutUint32(buf[:], uint32(len(response)))
		_, err = c.conn.Write(buf[:])
		if err != nil {
			return err
		}
		_, err = c.conn.Write(response)
		if err != nil {
			return err
		}
	}
}

var (
	dialdConfig = `# Diald config for {{.Hostname}}

[longsocksd]

hostname = "{{.LongsocksdHostname}}"
port = {{.LongsocksdPort}}

[diald]

ca = """
{{.CACertificate}}"""

certificate_file = "/usr/local/etc/longsocks.d/diald.crt"
private_key_file = "/usr/local/etc/longsocks.d/diald.key"
`
	dialdTmpl = template.Must(template.New("diald.conf").Parse(dialdConfig))

	hostInitCmd  = `diald init {{.ID}}`
	hostInitTmpl = template.Must(template.New("diald.init").Parse(hostInitCmd))

	hostRemoteInitCmd  = `diald remote-init {{.LongsocksdHostname}} {{.RemoteID}} {{.CertFingerprint}}`
	hostRemoteInitTmpl = template.Must(template.New("diald.remote.init").
				Parse(hostRemoteInitCmd))
)

func processCommand(args []string) ([]string, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no command")
	}
	switch args[0] {
	case "host-init":
		if len(args) != 2 {
			return nil, fmt.Errorf("usage: %v hostname", args[0])
		}
		host, err := NewRegistration(args[1])
		if err != nil {
			return nil, err
		}

		id := host.ID
		var remoteID string

		idParts := strings.Split(id, "-")
		if len(idParts) == 5 {
			remoteID = idParts[0]
		} else {
			remoteID = id
		}

		// Compute CA certificate fingerprint.

		key, err := longsocks.ParseHex(idParts[:1])
		if err != nil {
			return nil, fmt.Errorf("invalid registration ID: %w", err)
		}
		mac := hmac.New(sha256.New, key)
		mac.Write(identity.Cert.Raw)
		hmac := mac.Sum(nil)

		fingerprint := fmt.Sprintf("%X", hmac[:4])

		var values = struct {
			Hostname           string
			LongsocksdHostname string
			LongsocksdPort     int
			CACertificate      string
			ID                 string
			RemoteID           string
			CertFingerprint    string
		}{
			ID:                 id,
			LongsocksdHostname: config.Longsocksd.Hostname,
			RemoteID:           remoteID,
			CertFingerprint:    fingerprint,
		}

		config, err := makeDialdConfig(host)
		if err != nil {
			return nil, err
		}

		var cmd bytes.Buffer
		err = hostInitTmpl.Execute(&cmd, values)
		if err != nil {
			return nil, err
		}

		var cmd2 bytes.Buffer
		err = hostRemoteInitTmpl.Execute(&cmd2, values)
		if err != nil {
			return nil, err
		}

		return []string{
			"host-init",
			string(config),
			cmd.String(),
			cmd2.String(),
		}, nil

	case "ls", "list":
		result := []string{"list"}
		for k, v := range hosts {
			var id string
			if len(v.ID) != 0 {
				id = v.ID
			}
			value := fmt.Sprintf("%v|%v", k, id)
			result = append(result, value)
		}

		for k, v := range registrations {
			value := fmt.Sprintf("%v|%v", k, strings.Join(v.Names, ","))
			result = append(result, value)
		}

		return result, nil

	default:
		return nil, fmt.Errorf("unknown command: %v", args[0])
	}
}

func makeDialdConfig(diald *Diald) ([]byte, error) {
	var values = struct {
		Hostname           string
		LongsocksdHostname string
		LongsocksdPort     int
		CACertificate      string
	}{
		Hostname:           fmt.Sprintf("%v", diald.Names),
		LongsocksdHostname: config.Longsocksd.Hostname,
		LongsocksdPort:     config.Longsocksd.Port,
		CACertificate:      string(identity.CertPEM()),
	}

	var config bytes.Buffer
	err := dialdTmpl.Execute(&config, values)
	if err != nil {
		return nil, err
	}

	return config.Bytes(), nil
}
