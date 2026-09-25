// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"text/template"

	"github.com/markkurossi/go-libs/uuid"
	"github.com/markkurossi/longsocks"
)

type IPC struct {
	listener net.Listener
}

func NewIPCListener(path string) (*IPC, error) {
	os.RemoveAll(path)
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}

	return &IPC{
		listener: listener,
	}, nil
}

func (ipc *IPC) Run() {
	for {
		conn, err := ipc.listener.Accept()
		if err != nil {
			log.Printf("IPC accept: %v", err)
			continue
		}

		c := &Connection{
			conn: conn,
		}

		go c.msgLoop()
	}
}

type Connection struct {
	conn net.Conn
}

func (c *Connection) msgLoop() {
	err := c.processMessages()
	if err != nil {
		log.Printf("IPC: %v", err)
	}
	c.conn.Close()
}

func (c *Connection) processMessages() error {
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
	dialdTmp = template.Must(template.New("diald.conf").Parse(dialdConfig))

	hostInitCmd  = "diald init {{.ID}}"
	hostInitTmpl = template.Must(template.New("diald.init").Parse(hostInitCmd))
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
		var values = struct {
			Hostname           string
			LongsocksdHostname string
			LongsocksdPort     int
			CACertificate      string
			ID                 string
		}{
			Hostname:           fmt.Sprintf("%v", host.Names),
			LongsocksdHostname: config.Longsocksd.Hostname,
			LongsocksdPort:     config.Longsocksd.Port,
			ID:                 host.ID.String(),
			CACertificate:      string(identity.CertPEM()),
		}

		var config bytes.Buffer
		err = dialdTmp.Execute(&config, values)
		if err != nil {
			return nil, err
		}

		var cmd bytes.Buffer
		err = hostInitTmpl.Execute(&cmd, values)
		if err != nil {
			return nil, err
		}

		return []string{config.String(), cmd.String()}, nil

	case "ls", "list":
		var result []string
		for k, v := range hosts {
			var id string
			if uuid.Nil.Compare(v.ID) != 0 {
				id = v.ID.String()
			}
			value := fmt.Sprintf("%v|%v", k, id)
			result = append(result, value)
		}
		return result, nil

	default:
		return nil, fmt.Errorf("unknown command: %v", args[0])
	}
}
