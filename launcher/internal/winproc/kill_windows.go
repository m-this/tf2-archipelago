//go:build windows

package winproc

import (
	"fmt"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// KillUnder terminates every running process whose program lives under root,
// and reports what it killed.
//
// SteamCMD relaunches itself after it updates, so the process the launcher
// started is not the one still holding the files. Cancelling our own child
// leaves that second one running, and Windows then refuses to delete anything
// it has open. Only the install tree is searched, so an unrelated SteamCMD
// elsewhere on the machine is left alone.
func KillUnder(root string) ([]string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("cannot list the running processes: %w", err)
	}
	defer func() { _ = windows.CloseHandle(snapshot) }()

	var killed []string
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := windows.Process32First(snapshot, &entry); err != nil {
		return nil, err
	}
	self := windows.GetCurrentProcessId()
	for {
		if entry.ProcessID != self && entry.ProcessID != 0 {
			if path, ok := programPath(entry.ProcessID); ok && under(path, root) {
				if killProcess(entry.ProcessID) {
					killed = append(killed, path)
				}
			}
		}
		if err := windows.Process32Next(snapshot, &entry); err != nil {
			break
		}
	}
	return killed, nil
}

func programPath(pid uint32) (string, bool) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", false
	}
	defer func() { _ = windows.CloseHandle(handle) }()

	buf := make([]uint16, windows.MAX_LONG_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(handle, 0, &buf[0], &size); err != nil {
		return "", false
	}
	return windows.UTF16ToString(buf[:size]), true
}

func killProcess(pid uint32) bool {
	handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, pid)
	if err != nil {
		return false
	}
	defer func() { _ = windows.CloseHandle(handle) }()
	return windows.TerminateProcess(handle, 1) == nil
}

// under reports whether path sits inside root, comparing case-insensitively:
// Windows paths do.
func under(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}
