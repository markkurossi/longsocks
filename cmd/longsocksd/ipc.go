// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"

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
		fmt.Printf("args: %v\n", args)

		var responseValues []string
		if len(args) == 0 {
			responseValues = []string{"ERROR", "no command"}
		} else {
			switch args[0] {
			default:
				responseValues = []string{"ERROR", "unknown command", args[0]}
			}
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
