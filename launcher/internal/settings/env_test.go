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

// Compose passes SRCDS_START_MISSION empty when .env names none. That is not
// a mission, so the map SRCDS_STARTMAP names still decides where the run starts.
func TestAnEmptyStartMissionLeavesTheStartMap(t *testing.T) {
	t.Setenv("SRCDS_STARTMAP", "mvm_bigrock")
	t.Setenv("SRCDS_START_MISSION", "")
	if got := ApplyEnv(Defaults()).SrcdsStartMission; got == "" || got == Defaults().SrcdsStartMission {
		t.Errorf("start mission = %q, want a mission on mvm_bigrock", got)
	}
}
