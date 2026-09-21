//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package longsocks

import (
	"path/filepath"
)

const (
	EtcDir = "/usr/local/etc/longsocks.d"

	IPCListener = "/tmp/longsocks.sock"
)

func ConfigFile(file string) string {
	return filepath.Join(EtcDir, file)
}
