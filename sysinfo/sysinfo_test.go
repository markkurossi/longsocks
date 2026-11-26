//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package sysinfo

import (
	"context"
	"fmt"
	"testing"
)

func TestSysinfo(t *testing.T) {
	ctx := context.Background()

	h, err := Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("h: %#v\n", h)

	a, err := Agent(ctx)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("a: %#v\n", a)

	m, err := Memory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("m: %#v\n", m)

	s, err := Swap(ctx)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("s: %#v\n", s)
}
