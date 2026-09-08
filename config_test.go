//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package longsocks

import (
	"fmt"
	"testing"
)

var config = `
[longsocksd]

hostname = "nap.ephemelier.com"
port = 8103

certificate = """
-----BEGIN CERTIFICATE-----
...
-----END CERTIFICATE-----
"""

private_key_file = "/usr/local/etc/longsocks/proxy.key"

[socks]

listen = ":1080"

[http]

listen = ":8443"

[diald]

allowed_ports = [22, 80, 443, 8443]
`

func TestConfig(t *testing.T) {
	cfg, err := Parse([]byte(config))
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("cfg: %#v\n", cfg)
}
