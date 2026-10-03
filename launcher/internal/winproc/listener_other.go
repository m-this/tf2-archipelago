//go:build !windows

package winproc

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// listenState is TCP_LISTEN as /proc/net/tcp writes it.
const listenState = "0A"

// ListenerPID returns the process listening on this TCP port: the socket's
// inode out of /proc/net/tcp, then the process holding that inode open. Where
// there is no /proc, nothing is found.
func ListenerPID(port int) (int, bool) {
	inode := ""
	for _, table := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		if inode = listeningInode(table, port); inode != "" {
			break
		}
	}
	if inode == "" {
		return 0, false
	}
	want := "socket:[" + inode + "]"
	fds, _ := filepath.Glob("/proc/[0-9]*/fd/*")
	for _, fd := range fds {
		if link, err := os.Readlink(fd); err == nil && link == want {
			pid, err := strconv.Atoi(strings.Split(fd, "/")[2])
			return pid, err == nil
		}
	}
	return 0, false
}

func listeningInode(table string, port int) string {
	data, err := os.ReadFile(table)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n")[1:] {
		fields := strings.Fields(line)
		if len(fields) < 10 || fields[3] != listenState {
			continue
		}
		_, hexPort, ok := strings.Cut(fields[1], ":")
		if got, err := strconv.ParseUint(hexPort, 16, 16); ok && err == nil && int(got) == port {
			return fields[9]
		}
	}
	return ""
}
