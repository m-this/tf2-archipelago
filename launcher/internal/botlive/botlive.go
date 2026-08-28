/*
Package botlive applies a bot team to a server that is already running.

The Bots tab in the window and the one in the terminal edit the same team the
settings hold, and both apply it the same way: rewrite the files the mod reads
off disk, then tell the mod over RCON to pick the team up. This package is that
sequence, so neither interface owns it and a change lands in both.

The order matters. The loadout file is read when the mod is told to reseat, so
it has to be on disk first, or the team comes back holding what it held before.
*/
package botlive

import (
	"fmt"
	"reflect"

	"github.com/m-this/tf2-archipelago/launcher/internal/botloadout"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// Announcement is what the game chat says the moment a new team is applied, so
// the player knows the server took the save before the bots come back.
const Announcement = "Bot team changed, loading..."

/*
Commands is what to send, in order, to move a running server from one team to
another.

The blacklist goes with the composition because the two answer one question
between them: the composition names the seats it knows and the blacklist decides
what the mod draws for the rest.

Retyping the lineup is enough on its own. The mod acts on that convar when it
changes: it kicks the bots whose class the new list no longer asks for and keeps
the ones it does, so a one-seat change costs one bot its upgrades rather than
six.

sm_redbots_reseat is the blunt one, and only goes when the weapons moved. It
recycles the whole team, which a loadout change needs and a lineup change does
not, because a weapon is handed out on the way in and never again.
*/
func Commands(before, after settings.Settings) []string {
	var out []string
	// First, so the player reads it while the mod is still rebuilding the team.
	if teamMoved(before, after) {
		out = append(out, "say "+Announcement)
	}
	out = append(out, convars(after)...)
	if loadoutFile(before) != loadoutFile(after) {
		out = append(out, "sm_redbots_reseat")
	}
	return out
}

// teamMoved is whether these two settings ask for a different team at all. A
// save that changed nothing about the bots has nothing to announce.
func teamMoved(before, after settings.Settings) bool {
	return !reflect.DeepEqual(teamOf(before), teamOf(after))
}

// teamOf is the part of the settings Commands sends, so a save that left the
// bots alone compares equal.
func teamOf(s settings.Settings) []string {
	return append(convars(s), loadoutFile(s))
}

// convars is the lineup as the mod holds it, which is everything Commands has
// to retype for the team to change.
func convars(s settings.Settings) []string {
	return []string{
		fmt.Sprintf("sm_redbots_manager_defender_team_size %d", s.SrcdsBotTeamSize),
		fmt.Sprintf("sm_redbots_manager_class_blacklist %q", botloadout.Blacklist(s.SrcdsBotClassBlacklist)),
		fmt.Sprintf("sm_redbots_manager_team_composition %q", botloadout.Composition(s.SrcdsBotTeamComp, s.SrcdsBotClassBlacklist)),
		fmt.Sprintf("sm_redbots_manager_use_custom_loadouts %d", customLoadouts(s)),
	}
}

// loadoutFile is what the mod would read off disk for these settings, which is
// the only thing sm_redbots_reseat exists to pick up.
func loadoutFile(s settings.Settings) string {
	return LibraryOf(s).Render(s.SrcdsBotLoadouts, botloadout.Seats(s.SrcdsBotTeamComp, s.SrcdsBotSeatLoadouts))
}

// LibraryOf is the loadouts these settings can offer: the built-in presets and
// whatever the player has built. One place builds it, so the file, the tab and
// the convar cannot disagree about what a custom key means.
func LibraryOf(s settings.Settings) botloadout.Library {
	return botloadout.Library{Built: s.SrcdsBotCustomLoadouts}
}

// customLoadouts is whether the mod should read the loadout file at all, on the
// same terms the file is written on: it is removed when nothing is custom, and
// the convar has to agree or the mod looks for a file that is not there.
func customLoadouts(s settings.Settings) int {
	seats := botloadout.Seats(s.SrcdsBotTeamComp, s.SrcdsBotSeatLoadouts)
	if LibraryOf(s).Anything(s.SrcdsBotLoadouts, seats) {
		return 1
	}
	return 0
}

/*
	LiveOnly is whether the only thing that moved between these two settings is

the bot team, which is the case the running server does not have to restart for.

Everything else the launcher writes reaches the game through the command line or
through server.cfg, and the server reads both once at startup. The bot team is
the exception: the mod re-reads its lineup from a convar and its weapons from a
file, so Commands can hand it over without the mission ending.

Written as a comparison of everything else, not of the team: a setting added
later is not live until somebody says so, which is the safe way round.
*/
func LiveOnly(before, after settings.Settings) bool {
	if reflect.DeepEqual(before, after) {
		return false
	}
	return reflect.DeepEqual(withoutTeam(before), withoutTeam(after))
}

// withoutTeam is these settings with the bot team taken out, so what is left
// is everything a restart is still the only way to change.
func withoutTeam(s settings.Settings) settings.Settings {
	s.SrcdsBotTeamComp = nil
	s.SrcdsBotSeatLoadouts = nil
	s.SrcdsBotLoadouts = nil
	s.SrcdsBotClassBlacklist = nil
	s.SrcdsBotTeamSize = 0
	// The saved teams are the launcher's own list and reach the server through
	// nothing at all, so naming one is never a reason to restart either.
	s.SrcdsBotTeamPresets = nil
	return s
}
