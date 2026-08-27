package gamedata

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// The plugin matches on keys, not ids, and it is the one component that cannot
// be run here. Nothing in SourcePawn can check itself against these tables, so
// the tables check the SourcePawn instead: the source is read as text and the
// key literals in it are compared with what this package exports.
//
// The failure this prevents is quiet. A key the plugin does not know becomes a
// log line during a live wave and an unlock that never arrives.
const (
	unlocksSource = "../plugin/scripting/tf2_archipelago/unlocks.inc"
	bridgeSource  = "../plugin/scripting/tf2_archipelago/bridge.inc"
	pluginSource  = "../plugin/scripting/tf2_archipelago.sp"
)

// literals pulls the quoted strings out of one SourcePawn array or call.
func literals(t *testing.T, path, opening string) []string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	start := strings.Index(text, opening)
	if start == -1 {
		t.Fatalf("%s has no %q", path, opening)
	}
	end := strings.Index(text[start:], "};")
	if end == -1 {
		t.Fatalf("%s: %q is not closed", path, opening)
	}
	return regexp.MustCompile(`"([^"]*)"`).FindAllString(text[start:start+end], -1)
}

func unquoted(values []string) []string {
	plain := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.Trim(value, `"`); trimmed != "" {
			plain = append(plain, trimmed)
		}
	}
	return plain
}

func TestThePluginKnowsEveryClassKey(t *testing.T) {
	held := unquoted(literals(t, unlocksSource, "g_ClassKeys"))
	if len(held) != len(Classes) {
		t.Fatalf("the plugin lists %d class keys, this package has %d", len(held), len(Classes))
	}
	// The plugin indexes its array by the game's own class numbering, which is
	// not the order of the Classes table. Only the set has to agree.
	for _, class := range Classes {
		if !slices.Contains(held, class.Key) {
			t.Errorf("the plugin has no key %q", class.Key)
		}
	}
}

func TestThePluginKnowsEveryWeaponSlotKey(t *testing.T) {
	held := unquoted(literals(t, unlocksSource, "g_SlotKeys"))
	if len(held) != len(WeaponSlots) {
		t.Fatalf("the plugin lists %d slot keys, this package has %d", len(held), len(WeaponSlots))
	}
	// Order matters here and not for classes: the plugin uses the position in
	// this array as the game's weapon slot number.
	for i, slot := range WeaponSlots {
		if held[i] != slot.Key {
			t.Errorf("slot %d is %q in the plugin and %q here", i, held[i], slot.Key)
		}
	}
}

// The plugin's table is indexed by TF2's class enum, which is not the order of
// Classes, so each row names its class in a trailing comment and this reads
// that. A row whose comment is wrong fails here rather than handing a Medic a
// syringe gun and no Medigun for a whole evening.
var pluginSlotOrderRow = regexp.MustCompile(`\{([^{}]*)\},\s*//\s*(\w+)`)

func TestThePluginOpensSlotsInTheOrderThisPackageSets(t *testing.T) {
	body, err := os.ReadFile(unlocksSource)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	start := strings.Index(text, "g_SlotOrder")
	if start == -1 {
		t.Fatalf("%s has no g_SlotOrder", unlocksSource)
	}
	end := strings.Index(text[start:], "};")
	if end == -1 {
		t.Fatalf("%s: g_SlotOrder is not closed", unlocksSource)
	}
	rows := pluginSlotOrderRow.FindAllStringSubmatch(text[start:start+end], -1)

	// One row per class, plus the row TFClass_Unknown indexes.
	if len(rows) != len(Classes)+1 {
		t.Fatalf("the plugin lists %d slot orders, this package has %d classes", len(rows), len(Classes))
	}

	held := make(map[string][]WeaponSlotID, len(rows))
	for _, row := range rows {
		order := make([]WeaponSlotID, 0, len(WeaponSlots))
		for name := range strings.SplitSeq(row[1], ",") {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			slot, ok := weaponSlotByPluginName(name)
			if !ok {
				t.Fatalf("the plugin names a slot %q that this package does not have", name)
			}
			order = append(order, slot)
		}
		held[row[2]] = order
	}

	for _, class := range Classes {
		order, ok := held[class.Key]
		if !ok {
			t.Errorf("the plugin has no slot order for %q", class.Key)
			continue
		}
		if !slices.Equal(order, class.SlotOrder) {
			t.Errorf("%s opens %v in the plugin and %v here", class.Key, order, class.SlotOrder)
		}
	}
}

// The plugin spells a slot Slot_Primary where this package keys it "primary".
func weaponSlotByPluginName(name string) (WeaponSlotID, bool) {
	for _, slot := range WeaponSlots {
		if name == "Slot_"+slot.Name {
			return slot.ID, true
		}
	}
	return 0, false
}

func TestThePluginHandlesEveryGrantKind(t *testing.T) {
	body, err := os.ReadFile(bridgeSource)
	if err != nil {
		t.Fatal(err)
	}
	handled := string(body)
	for _, kind := range ItemKinds {
		if !strings.Contains(handled, `"`+kind.Key()+`"`) {
			t.Errorf("the plugin does not handle a grant of kind %q", kind.Key())
		}
	}
}

func TestThePluginReportsEveryObjectiveKind(t *testing.T) {
	body, err := os.ReadFile(pluginSource)
	if err != nil {
		t.Fatal(err)
	}
	reported := string(body)
	for _, kind := range ObjectiveKinds {
		if !strings.Contains(reported, `"`+kind.Key()+`"`) {
			t.Errorf("the plugin never reports an objective of kind %q", kind.Key())
		}
	}
}
