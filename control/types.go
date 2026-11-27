//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

// Package control defines the control protocol messages.
package control

import (
	"github.com/markkurossi/longsocks/sysinfo"
)

//go:generate stringer -type=MsgType -trimprefix=Msg

// MsgType defines control messages.
type MsgType byte

// Control messages.
const (
	MsgError MsgType = iota
	MsgSuccess
	MsgHostInit
	MsgHostInitResp
	MsgHostRemoteInit
	MsgHostRemoteInitResp
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

// HostInit defines a host init message.
type HostInit struct {
	CSR []byte
}

// HostInitResp defines a host init response.
type HostInitResp struct {
	Cert []byte
}

// HostRemoteInit defines a host remote init message.
type HostRemoteInit struct {
	CSR []byte
}

// HostRemoteInitResp defines a host remote init response.
type HostRemoteInitResp struct {
	Config []byte
	Cert   []byte
}

// CtrlCh defines a control connection message.
type CtrlCh struct {
	Version      uint8
	AllowedPorts []int
	AgentInfo    sysinfo.AgentInfo
	HostInfo     sysinfo.HostInfo
	MemInfo      sysinfo.MemInfo
	SwapInfo     sysinfo.SwapInfo
}

// AppCh defines an application control message.
type AppCh struct {
	Cookie string
}

// Ping defines a keepalive ping request.
type Ping struct {
	Time uint64
}

// Pong defines a ping response.
type Pong struct {
	Time     uint64
	Elapsed  uint64
	MemInfo  sysinfo.MemInfo
	SwapInfo sysinfo.SwapInfo
}

// ConnReq defines a connection request.
type ConnReq struct {
	IP       []byte
	Hostname string
	Port     uint16
	Cookie   string
}
