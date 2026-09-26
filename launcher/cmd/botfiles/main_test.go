package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/m-this/tf2-archipelago/launcher/internal/botfiles"
)

// The seat a player named reaches the file the mod reads, and the exit line is
// the convar that makes the mod read it. Both halves failed in the stack: the
// file was never written, and nothing turned the convar on.
func TestASeatLoadoutIsWrittenAndAnnounced(t *testing.T) {
	root := staged(t)
	t.Setenv("SRCDS_BOT_TEAM_COMP", "pyro,engineer")
	t.Setenv("SRCDS_BOT_SEAT_LOADOUTS", "custom:burn,")
	t.Setenv("SRCDS_BOT_CUSTOM_LOADOUTS",
		`{"burn":{"Class":"pyro","Primary":594,"Second":1180,"Melee":-1,"PDA2":-1}}`)

	var out, errs bytes.Buffer
	if code := run([]string{"-root", root}, &out, &errs); code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, errs.String())
	}
	if got := strings.TrimSpace(out.String()); got != "1" {
		t.Fatalf("said %q, want 1 so the mod reads the file", got)
	}

	body, err := os.ReadFile(loadoutPath(root))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"seats"`, `"class"`, "pyro", "594", "1180"} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("loadout.cfg has no %q:\n%s", want, body)
		}
	}
}

/*
A stack that picked no loadout leaves the file the image shipped alone.

The mod ships an example there and reads it only when the convar says so, so
saying 0 is the whole of "nothing was chosen". Deleting the example to say it
would lose the one copy of the format anybody reads before editing by hand.
*/
func TestNoLoadoutLeavesTheShippedFileAndSaysSo(t *testing.T) {
	root := staged(t)
	shipped := []byte("// the example the mod ships\n")
	if err := os.WriteFile(loadoutPath(root), shipped, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SRCDS_BOT_TEAM_COMP", "pyro,engineer")

	var out, errs bytes.Buffer
	if code := run([]string{"-root", root}, &out, &errs); code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, errs.String())
	}
	if got := strings.TrimSpace(out.String()); got != "0" {
		t.Fatalf("said %q, want 0", got)
	}
	body, err := os.ReadFile(loadoutPath(root))
	if err != nil || !bytes.Equal(body, shipped) {
		t.Fatalf("the shipped example was rewritten: %s, %v", body, err)
	}
}

func TestLiveClearReplacesTheOldCardFile(t *testing.T) {
	root := staged(t)
	if err := os.WriteFile(loadoutPath(root), []byte(`"giant" "1"`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SRCDS_BOT_TEAM_COMP", "soldier")
	t.Setenv("SRCDS_BOT_SEAT_NAMES", "")
	var out, errs bytes.Buffer
	if code := run([]string{"-root", root, "-live"}, &out, &errs); code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, errs.String())
	}
	if got := strings.TrimSpace(out.String()); got != "0" {
		t.Fatalf("said %q, want 0", got)
	}
	body, err := os.ReadFile(loadoutPath(root))
	if err != nil || !strings.Contains(string(body), `"loadout"`) || strings.Contains(string(body), `"giant"`) {
		t.Fatalf("old giant card survived clearing: %s, %v", body, err)
	}
}

func TestARootThatWasNotGivenIsRefused(t *testing.T) {
	var out, errs bytes.Buffer
	if code := run(nil, &out, &errs); code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(errs.String(), "-root") {
		t.Fatalf("the refusal does not name the flag: %s", errs.String())
	}
}

func staged(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Dir(loadoutPath(root)), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func loadoutPath(root string) string { return botfiles.LoadoutPath(root) }
