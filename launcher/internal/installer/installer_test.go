package installer

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
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
