package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConsoleLogSuppliesFakeIPAddressMissingFromStdout(t *testing.T) {
	path := filepath.Join(t.TempDir(), ConsoleLogName)
	body := "Connection to Steam servers successful.\n" +
		"FakeIP allocation succeeded: 169.254.232.222:18344, 18345\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	var lines []Line
	sink := dedupeFakeIPSink(func(line Line) { lines = append(lines, line) })
	offset := readConsoleFakeIP(path, 0, sink)
	if len(lines) != 1 || FakeIPAddress(lines[0].Text) != "169.254.232.222:18344" {
		t.Fatalf("console signals = %#v", lines)
	}

	// The normal case also reaches stdout. Reading the same allocation from
	// both places must not put a duplicate line in the launcher log.
	sink(Line{Source: "srcds", Text: "FakeIP allocation succeeded: 169.254.232.222:18344, 18345"})
	if len(lines) != 1 {
		t.Fatalf("duplicate allocation reached the sink: %#v", lines)
	}

	if next := readConsoleFakeIP(path, offset, sink); next != offset || len(lines) != 1 {
		t.Fatalf("unchanged log was read again: offset %d -> %d, lines %#v", offset, next, lines)
	}
}
