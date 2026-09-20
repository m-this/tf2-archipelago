package installer

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/m-this/tf2-archipelago/launcher/internal/assets"
)

func withSigmodPin(t *testing.T, data []byte) {
	t.Helper()
	oldVersion, oldSHA := assets.SigsegvMVMVersion, assets.SigsegvMVMSHA256
	assets.SigsegvMVMVersion = "test-version"
	assets.SigsegvMVMSHA256 = fmt.Sprintf("%x", sha256.Sum256(data))
	t.Cleanup(func() {
		assets.SigsegvMVMVersion, assets.SigsegvMVMSHA256 = oldVersion, oldSHA
	})
}

func fakeSigmodZip(t *testing.T) []byte {
	t.Helper()
	return zipWith(t, map[string]string{
		"addons/sourcemod/extensions/sigsegv.ext.2.tf2.so":     "32-bit extension",
		"addons/sourcemod/extensions/x64/sigsegv.ext.2.tf2.so": "64-bit extension",
		"addons/sourcemod/extensions/sigsegv.autoload":         "",
		"addons/sourcemod/gamedata/sigsegv/population.txt":     "gamedata",
		"cfg/sigsegv_convars.cfg":                              "configuration",
	})
}

func writeFakeSourcemod(t *testing.T, modDir string) {
	t.Helper()
	for _, relative := range sourcemodFiles(runtime.GOOS) {
		path := filepath.Join(modDir, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("sourcemod"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

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

func TestDisableSourceModMapRotationIsIdempotent(t *testing.T) {
	modDir := t.TempDir()
	active := filepath.Join(modDir, "addons", "sourcemod", "plugins", "nextmap.smx")
	if err := os.MkdirAll(filepath.Dir(active), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(active, []byte("stock nextmap"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := disableSourceModMapRotation(modDir); err != nil {
		t.Fatal(err)
	}
	disabled := filepath.Join(filepath.Dir(active), "disabled", "nextmap.smx")
	if body, err := os.ReadFile(disabled); err != nil || string(body) != "stock nextmap" {
		t.Fatalf("disabled nextmap = %q, %v", body, err)
	}
	if err := disableSourceModMapRotation(modDir); err != nil {
		t.Fatal(err)
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
	completed := false
	if err := installCommunityZip(archive, modDir, nil, func(format string, args ...any) {
		completed = completed || strings.Contains(fmt.Sprintf(format, args...), "100%")
	}); err != nil {
		t.Fatal(err)
	}
	if !completed {
		t.Error("community archive extraction did not report completion progress")
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

func TestCommunityDownloadManifestIncludesIconsAndAliasedTextures(t *testing.T) {
	modDir := t.TempDir()
	populationDir := filepath.Join(modDir, "scripts", "population")
	materialDir := filepath.Join(modDir, "materials", "hud")
	if err := os.MkdirAll(populationDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(materialDir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(populationDir, "mvm_kelly_rc1b_adv_test.pop"),
		"#base robot_test.pop\nClassIcon stock_icon\n")
	write(filepath.Join(populationDir, "robot_test.pop"), "ClassIcon Engineer_Ranger_Test_Giant\n")
	write(filepath.Join(materialDir, "leaderboard_class_engineer_ranger_test_giant.vmt"),
		`"UnlitGeneric" { "$baseTexture" "hud/leaderboard_class_engineer_ranger_test" }`)
	write(filepath.Join(materialDir, "leaderboard_class_engineer_ranger_test.vtf"), "texture")

	if err := writeCommunityDownloadManifests(modDir); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(modDir, "addons", "sourcemod", "data", "tf2_archipelago", "downloads", "mvm_kelly_rc1b.txt"))
	if err != nil {
		t.Fatal(err)
	}
	want := "materials/hud/leaderboard_class_engineer_ranger_test.vtf\n" +
		"materials/hud/leaderboard_class_engineer_ranger_test_giant.vmt\n"
	if string(body) != want {
		t.Fatalf("manifest = %q, want %q", body, want)
	}
}

func TestCommunityInstallSkipsUnsupportedMissionsButKeepsSharedPopulationFiles(t *testing.T) {
	root := t.TempDir()
	archive := filepath.Join(root, "archive-assets.zip")
	if err := os.WriteFile(archive, zipWith(t, map[string]string{
		"tf/download/scripts/population/mvm_lotus_b6_adv_mud.pop":      "supported",
		"tf/download/scripts/population/mvm_lotus_b6_adv_ledmotif.pop": "requires RafMod",
		"tf/download/scripts/population/robot_giant.pop":               "shared template",
		"tf/download/scripts/population/mvm_some_user_map.pop":         "unrelated",
	}), 0o644); err != nil {
		t.Fatal(err)
	}
	modDir := filepath.Join(root, "server", "tf")
	if err := installCommunityZip(archive, modDir, nil, func(string, ...any) {}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"mvm_lotus_b6_adv_mud.pop",
		"robot_giant.pop",
		"mvm_some_user_map.pop",
	} {
		if _, err := os.Stat(filepath.Join(modDir, "scripts", "population", name)); err != nil {
			t.Errorf("missing %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(modDir, "scripts", "population", "mvm_lotus_b6_adv_ledmotif.pop")); !os.IsNotExist(err) {
		t.Errorf("unsupported mission was installed: %v", err)
	}
}

func TestCommunityInstallIncludesSigmodMissionsOnlyWithVerifiedMod(t *testing.T) {
	root := t.TempDir()
	archive := filepath.Join(root, "archive-assets.zip")
	if err := os.WriteFile(archive, zipWith(t, map[string]string{
		"tf/download/scripts/population/mvm_lotus_b6_adv_ledmotif.pop": "SigMod mission",
	}), 0o644); err != nil {
		t.Fatal(err)
	}
	without := filepath.Join(root, "without", "tf")
	with := filepath.Join(root, "with", "tf")
	if err := installCommunityZip(archive, without, nil, func(string, ...any) {}); err != nil {
		t.Fatal(err)
	}
	if err := installCommunityZip(archive, with, []string{sigmodKey}, func(string, ...any) {}); err != nil {
		t.Fatal(err)
	}
	relative := filepath.Join("scripts", "population", "mvm_lotus_b6_adv_ledmotif.pop")
	if _, err := os.Stat(filepath.Join(without, relative)); !os.IsNotExist(err) {
		t.Errorf("SigMod mission installed without SigMod: %v", err)
	}
	if _, err := os.Stat(filepath.Join(with, relative)); err != nil {
		t.Errorf("SigMod mission missing with SigMod: %v", err)
	}
}

func TestInstallServerModsUsesVerifiedCacheAndDetectsTheInstall(t *testing.T) {
	data := fakeSigmodZip(t)
	withSigmodPin(t, data)
	root := t.TempDir()
	modDir := filepath.Join(root, "tf-dedicated", "tf")
	writeFakeSourcemod(t, modDir)
	cache := filepath.Join(root, "downloads", "sigsegv-mvm-test-version-linux.zip")
	if err := os.MkdirAll(filepath.Dir(cache), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cache, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := installServerMods(context.Background(), root, modDir, []string{sigmodKey}, func(string, ...any) {}); err != nil {
		t.Fatal(err)
	}
	if got := ReadyServerMods(root); len(got) != 1 || got[0] != sigmodKey {
		t.Fatalf("ready server mods = %v", got)
	}
	if err := os.Remove(filepath.Join(modDir, "addons", "sourcemod", "gamedata", "sigsegv", "population.txt")); err != nil {
		t.Fatal(err)
	}
	if got := ReadyServerMods(root); len(got) != 0 {
		t.Fatalf("incomplete SigMod reported ready: %v", got)
	}
}

func TestComposeSigmodStampMatchesLauncherReceipt(t *testing.T) {
	root := t.TempDir()
	for _, relative := range sigmodFiles(runtime.GOOS) {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("contents of "+relative), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	version, archiveSHA := "compose-test-version", strings.Repeat("a", 64)
	script := filepath.Join("..", "..", "..", "deploy", "sigmod-stamp.sh")
	if output, err := exec.Command("sh", script, root, version, archiveSHA).CombinedOutput(); err != nil {
		t.Fatalf("write Compose SigMod stamp: %v\n%s", err, output)
	}
	oldVersion, oldSHA := assets.SigsegvMVMVersion, assets.SigsegvMVMSHA256
	assets.SigsegvMVMVersion, assets.SigsegvMVMSHA256 = version, archiveSHA
	t.Cleanup(func() { assets.SigsegvMVMVersion, assets.SigsegvMVMSHA256 = oldVersion, oldSHA })
	want, err := sigmodStamp(root)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, "addons", ".tf2ap-sigsegv-mvm.stamp"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("Compose SigMod stamp differs from launcher receipt\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestCommunityInstallRemovesUnsupportedMissionsFromExistingTrees(t *testing.T) {
	modDir := t.TempDir()
	populationDir := filepath.Join(modDir, "scripts", "population")
	if err := os.MkdirAll(populationDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"mvm_lotus_b6_adv_mud.pop",
		"mvm_lotus_b6_adv_ledmotif.pop",
		"robot_standard.pop",
	} {
		if err := os.WriteFile(filepath.Join(populationDir, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	removed, err := removeUnsupportedCommunityPopfiles(modDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("removed %d files, want 1", removed)
	}
	if _, err := os.Stat(filepath.Join(populationDir, "mvm_lotus_b6_adv_ledmotif.pop")); !os.IsNotExist(err) {
		t.Errorf("unsupported installed mission remains: %v", err)
	}
	for _, name := range []string{"mvm_lotus_b6_adv_mud.pop", "robot_standard.pop"} {
		if _, err := os.Stat(filepath.Join(populationDir, name)); err != nil {
			t.Errorf("compatible file %s was removed: %v", name, err)
		}
	}
}

func TestDownloadCommunityArchivesDownloadsOnlyTheSelectedPack(t *testing.T) {
	data := zipWith(t, map[string]string{
		"tf/download/maps/mvm_example.bsp": "map",
	})
	withPotatoArchivePin(t, data)
	withoutGitHubParts(t, "archive-assets.zip")
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

func withPotatoArchivePin(t *testing.T, data []byte) {
	t.Helper()
	old := communityArchiveSHA256["archive-assets.zip"]
	digest := sha256.Sum256(data)
	communityArchiveSHA256["archive-assets.zip"] = fmt.Sprintf("%x", digest)
	t.Cleanup(func() { communityArchiveSHA256["archive-assets.zip"] = old })
}

func TestCommunityArchiveMismatchNeedsExplicitApprovalForExactBytes(t *testing.T) {
	wanted := zipWith(t, map[string]string{"tf/download/maps/map.bsp": "expected"})
	changed := zipWith(t, map[string]string{"tf/download/maps/map.bsp": "changed"})
	withPotatoArchivePin(t, wanted)
	withoutGitHubParts(t, "archive-assets.zip")
	oldClient := communityHTTPClient
	communityHTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(changed)), ContentLength: int64(len(changed))}, nil
	})}
	t.Cleanup(func() { communityHTTPClient = oldClient })
	path := filepath.Join(t.TempDir(), "archive-assets.zip")
	err := DownloadCommunityArchives(context.Background(), []string{path}, func(string, ...any) {})
	var mismatch *CommunityArchiveHashMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("download error = %v, want hash mismatch", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("mismatched pack became usable: %v", err)
	}
	if got := PendingCommunityArchiveHashMismatches([]string{path}); !slices.Equal(got, []string{"archive-assets.zip"}) {
		t.Fatalf("pending packs = %v", got)
	}
	if _, err := IgnoreCommunityArchiveHashMismatch([]string{path}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCommunityArchives([]string{path}, func(string, ...any) {}); err != nil {
		t.Fatalf("approved bytes rejected: %v", err)
	}
	if err := os.WriteFile(path, zipWith(t, map[string]string{"tf/download/maps/map.bsp": "changed again"}), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCommunityArchives([]string{path}, func(string, ...any) {}); !errors.As(err, &mismatch) {
		t.Fatalf("changed bytes inherited approval: %v", err)
	}
}

func withoutGitHubParts(t *testing.T, name string) {
	t.Helper()
	old := communityGitHubParts[name]
	delete(communityGitHubParts, name)
	t.Cleanup(func() { communityGitHubParts[name] = old })
}

func TestGitHubSplitArchiveReassemblesAndFallsBackToPotato(t *testing.T) {
	data := zipWith(t, map[string]string{"tf/download/maps/map.bsp": "map"})
	withPotatoArchivePin(t, data)
	cut := len(data) / 2
	parts := [][]byte{data[:cut], data[cut:]}
	old := communityGitHubParts["archive-assets.zip"]
	communityGitHubParts["archive-assets.zip"] = []communityPart{
		{URL: "https://github.test/part-0", Size: int64(len(parts[0])), SHA256: fmt.Sprintf("%x", sha256.Sum256(parts[0]))},
		{URL: "https://github.test/part-1", Size: int64(len(parts[1])), SHA256: fmt.Sprintf("%x", sha256.Sum256(parts[1]))},
	}
	t.Cleanup(func() { communityGitHubParts["archive-assets.zip"] = old })
	oldClient := communityHTTPClient
	fallback := false
	communityHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var body []byte
		switch req.URL.Host {
		case "github.test":
			if strings.HasSuffix(req.URL.Path, "part-0") {
				body = parts[0]
			} else {
				body = parts[1]
			}
		case "dlarchive.potato.tf":
			fallback = true
			body = data
		default:
			return nil, fmt.Errorf("unexpected URL: %s", req.URL)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(body)), ContentLength: int64(len(body))}, nil
	})}
	t.Cleanup(func() { communityHTTPClient = oldClient })
	path := filepath.Join(t.TempDir(), "archive-assets.zip")
	if err := downloadCommunityArchive(context.Background(), path, func(string, ...any) {}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, data) || fallback {
		t.Fatalf("GitHub reconstruction failed: read error %v, fallback %t", err, fallback)
	}
	// Corrupt the first part. The mirror must be tried and must reconstruct
	// exactly the same archive, without accepting the bad GitHub bytes.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	parts[0] = bytes.Clone(parts[0])
	parts[0][0] ^= 1
	if err := downloadCommunityArchive(context.Background(), path, func(string, ...any) {}); err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(path)
	if err != nil || !bytes.Equal(got, data) || !fallback {
		t.Fatalf("Potato fallback failed: read error %v, fallback %t", err, fallback)
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
	err := installCommunityArchives([]string{archive}, filepath.Join(root, "server", "tf"), nil, func(string, ...any) {})
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

	data := zipWith(t, map[string]string{
		"tf/download/maps/mvm_example.bsp": "map",
	})
	withPotatoArchivePin(t, data)
	if err := os.WriteFile(missing, data, 0o644); err != nil {
		t.Fatal(err)
	}
	got := AvailableCommunityArchives([]string{missing, invalid})
	if len(got) != 1 || got[0] != missing {
		t.Fatalf("available archives = %v, want only %s", got, missing)
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

// A directory is not a mod that loads. The launcher looks for the files the
// engine reads, and names the one that is gone.
func TestTheLoaderFilesDecideWhetherToReinstall(t *testing.T) {
	for _, goos := range []string{"linux", "windows"} {
		modDir := t.TempDir()
		files := append(metamodFiles(goos), sourcemodFiles(goos)...)
		if got := firstMissing(modDir, metamodFiles(goos)); got != "addons/metamod.vdf" {
			t.Errorf("%s: empty tree reports %q", goos, got)
		}
		for _, relative := range files {
			path := filepath.Join(modDir, filepath.FromSlash(relative))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if got := firstMissing(modDir, metamodFiles(goos)); got != "" {
			t.Errorf("%s: full tree reports %q", goos, got)
		}
		loader := filepath.Join(modDir, filepath.FromSlash(sourcemodFiles(goos)[0]))
		if err := os.Remove(loader); err != nil {
			t.Fatal(err)
		}
		if got := firstMissing(modDir, sourcemodFiles(goos)); got != sourcemodFiles(goos)[0] {
			t.Errorf("%s: a missing loader reports %q", goos, got)
		}
	}
}

// Upstream's SigMod release has no Windows binary in it, so a Windows launcher
// that fell back to package-linux.zip would download 30 MB and install nothing
// SourceMod can load. The pins are separate releases of separate repositories.
func TestSigmodPackageIsPerPlatform(t *testing.T) {
	windows := sigmodURL("windows", "20260918")
	linux := sigmodURL("linux", "20250703")
	if !strings.Contains(windows, "sigsegv-mvm-win/releases/download/20260918/package-windows.zip") {
		t.Errorf("Windows SigMod URL = %q", windows)
	}
	if !strings.Contains(linux, "rafradek/sigsegv-mvm/releases/download/20250703/package-linux.zip") {
		t.Errorf("Linux SigMod URL = %q", linux)
	}
	if slices.Contains(sigmodFiles("windows"), "addons/sourcemod/extensions/x64/sigsegv.ext.2.tf2.so") {
		t.Error("the Windows install is judged by a Linux extension")
	}
	for _, want := range []string{
		"addons/sourcemod/extensions/sigsegv.ext.2.tf2.dll",
		"addons/sourcemod/gamedata/sigsegv/windows.txt",
	} {
		if !slices.Contains(sigmodFiles("windows"), want) {
			t.Errorf("the Windows install does not require %s", want)
		}
	}
}

// A mod the catalog has no build for here is what an earlier release left
// behind, so the question is never whether to install it.
func TestBuildsOnFollowsTheCatalog(t *testing.T) {
	for _, goos := range []string{"linux", "windows"} {
		if !buildsOn(sigmodKey, goos) {
			t.Errorf("SigMod has no %s build", goos)
		}
	}
	if buildsOn("not-a-mod", "linux") {
		t.Error("an unknown mod claims a build")
	}
	if buildsOn(sigmodKey, "darwin") {
		t.Error("SigMod claims a build on a platform nothing ships for")
	}
}

/*
The autoload marker is launcher state, not part of the package.

apw-5g4.14: SourceMod loads an extension because a file sits beside it, so
installing SigMod was loading it and a host whose server crashed could not turn
it off. Hashing the marker into the receipt would make turning it off read as a
broken install and reinstall it on the next start.
*/
func TestTheAutoloadMarkerIsNotPartOfTheInstall(t *testing.T) {
	for _, goos := range []string{"linux", "windows"} {
		if slices.Contains(sigmodFiles(goos), sigmodAutoload) {
			t.Errorf("the %s install is judged by the autoload marker", goos)
		}
	}
}

// Off, and on, and off again: this runs on every start.
func TestSetServerModLoadingWritesAndRemovesTheMarker(t *testing.T) {
	installRoot := t.TempDir()
	modDir := filepath.Join(installRoot, "tf-dedicated", "tf")
	for _, relative := range sigmodFiles(runtime.GOOS) {
		path := filepath.Join(modDir, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	marker := filepath.Join(modDir, filepath.FromSlash(sigmodAutoload))

	if err := SetServerModLoading(installRoot, []string{sigmodKey}); err != nil {
		t.Fatalf("load: %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("the marker was not written: %v", err)
	}
	if err := SetServerModLoading(installRoot, nil); err != nil {
		t.Fatalf("unload: %v", err)
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Error("the marker survived being turned off")
	}
	// Twice, because nothing guarantees which state a start begins in.
	if err := SetServerModLoading(installRoot, nil); err != nil {
		t.Fatalf("unload twice: %v", err)
	}
}

// A marker beside no extension makes SourceMod complain on every start.
func TestSetServerModLoadingWritesNoMarkerWithoutTheExtension(t *testing.T) {
	installRoot := t.TempDir()
	if err := SetServerModLoading(installRoot, []string{sigmodKey}); err != nil {
		t.Fatalf("load: %v", err)
	}
	marker := filepath.Join(installRoot, "tf-dedicated", "tf", filepath.FromSlash(sigmodAutoload))
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Error("a marker was written beside no extension")
	}
}

// The autoload marker is why removing the files is the fix and unticking the
// mod is not: SourceMod loads any extension beside one on every start.
func TestRemoveSigmodTakesTheWholeInstallOffTheDisk(t *testing.T) {
	modDir := t.TempDir()
	written := []string{
		"addons/sourcemod/extensions/sigsegv.ext.2.tf2.dll",
		"addons/sourcemod/extensions/sigsegv.autoload",
		"addons/sourcemod/gamedata/sigsegv/windows.txt",
		"addons/sourcemod/gamedata/sigsegv/misc.txt",
		"cfg/sigsegv_convars.cfg",
		"addons/.tf2ap-sigsegv-mvm.stamp",
	}
	keep := "addons/sourcemod/gamedata/tf2_archipelago.txt"
	for _, relative := range append(written, keep) {
		path := filepath.Join(modDir, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	removed, err := removeSigmod(modDir)
	if err != nil {
		t.Fatalf("removeSigmod: %v", err)
	}
	if !removed {
		t.Error("removeSigmod found no install to remove")
	}
	for _, relative := range written {
		if _, err := os.Stat(filepath.Join(modDir, filepath.FromSlash(relative))); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s is still there", relative)
		}
	}
	if _, err := os.Stat(filepath.Join(modDir, filepath.FromSlash(keep))); err != nil {
		t.Errorf("removeSigmod took %s with it: %v", keep, err)
	}

	// Twice is not an error, and the second time reports nothing to remove:
	// this runs on every start.
	removed, err = removeSigmod(modDir)
	if err != nil {
		t.Fatalf("removeSigmod on a clean directory: %v", err)
	}
	if removed {
		t.Error("removeSigmod reported an install it had already taken off")
	}
}
