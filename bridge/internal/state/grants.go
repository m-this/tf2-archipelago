package state

import (
	"slices"

	"git-ssh.croque.top/mathis/tf2-archipelago/gamedata"
)

// Grant is one received item, in the plugin's vocabulary. The plugin is never
// told an item id: it is told that a class is now playable, or that a loadout
// slot opened, and it applies that.
type Grant struct {
	// Seq is the grant's position in the run, counting from 1. The plugin
	// long-polls for anything past the sequence it last applied.
	Seq  int    `json:"seq"`
	Kind string `json:"kind"`

	// Key names what was granted: a class key, a slot key, a pop file. Empty
	// for a grant whose payload is a number.
	Key string `json:"key,omitempty"`

	// Name is the same thing spelled for a player, because the plugin
	// announces grants in chat and "Ctrl+Alt+Destruction" reads better than
	// "mvm_coaltown_advanced".
	Name string `json:"name,omitempty"`

	// Amount is the credits a cash bundle is worth. Zero elsewhere.
	Amount int `json:"amount,omitempty"`
}

// Unlocks is everything that should be true right now, which is what a plugin
// asks for after a reload or a map change.
//
// Credits are deliberately absent: a cash bundle is applied once when it
// arrives, and re-applying it on every map change would print money.
type Unlocks struct {
	Seq      int      `json:"seq"`
	Classes  []string `json:"classes"`
	Slots    []string `json:"slots"`
	Missions []string `json:"missions"`
}

// grantsFrom turns the ordered list of received item ids into grants. It is a
// pure function of that list, which is what makes the sequence numbers stable
// across a restart: the list is what gets persisted, the grants are derived.
//
// An id the tables do not know is skipped. That means a seed generated against
// a newer gamedata than this binary, which is a real possibility once anything
// ships, and dropping the item beats crashing the bridge mid-wave.
func grantsFrom(itemIDs []int64) []Grant {
	grants := make([]Grant, 0, len(itemIDs))
	slotsGranted := 0
	for _, id := range itemIDs {
		item, known := gamedata.ItemByID(id)
		if !known {
			continue
		}
		grant, ok := grantFor(item, slotsGranted)
		if !ok {
			continue
		}
		if item.Kind == gamedata.ItemWeaponSlot {
			slotsGranted++
		}
		grant.Seq = len(grants) + 1
		grants = append(grants, grant)
	}
	return grants
}

// grantFor is where the progressive weapon slot stops being progressive: copy
// n of one item name becomes the nth slot in gamedata's order.
func grantFor(item gamedata.Item, slotsGranted int) (Grant, bool) {
	switch item.Kind {
	case gamedata.ItemMissionTicket:
		mission, ok := gamedata.MissionByID(item.Mission)
		if !ok {
			return Grant{}, false
		}
		return Grant{Kind: item.Kind.Key(), Key: mission.PopFile, Name: mission.Name}, true

	case gamedata.ItemClass:
		class, ok := gamedata.ClassByID(item.Class)
		if !ok {
			return Grant{}, false
		}
		return Grant{Kind: item.Kind.Key(), Key: class.Key, Name: class.Name}, true

	case gamedata.ItemWeaponSlot:
		if slotsGranted >= len(gamedata.WeaponSlots) {
			return Grant{}, false
		}
		slot := gamedata.WeaponSlots[slotsGranted]
		return Grant{Kind: item.Kind.Key(), Key: slot.Key, Name: slot.Name}, true

	case gamedata.ItemCredits:
		return Grant{Kind: item.Kind.Key(), Amount: int(item.Credits), Name: item.Name}, true

	default:
		return Grant{}, false
	}
}

// unlocksFrom collapses the grant history into the set that should hold now,
// keeping the order they were received in.
func unlocksFrom(grants []Grant) Unlocks {
	unlocks := Unlocks{
		Seq:      len(grants),
		Classes:  []string{},
		Slots:    []string{},
		Missions: []string{},
	}
	for _, grant := range grants {
		switch grant.Kind {
		case gamedata.ItemClass.Key():
			unlocks.Classes = appendOnce(unlocks.Classes, grant.Key)
		case gamedata.ItemWeaponSlot.Key():
			unlocks.Slots = appendOnce(unlocks.Slots, grant.Key)
		case gamedata.ItemMissionTicket.Key():
			unlocks.Missions = appendOnce(unlocks.Missions, grant.Key)
		}
	}
	return unlocks
}

func appendOnce(keys []string, key string) []string {
	if slices.Contains(keys, key) {
		return keys
	}
	return append(keys, key)
}
