//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/markkurossi/longsocks"
	"github.com/markkurossi/longsocks/control"
)

func processControl(conn net.Conn) error {
	defer conn.Close()

	var hdr [5]byte
	data, err := longsocks.Marshal(control.CtrlCh{
		Msg: "Hello, longsocksd!",
	})
	if err != nil {
		return err
	}
	hdr[0] = byte(control.MsgCtrlCh)
	bo.PutUint32(hdr[1:], uint32(len(data)))
	_, err = conn.Write(hdr[:])
	if err != nil {
		return err
	}
	_, err = conn.Write(data)
	if err != nil {
		return err
	}

	for {
		_, err := conn.Read(hdr[:])
		if err != nil {
			return err
		}
		msgType := control.MsgType(hdr[0])
		l := int(bo.Uint32(hdr[1:]))

		if l > len(data) {
			data = make([]byte, l)
		}
		_, err = conn.Read(data[:l])
		if err != nil {
			return err
		}
		log.Printf("%v:\n%s", msgType, hex.Dump(data[:l]))

		var response interface{}

		switch msgType {
		case control.MsgPing:
			response = control.Pong{
				Time: uint64(time.Now().UnixMicro()),
			}
		default:
			response = control.Error{
				Error: fmt.Sprintf("unknown message %v", msgType),
			}
		}
		err = control.Send(conn, response)
		if err != nil {
			return err
		}
	}
}
