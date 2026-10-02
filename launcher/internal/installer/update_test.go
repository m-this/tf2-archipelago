package installer

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/m-this/tf2-archipelago/launcher/internal/assets"
)

// fakeSteamcmd writes a steamcmd.sh that records every command line and
// answers each app_update with the next of outcomes: "ok", "stuck" (state
// 0x6), or "lost" (state 0x602).
func fakeSteamcmd(t *testing.T, outcomes ...string) (steamcmdDir, gameDir string, calls func() []string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake SteamCMD is a shell script")
	}
	root := t.TempDir()
	steamcmdDir = filepath.Join(root, "steamcmd")
	gameDir = filepath.Join(root, "tf-dedicated")
	for _, dir := range []string{steamcmdDir, filepath.Join(gameDir, "steamapps")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(gameDir, "srcds_run"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(steamcmdDir, "outcomes"), []byte(strings.Join(outcomes, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	script := `#!/bin/sh
dir=$(dirname "$0")
echo "$*" >> "$dir/calls.log"
case "$*" in *app_update*) ;; *) exit 0 ;; esac
n=$(grep -c app_update "$dir/calls.log")
case "$(sed -n "${n}p" "$dir/outcomes")" in
stuck) echo "Error! App '232250' state is 0x6 after update job."; exit 8 ;;
lost) echo "Error! App '232250' state is 0x602 after update job."; exit 8 ;;
ok) echo "Success! App '232250' fully installed."; exit 0 ;;
*) echo "unexpected app_update number $n"; exit 99 ;;
esac
`
	if err := os.WriteFile(filepath.Join(steamcmdDir, "steamcmd.sh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	calls = func() []string {
		data, err := os.ReadFile(filepath.Join(steamcmdDir, "calls.log"))
		if err != nil {
			return nil
		}
		return strings.Split(strings.TrimSpace(string(data)), "\n")
	}
	return steamcmdDir, gameDir, calls
}

func writeManifest(t *testing.T, gameDir string) string {
	t.Helper()
	path := filepath.Join(gameDir, "steamapps", "appmanifest_232250.acf")
	if err := os.WriteFile(path, []byte(`"AppState" { "StateFlags" "6" }`), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func appUpdates(calls []string) []string {
	return slices.DeleteFunc(slices.Clone(calls), func(call string) bool {
		return !strings.Contains(call, "+app_update")
	})
}

func setAside(t *testing.T, gameDir string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(gameDir, "steamapps", "appmanifest_232250.acf.0x6-*"))
	if err != nil {
		t.Fatal(err)
	}
	return matches
}

func discard(string, ...any) {}

func TestUpdateGameUpdatesAnInstalledGame(t *testing.T) {
	_, gameDir, calls := fakeSteamcmd(t, "ok")

	if err := UpdateGame(context.Background(), filepath.Dir(gameDir), discard); err != nil {
		t.Fatalf("UpdateGame: %v", err)
	}
	updates := appUpdates(calls())
	if len(updates) != 1 {
		t.Fatalf("ran %d app_update, want 1: %v", len(updates), calls())
	}
	if strings.Contains(updates[0], "validate") {
		t.Errorf("the routine update validates 14 GB: %s", updates[0])
	}
	if !strings.Contains(updates[0], "+force_install_dir "+gameDir) {
		t.Errorf("the update does not target the game dir: %s", updates[0])
	}
}

func TestUpdateGameNeedsAnInstalledGame(t *testing.T) {
	steamcmdDir, gameDir, calls := fakeSteamcmd(t, "ok")
	if err := os.Remove(filepath.Join(gameDir, "srcds_run")); err != nil {
		t.Fatal(err)
	}

	if err := UpdateGame(context.Background(), filepath.Dir(steamcmdDir), discard); err == nil {
		t.Error("updated a game that is not installed")
	}
	if got := calls(); len(got) != 0 {
		t.Errorf("ran SteamCMD for a game that is not installed: %v", got)
	}
}

// Updating is a start's job. Ensure also runs from Settings, where a TF2
// download would be a surprise.
func TestEnsureDoesNotUpdateAnInstalledGame(t *testing.T) {
	steamcmdDir, gameDir, calls := fakeSteamcmd(t, "ok")
	withAssetVersions(t)
	modDir := filepath.Join(gameDir, "tf")
	writeFakeMetamod(t, modDir)
	writeFakeSourcemod(t, modDir)

	result, err := Ensure(context.Background(), filepath.Dir(steamcmdDir), nil, nil, discard)
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if !result.Done.GameInstalled {
		t.Error("Ensure did not see the installed game")
	}
	if got := calls(); len(got) != 0 {
		t.Errorf("Ensure ran SteamCMD on an installed game: %v", got)
	}
}

func withAssetVersions(t *testing.T) {
	t.Helper()
	for _, version := range []*string{
		&assets.SourcemodVersion, &assets.MetamodVersion, &assets.RipextVersion,
		&assets.ArchipelagoVersion, &assets.SigsegvMVMVersion, &assets.SigsegvMVMSHA256,
	} {
		old := *version
		*version = "test"
		t.Cleanup(func() { *version = old })
	}
}

func writeFakeMetamod(t *testing.T, modDir string) {
	t.Helper()
	for _, relative := range metamodFiles(runtime.GOOS) {
		path := filepath.Join(modDir, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("metamod"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestUpdateGameRecoversOnceFromStateZeroX6(t *testing.T) {
	steamcmdDir, gameDir, calls := fakeSteamcmd(t, "stuck", "ok")
	manifest := writeManifest(t, gameDir)

	if err := updateGame(context.Background(), steamcmdDir, gameDir, discard); err != nil {
		t.Fatalf("updateGame: %v", err)
	}
	if _, err := os.Stat(manifest); err == nil {
		t.Error("the stuck manifest is still in place")
	}
	if aside := setAside(t, gameDir); len(aside) != 1 {
		t.Errorf("want the manifest kept aside once, got %v", aside)
	}
	got := calls()
	if n := len(appUpdates(got)); n != 2 {
		t.Fatalf("ran %d app_update, want 2: %v", n, got)
	}
	// The warm-up between the two updates: a bare +quit, then a login.
	last := slices.IndexFunc(got[1:], func(call string) bool { return strings.Contains(call, "+app_update") }) + 1
	if want := []string{"+quit", "+login anonymous +quit"}; !slices.Equal(got[1:last], want) {
		t.Errorf("runs between the updates = %q, want %q", got[1:last], want)
	}
}

func TestUpdateGameStopsAfterASecondStateZeroX6(t *testing.T) {
	steamcmdDir, gameDir, calls := fakeSteamcmd(t, "stuck", "stuck", "ok")
	manifest := writeManifest(t, gameDir)

	err := updateGame(context.Background(), steamcmdDir, gameDir, discard)
	if err == nil {
		t.Fatal("a second 0x6 was not an error")
	}
	if !strings.Contains(err.Error(), manifest) {
		t.Errorf("the error does not name the manifest %s: %v", manifest, err)
	}
	if n := len(appUpdates(calls())); n != 2 {
		t.Errorf("ran %d app_update, want 2: %v", n, calls())
	}
	if aside := setAside(t, gameDir); len(aside) != 1 {
		t.Errorf("want the manifest kept aside once, got %v", aside)
	}
}

func TestUpdateGameCarriesOnWhenSteamCannotUpdate(t *testing.T) {
	steamcmdDir, gameDir, calls := fakeSteamcmd(t, "lost", "lost", "ok")
	manifest := writeManifest(t, gameDir)

	if err := updateGame(context.Background(), steamcmdDir, gameDir, discard); err != nil {
		t.Fatalf("updateGame: %v", err)
	}
	if n := len(appUpdates(calls())); n != 2 {
		t.Errorf("ran %d app_update, want 2: %v", n, calls())
	}
	if _, err := os.Stat(manifest); err != nil {
		t.Errorf("a 0x602 moved the manifest: %v", err)
	}
	if aside := setAside(t, gameDir); len(aside) != 0 {
		t.Errorf("a 0x602 set the manifest aside: %v", aside)
	}
}

func TestUpdateGameStartsTheInstalledBuildWhenTheStuckManifestIsMissing(t *testing.T) {
	steamcmdDir, gameDir, calls := fakeSteamcmd(t, "stuck", "ok")

	if err := updateGame(context.Background(), steamcmdDir, gameDir, discard); err != nil {
		t.Fatalf("updateGame: %v", err)
	}
	if n := len(appUpdates(calls())); n != 1 {
		t.Errorf("ran %d app_update, want 1: %v", n, calls())
	}
	if aside := setAside(t, gameDir); len(aside) != 0 {
		t.Errorf("set aside a manifest that was not there: %v", aside)
	}
}

func TestUpdateGameStopsWhenCancelled(t *testing.T) {
	steamcmdDir, gameDir, calls := fakeSteamcmd(t, "ok")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := updateGame(ctx, steamcmdDir, gameDir, discard); err == nil {
		t.Fatal("a cancelled update reported success")
	}
	if got := calls(); len(got) != 0 {
		t.Errorf("a cancelled update ran SteamCMD: %v", got)
	}
}

func TestStuckUpdateLine(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"Error! App '232250' state is 0x6 after update job.", true},
		{"Error! App '232250' state is 0x6", true},
		{"Error! App '232250' state is 0x602 after update job.", false},
		{"Error! App '232250' state is 0x202 after update job.", false},
		{"Success! App '232250' fully installed.", false},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := stuckUpdate(tt.line); got != tt.want {
				t.Errorf("stuckUpdate(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}
