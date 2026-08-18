//go:build windows

package winproc

import (
	"fmt"
	"os"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	consoleOnce sync.Once
	consoleIn   *os.File
	consoleErr  error
)

// ConsoleStdin returns a console input handle for a child process, allocating
// a hidden console for this program the first time it is asked.
//
// srcds runs with -console, and that reads the console input buffer through
// GetNumberOfConsoleInputEvents. A pipe or NUL is not one, so the server dies
// on its first read with:
//
//	CTextConsoleWin32::GetLine: !GetNumberOfConsoleInputEvents
//	FATAL ERROR
//
// The launcher is linked for the windows subsystem and starts with no console
// at all, so there is nothing to inherit. AllocConsole makes one; hiding its
// window keeps it off the screen. The child inherits it and reads from it,
// while its output still comes back through the pipes the caller set.
func ConsoleStdin() (*os.File, error) {
	consoleOnce.Do(func() {
		kernel32 := windows.NewLazySystemDLL("kernel32.dll")
		getConsoleWindow := kernel32.NewProc("GetConsoleWindow")
		allocConsole := kernel32.NewProc("AllocConsole")

		if window, _, _ := getConsoleWindow.Call(); window == 0 {
			if ret, _, err := allocConsole.Call(); ret == 0 {
				consoleErr = fmt.Errorf("cannot allocate a console: %w", err)
				return
			}
		}
		if window, _, _ := getConsoleWindow.Call(); window != 0 {
			user32 := windows.NewLazySystemDLL("user32.dll")
			const swHide = 0
			user32.NewProc("ShowWindow").Call(window, swHide)
		}

		name, err := windows.UTF16PtrFromString("CONIN$")
		if err != nil {
			consoleErr = err
			return
		}
		security := &windows.SecurityAttributes{InheritHandle: 1}
		security.Length = uint32(unsafe.Sizeof(*security))
		handle, err := windows.CreateFile(name,
			windows.GENERIC_READ|windows.GENERIC_WRITE,
			windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
			security, windows.OPEN_EXISTING, 0, 0)
		if err != nil {
			consoleErr = fmt.Errorf("cannot open the console input: %w", err)
			return
		}
		consoleIn = os.NewFile(uintptr(handle), "CONIN$")
	})
	return consoleIn, consoleErr
}
