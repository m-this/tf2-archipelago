package settings

import (
	"reflect"
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
		"InstallRoot":         "TF2AP_INSTALL_ROOT",
		"TestMode":            "TF2AP_TEST_MODE",
		"ArchipelagoDir":      "TF2AP_ARCHIPELAGO_DIR",
		"APHost":              "AP_HOST",
		"APPort":              "AP_PORT",
		"APTls":               "AP_TLS",
		"APSlotName":          "AP_SLOT_NAME",
		"APPassword":          "AP_PASSWORD",
		"SrcdsHostname":       "SRCDS_HOSTNAME",
		"SrcdsRconPw":         "SRCDS_RCONPW",
		"SrcdsPw":             "SRCDS_PW",
		"SrcdsPort":           "SRCDS_PORT",
		"SrcdsMaxPlayers":     "SRCDS_MAXPLAYERS",
		"SrcdsStartMap":       "SRCDS_STARTMAP",
		"SrcdsToken":          "SRCDS_TOKEN",
		"SrcdsLan":            "SRCDS_LAN",
		"SrcdsAdminSteamIDs":  "SRCDS_ADMIN_STEAMIDS",
		"SrcdsBots":           "SRCDS_BOTS",
		"SrcdsBotTeamSize":    "SRCDS_BOT_TEAM_SIZE",
		"MvmMissionCount":     "MVM_MISSION_COUNT",
		"MvmDifficulty":       "MVM_DIFFICULTY",
		"MvmGoal":             "MVM_GOAL",
		"MvmMissionsanityPct": "MVM_MISSIONSANITY_PERCENTAGE",
		"MvmDeathLink":        "MVM_DEATH_LINK",
		"MetricsPort":         "BRIDGE_METRICS_PORT",
	}
	known := map[string]bool{}
	for _, name := range EnvNames {
		known[name] = true
	}

	for structField := range reflect.TypeFor[Settings]().Fields() {
		field := structField.Name
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
	mapped := map[string]bool{"AP_ROOM": true}
	for _, name := range byField {
		mapped[name] = true
	}
	for _, name := range EnvNames {
		if !mapped[name] {
			t.Errorf("%s is listed but no field uses it", name)
		}
	}
}
