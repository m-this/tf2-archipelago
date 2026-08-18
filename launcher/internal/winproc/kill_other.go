//go:build !windows

package winproc

// KillUnder does nothing here. The launcher's window, and the process tree it
// has to clean up after, are Windows only.
func KillUnder(string) ([]string, error) { return nil, nil }
