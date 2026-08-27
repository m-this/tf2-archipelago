package state

import (
	"slices"

	"github.com/m-this/tf2-archipelago/gamedata"
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

	// Amount is the credits a cash bundle is worth. Zero elsewhere, and always
	// on the wire: SourcePawn errors out on a key that is not there, so an
	// omitted zero would abort the callback applying the grant.
	Amount int `json:"amount"`

	// OneShot marks an effect rather than state: applying it twice is not the
	// same as applying it once, so the bridge stops sending it once the plugin
	// acknowledges it. Off the wire on purpose; the plugin acknowledges
	// everything it applied and does not need to tell them apart.
	OneShot bool `json:"-"`
}

// Unlocks is everything that should be true right now, which is what a plugin
// asks for after a reload or a map change.
//
// Keyed by grant kind rather than by a field per kind: a kind added to gamedata
// then appears here with no change on either side of the wire, and a plugin too
// old to know it ignores it instead of failing to parse. Effects are absent,
// because they are not state. Re-applying a cash bundle on every map change
// would print money.
type Unlocks struct {
	// ResumeFrom is the sequence the plugin polls from next, and it is the
	// acknowledged one rather than how far the item list reaches.
	//
	// The difference is a lost effect. The unlock set carries state only, so a
	// cash bundle that arrived while no plugin was listening is not in it. A
	// cursor set to the length of the item list would sit above that bundle and
	// the bridge would never hand it over. Resuming from the acknowledged
	// sequence re-sends some state the plugin already has, which costs nothing,
	// and every effect it never got.
	ResumeFrom int                 `json:"resume_from"`
	ByKind     map[string][]string `json:"unlocks"`
}

// Of is the keys held for one kind, and nothing for a kind that is not state.
func (u Unlocks) Of(kind gamedata.ItemKind) []string { return u.ByKind[kind.Key()] }

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
		grant.OneShot = item.Kind.OneShot()
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

	case gamedata.ItemWeaponBuff:
		buff, ok := gamedata.WeaponBuffByID(item.WeaponBuff)
		if !ok {
			return Grant{}, false
		}
		return Grant{Kind: item.Kind.Key(), Key: buff.Key, Name: item.Name}, true

	default:
		return Grant{}, false
	}
}

// unlocksFrom collapses the grant history into the current set, in the order
// received.
//
// Every state kind gets an entry even when it is empty, so the shape a plugin
// parses does not change with what the run happens to hold.
func unlocksFrom(grants []Grant, resumeFrom int) Unlocks {
	unlocks := Unlocks{ResumeFrom: resumeFrom, ByKind: make(map[string][]string, len(gamedata.ItemKinds))}
	for _, kind := range gamedata.ItemKinds {
		if !kind.OneShot() {
			unlocks.ByKind[kind.Key()] = []string{}
		}
	}
	for _, grant := range grants {
		held, state := unlocks.ByKind[grant.Kind]
		if !state {
			continue
		}
		// Numeric weapon effects count copies. Preserve their duplicate keys so
		// a plugin rebuilding state after a map change restores every level.
		if grant.Kind == gamedata.ItemWeaponBuff.Key() {
			unlocks.ByKind[grant.Kind] = append(held, grant.Key)
		} else {
			unlocks.ByKind[grant.Kind] = appendOnce(held, grant.Key)
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
