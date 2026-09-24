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
	"slices"
	"strconv"
	"strings"

	"github.com/m-this/tf2-archipelago/gamedata"
	"github.com/m-this/tf2-archipelago/launcher/internal/botcards"
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
	moved := TeamMoved(before, after)
	oldCards, newCards := cardStates(before), cardStates(after)
	if (len(oldCards) > 0 || len(newCards) > 0) && !manualSeatsMoved(before, after) {
		// Reload first, while each surviving bot still has its old name. The
		// mod rebinds that name to the card's new priority seat without a kick.
		out = append(out, "sm_redbots_rebind_seats")
		oldSeats := SeatsOf(before)
		for _, seat := range slices.Backward(oldSeats) {
			name := seat.Name
			old, isCard := oldCards[name]
			if !isCard {
				continue
			}
			if next, kept := newCards[name]; !kept || !reflect.DeepEqual(old, next) {
				out = append(out, "sm_ap_botcards_evict "+strconv.Quote(name))
			}
		}
		out = append(out, convars(after)...)
		// A priority drag may move an existing card below the human-adjusted
		// cutoff even though its identity and loadout did not change. Rebind
		// alone preserves that bot incorrectly; reconcile frees only the lower
		// priority seats needed by selected cards now above the cutoff.
		out = append(out, "sm_ap_botcards_reconcile")
		// Do not claim success in chat until the live reconciliation command
		// has answered. The admin stops sending on an RCON refusal.
		if moved {
			out = append(out, "say "+Announcement)
		}
		return out
	}
	// Legacy manual seats retain their existing early announcement. The
	// defender manager can defer their full reseat until a wave break.
	if moved {
		out = append(out, "say "+Announcement)
	}
	out = append(out, convars(after)...)
	switch {
	case weaponsFile(before) != weaponsFile(after):
		// A reseat reads both files on the way and names every bot it builds,
		// so it covers a name that moved with the weapons.
		out = append(out, "sm_redbots_reseat")
	case namesMoved(before, after):
		out = append(out, "sm_redbots_reload_names")
	}
	return out
}

type cardState struct {
	Seat     botloadout.Seat
	Weapons  botloadout.Loadout
	Fallback string
}

func cardStates(s settings.Settings) map[string]cardState {
	out := map[string]cardState{}
	for _, seat := range SeatsOf(s) {
		if !seat.Card {
			continue
		}
		class, ok := botloadout.ClassByKey(seat.Class)
		if !ok {
			continue
		}
		out[seat.Name] = cardState{seat, LibraryOf(s).Loadout(class, seat.Loadout), s.SrcdsBotLoadouts[seat.Class]}
	}
	return out
}

func manualSeatsMoved(before, after settings.Settings) bool {
	manual := func(s settings.Settings) []botloadout.Seat {
		var seats []botloadout.Seat
		for _, seat := range SeatsOf(s) {
			if !seat.Card {
				seats = append(seats, seat)
			}
		}
		return seats
	}
	old, next := manual(before), manual(after)
	used := make([]bool, len(old))
	for _, seat := range next {
		found := false
		for i, previous := range old {
			if !used[i] && reflect.DeepEqual(previous, seat) {
				used[i], found = true, true
				break
			}
		}
		if !found {
			return true
		}
	}
	return len(next) > 0 && (!reflect.DeepEqual(before.SrcdsBotLoadouts, after.SrcdsBotLoadouts) ||
		!reflect.DeepEqual(before.SrcdsBotCustomLoadouts, after.SrcdsBotCustomLoadouts))
}

/*
namesMoved is whether the bots would be called anything different.

Kept apart from the weapons because the answers cost different things. A weapon
is handed out on the way in and never again, so changing one means recycling the
team and losing the upgrades it bought. A name is a string on a player, and
sm_redbots_reload_names sets it in place: renaming a seat between waves should
not cost the wave.
*/
func namesMoved(before, after settings.Settings) bool {
	return !slices.Equal(before.SrcdsBotSeatNames, after.SrcdsBotSeatNames) ||
		!slices.Equal(before.SrcdsBotNamesExcluded, after.SrcdsBotNamesExcluded) ||
		!slices.Equal(before.SrcdsBotNamesAdded, after.SrcdsBotNamesAdded)
}

// weaponsFile is the loadout file with the names taken out of it, which is what
// decides whether the team has to be rebuilt.
func weaponsFile(s settings.Settings) string {
	var cards strings.Builder
	for index, seat := range SeatsOf(s) {
		if card, ok := botcards.BySeat(seat.Class, seat.Name); ok {
			fmt.Fprintf(&cards, "%d:%s:%t:%t;", index, card.ID, seat.Robot, seat.Giant)
		}
	}
	s.SrcdsBotSeatNames = nil
	return loadoutFile(s) + cards.String()
}

// TeamMoved is whether these two settings ask for a different team at all. A
// save that changed nothing about the bots has nothing to announce and nothing
// to send.
func TeamMoved(before, after settings.Settings) bool {
	return !reflect.DeepEqual(teamOf(before), teamOf(after))
}

// teamOf is the part of the settings Commands sends, so a save that left the
// bots alone compares equal.
func teamOf(s settings.Settings) []string {
	return append(convars(s), loadoutFile(s),
		strings.Join(s.SrcdsBotNamesExcluded, ","), strings.Join(s.SrcdsBotNamesAdded, ","))
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
	return LibraryOf(s).Render(s.SrcdsBotLoadouts, SeatsOf(s))
}

// SeatsOf decorates named collectible seats with their gameplay card data.
// The exported file is shared by native installs and the Compose hot-apply.
func SeatsOf(s settings.Settings) []botloadout.Seat {
	seats := botloadout.Seats(s.SrcdsBotTeamComp, s.SrcdsBotSeatLoadouts, s.SrcdsBotSeatNames)
	for i := range seats {
		card, ok := botcards.BySeat(seats[i].Class, seats[i].Name)
		if !ok {
			continue
		}
		tier := card.Tier
		rolled, rolledTier, _, valid := botcards.ParseItemName(s.SrcdsBotCardRolls[card.ID])
		if s.MvmBotCards && !s.TestMode && (!valid || rolled.ID != card.ID) {
			continue
		}
		if valid && rolled.ID == card.ID {
			tier = rolledTier
		}
		seats[i].Giant = slices.Contains(s.SrcdsBotGiantCards, card.ID)
		seats[i].Robot = seats[i].Giant || !slices.Contains(s.SrcdsBotHumanCards, card.ID)
		seats[i].Card = true
		seats[i].Tier = tier.Stacks()
		seats[i].Cosmetic = card.Cosmetic
		seats[i].Unusual = tier == botcards.Legendary && card.Cosmetic != 0
		seats[i].UnusualEffect = card.UnusualEffect
		if seats[i].Unusual && seats[i].UnusualEffect == 0 {
			seats[i].UnusualEffect = 13
		}
		for _, id := range botcards.BuffsFor(card, tier) {
			buff, ok := gamedata.WeaponBuffByID(id)
			if ok && buff.Eligible {
				seats[i].Innates = append(seats[i].Innates, botloadout.Innate{
					Effect: int(buff.EffectID) - 1, Stacks: tier.Stacks(),
				})
			}
		}
	}
	return seats
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
	seats := SeatsOf(s)
	if LibraryOf(s).Anything(s.SrcdsBotLoadouts, seats) {
		return 1
	}
	return 0
}

/*
	WithoutTeam is these settings with the bot team taken out, so what is left is

everything the mod does not re-read.

There used to be a LiveOnly here that compared two of these and answered whether
the bot team was the only thing that moved. saveplan asks a wider question, of
which that was one answer, and asking it twice in two packages is how the two
drift apart.
*/
func WithoutTeam(s settings.Settings) settings.Settings {
	s.SrcdsBotTeamComp = nil
	s.SrcdsBotSeatLoadouts = nil
	s.SrcdsBotGiantCards = nil
	s.SrcdsBotHumanCards = nil
	s.SrcdsBotLoadouts = nil
	s.SrcdsBotClassBlacklist = nil
	s.SrcdsBotTeamSize = 0
	// The saved teams are the launcher's own list and reach the server through
	// nothing at all, so naming one is never a reason to restart either.
	s.SrcdsBotTeamPresets = nil
	return s
}
