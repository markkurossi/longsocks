//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/markkurossi/longsocks"
	"github.com/markkurossi/longsocks/control"
	"github.com/markkurossi/longsocks/sysinfo"
)

func processControl(conn net.Conn, dialer *tls.Dialer, addr string) error {
	defer conn.Close()

	ctx := context.Background()
	agentInfo, err := sysinfo.Agent(ctx)
	if err != nil {
		return err
	}
	hostInfo, err := sysinfo.Host(ctx)
	if err != nil {
		return err
	}
	memInfo, err := sysinfo.Memory(ctx)
	if err != nil {
		return err
	}
	swapInfo, err := sysinfo.Swap(ctx)
	if err != nil {
		return err
	}

	err = control.Send(conn, control.CtrlCh{
		Version:      0,
		AllowedPorts: config.Diald.AllowedPorts,
		AgentInfo:    *agentInfo,
		HostInfo:     *hostInfo,
		MemInfo:      *memInfo,
		SwapInfo:     *swapInfo,
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
			reqTime := time.Now().UnixMicro()

			memInfo, _ := sysinfo.Memory(ctx)
			swapInfo, _ := sysinfo.Swap(ctx)

			respTime := time.Now().UnixMicro()
			elapsed := respTime - reqTime

			pong := control.Pong{
				Time:    uint64(reqTime + elapsed/2),
				Elapsed: uint64(elapsed),
			}
			if memInfo != nil {
				pong.MemInfo = *memInfo
			}
			if swapInfo != nil {
				pong.SwapInfo = *swapInfo
			}
			response = pong

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
