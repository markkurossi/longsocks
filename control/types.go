//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

// Package control defines the control protocol messages.
package control

//go:generate stringer -type=MsgType -trimprefix=Msg

type MsgType byte

const (
	MsgError MsgType = iota
	MsgOk
	MsgHostInit
	MsgHostInitResp
	MsgCtrlCh
	MsgAppCh
	MsgPing
	MsgPong
)

type Error struct {
	Error string
}

type HostInit struct {
	CSR []byte
}

type HostInitResp struct {
	Cert []byte
}

type CtrlCh struct {
	Msg string
}

type Ping struct {
	Time uint64
}

type Pong struct {
	Time uint64
}
