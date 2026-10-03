//go:build windows

package winproc

import (
	"encoding/binary"
	"unsafe"

	"golang.org/x/sys/windows"
)

var getExtendedTCPTable = windows.NewLazySystemDLL("iphlpapi.dll").NewProc("GetExtendedTcpTable")

// tcpTableOwnerPIDListener is TCP_TABLE_OWNER_PID_LISTENER.
const tcpTableOwnerPIDListener = 3

// ListenerPID returns the process listening on this TCP port over IPv4, read
// off the table netstat -ano prints.
func ListenerPID(port int) (int, bool) {
	var size uint32
	for range 3 {
		buf := make([]byte, max(size, 4))
		code, _, _ := getExtendedTCPTable.Call(
			uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)),
			0, windows.AF_INET, tcpTableOwnerPIDListener, 0)
		if code == uintptr(windows.ERROR_INSUFFICIENT_BUFFER) {
			continue
		}
		if code != 0 {
			return 0, false
		}
		return ownerOf(buf, port)
	}
	return 0, false
}

// ownerOf walks MIB_TCPTABLE_OWNER_PID: a count, then rows of six DWORDs with
// the local port in network order and the owner last.
func ownerOf(buf []byte, port int) (int, bool) {
	const row = 24
	count := int(binary.LittleEndian.Uint32(buf))
	for i := range count {
		at := 4 + i*row
		if at+row > len(buf) {
			break
		}
		local := binary.BigEndian.Uint16(buf[at+8:])
		if int(local) == port {
			return int(binary.LittleEndian.Uint32(buf[at+20:])), true
		}
	}
	return 0, false
}
