//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package control

//go:generate stringer -type=MsgType -trimprefix=Msg

type MsgType byte

const (
	MsgError MsgType = iota
	MsgOk
	MsgHostInit
	MsgHostInitResp
)

type HostInit struct {
	CSR []byte
}

type HostInitResp struct {
	Cert []byte
}
