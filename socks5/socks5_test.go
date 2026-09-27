// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package socks5

import (
	"bytes"
	"net"
	"testing"
)

var responseTests = []struct {
	reply  Reply
	ip     net.IP
	addr   string
	port   int
	result []byte
}{
	{
		ReplySucceeded, nil, "", 22, []byte{
			0x05, byte(ReplySucceeded), 0x00,
			0x01, 0x00, 0x00, 0x00, 0x00,
			0x00, 22,
		},
	},
	{
		ReplySucceeded, nil, "", 8443, []byte{
			0x05, byte(ReplySucceeded), 0x00,
			0x01, 0x00, 0x00, 0x00, 0x00,
			0x20, 0xfb,
		},
	},
	{
		ReplySucceeded, net.IP([]byte{127, 0, 0, 1}), "", 8443, []byte{
			0x05, byte(ReplySucceeded), 0x00,
			0x01, 127, 0, 0, 1,
			0x20, 0xfb,
		},
	},
	{
		ReplySucceeded, nil, "iki.fi", 8443, []byte{
			0x05, byte(ReplySucceeded), 0x00,
			0x03, 0x06, 'i', 'k', 'i', '.', 'f', 'i',
			0x20, 0xfb,
		},
	},
	{
		ReplySucceeded, net.IP([]byte{
			0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 1}), "", 8443, []byte{
			0x05, byte(ReplySucceeded), 0x00,
			0x04, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1,
			0x20, 0xfb,
		},
	},
}

func TestResponse(t *testing.T) {
	for idx, test := range responseTests {
		var w bytes.Buffer

		err := Response(&w, test.reply, test.ip, test.addr, test.port)
		if err != nil {
			t.Errorf("test-%d: Response: %v", idx, err)
			continue
		}
		if !bytes.Equal(w.Bytes(), test.result) {
			t.Errorf("test-%d: got %x, expected %x", idx, w.Bytes(),
				test.result)
		}
	}
}
