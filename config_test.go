//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package longsocks

import (
	"testing"
)

var config = `
[longsocksd]

hostname = "nap.ephemelier.com"
port = 8103

certificate_file = "/usr/local/etc/longsocks.d/proxy.crt"
private_key_file = "/usr/local/etc/longsocks.d/proxy.key"

[longsocksd.certificate]

country = ["FI"]
organization = ["Ephemelier"]
common_name = "Longsocks"

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
	_ = cfg
}
