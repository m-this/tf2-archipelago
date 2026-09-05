//go:build windows

package tailscalefastdl

import (
	"os"
	"path/filepath"
)

// installedExecutablePath covers the standard machine-wide Windows install
// when the Tailscale directory was not added to this process's PATH.
func installedExecutablePath() string {
	for _, name := range []string{"ProgramW6432", "ProgramFiles"} {
		root, _ := os.LookupEnv(name)
		if root == "" {
			continue
		}
		path := filepath.Join(root, "Tailscale", "tailscale.exe")
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}
