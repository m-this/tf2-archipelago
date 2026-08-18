//go:build !windows

// Package winproc holds the one thing the launcher needs from Windows process
// creation. Away from Windows there is nothing to do.
package winproc

import "os/exec"

// HideConsole does nothing here: no console is allocated.
func HideConsole(*exec.Cmd) {}
