package runtime

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// smlogPoll is how often the newest SourceMod error log is read again.
const smlogPoll = 2 * time.Second

// watchSourcemodErrors feeds SourceMod's error log into the sink.
//
// SourceMod writes plugin errors to addons/sourcemod/logs/errors_*.log and not
// to the console, so a plugin that throws leaves nothing in the window. That
// is the file to read when the server stops answering: the last lines in it
// name the plugin and the line it died on.
func watchSourcemodErrors(ctx context.Context, gameDir string, sink Sink) {
	dir := filepath.Join(gameDir, "tf", "addons", "sourcemod", "logs")
	var (
		current string
		offset  int64
	)
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(smlogPoll):
		}

		newest := newestErrorLog(dir)
		if newest == "" {
			continue
		}
		if newest != current {
			current, offset = newest, 0
		}
		offset = readFrom(newest, offset, sink)
	}
}

func newestErrorLog(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var names []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "errors_") && strings.HasSuffix(entry.Name(), ".log") {
			names = append(names, entry.Name())
		}
	}
	if len(names) == 0 {
		return ""
	}
	// The names carry the date, so the last one in order is the current one.
	sort.Strings(names)
	return filepath.Join(dir, names[len(names)-1])
}

// readFrom emits whatever was appended since offset, and returns the new one.
func readFrom(path string, offset int64, sink Sink) int64 {
	file, err := os.Open(path)
	if err != nil {
		return offset
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil || info.Size() < offset {
		return 0 // rotated or truncated
	}
	if _, err := file.Seek(offset, 0); err != nil {
		return offset
	}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			sink(Line{At: time.Now(), Source: "sourcemod", Text: line})
		}
	}
	return info.Size()
}
