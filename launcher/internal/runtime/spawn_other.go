//go:build !windows

package runtime

import "os/exec"

// hideConsole does nothing away from Windows: no console is allocated there.
func hideConsole(*exec.Cmd) {}
