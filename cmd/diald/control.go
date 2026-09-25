//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/markkurossi/longsocks"
	"github.com/markkurossi/longsocks/control"
)

func processControl(conn net.Conn, dialer *tls.Dialer, addr string) error {
	defer conn.Close()

	err := control.Send(conn, control.CtrlCh{
		Msg: "Hello, longsocksd!",
	})
	if err != nil {
		return err
	}

	var hdr [5]byte
	var data []byte

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

		var response interface{}

		switch msgType {
		case control.MsgPing:
			response = control.Pong{
				Time: uint64(time.Now().UnixMicro()),
			}

		case control.MsgConnReq:
			var msg control.ConnReq
			_, err = longsocks.UnmarshalFrom(data[:l], &msg)
			if err != nil {
				return err
			}

			err = appConnection(msg, dialer, addr)
			if err != nil {
				response = control.Error{
					Error: err.Error(),
				}
			} else {
				response = control.Success{}
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

func appConnection(req control.ConnReq, dialer *tls.Dialer, addr string) error {
	target := fmt.Sprintf(":%v", req.Port)
	client, err := net.Dial("tcp", target)
	if err != nil {
		return err
	}

	server, err := dialer.Dial("tcp", addr)
	if err != nil {
		client.Close()
		return err
	}

	log.Printf("app connection to %v", target)

	err = control.Send(server, control.AppCh{
		Cookie: req.Cookie,
	})
	if err != nil {
		client.Close()
		server.Close()
		return err
	}

	go control.Relay(client, server)

	return nil
}
