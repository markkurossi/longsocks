//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

// Package control defines the control protocol messages.
package control

//go:generate stringer -type=MsgType -trimprefix=Msg

// MsgType defines control messages.
type MsgType byte

// Control messages.
const (
	MsgError MsgType = iota
	MsgSuccess
	MsgHostInit
	MsgHostInitResp
	MsgCtrlCh
	MsgAppCh
	MsgPing
	MsgPong
	MsgConnReq
)

// Error defines error message.
type Error struct {
	Error string
}

// Success defines a success message.
type Success struct {
}

// HostInit defines host init message.
type HostInit struct {
	CSR []byte
}

// HostInitResp defines host init response.
type HostInitResp struct {
	Cert []byte
}

// CtrlCh defines control connection message.
type CtrlCh struct {
	Msg string
}

type AppCh struct {
	Cookie string
}

// Ping defines keepalive ping request.
type Ping struct {
	Time uint64
}

// Pong defines ping response.
type Pong struct {
	Time uint64
}

type ConnReq struct {
	IP       []byte
	Hostname string
	Port     uint16
	Cookie   string
}
