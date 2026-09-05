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
)

// Trap is one negative effect. Key is the word on the wire to the plugin, which
// matches on it and never sees the id.
type Trap struct {
	ID   TrapID
	Key  string
	Name string
}

// Traps is every trap this project can fire. The other five from
// docs/en/spec.md are not built.
var Traps = []Trap{
	{TrapTeamJarate, "team_jarate", "Team Jarate"},
}

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
