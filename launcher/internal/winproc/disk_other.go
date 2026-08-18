//go:build !windows

package winproc

import "golang.org/x/sys/unix"

// FreeBytes reports the free space available on the file system that holds
// path.
func FreeBytes(path string) (uint64, bool) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, false
	}
	return stat.Bavail * uint64(stat.Bsize), true
}
