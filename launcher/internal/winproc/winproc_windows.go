//go:build windows

// Package winproc holds the one thing the launcher needs from Windows process
// creation: starting a console program without giving it a console.
package winproc

import (
	"os/exec"
	"syscall"
)

// createNoWindow is CREATE_NO_WINDOW.
const createNoWindow = 0x08000000

// HideConsole keeps a console program from opening a console window.
//
// The game server and SteamCMD are console programs and the launcher is not,
// so Windows gives each of them a console of its own. That console steals the
// foreground, and it leaves the launcher's window half laid out: the toolbar
// and the log keep their old size while the rest resizes around them. Their
// output still reaches the log view, because that comes through the pipes
// either way.
func HideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
}
