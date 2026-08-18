//go:build windows

package winproc

import "golang.org/x/sys/windows"

// FreeBytes reports the free space available on the volume that holds path,
// and whether Windows would say.
func FreeBytes(path string) (uint64, bool) {
	wide, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, false
	}
	var free, total, totalFree uint64
	if err := windows.GetDiskFreeSpaceEx(wide, &free, &total, &totalFree); err != nil {
		return 0, false
	}
	return free, true
}
