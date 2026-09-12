package runtime

import (
	"bufio"
	"context"
	"os"
	"sync"
	"time"
)

const consoleSignalPoll = 500 * time.Millisecond

// watchConsoleFakeIP follows the console written by -condebug for the relay
// allocation that srcds does not reliably copy to its stdout pipe.
func watchConsoleFakeIP(ctx context.Context, path string, sink Sink, wg *sync.WaitGroup) {
	defer wg.Done()
	var offset int64
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(consoleSignalPoll):
			offset = readConsoleFakeIP(path, offset, sink)
		}
	}
}

func readConsoleFakeIP(path string, offset int64, sink Sink) int64 {
	file, err := os.Open(path)
	if err != nil {
		return offset
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return offset
	}
	if info.Size() < offset {
		offset = 0
	}
	if _, err := file.Seek(offset, 0); err != nil {
		return offset
	}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if FakeIPAddress(line) != "" {
			sink(Line{At: time.Now(), Source: "srcds", Text: line})
		}
	}
	return info.Size()
}

// dedupeFakeIPSink makes the stdout pipe and console watcher one source for
// relay allocations without suppressing any other repeated server output.
func dedupeFakeIPSink(sink Sink) Sink {
	var mu sync.Mutex
	seen := make(map[string]struct{})
	return func(line Line) {
		if address := FakeIPAddress(line.Text); address != "" {
			mu.Lock()
			if _, ok := seen[address]; ok {
				mu.Unlock()
				return
			}
			seen[address] = struct{}{}
			mu.Unlock()
		}
		sink(line)
	}
}
