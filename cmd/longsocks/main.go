//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/markkurossi/longsocks"
)

func main() {
	daemon := flag.Bool("d", false, "daemon control")
	flag.Parse()

	if *daemon {
		err := daemonControl(flag.Args())
		if err != nil {
			log.Fatal(err)
		}
	}
}

func daemonControl(args []string) error {
	conn, err := net.Dial("unix", longsocks.IPCListener)
	if err != nil {
		return err
	}
	req, err := longsocks.Marshal(args)
	if err != nil {
		return err
	}
	var buf [4]byte

	longsocks.BO.PutUint32(buf[:], uint32(len(req)))
	_, err = conn.Write(buf[:])
	if err != nil {
		return err
	}
	_, err = conn.Write(req)
	if err != nil {
		return err
	}

	_, err = conn.Read(buf[:])
	if err != nil {
		return err
	}
	l := int(longsocks.BO.Uint32(buf[:]))
	resp := make([]byte, l)
	_, err = conn.Read(resp)
	if err != nil {
		return err
	}

	var values []string
	n, err := longsocks.UnmarshalFrom(resp, &values)
	if err != nil {
		return err
	}
	if n != l {
		return fmt.Errorf("unmarshal: %v vs. %v", n, l)
	}
	if len(values) < 2 {
		return fmt.Errorf("unexpected result: %v", values)
	}
	switch values[0] {
	case "OK":
		fmt.Printf(`
Save this configuration file to host at /usr/local/etc/longsocksd/diald.toml:

----------------------------------------------------------------------
%s----------------------------------------------------------------------
`, values[1])

		fmt.Printf(`
Run this command at host to init diald:

----------------------------------------------------------------------
%s
----------------------------------------------------------------------
`,
			values[2])

	default:
		return fmt.Errorf("%v", values)
	}

	return nil
}
