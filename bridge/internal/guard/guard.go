/*
Package guard turns a panic on a goroutine nobody waits on into a line.

The bridge runs its pump and its session on their own goroutines, and Go takes
the whole process down for a panic on any of them. The launcher hosts the
bridge, so that is the launcher closing and the server somebody is playing on
stopping with it.

Not a swallow. The panic and its stack go to the logger the caller already has,
which is louder than the silent exit it replaces.
*/
package guard

import (
	"fmt"
	"log/slog"
	"runtime/debug"
)

// Run does the work, and reports a panic rather than letting it end the process.
func Run(name string, logger *slog.Logger, work func()) {
	defer func() {
		reason := recover()
		if reason == nil {
			return
		}
		if logger != nil {
			logger.Error("contained a panic",
				"where", name,
				"reason", fmt.Sprint(reason),
				"stack", string(debug.Stack()))
		}
	}()
	work()
}
