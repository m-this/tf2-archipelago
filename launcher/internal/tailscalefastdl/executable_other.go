//go:build !windows

package tailscalefastdl

// Non-Windows packages install the CLI on PATH.
func installedExecutablePath() string { return "" }
