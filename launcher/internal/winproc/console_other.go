//go:build !windows

package winproc

import "os"

// ConsoleStdin returns nothing here.
func ConsoleStdin() (*os.File, error) { return nil, ErrNoConsole }
