//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package control

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/markkurossi/longsocks"
)

var bo = binary.BigEndian

func RPC(conn net.Conn, req interface{}) (MsgType, []byte, error) {
	err := Send(conn, req)
	if err != nil {
		return MsgError, nil, err
	}

	return Recv(conn)
}

func Send(conn net.Conn, req interface{}) error {
	data, err := longsocks.Marshal(req)
	if err != nil {
		return err
	}

	var msgType MsgType
	switch req.(type) {
	case Error:
		msgType = MsgError
	case HostInit:
		msgType = MsgHostInit
	case Ping:
		msgType = MsgPing
	case Pong:
		msgType = MsgPong
	default:
		return fmt.Errorf("unknown request [%T]", req)
	}

	var hdr [5]byte
	hdr[0] = byte(msgType)
	bo.PutUint32(hdr[1:], uint32(len(data)))

	_, err = conn.Write(hdr[:])
	if err != nil {
		return err
	}
	_, err = conn.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func Recv(conn net.Conn) (MsgType, []byte, error) {

	var hdr [5]byte

	_, err := conn.Read(hdr[:])
	if err != nil {
		return MsgError, nil, err
	}
	l := bo.Uint32(hdr[1:])
	data := make([]byte, l)
	_, err = conn.Read(data)

	return MsgType(hdr[0]), data, err
}
