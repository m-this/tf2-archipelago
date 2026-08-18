//go:build windows

package ui

import (
	"os"
	"syscall"
)

const attachParentProcess = ^uint32(0) // ATTACH_PARENT_PROCESS

// AttachConsole reattaches the standard streams to the terminal that started
// the exe, and reports whether there was one.
//
// The exe is linked for the windows subsystem, so double-clicking it opens the
// window and no console. That also means a run from cmd.exe has nowhere to
// print, which would make every flag look broken. Attaching to the parent's
// console gives those flags their output back.
func AttachConsole() bool {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	if ret, _, _ := kernel32.NewProc("AttachConsole").Call(uintptr(attachParentProcess)); ret == 0 {
		return false
	}
	for _, bind := range []struct {
		name   string
		stream **os.File
	}{
		{"CONOUT$", &os.Stdout},
		{"CONOUT$", &os.Stderr},
		{"CONIN$", &os.Stdin},
	} {
		file, err := os.OpenFile(bind.name, os.O_RDWR, 0)
		if err != nil {
			continue
		}
		*bind.stream = file
	}
	return true
}
