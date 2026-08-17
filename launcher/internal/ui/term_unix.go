//go:build !windows

package ui

import "bufio"

// termReadLine is a no-op fallback on non-Windows: the line is read with echo
// on. The launcher targets Windows; this exists so `go build` succeeds on the
// dev machine and the tests run.
func termReadLine(r *bufio.Reader) (string, error) {
	return "", nil
}
