package settings

import (
	"reflect"
	"strings"
	"testing"
)

func TestApplyEnvOverridesTheFile(t *testing.T) {
	saved := Defaults()
	saved.APHost = "saved.example"
	saved.SrcdsPort = 27015

	t.Setenv("AP_HOST", "archipelago.gg")
	t.Setenv("AP_PORT", "38281")
	t.Setenv("SRCDS_BOT_TEAM_SIZE", "4")

	got := ApplyEnv(saved)
	if got.APHost != "archipelago.gg" {
		t.Errorf("AP_HOST: got %q, want archipelago.gg", got.APHost)
	}
	if got.APPort != 38281 {
		t.Errorf("AP_PORT: got %d, want 38281", got.APPort)
	}
	if got.SrcdsBotTeamSize != 4 {
		t.Errorf("SRCDS_BOT_TEAM_SIZE: got %d, want 4", got.SrcdsBotTeamSize)
	}
	if got.SrcdsPort != 27015 {
		t.Errorf("an unset variable changed SrcdsPort to %d", got.SrcdsPort)
	}
}

func TestApplyEnvBooleanSpellings(t *testing.T) {
	for _, value := range []string{"0", "false", "no", "off"} {
		t.Setenv("SRCDS_BOTS", value)
		s := Defaults()
		if ApplyEnv(s).SrcdsBots {
			t.Errorf("SRCDS_BOTS=%s left the bots on", value)
		}
	}
	for _, value := range []string{"1", "true", "yes", "on"} {
		t.Setenv("SRCDS_BOTS", value)
		s := Defaults()
		s.SrcdsBots = false
		if !ApplyEnv(s).SrcdsBots {
			t.Errorf("SRCDS_BOTS=%s left the bots off", value)
		}
	}
}

// A value that is not a number must not silently zero the setting: an empty
// AP_PORT would make the launcher ask for a room address it already had.
func TestApplyEnvIgnoresGarbageNumbers(t *testing.T) {
	t.Setenv("AP_PORT", "")
	s := Defaults()
	s.APPort = 38281
	if got := ApplyEnv(s).APPort; got != 38281 {
		t.Errorf("empty AP_PORT changed the port to %d", got)
	}
}

// Every field the launcher saves has to be reachable from the environment, so a
// shortcut, a .bat or a CI job can set any of them. A field added without a
// variable fails here rather than being discovered by somebody scripting it.
func TestEveryFieldHasAnEnvVar(t *testing.T) {
	// The variable that carries a whole room address covers three fields at
	// once, and the rest map one to one.
	byField := map[string]string{
		"InstallRoot":                "TF2AP_INSTALL_ROOT",
		"CommunityContentDir":        "TF2AP_COMMUNITY_CONTENT_DIR",
		"CommunityPacks":             "TF2AP_COMMUNITY_PACKS",
		"TestMode":                   "TF2AP_TEST_MODE",
		"ArchipelagoDir":             "TF2AP_ARCHIPELAGO_DIR",
		"APHost":                     "AP_HOST",
		"APPort":                     "AP_PORT",
		"APTls":                      "AP_TLS",
		"APSlotName":                 "AP_SLOT_NAME",
		"APPassword":                 "AP_PASSWORD",
		"SrcdsHostname":              "SRCDS_HOSTNAME",
		"SrcdsRconPw":                "SRCDS_RCONPW",
		"SrcdsPw":                    "SRCDS_PW",
		"SrcdsPort":                  "SRCDS_PORT",
		"SrcdsMaxPlayers":            "SRCDS_MAXPLAYERS",
		"SrcdsStartMission":          "SRCDS_START_MISSION",
		"SrcdsStartMap":              "SRCDS_STARTMAP",
		"SrcdsToken":                 "SRCDS_TOKEN",
		"SrcdsReach":                 "SRCDS_REACH",
		"SrcdsAdminSteamIDs":         "SRCDS_ADMIN_STEAMIDS",
		"SrcdsBots":                  "SRCDS_BOTS",
		"SrcdsBotTeamSize":           "SRCDS_BOT_TEAM_SIZE",
		"SrcdsBotClassBlacklist":     "SRCDS_BOT_CLASS_BLACKLIST",
		"SrcdsBotLoadouts":           "SRCDS_BOT_LOADOUTS",
		"SrcdsBotTeamComp":           "SRCDS_BOT_TEAM_COMP",
		"SrcdsBotSeatLoadouts":       "SRCDS_BOT_SEAT_LOADOUTS",
		"SrcdsBotHats":               "SRCDS_BOT_HATS",
		"SrcdsBotHatEffects":         "SRCDS_BOT_HAT_EFFECTS",
		"BotUpgradesChat":            "TF2AP_BOT_UPGRADES_CHAT",
		"MvmMissionCount":            "MVM_MISSION_COUNT",
		"MvmDifficulty":              "MVM_DIFFICULTY",
		"MvmGoal":                    "MVM_GOAL",
		"MvmMissionsanityPct":        "MVM_MISSIONSANITY_PERCENTAGE",
		"MvmDeathLink":               "MVM_DEATH_LINK",
		"MvmExcludedMissions":        "MVM_EXCLUDED_MISSIONS",
		"MvmStartMission":            "MVM_START_MISSION",
		"MvmStartClass":              "MVM_START_CLASS",
		"MvmMissionTicketImportance": "MVM_MISSION_TICKET_IMPORTANCE",
		"MvmClassUnlockImportance":   "MVM_CLASS_UNLOCK_IMPORTANCE",
		"MvmWeaponSlotImportance":    "MVM_WEAPON_SLOT_IMPORTANCE",
		"MvmWeaponBuffImportance":    "MVM_WEAPON_BUFF_IMPORTANCE",
		"MvmCashRewards":             "MVM_CASH_REWARDS",
		"MvmWeaponBuffPct":           "MVM_WEAPON_BUFF_PERCENTAGE",
		"MvmWeaponBuffStackChance":   "MVM_WEAPON_BUFF_STACK_CHANCE",
		"MetricsPort":                "BRIDGE_METRICS_PORT",
		"SrcdsBluHealthPct":          "SRCDS_BLU_HEALTH_PCT",
	}
	// Fields kept only to read a config file written by an older build. They
	// are never saved and never asked for, so there is nothing to set.
	legacy := map[string]bool{"SrcdsLanLegacy": true}

	// Fields the config file holds and the environment does not. A saved team
	// is something somebody named in front of the window; a compose stack
	// names its team in SRCDS_BOT_TEAM_COMP and has nowhere to click. A built
	// loadout is the same: four item indexes and a name, put together in a
	// menu, and a seat names one with the custom: prefix in the team it is
	// already setting.
	windowOnly := map[string]bool{
		"SrcdsBotTeamPresets":    true,
		"SrcdsBotCustomLoadouts": true,
	}

	known := map[string]bool{}
	for _, name := range EnvNames {
		known[name] = true
	}

	for structField := range reflect.TypeFor[Settings]().Fields() {
		field := structField.Name
		if legacy[field] || windowOnly[field] {
			continue
		}
		name, ok := byField[field]
		if !ok {
			t.Errorf("%s has no environment variable; add one to ApplyEnv and to this table", field)
			continue
		}
		if !known[name] {
			t.Errorf("%s maps to %s, which is not in EnvNames", field, name)
		}
	}

	// And the other way: a name in the list that nothing reads is a lie in the
	// output of -env.
	// AP_ROOM carries a whole address, and SRCDS_LAN is the older spelling of
	// two of the three reaches. Both are read, neither owns a field.
	mapped := map[string]bool{"AP_ROOM": true, "SRCDS_LAN": true}
	for _, name := range byField {
		mapped[name] = true
	}
	for _, name := range EnvNames {
		if !mapped[name] {
			t.Errorf("%s is listed but no field uses it", name)
		}
	}
}

func TestApplyEnvReadsTheLists(t *testing.T) {
	t.Setenv("SRCDS_BOT_CLASS_BLACKLIST", "sniper, spy")
	t.Setenv("SRCDS_BOT_LOADOUTS", "scout=milk,soldier=banner")
	t.Setenv("MVM_EXCLUDED_MISSIONS", "mvm_ghost_town_666")
	t.Setenv("SRCDS_STARTMAP", "mvm_bigrock")

	got := ApplyEnv(Defaults())
	if !reflect.DeepEqual(got.SrcdsBotClassBlacklist, []string{"sniper", "spy"}) {
		t.Errorf("blacklist = %v", got.SrcdsBotClassBlacklist)
	}
	if got.SrcdsBotLoadouts["scout"] != "milk" || got.SrcdsBotLoadouts["soldier"] != "banner" {
		t.Errorf("loadouts = %v", got.SrcdsBotLoadouts)
	}
	if !reflect.DeepEqual(got.MvmExcludedMissions, []string{"mvm_ghost_town_666"}) {
		t.Errorf("excluded = %v", got.MvmExcludedMissions)
	}
	if got.SrcdsStartMission != "mvm_bigrock" {
		t.Errorf("start mission from SRCDS_STARTMAP = %q", got.SrcdsStartMission)
	}
}

// The default reach leaves the local network, so a server that is meant to stay
// on it has to be able to say so, in either spelling.
func TestTheEnvironmentCanKeepTheServerLocal(t *testing.T) {
	cases := []struct {
		name string
		vars map[string]string
	}{
		{"SRCDS_REACH", map[string]string{"SRCDS_REACH": "lan"}},
		{"SRCDS_LAN", map[string]string{"SRCDS_LAN": "1"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for name, value := range c.vars {
				t.Setenv(name, value)
			}
			if got := ApplyEnv(Defaults()).SrcdsReach; got != ReachLan {
				t.Errorf("reach: got %q, want %q", got, ReachLan)
			}
		})
	}
}

/* A seat left on the draw survives the environment.
 *
 * The two seat lists are positional: entry n is seat n, and a seat the mod
 * draws for itself is an empty entry. Reading them the way the blacklist is
 * read drops the empties, which moves every seat after a draw up one, and a
 * compose stack that named seat 4 got it played as seat 1.
 */
func TestSeatListsKeepTheirHoles(t *testing.T) {
	t.Setenv("SRCDS_BOT_TEAM_COMP", ",engineer, ,heavyweapons,")
	t.Setenv("SRCDS_BOT_SEAT_LOADOUTS", ",ranger,,brass")
	t.Setenv("SRCDS_BOT_CLASS_BLACKLIST", "sniper, ,spy")

	s := ApplyEnv(Settings{})

	if got := strings.Join(s.SrcdsBotTeamComp, "|"); got != "|engineer||heavyweapons" {
		t.Errorf("team comp = %q", got)
	}
	if got := strings.Join(s.SrcdsBotSeatLoadouts, "|"); got != "|ranger||brass" {
		t.Errorf("seat loadouts = %q", got)
	}
	// The blacklist is a set and has no seats, so it drops its empties.
	if got := strings.Join(s.SrcdsBotClassBlacklist, "|"); got != "sniper|spy" {
		t.Errorf("blacklist = %q", got)
	}
}
