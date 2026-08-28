package runtime

import (
	"fmt"
	"runtime/debug"
)

/*
Guard runs work that nobody is waiting on, and turns a panic in it into a line
rather than a dead launcher.

Every one of these goroutines outlives the call that started it: the bridge, the
server, the SourceMod error watcher. Go takes the whole process down for a panic
on any goroutine, so a nil map in the log watcher closed the window, stopped
supervising the server somebody was playing on, and left them with a mission to
replay.

Not a swallow. The panic and its stack go to the sink the caller reads, which is
the log a debug bundle carries, so it is louder here than it was as a silent
exit. What it does not do is take the run with it.
*/
func Guard(name string, say func(string), work func()) {
	defer func() {
		reason := recover()
		if reason == nil {
			return
		}
		say(fmt.Sprintf("%s panicked and was contained: %v\n%s", name, reason, debug.Stack()))
	}()
	work()
}
