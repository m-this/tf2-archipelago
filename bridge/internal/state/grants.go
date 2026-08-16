package state

import (
	"slices"

	"git-ssh.croque.top/mathis/tf2-archipelago/gamedata"
)

// Grant is one received item in the plugin's vocabulary. The plugin is never
// told an item id, only that a class is playable or a loadout slot opened.
type Grant struct {
	// Seq is the item's position in what Archipelago has sent, counting from 1,
	// and the cursor the plugin long-polls on.
	//
	// It counts items rather than grants on purpose. An id this binary cannot
	// read is skipped, so counting grants would renumber every later one the
	// day a larger gamedata makes that id readable, and the plugin would
	// reapply some grants and miss others without noticing.
	Seq  int    `json:"seq"`
	Kind string `json:"kind"`

	// Key names what was granted: a class key, a slot key, a pop file. Empty
	// for a grant whose payload is a number.
	Key string `json:"key,omitempty"`

	// Name is the same thing spelled for a player; the plugin announces grants
	// in chat.
	Name string `json:"name,omitempty"`

	// Amount is the credits a cash bundle is worth. Zero elsewhere.
	Amount int `json:"amount,omitempty"`
}

// Unlocks is everything that should be true right now, which is what a plugin
// asks for after a reload or a map change. Credits are absent on purpose:
// re-applying a cash bundle on every map change would print money.
type Unlocks struct {
	Seq      int      `json:"seq"`
	Classes  []string `json:"classes"`
	Slots    []string `json:"slots"`
	Missions []string `json:"missions"`
}

// grantsFrom derives grants from the persisted item list, which is what keeps
// sequence numbers stable across a restart. An id the tables do not know means
// a seed from a newer gamedata, and skipping it beats crashing mid-wave. The
// skip leaves a gap in the sequence rather than moving what follows it.
func grantsFrom(itemIDs []int64) []Grant {
	grants := make([]Grant, 0, len(itemIDs))
	slotsGranted := 0
	for index, id := range itemIDs {
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
		grant.Seq = index + 1
		grants = append(grants, grant)
	}
	return grants
}

// grantFor is where the progressive weapon slot stops being progressive: copy n
// becomes the nth slot in gamedata's order.
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

// unlocksFrom collapses the grant history into the current set, in the order
// received. Seq is how far through the item list the set reaches, so it counts
// the same things Grant.Seq does.
func unlocksFrom(grants []Grant, itemCount int) Unlocks {
	unlocks := Unlocks{
		Seq:      itemCount,
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
