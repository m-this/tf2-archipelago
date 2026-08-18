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
// SteamCMD is a console program and the launcher is not, so Windows would give
// it a console of its own, which appears in front of the player. Its output
// still reaches the log view: that comes through the pipes either way.
//
// Not for the game server. srcds runs with -console and reads the console
// input buffer, so denying it a console kills it on the first read. It gets
// the hidden one this program allocates instead; see ConsoleStdin.
func HideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
}
