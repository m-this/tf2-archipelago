package generate

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// A fake generator: writes an archive where it is told to and prints a line,
// which is what the real one does from this side.
func fakeGenerator(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	script := `#!/bin/sh
out=""
while [ $# -gt 0 ]; do
  case "$1" in --outputpath) out="$2"; shift;; esac
  shift
done
echo "Creating final archive at $out/AP_123.zip"
: > "$out/AP_123.zip"
`
	path := filepath.Join(dir, "ArchipelagoGenerate")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("cannot write the fake generator: %v", err)
	}
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh")
	}
	return dir
}

func TestRunFindsTheAppAndReturnsTheArchive(t *testing.T) {
	appDir := fakeGenerator(t)

	s := settings.Defaults()
	s.InstallRoot = t.TempDir()
	s.APSlotName = "tester"

	var lines []string
	result, err := Run(context.Background(), Options{
		Settings:           s,
		AppDir:             appDir,
		Apworld:            []byte("PK-fake"),
		ArchipelagoVersion: "0.6.7",
		Log:                func(line string) { lines = append(lines, line) },
		Timeout:            30 * time.Second,
	})
	if err != nil {
		t.Fatalf("Run: %v\n%s", err, strings.Join(lines, "\n"))
	}
	if filepath.Base(result.Archive) != "AP_123.zip" {
		t.Errorf("archive is %s", result.Archive)
	}
	if result.AppDir != appDir {
		t.Errorf("app dir is %s, want %s", result.AppDir, appDir)
	}

	// The apworld went where the app looks, and the player file went where
	// the generator was pointed.
	if _, err := os.Stat(filepath.Join(appDir, "custom_worlds", "tf2_mvm.apworld")); err != nil {
		t.Errorf("the apworld was not installed: %v", err)
	}
	yaml, err := os.ReadFile(filepath.Join(s.InstallRoot, "generate", "players", "tf2.yaml"))
	if err != nil {
		t.Fatalf("no player file: %v", err)
	}
	if !strings.Contains(string(yaml), `name: "tester"`) {
		t.Errorf("the player file has the wrong slot:\n%s", yaml)
	}
}

func TestRunWithoutTheApp(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	s := settings.Defaults()
	s.InstallRoot = t.TempDir()
	_, err := Run(context.Background(), Options{Settings: s, AppDir: filepath.Join(t.TempDir(), "nowhere")})
	if err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("got %v, want the not-installed error", err)
	}
}
