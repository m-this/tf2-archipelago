package gamedata

// A trap is an item with a negative effect, a first-class Archipelago
// classification. Another player opens a chest and this team pays for it, which
// is what makes a trap worth having in a multiworld.

// TrapID identifies one trap. It alone decides the item id, so a value here
// never changes.
type TrapID uint16

const (
	// TrapTeamJarate soaks everyone on RED, bots included.
	TrapTeamJarate TrapID = 1

	// Contributed by Lee Harrold. Each hits everyone on RED, like Jarate.
	TrapTeamMadMilk        TrapID = 2
	TrapTeamOnFire         TrapID = 3
	TrapTeamSlowDown       TrapID = 4
	TrapTeamBleed          TrapID = 5
	TrapTeamMarkedForDeath TrapID = 6
	TrapTeamGas            TrapID = 7
)

// Trap is one negative effect. Key is the word on the wire to the plugin, which
// matches on it and never sees the id.
type Trap struct {
	ID   TrapID
	Key  string
	Name string
}

// Traps is every trap this project can fire. A seed only holds the ones its
// traps option names, and a seed that names none holds Team Jarate alone,
// which is every seed made before the others existed.
var Traps = []Trap{
	{TrapTeamJarate, "team_jarate", "Team Jarate"},
	{TrapTeamMadMilk, "team_mad_milk", "Team Mad Milk"},
	{TrapTeamOnFire, "team_on_fire", "Team On Fire"},
	{TrapTeamSlowDown, "team_slow_down", "Team Slow Down"},
	{TrapTeamBleed, "team_bleed", "Team Bleed"},
	{TrapTeamMarkedForDeath, "team_marked_for_death", "Team Marked For Death"},
	{TrapTeamGas, "team_gas", "Team Gas"},
}

// TrapDefault is what a seed that names no trap draws from.
const TrapDefault = TrapTeamJarate

// ItemName is what the multiworld calls the item that fires this trap.
func (t Trap) ItemName() string { return "Trap: " + t.Name }

// TrapByID resolves the payload of a trap item into the trap the plugin is told about.
func TrapByID(id TrapID) (Trap, bool) {
	for _, trap := range Traps {
		if trap.ID == id {
			return trap, true
		}
	}
	return Trap{}, false
}
