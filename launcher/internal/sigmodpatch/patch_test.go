package sigmodpatch

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestPatchSigmodLoadoutMenu(t *testing.T) {
	root := t.TempDir()
	for _, relative := range ExtensionPaths("linux") {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		body := append(append([]byte("before\x00"), sigmodLoadoutMenuBroken...), "after"...)
		if err := os.WriteFile(path, body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := Patch(root, "linux"); err != nil {
		t.Fatal(err)
	}
	if err := Patch(root, "linux"); err != nil {
		t.Fatalf("second patch should be harmless: %v", err)
	}
	if !Ready(root, "linux") {
		t.Fatal("patched extensions were not recognized")
	}
	for _, relative := range ExtensionPaths("linux") {
		body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		want := append(append([]byte("before\x00"), sigmodLoadoutMenuFixed...), "after"...)
		if !bytes.Equal(body, want) {
			t.Fatalf("unexpected bytes in %s", relative)
		}
	}
}

// The Linux package is built from source that carries the menu fix, so it
// arrives with the corrected string and nothing left to patch.
func TestRebuiltLinuxPackageIsAlreadyFixed(t *testing.T) {
	root := t.TempDir()
	for _, relative := range ExtensionPaths("linux") {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		body := append(append([]byte("Extra loadout items\x00"), sigmodLoadoutMenuFixed...), "after"...)
		if err := os.WriteFile(path, body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := Patch(root, "linux"); err != nil {
		t.Fatal(err)
	}
	if !Ready(root, "linux") {
		t.Fatal("a package built with the fix was not recognized")
	}
}

func TestPatchSigmodLoadoutMenuRefusesUnknownBinary(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, filepath.FromSlash(ExtensionPaths("windows")[0]))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("unknown binary"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Patch(root, "windows"); err == nil {
		t.Fatal("unknown SigMod binary should require a new patch assessment")
	}
}

func TestPatchSigmodLoadoutMenuWindows(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, filepath.FromSlash(ExtensionPaths("windows")[0]))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, sigmodLoadoutMenuBroken, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Patch(root, "windows"); err != nil {
		t.Fatal(err)
	}
	if !Ready(root, "windows") {
		t.Fatal("patched Windows extension was not recognized")
	}
}
