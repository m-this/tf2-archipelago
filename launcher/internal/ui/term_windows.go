//go:build windows

package ui

import (
	"bufio"

	"golang.org/x/sys/windows"
)

// termReadLine reads a line from the console without echoing it, for password
// entry. Windows-only: the launcher's primary target.
func termReadLine(r *bufio.Reader) (string, error) {
	handle := windows.Handle(windows.Stdin)
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return "", err
	}
	if err := windows.SetConsoleMode(handle, mode&^windows.ENABLE_ECHO_INPUT); err != nil {
		return "", err
	}
	defer windows.SetConsoleMode(handle, mode)
	return r.ReadString('\n')
}
