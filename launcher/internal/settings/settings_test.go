package settings

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSaveLoadRoundTrip writes a config, reads it back, and checks the values
// survive. Uses a temp HOME so it does not touch the operator's real config.
func TestSaveLoadRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("APPDATA", filepath.Join(t.TempDir(), "AppData", "Roaming"))

	s := Defaults()
	s.SrcdsRconPw = "hunter2"
	s.APPort = 12345
	s.APHost = "archipelago.gg"
	s.APTls = true
	s.SrcdsAdminSteamIDs = "76561198014216803"

	if err := Save(s); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.SrcdsRconPw != s.SrcdsRconPw {
		t.Errorf("RCON password: got %q, want %q", loaded.SrcdsRconPw, s.SrcdsRconPw)
	}
	if loaded.APPort != s.APPort {
		t.Errorf("AP port: got %d, want %d", loaded.APPort, s.APPort)
	}
	if loaded.APHost != s.APHost {
		t.Errorf("AP host: got %q, want %q", loaded.APHost, s.APHost)
	}
	if loaded.APTls != s.APTls {
		t.Errorf("AP TLS: got %v, want %v", loaded.APTls, s.APTls)
	}
}

// TestLoadMissingReturnsDefaults checks that a missing config file yields the
// defaults rather than an error.
func TestLoadMissingReturnsDefaults(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("APPDATA", filepath.Join(t.TempDir(), "AppData", "Roaming"))

	s, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	d := Defaults()
	if s.SrcdsPort != d.SrcdsPort {
		t.Errorf("SrcdsPort: got %d, want %d", s.SrcdsPort, d.SrcdsPort)
	}
	if s.MvmDifficulty != d.MvmDifficulty {
		t.Errorf("MvmDifficulty: got %q, want %q", s.MvmDifficulty, d.MvmDifficulty)
	}
}

// TestApplyDefaultsFillsZero checks that an old config missing a field gets the
// default for it.
func TestApplyDefaultsFillsZero(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("APPDATA", filepath.Join(t.TempDir(), "AppData", "Roaming"))

	s := Settings{SrcdsRconPw: "x"}
	if err := Save(s); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.SrcdsMaxPlayers != 32 {
		t.Errorf("SrcdsMaxPlayers: got %d, want 32 (default)", loaded.SrcdsMaxPlayers)
	}
}

// TestConfigFilePermissions checks the config file is 0600: it holds the RCON
// password.
func TestConfigFilePermissions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("APPDATA", filepath.Join(t.TempDir(), "AppData", "Roaming"))

	if err := Save(Defaults()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	path, _ := Path()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("config file mode: got %o, want 0600", info.Mode().Perm())
	}
}
