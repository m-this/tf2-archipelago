package debugbundle

import (
	"archive/zip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	apruntime "github.com/m-this/tf2-archipelago/launcher/internal/runtime"
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

// The run before this one is in the bundle too. A player who hits a bug
// restarts the server before going to look for the button, so the run that
// broke is usually not the run the bundle is made from.
func TestWriteKeepsThePreviousRun(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, apruntime.LogFileName), "this run\n")
	write(t, filepath.Join(root, apruntime.LogPreviousName), "the run that broke\n")

	s := settings.Defaults()
	s.InstallRoot = root
	path, err := Write(s, nil, time.Date(2026, 8, 18, 1, 2, 3, 0, time.UTC))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	held := read(t, path)
	if !strings.Contains(held[apruntime.LogPreviousName], "the run that broke") {
		t.Errorf("the previous run is not in the bundle: %v", keys(held))
	}
}

// What the bridge says about the run, and what the bundle says when it says
// nothing. Both are worth having: no answer means the state came from nowhere,
// rather than the run being empty.
func TestWriteCarriesTheBridgeState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			_, _ = io.WriteString(w, `{"connected":true,"slot":"tf2","checks":7}`)
		case "/missions":
			_, _ = io.WriteString(w, `{"missions":[{"popfile":"mvm_decoy","cleared":true}]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	previous := bridgeURL
	bridgeURL = server.URL
	defer func() { bridgeURL = previous }()

	s := settings.Defaults()
	s.InstallRoot = t.TempDir()
	path, err := Write(s, nil, time.Date(2026, 8, 18, 1, 2, 3, 0, time.UTC))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	held := read(t, path)
	if !strings.Contains(held["bridge.json"], `"checks": 7`) {
		t.Errorf("the bundle does not carry the run: %q", held["bridge.json"])
	}
	if !strings.Contains(held["bridge.json"], "mvm_decoy") {
		t.Errorf("the bundle does not carry the missions: %q", held["bridge.json"])
	}
}

func TestWriteSaysWhenTheBridgeDidNotAnswer(t *testing.T) {
	previous := bridgeURL
	// Nothing listens here: a bundle made after everything was stopped.
	bridgeURL = "http://127.0.0.1:1"
	defer func() { bridgeURL = previous }()

	s := settings.Defaults()
	s.InstallRoot = t.TempDir()
	path, err := Write(s, nil, time.Date(2026, 8, 18, 1, 2, 3, 0, time.UTC))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if held := read(t, path); !strings.Contains(held["bridge.json"], "error") {
		t.Errorf("a refused bridge left no trace: %q", held["bridge.json"])
	}
}

// A crash that leaves no line in any log leaves a minidump, and that file is
// the only one naming the function the server died in. It was not collected,
// so the first crash report to arrive could not be taken any further.
func TestWriteCollectsTheCrashDumps(t *testing.T) {
	root := t.TempDir()
	game := filepath.Join(root, "tf-dedicated", "tf")
	write(t, filepath.Join(game, "crash_20260821.mdmp"), "MDMP fake")
	write(t, filepath.Join(root, "tf-dedicated", "srcds_20260821.mdmp"), "MDMP beside the exe")

	s := settings.Defaults()
	s.InstallRoot = root
	path, err := Write(s, nil, time.Date(2026, 8, 21, 23, 4, 35, 0, time.UTC))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	held := read(t, path)
	for _, name := range []string{"crashes/crash_20260821.mdmp", "crashes/srcds_20260821.mdmp"} {
		if _, ok := held[name]; !ok {
			t.Errorf("the bundle has no %s (it holds %v)", name, keys(held))
		}
	}
}

// Which defender bots were playing. A crash report that does not say cannot be
// read: the mod is where most of the crashes have been.
func TestTheSummaryNamesTheBotsVersion(t *testing.T) {
	s := settings.Defaults()
	s.InstallRoot = t.TempDir()
	path, err := Write(s, map[string]string{"defenderbots": "v2.0.0"}, time.Now())
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if held := read(t, path); !strings.Contains(held["summary.txt"], "v2.0.0") {
		t.Errorf("the summary does not name the bots: %q", held["summary.txt"])
	}
}

/*
An access violation Breakpad does not catch leaves its dump with Windows, not
beside the binary. k-kaneta's bundle carried two 0xc0000005 and no dump, which
is why apw-eei is still inference.
*/
func TestCrashDumpsComeFromWindowsErrorReportingToo(t *testing.T) {
	local := t.TempDir()

	wer := filepath.Join(local, "CrashDumps")
	if err := os.MkdirAll(wer, 0o755); err != nil {
		t.Fatal(err)
	}
	dump := filepath.Join(wer, "srcds.exe.1234.dmp")
	if err := os.WriteFile(dump, []byte("dump"), 0o600); err != nil {
		t.Fatal(err)
	}

	game := filepath.Join(t.TempDir(), "tf-dedicated", "tf")
	if err := os.MkdirAll(game, 0o755); err != nil {
		t.Fatal(err)
	}

	found := newestCrashDumps(game, wer, 3)
	if !slices.Contains(found, dump) {
		t.Errorf("the dump Windows wrote was not collected: %v", found)
	}
}

/*
Windows Error Reporting keeps a dump for every program on the machine. Two
bundles carried GameBar, Refunct and THPS12 dumps under crashes/, and the summary
told the reader to open them first, while the server's own crash had left none.
Only the game server's dumps come out of that directory.
*/
func TestOtherProgramsDumpsStayOutOfTheBundle(t *testing.T) {
	wer := filepath.Join(t.TempDir(), "CrashDumps")
	if err := os.MkdirAll(wer, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"GameBar.exe.14908-tail.dmp", "Refunct-Win32-Shipping.exe.14640-tail.dmp", "srcds.exe.1234.dmp", "tf2ap.exe.77.dmp"} {
		if err := os.WriteFile(filepath.Join(wer, name), []byte("dump"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	game := filepath.Join(t.TempDir(), "tf-dedicated", "tf")
	if err := os.MkdirAll(game, 0o755); err != nil {
		t.Fatal(err)
	}

	found := newestCrashDumps(game, wer, 5)
	if len(found) != 2 {
		t.Fatalf("collected %v, want only the game server's and the launcher's", found)
	}
	for _, path := range found {
		if strings.Contains(path, "GameBar") || strings.Contains(path, "Refunct") {
			t.Errorf("another program's dump was collected: %s", path)
		}
	}

	// Breakpad names its dumps beside the binary by a GUID, so the game
	// directories keep whatever they hold.
	guid := filepath.Join(game, "a1b2c3d4-0000-1111-2222-333344445555.mdmp")
	if err := os.WriteFile(guid, []byte("dump"), 0o600); err != nil {
		t.Fatal(err)
	}
	if found := newestCrashDumps(game, wer, 5); !slices.Contains(found, guid) {
		t.Errorf("Breakpad's dump was dropped: %v", found)
	}
}
