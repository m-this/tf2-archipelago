package provision

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func installWith(t *testing.T, mods []string, convars string) string {
	t.Helper()
	game := t.TempDir()
	cfg := filepath.Join(game, "cfg")
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfg, "sigsegv_convars.cfg")
	if err := os.WriteFile(path, []byte(convars), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := (Server{ServerCfg: "hostname x\n", Mods: mods}).Install(game); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

// SigMod's convar file runs after server.cfg, so the "1" it ships would undo
// the line server.cfg writes. Only that value changes; the rest of the file
// stays the operator's.
func TestInstallTakesREDOutOfSigModsRobotLimit(t *testing.T) {
	got := installWith(t, []string{"sigsegv-mvm"},
		"sig_mvm_robot_limit 1\nsig_mvm_robot_limit_fix_red \"1\"\n")
	if want := "sig_mvm_robot_limit 1\nsig_mvm_robot_limit_fix_red \"0\"\n"; got != want {
		t.Fatalf("convars = %q, want %q", got, want)
	}
}

func TestInstallLeavesSigModsConvarsAloneWhenSigModIsNotLoaded(t *testing.T) {
	got := installWith(t, nil, "sig_mvm_robot_limit_fix_red \"1\"\n")
	if !strings.Contains(got, `"1"`) {
		t.Fatalf("convars rewritten with SigMod not loaded: %q", got)
	}
}

func TestInstallReportsOnlyAServerCfgThatChanged(t *testing.T) {
	game := t.TempDir()
	server := Server{ServerCfg: "hostname x\n"}
	for i, want := range []bool{true, false} {
		changed, err := server.Install(game)
		if err != nil {
			t.Fatal(err)
		}
		if changed != want {
			t.Fatalf("install %d reported changed=%v, want %v", i+1, changed, want)
		}
	}
}
