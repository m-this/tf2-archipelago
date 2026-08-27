package installer

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
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

func TestInstallCommunityZipStripsTFDownload(t *testing.T) {
	root := t.TempDir()
	archive := filepath.Join(root, "archive-assets.zip")
	if err := os.WriteFile(archive, zipWith(t, map[string]string{
		"tf/download/maps/mvm_example.bsp":                        "map",
		"tf/download/scripts/population/mvm_example_adv_test.pop": "pop",
		"outside.txt": "ignored",
	}), 0o644); err != nil {
		t.Fatal(err)
	}
	modDir := filepath.Join(root, "server", "tf")
	if err := installCommunityZip(archive, modDir); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"maps/mvm_example.bsp", "scripts/population/mvm_example_adv_test.pop"} {
		if _, err := os.Stat(filepath.Join(modDir, filepath.FromSlash(name))); err != nil {
			t.Errorf("missing %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(modDir, "outside.txt")); !os.IsNotExist(err) {
		t.Errorf("non-TF archive entry was installed: %v", err)
	}
}

func TestDownloadCommunityArchivesDownloadsOnlyTheSelectedPack(t *testing.T) {
	data := zipWith(t, map[string]string{
		"tf/download/maps/mvm_example.bsp": "map",
	})
	requests := 0
	oldClient := communityHTTPClient
	communityHTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++
		return &http.Response{
			StatusCode:    http.StatusOK,
			Body:          io.NopCloser(bytes.NewReader(data)),
			ContentLength: int64(len(data)),
		}, nil
	})}
	t.Cleanup(func() { communityHTTPClient = oldClient })

	root := t.TempDir()
	archive := filepath.Join(root, "packs", "archive-assets.zip")
	if err := DownloadCommunityArchives(context.Background(), []string{archive}, func(string, ...any) {}); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("download requests = %d, want one selected pack", requests)
	}
	if _, err := os.Stat(archive); err != nil {
		t.Errorf("missing %s: %v", archive, err)
	}
	if _, err := os.Stat(filepath.Join(root, "packs", "mlarchive-assets.zip")); !os.IsNotExist(err) {
		t.Errorf("an unselected pack was downloaded: %v", err)
	}
}

func TestInstallCommunityArchivesNeverDownloadsAMissingPack(t *testing.T) {
	requested := false
	oldClient := communityHTTPClient
	communityHTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		requested = true
		return nil, errors.New("network must not be used")
	})}
	t.Cleanup(func() { communityHTTPClient = oldClient })

	root := t.TempDir()
	archive := filepath.Join(root, "archive-assets.zip")
	err := installCommunityArchives([]string{archive}, filepath.Join(root, "server", "tf"), func(string, ...any) {})
	if err == nil {
		t.Fatal("Start accepted a missing selected pack")
	}
	if requested {
		t.Fatal("Start attempted to download a missing community pack")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestDownloadCommunityArchiveRejectsAnUnknownPack(t *testing.T) {
	err := downloadCommunityArchive(context.Background(), filepath.Join(t.TempDir(), "other.zip"), func(string, ...any) {})
	if err == nil {
		t.Fatal("an unknown pack was downloaded")
	}
}

func TestValidateCommunityArchivesRejectsAnInvalidCachedPack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive-assets.zip")
	if err := os.WriteFile(path, []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCommunityArchives([]string{path}, func(string, ...any) {}); err == nil {
		t.Fatal("an invalid cached pack was accepted")
	}
}

func TestAvailableCommunityArchivesRequiresAValidLocalZIP(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "archive-assets.zip")
	invalid := filepath.Join(root, "mlarchive-assets.zip")
	if err := os.WriteFile(invalid, []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := AvailableCommunityArchives([]string{missing, invalid}); len(got) != 0 {
		t.Fatalf("unavailable archives reported as ready: %v", got)
	}

	if err := os.WriteFile(missing, zipWith(t, map[string]string{
		"tf/download/maps/mvm_example.bsp": "map",
	}), 0o644); err != nil {
		t.Fatal(err)
	}
	got := AvailableCommunityArchives([]string{missing, invalid})
	if len(got) != 1 || got[0] != missing {
		t.Fatalf("available archives = %v, want only %s", got, missing)
	}
}

func TestInstallPluginIncludesNativeProjectileGameData(t *testing.T) {
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
