//
// Copyright (c) 2025-2026 Markku Rossi
//
// All rights reserved.
//

package longsocks

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"testing"
	"time"
)

type physicalID uint64
type logicalID uint64

const (
	rootPtrMagic = uint64(0x7b5368616465737d)
	RootPtrSize  = 112
)

type rootPointer struct {
	Magic        uint64
	Flags        uint16
	Depth        uint16
	PageSize     uint32
	Timestamp    uint64
	Generation   uint64
	NextPhysical uint64
	NextLogical  uint64
	PageTable    physicalID
	Freelist     physicalID
	Snapshots    physicalID
	BlobFreelist logicalID
	BsobFreelist logicalID
	UserData     uint64
	Checksum     [16]byte
}

func TestMarshalValues(t *testing.T) {
	rp := rootPointer{
		Magic:        rootPtrMagic,
		Depth:        1,
		PageSize:     4096,
		Timestamp:    uint64(time.Now().UnixNano()),
		Generation:   1,
		NextPhysical: 2,
		NextLogical:  3,
	}

	var buf [1024]byte

	n, err := MarshalTo(buf[:], &rp)
	if err != nil {
		t.Fatal(err)
	}
	if n != RootPtrSize {
		t.Errorf("marshalled %v, expected %v", n, RootPtrSize)
	}

	var rp2 rootPointer

	n2, err := UnmarshalFrom(buf[:], &rp2)
	if err != nil {
		t.Fatal(err)
	}
	if n2 != n {
		t.Errorf("unmarshalled %v, expected %v", n2, n)
	}

	if !reflect.DeepEqual(&rp, &rp2) {
		t.Errorf("rp != rp2")
	}
}

func TestMarshalArray(t *testing.T) {
	values := []string{"a", "bc", "def"}
	var buf [1024]byte

	n, err := MarshalTo(buf[:], values)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("marshal:\n%s", hex.Dump(buf[:n]))

	var v2 []string
	n2, err := UnmarshalFrom(buf[:], &v2)
	if err != nil {
		t.Fatal(err)
	}
	if n2 != n {
		t.Errorf("n2=%v != n=%v", n2, n)
	}
	if len(values) != len(v2) {
		t.Fatalf("len(values)=%v != len(v2)=%v", len(values), len(v2))
	}
	for idx, v := range values {
		if v2[idx] != v {
			t.Errorf("v2[%v]=%v != values[%v]=%v", idx, v2[idx], idx, v)
		}
	}
}
