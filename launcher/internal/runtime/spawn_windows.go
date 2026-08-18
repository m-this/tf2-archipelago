//go:build windows

package runtime

import (
	"os/exec"
	"syscall"
)

// createNoWindow is CREATE_NO_WINDOW: the child runs with no console of its
// own.
const createNoWindow = 0x08000000

// hideConsole keeps the game server from opening a console window.
//
// srcds.exe is a console program and the launcher is not, so Windows gives the
// child a console of its own. That console steals the foreground, and it
// leaves this window half laid out: the toolbar and the log keep their old
// size while the rest of the window resizes around them. The output still
// reaches the log view, because that comes through the pipes either way.
func hideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
}
