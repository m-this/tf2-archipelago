package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/m-this/tf2-archipelago/gamedata"
)

// Uses a temp HOME so it does not touch the operator's real config.
func TestSaveLoadRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("APPDATA", filepath.Join(t.TempDir(), "AppData", "Roaming"))

	s := Defaults()
	s.SrcdsRconPw = "hunter2"
	s.APPort = 12345
	s.APHost = "archipelago.gg"
	s.APRoomURL = "https://archipelago.gg/room/demo-room"
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
	if loaded.APRoomURL != s.APRoomURL {
		t.Errorf("AP room URL: got %q, want %q", loaded.APRoomURL, s.APRoomURL)
	}
	if loaded.APTls != s.APTls {
		t.Errorf("AP TLS: got %v, want %v", loaded.APTls, s.APTls)
	}
}

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

// The config file is 0600: it holds the RCON password.
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

// A file from before the start mission existed names a map. The mission the
// server starts on is then that map's first mission.
func TestLoadMigratesTheStartMap(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("APPDATA", filepath.Join(t.TempDir(), "AppData", "Roaming"))

	path, _ := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"srcds_start_map": "mvm_coaltown"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.SrcdsStartMission != "mvm_coaltown" {
		t.Errorf("start mission = %q, want the first Coal Town mission", loaded.SrcdsStartMission)
	}
	if loaded.SrcdsStartMap != "" {
		t.Errorf("the start map survived the migration: %q", loaded.SrcdsStartMap)
	}
}

func TestLoadMovesAnUnsupportedCommunityStartToSafeDefaults(t *testing.T) {
	var unsupported string
	for _, mission := range gamedata.Missions {
		if gamedata.IsCommunityMission(mission.ID) && !gamedata.IsPlayableMission(mission.ID) {
			unsupported = mission.PopFile
			break
		}
	}
	if unsupported == "" {
		t.Fatal("the catalog has no unavailable community mission")
	}
	body := fmt.Sprintf(`{
		"srcds_start_mission":%q,
		"mvm_start_mission":%q
	}`, unsupported, unsupported)
	loaded, err := parse([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if loaded.SrcdsStartMission != Defaults().SrcdsStartMission {
		t.Errorf("server start = %q, want %q", loaded.SrcdsStartMission, Defaults().SrcdsStartMission)
	}
	if loaded.MvmStartMission != "" {
		t.Errorf("run still starts on unsupported mission %q", loaded.MvmStartMission)
	}
}

func TestLoadMovesARemovedStartMissionToSafeDefaults(t *testing.T) {
	loaded, err := parse([]byte(`{
		"srcds_start_mission":"mvm_removed_mission",
		"mvm_start_mission":"mvm_removed_mission"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if loaded.SrcdsStartMission != Defaults().SrcdsStartMission {
		t.Errorf("server start = %q, want %q", loaded.SrcdsStartMission, Defaults().SrcdsStartMission)
	}
	if loaded.MvmStartMission != "" {
		t.Errorf("run still starts on removed mission %q", loaded.MvmStartMission)
	}
}

// A config file written before the appearance switches existed does not
// mention them, and a bool nobody wrote reads back as false. Every install that
// had ever saved a setting opened the next version with the bots undressed.
func TestAConfigThatPredatesTheAppearanceSwitchesGetsTheDefaults(t *testing.T) {
	s, err := parse([]byte(`{"srcds_bot_team_size": 6}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if !s.SrcdsBotHats || !s.SrcdsBotHatEffects {
		t.Errorf("hats %v, effects %v; want both on", s.SrcdsBotHats, s.SrcdsBotHatEffects)
	}
}

// A file that says no is not a file that says nothing: unticking one has to
// survive the next start.
func TestTheAppearanceSwitchesStayOffWhenTheFileSaysSo(t *testing.T) {
	s, err := parse([]byte(`{"srcds_bot_hats": false, "srcds_bot_hat_effects": true}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if s.SrcdsBotHats {
		t.Error("hats were off in the file and came back on")
	}
	if !s.SrcdsBotHatEffects {
		t.Error("effects were on in the file and came back off")
	}
}

func TestOldConfigGetsRewardDefaults(t *testing.T) {
	s, err := parse([]byte(`{"mvm_mission_count": 8}`))
	if err != nil {
		t.Fatal(err)
	}
	if s.MvmMissionTicketImportance != "progression" ||
		s.MvmClassUnlockImportance != "progression" ||
		s.MvmWeaponSlotImportance != "progression" ||
		s.MvmWeaponBuffImportance != "useful" {
		t.Errorf("reward importance defaults = %+v", s)
	}
	if s.MvmCashRewards || s.MvmWeaponBuffPct != 75 || s.MvmWeaponBuffStackChance != 25 || s.MvmTrapPct != 1 {
		t.Errorf("reward defaults: cash=%v, buffs=%d, stack=%d, traps=%d", s.MvmCashRewards, s.MvmWeaponBuffPct, s.MvmWeaponBuffStackChance, s.MvmTrapPct)
	}
}

func TestExplicitZeroRewardPercentagesSurvive(t *testing.T) {
	s, err := parse([]byte(`{"mvm_weapon_buff_percentage": 0, "mvm_weapon_buff_stack_chance": 0, "mvm_trap_percentage": 0}`))
	if err != nil {
		t.Fatal(err)
	}
	if s.MvmWeaponBuffPct != 0 || s.MvmWeaponBuffStackChance != 0 || s.MvmTrapPct != 0 {
		t.Errorf("explicit zeros became buffs=%d, stack=%d, traps=%d", s.MvmWeaponBuffPct, s.MvmWeaponBuffStackChance, s.MvmTrapPct)
	}
}

func TestOldConfigGetsMissionModifierBounds(t *testing.T) {
	s, err := parse([]byte(`{"mvm_mission_count": 8}`))
	if err != nil {
		t.Fatal(err)
	}
	if s.MvmModifierMin != 1 || s.MvmModifierMax != 2 {
		t.Errorf("modifier bounds = %d..%d, want 1..2", s.MvmModifierMin, s.MvmModifierMax)
	}
}

func TestExplicitZeroMissionModifierBoundsSurvive(t *testing.T) {
	s, err := parse([]byte(`{"mvm_minimum_mission_modifiers": 0, "mvm_maximum_mission_modifiers": 0}`))
	if err != nil {
		t.Fatal(err)
	}
	if s.MvmModifierMin != 0 || s.MvmModifierMax != 0 {
		t.Errorf("modifier bounds = %d..%d, want 0..0", s.MvmModifierMin, s.MvmModifierMax)
	}
}

/*
	An install folder the launcher could not act on is refused at Save.

It is a setting a player can type into now, and the two ways to get it wrong
both go wrong somewhere else: an empty one sends MkdirAll at the process's
working directory, and a relative one lands wherever the .exe was started from.
Neither failure names the box that caused it, which is the whole reason the
settings screen refuses it instead.
*/
func TestPersistRefusesAnInstallFolderItCannotUse(t *testing.T) {
	for _, bad := range []struct{ name, root string }{
		{"empty", ""},
		{"only spaces", "   "},
		{"relative", "tf2-archipelago"},
	} {
		t.Run(bad.name, func(t *testing.T) {
			s := Defaults()
			s.InstallRoot = bad.root
			if _, err := Persist(s); err == nil {
				t.Errorf("Persist took an install folder of %q", bad.root)
			}
		})
	}
}

// And it takes a real one, writing the file where Path says and filling in the
// RCON password on the way. What comes back is what was written, not what went
// in: a caller that kept its own copy would hold a password the server does not
// have.
func TestPersistWritesAndReturnsWhatItWrote(t *testing.T) {
	// Both, because Path reads os.UserConfigDir and that is a different
	// variable on Windows than everywhere else.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("AppData", t.TempDir())

	s := Defaults()
	s.InstallRoot = t.TempDir()
	s.SrcdsRconPw = ""

	written, err := Persist(s)
	if err != nil {
		t.Fatalf("Persist: %v", err)
	}
	if written.SrcdsRconPw == "" {
		t.Error("Persist returned settings with no RCON password, so the launcher cannot drive its own server")
	}

	path, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Persist reported success and wrote no file: %v", err)
	}
	if !strings.Contains(string(body), written.SrcdsRconPw) {
		t.Error("the file on disk does not hold the password Persist handed back")
	}
}

// A settings file written by 1.16.0 holds a boolean here, and losing the file
// over it would take the room and the bots with it.
func TestClassSlotsReadsTheBooleanItUsedToBe(t *testing.T) {
	for _, one := range []struct {
		body string
		want ClassSlots
	}{
		{`{"mvm_class_weapon_slots": true}`, ClassSlotsProgressive},
		{`{"mvm_class_weapon_slots": false}`, ClassSlotsOff},
		{`{"mvm_class_weapon_slots": "any_order"}`, ClassSlotsAnyOrder},
		{`{"mvm_class_weapon_slots": "nonsense"}`, ClassSlotsOff},
		{`{}`, ""},
	} {
		var s Settings
		if err := json.Unmarshal([]byte(one.body), &s); err != nil {
			t.Fatalf("%s: %v", one.body, err)
		}
		if s.MvmClassWeaponSlots != one.want {
			t.Errorf("%s read as %q, want %q", one.body, s.MvmClassWeaponSlots, one.want)
		}
	}
}

func TestClassSlotsOnCoversBothPerClassModes(t *testing.T) {
	if !ClassSlotsProgressive.On() || !ClassSlotsAnyOrder.On() {
		t.Error("a per-class mode did not count as per class")
	}
	if ClassSlotsOff.On() || ClassSlots("").On() {
		t.Error("off counted as per class")
	}
}

// A settings file from before the six traps after Team Jarate never mentions
// mvm_traps. It has to write the player file it always meant: Jarate alone.
func TestASettingsFileFromBeforeTheNewTrapsStillMeansJarateAlone(t *testing.T) {
	s, err := parse([]byte(`{"mvm_mission_count": 8, "mvm_trap_percentage": 20}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := TrapKeys(s); !slices.Equal(got, []string{"team_jarate"}) {
		t.Errorf("an old file draws %v, want team_jarate alone", got)
	}
	if yaml := PlayerYAML(s, ""); !strings.Contains(yaml, "  trap_percentage: 20\n  traps:\n    - \"team_jarate\"\n") {
		t.Errorf("an old file writes traps as:\n%s", yaml)
	}
}

func TestTrapKeys(t *testing.T) {
	tests := []struct {
		name  string
		saved []string
		want  []string
	}{
		{"none means Jarate alone", nil, []string{"team_jarate"}},
		{"an empty list means Jarate alone", []string{}, []string{"team_jarate"}},
		{"only keys gamedata knows", []string{"team_gas", "team_nonsense"}, []string{"team_gas"}},
		{"nothing known means Jarate alone", []string{"team_nonsense"}, []string{"team_jarate"}},
		{"in gamedata's order", []string{"team_gas", "team_jarate", "team_bleed"}, []string{"team_jarate", "team_bleed", "team_gas"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Defaults()
			s.MvmTraps = tt.saved
			if got := TrapKeys(s); !slices.Equal(got, tt.want) {
				t.Errorf("TrapKeys(%v) = %v, want %v", tt.saved, got, tt.want)
			}
		})
	}
}

func TestExplicitlyNoTrapSurvivesASave(t *testing.T) {
	s, err := parse([]byte(`{"mvm_traps": []}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.MvmTraps) != 0 {
		t.Errorf("an empty trap list loaded as %v", s.MvmTraps)
	}
}
