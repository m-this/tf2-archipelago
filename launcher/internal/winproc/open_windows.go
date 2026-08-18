//go:build windows

package winproc

import (
	"context"
	"fmt"
	"os/exec"

	"golang.org/x/sys/windows"
)

// Open shows a file or a folder to the player, in whatever program Windows
// uses for it.
//
// A .yaml has no program associated with it on a fresh Windows, and
// ShellExecute fails rather than asking. Notepad opens it in that case, which
// is what a player wants from a button called "open".
func Open(path string) error {
	verb, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	target, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	const swShowNormal = 1
	if err := windows.ShellExecute(0, verb, target, nil, nil, swShowNormal); err == nil {
		return nil
	}
	if err := exec.CommandContext(context.Background(), "notepad.exe", path).Start(); err != nil {
		return fmt.Errorf("cannot open %s: %w", path, err)
	}
	return nil
}
