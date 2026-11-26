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
	EtcDir    = "/etc/longsocks"
	VarLibDir = "/var/lib/longsocks"
	RunDir    = "/run/longsocks"

	IPCListener = RunDir + "/longsocks.sock"

	PrivateKeyName       = "proxy.key"
	CertificateName      = "proxy.crt"
	DialdPrivateKeyName  = "diald.key"
	DialdCertificateName = "diald.crt"
)

type ConfigDirs struct {
	Etc    string
	VarLib string
	Run    string
}

var Dirs = &ConfigDirs{
	Etc:    EtcDir,
	VarLib: VarLibDir,
	Run:    RunDir,
}

func ConfigFile(file string) string {
	return filepath.Join(Dirs.Etc, file)
}
