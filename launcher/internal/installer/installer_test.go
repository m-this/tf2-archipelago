package installer

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-this/tf2-archipelago/launcher/internal/assets"
)

func zipWith(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, body := range entries {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("cannot create %s: %v", name, err)
		}
		if _, err := f.Write([]byte(body)); err != nil {
			t.Fatalf("cannot write %s: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("cannot close the zip: %v", err)
	}
	return buf.Bytes()
}

// SourceMod, Metamod, ripext and the bots all root at addons/, and all four
// belong under tf/. Unpacking them next to srcds.exe installs nothing the
// server ever loads.
func TestUnzipToKeepsTheArchiveLayout(t *testing.T) {
	root := t.TempDir()
	modDir := filepath.Join(root, "tf-dedicated", "tf")
	data := zipWith(t, map[string]string{
		"addons/metamod.vdf":                     "vdf",
		"addons/sourcemod/plugins/tf2utils.smx":  "smx",
		"addons/sourcemod/extensions/a.tf2.dll":  "dll",
		"cfg/sourcemod/tf2_archipelago.cfg":      "cfg",
		"addons/sourcemod/configs/defbots/n.txt": "names",
	})

	if err := unzipTo(data, modDir); err != nil {
		t.Fatalf("unzipTo: %v", err)
	}
	for _, want := range []string{
		"addons/metamod.vdf",
		"addons/sourcemod/plugins/tf2utils.smx",
		"addons/sourcemod/extensions/a.tf2.dll",
		"cfg/sourcemod/tf2_archipelago.cfg",
	} {
		if _, err := os.Stat(filepath.Join(modDir, filepath.FromSlash(want))); err != nil {
			t.Errorf("missing %s: %v", want, err)
		}
	}
}

func TestUnzipToRejectsAnEscapingEntry(t *testing.T) {
	dir := t.TempDir()
	data := zipWith(t, map[string]string{"../escaped.txt": "no"})
	if err := unzipTo(data, filepath.Join(dir, "game")); err == nil {
		t.Fatal("an entry outside the install directory was accepted")
	}
	if _, err := os.Stat(filepath.Join(dir, "escaped.txt")); err == nil {
		t.Error("the entry was written outside the install directory")
	}
}

/*
	The installer writes the plugin and the game data it needs beside it. A

plugin without its game data loads and then cannot find the natives it was
compiled against, which reads as the plugin being broken.

Skipped without the real assets. `make embed` fetches ripext and the ordinary
build stands in an empty zip for it, so this runs on a full build and in the
release job rather than on every `go test`.
*/
func TestInstallPluginIncludesNativeProjectileGameData(t *testing.T) {
	if len(assets.RipextZip()) < 1024 {
		t.Skip("the embedded ripext is a placeholder; run make embed for the real one")
	}
	modDir := filepath.Join(t.TempDir(), "tf")
	if err := installRipextAndPlugin(modDir); err != nil {
		t.Fatal(err)
	}
	wants := map[string][]byte{
		filepath.Join("addons", "sourcemod", "plugins", "tf2_archipelago.smx"):  assets.Plugin(),
		filepath.Join("addons", "sourcemod", "gamedata", "tf2_archipelago.txt"): assets.PluginGameData(),
	}
	for relative, want := range wants {
		got, err := os.ReadFile(filepath.Join(modDir, relative))
		if err != nil {
			t.Fatalf("read %s: %v", relative, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s differs from embedded asset", relative)
		}
	}
}

func TestCleanKeepsWhatCannotBeFetchedAgain(t *testing.T) {
	root := t.TempDir()
	write := func(parts ...string) string {
		path := filepath.Join(append([]string{root}, parts...)...)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("cannot create %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("cannot write %s: %v", path, err)
		}
		return path
	}

	gone := []string{
		write("steamcmd", "steamcmd.exe"),
		write("tf-dedicated", "tf", "addons", "sourcemod", "plugins", "tf2_archipelago.smx"),
		write("tf-dedicated", "steamapps", "appmanifest_232250.acf"),
	}
	kept := []string{
		write("tf-dedicated", "srcds.exe"),
		write("tf-dedicated", "tf", "maps", "mvm_decoy.bsp"),
		write("bridge-state", "bridge.json"),
		write("tf2.yaml"),
	}

	removed, err := Clean(root)
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if len(removed) != 3 {
		t.Errorf("removed %d directories, want 3: %v", len(removed), removed)
	}
	for _, path := range gone {
		if _, err := os.Stat(path); err == nil {
			t.Errorf("%s survived", path)
		}
	}
	for _, path := range kept {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("%s was removed: %v", path, err)
		}
	}
}

// A repair on a half-installed tree must not fail on what is not there.
func TestCleanOnAnEmptyRoot(t *testing.T) {
	removed, err := Clean(t.TempDir())
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if len(removed) != 0 {
		t.Errorf("removed %v from an empty root", removed)
	}
}
