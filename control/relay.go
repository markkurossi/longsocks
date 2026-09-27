//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package control

import (
	"io"
	"net"
	"sync"
)

// CloseWriter defines interface with CloseWrite function.
type CloseWriter interface {
	CloseWrite() error
}

// Relay connects connections c1 and c2 bidirectionally. It trigger
// half-close on both directions if the connections implement
// CloseWriter. The connections are closed once both directions have
// been closed.
func Relay(c1, c2 net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	// Stream c1 -> c2.
	go func() {
		defer wg.Done()
		io.Copy(c2, c1)

		// Signal to c2 that no more data is coming from c1.
		if tc, ok := c2.(CloseWriter); ok {
			tc.CloseWrite()
		}
	}()

	// Stream c2 -> c1.
	go func() {
		defer wg.Done()
		io.Copy(c1, c2)

		// Signal to c1 that no more data is coming from c2.
		if tc, ok := c1.(CloseWriter); ok {
			tc.CloseWrite()
		}
	}()

	wg.Wait()

	c1.Close()
	c2.Close()
}
