package debugbundle

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("cannot create %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("cannot write %s: %v", path, err)
	}
}

func read(t *testing.T, path string) map[string]string {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("cannot open the bundle: %v", err)
	}
	defer func() { _ = reader.Close() }()

	held := map[string]string{}
	for _, file := range reader.File {
		opened, err := file.Open()
		if err != nil {
			t.Fatalf("cannot read %s: %v", file.Name, err)
		}
		body, _ := io.ReadAll(opened)
		_ = opened.Close()
		held[file.Name] = string(body)
	}
	return held
}

func TestWriteCollectsTheFilesAndHidesTheSecrets(t *testing.T) {
	root := t.TempDir()
	game := filepath.Join(root, "tf-dedicated", "tf")
	write(t, filepath.Join(root, "launcher.log"), "13:37:00 launcher started\n")
	write(t, filepath.Join(root, settings.PlayerFileName), "name: \"mathis\"\n")
	write(t, filepath.Join(game, "console.log"), "Server is hibernating\n")
	write(t, filepath.Join(game, "cfg", "server.cfg"), "hostname \"x\"\n")
	write(t, filepath.Join(game, "addons", "sourcemod", "logs", "errors_20260818.log"), "L 08/18/2026: [SM] Exception\n")

	s := settings.Defaults()
	s.InstallRoot = root
	s.APHost, s.APPort = "archipelago.gg", 12345
	s.SrcdsRconPw = "the-rcon-secret"
	s.SrcdsPw = "the-join-secret"
	s.APPassword = "the-room-secret"
	s.SrcdsToken = "the-token-secret"

	stamp := time.Date(2026, 8, 18, 17, 4, 5, 0, time.UTC)
	path, err := Write(s, map[string]string{"sourcemod": "1.12.0-git7246"}, stamp)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if want := filepath.Join(root, "debug-logs-2026-08-18-170405.zip"); path != want {
		t.Errorf("wrote %s, want %s", path, want)
	}

	held := read(t, path)
	for _, name := range []string{
		"summary.txt", "config.json", "launcher.log", "tf2.yaml",
		"console.log", "server.cfg", "sourcemod/errors_20260818.log",
	} {
		if _, ok := held[name]; !ok {
			t.Errorf("the bundle has no %s (it holds %v)", name, keys(held))
		}
	}
	if !strings.Contains(held["sourcemod/errors_20260818.log"], "[SM] Exception") {
		t.Error("the SourceMod error did not make it in")
	}
	if !strings.Contains(held["summary.txt"], "1.12.0-git7246") {
		t.Error("the summary does not name the versions")
	}

	// The bundle is made to be posted in a chat channel.
	whole := strings.Join([]string{held["summary.txt"], held["config.json"]}, "\n")
	for _, secret := range []string{"the-rcon-secret", "the-join-secret", "the-room-secret", "the-token-secret"} {
		if strings.Contains(whole, secret) {
			t.Errorf("%s is in the bundle", secret)
		}
	}
}

// A run that never got as far as a console log still has to produce a bundle:
// the missing files are themselves a clue.
func TestWriteOnAnEmptyRoot(t *testing.T) {
	s := settings.Defaults()
	s.InstallRoot = t.TempDir()
	path, err := Write(s, nil, time.Date(2026, 8, 18, 1, 2, 3, 0, time.UTC))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	held := read(t, path)
	if _, ok := held["summary.txt"]; !ok {
		t.Errorf("no summary in a bundle from an empty root: %v", keys(held))
	}
}

func keys(m map[string]string) []string {
	var out []string
	for name := range m {
		out = append(out, name)
	}
	return out
}
