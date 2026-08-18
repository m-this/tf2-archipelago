package settings

import "testing"

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
