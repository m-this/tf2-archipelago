//go:build !windows

package ui

// AttachConsole is a no-op away from Windows: the streams are already the
// terminal's.
func AttachConsole() bool { return true }
