package winproc

import (
	"errors"
	"fmt"
	"os"
	"time"
)

// ErrNoConsole says this platform has no console input buffer to hand a child.
// Only Windows has one, and only srcds asks for it.
var ErrNoConsole = errors.New("console input is a Windows idea")

// ConsoleStdinTimeout is ConsoleStdin with a deadline. Allocating a console is
// a call into Windows that a broken console host can sit on, and neither the
// window nor the server start is allowed to wait for it.
func ConsoleStdinTimeout(limit time.Duration) (*os.File, error) {
	type result struct {
		file *os.File
		err  error
	}
	done := make(chan result, 1)
	go func() {
		/* This one goroutine outlives the timeout below when the console host
		   hangs, so a panic in ConsoleStdin would land on nobody. Recovered
		   into the channel instead, where the caller reads it as the failure
		   it is. */
		defer func() {
			if reason := recover(); reason != nil {
				done <- result{nil, fmt.Errorf("the console probe panicked: %v", reason)}
			}
		}()
		file, err := ConsoleStdin()
		done <- result{file, err}
	}()
	select {
	case got := <-done:
		return got.file, got.err
	case <-time.After(limit):
		return nil, errors.New("the console did not come up in time")
	}
}
