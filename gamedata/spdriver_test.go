package gamedata

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/spshell"
)

/*
Running plugin functions instead of reading them.

The tests beside this one check the plugin by looking for substrings in its
source: that WeaponBuffs_AttackEnemyProjectiles still contains "chosen =
projectile", that the sweep still mentions TE_SetupSparks. They were written
because nothing in SourcePawn could check itself, and they are the best that can
be done by reading. It is not much. Renaming a local breaks them without
changing behaviour, and changing the arithmetic passes them as long as the names
survive, which is the wrong way round for both.

So the toolchain the defender mod built for its generated code is pointed at
this plugin's hand-written code. spcomp compiles a driver and SourcePawn's
standalone VM runs it, and what comes back is what the function computes, on
inputs this test chose. No game server, no map, no client.

The driver includes weapon_buffs_math.inc rather than pasting text out of it.
That file exists for this: it holds the decisions that need no native, it
includes nothing itself, and the plugin includes it too. So what runs here is
the text the plugin compiles, not a copy and not an extract.

What stays next door needs the engine. A function that reads an entity property,
makes an SDKCall or asks TF2 anything cannot run without a server, and those are
still checked by reading, which is the best that can be done for them. Moving
one here means first making it need nothing, which is a change to the plugin
rather than to its tests.
*/

// requireEnv turns an absent toolchain from a skip into a failure. make check
// sets it; a developer with no clang gets the skip and a message naming what to
// run. The defender mod has its own variable for its own gate, which is why
// spshell takes the name rather than owning it.
const requireEnv = "TF2AP_REQUIRE_SPSHELL"

const (
	pluginDir   = "../plugin/scripting/tf2_archipelago"
	mathSource  = pluginDir + "/weapon_buffs_math.inc"
	dataSource  = pluginDir + "/weapon_buffs_data.inc"
	buffsSource = pluginDir + "/weapon_buffs.inc"
)

// driverIncludes are the plugin files a driver compiles against, in the order
// they are included. They ship with the plugin; none is written here.
var driverIncludes = []string{
	"weapon_buffs_data.inc",
	"weapon_buffs_math.inc",
	"bridge_grants_math.inc",
	"mission_modifiers_math.inc",
}

/*
	driver is a standalone plugin built around the plugin's own math include.

body is the main: whatever it prints comes back as cells, in order. A float goes
out as view_as<int> so the bits arrive rather than a rounded decimal.
*/
type driver struct {
	body string
}

func (d driver) source() string {
	var b strings.Builder
	b.WriteString("#pragma semicolon 1\n#pragma newdecls required\n\n")
	// spshell's own builtin, and the only native any driver here needs.
	b.WriteString("native void printnum(int n);\n\n")
	for _, name := range driverIncludes {
		fmt.Fprintf(&b, "#include %q\n", name)
	}
	b.WriteString("\npublic int main()\n{\n")
	b.WriteString(d.body)
	b.WriteString("\n    return 0;\n}\n")
	return b.String()
}

// run compiles the driver and returns the cells it printed, in order.
func (d driver) run(t *testing.T) []int32 {
	t.Helper()
	tc := spshell.ForTestRequiring(t, requireEnv)

	dir := t.TempDir()
	path := filepath.Join(dir, "driver.sp")
	if err := os.WriteFile(path, []byte(d.source()), 0o600); err != nil {
		t.Fatal(err)
	}
	// The plugin's own files, copied beside the driver rather than reached for
	// on an include path: spshell.Run puts one injected directory first and the
	// driver has to find them there.
	for _, name := range driverIncludes {
		body, err := os.ReadFile(filepath.Join(pluginDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	cells, err := tc.Run(context.Background(), path, nil)
	if err != nil {
		t.Fatalf("running the driver: %v\n\n%s", err, d.source())
	}
	return cells
}

// The SourcePawn grant cursor must count each received item position once.
// Held cash or a trap makes the bridge return the same later buff positions on
// every retry; the old plugin could increment a legitimate x2 to x44 this way.
func TestStateGrantCursorIgnoresRuntimeReplays(t *testing.T) {
	got := driver{body: `
    int seen = 4; // The unlock snapshot already holds two copies.
    int levels = 2;
    for (int retry = 0; retry < 21; retry++)
    {
        int next = Bridge_NextStateGrantSeq(seen, 2);
        if (next != seen) { levels++; seen = next; }
        next = Bridge_NextStateGrantSeq(seen, 4);
        if (next != seen) { levels++; seen = next; }
    }
    printnum(levels);
    printnum(seen);

    // One genuinely new copy arrives while the same effect is still held.
    for (int retry = 0; retry < 21; retry++)
    {
        int next = Bridge_NextStateGrantSeq(seen, 5);
        if (next != seen) { levels++; seen = next; }
    }
    printnum(levels);
    printnum(seen);

    // An older replay cannot lower the cursor before another new copy.
    int next = Bridge_NextStateGrantSeq(seen, 3);
    if (next != seen) { levels++; seen = next; }
    next = Bridge_NextStateGrantSeq(seen, 6);
    if (next != seen) { levels++; seen = next; }
    printnum(levels);
    printnum(seen);
`}.run(t)
	want := []int32{2, 4, 3, 5, 4, 6}
	if len(got) != len(want) {
		t.Fatalf("grant cursor returned %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("grant cursor returned %v, want %v", got, want)
		}
	}
}

// A reload can ask from an acknowledgement that predates the unlock snapshot
// even when the run contains no cash or traps. Those copies are already in the
// menu and must not be counted by the first grant poll a second time.
func TestStateGrantCursorIgnoresSnapshotOverlapWithoutEffects(t *testing.T) {
	got := driver{body: `
    int seen = 2; // The unlock snapshot contains two buff copies.
    int levels = 2;
    for (int seq = 1; seq <= 2; seq++)
    {
        int next = Bridge_NextStateGrantSeq(seen, seq);
        if (next != seen) { levels++; seen = next; }
    }
    printnum(levels);
    printnum(seen);
`}.run(t)
	if len(got) != 2 || got[0] != 2 || got[1] != 2 {
		t.Fatalf("snapshot overlap returned %v, want [2 2]", got)
	}
}

/*
	The projectile destruction cooldown is the arithmetic, not the wording.

One level destroys a projectile every ProjectileDestructionBaseCooldown seconds,
each level after takes a step off, and it never goes below the floor. Three
constants and a max, and every one of them is a number a reader can get wrong
while leaving the source looking right.

The expected values are worked out here from the same #defines the plugin
carries, so this does not pin today's numbers: it pins the rule. Changing the
base cooldown moves both sides. Changing the subtraction to a division, or
inverting the floor's comparison so every cooldown collapses to the minimum,
moves one. The second of those leaves every string the scraping tests watch for
in place, and they pass on it.
*/
func TestProjectileDestructionCooldownIsTheDeclaredCurve(t *testing.T) {
	base := floatDefine(t, mathSource, "ProjectileDestructionBaseCooldown")
	step := floatDefine(t, mathSource, "ProjectileDestructionCooldownStep")
	floor := floatDefine(t, mathSource, "ProjectileDestructionMinimumCooldown")

	levels := []int{1, 2, 3, 4, 5, 8, 20}
	var calls strings.Builder
	for _, level := range levels {
		fmt.Fprintf(&calls, "    printnum(view_as<int>(WeaponBuffs_ProjectileDestructionCooldown(%d)));\n", level)
	}

	got := driver{body: calls.String()}.run(t)
	if len(got) != len(levels) {
		t.Fatalf("%d levels went in and %d answers came out", len(levels), len(got))
	}
	for i, level := range levels {
		want := max32(base-float32(level-1)*step, floor)
		if bitsToFloat(got[i]) != want {
			t.Errorf("level %d cools down in %v, wanted %v", level, bitsToFloat(got[i]), want)
		}
	}
}

/*
	A passive effect is the four the plugin names and nothing else.

WeaponBuffs_IsPassiveEffect decides which effects are applied at spawn rather
than on a hit or a kill, and getting it wrong is silent: an effect that falls
out of the passive set never applies, and the player reports a buff that does
nothing.

Every effect ID the generated table holds is tried, not the three that are
expected, because what matters is the answer for the ones nobody thought about.
The same goes for the on-kill set below.
*/
func TestOnlyTheNamedEffectsArePassive(t *testing.T) {
	assertEffectSet(t, "WeaponBuffs_IsPassiveEffect",
		"MoveSpeedEffect", "JumpHeightEffect", "ActiveHealthRegenEffect", "MaxHealthEffect")
}

// An on-kill effect is paid when the attacker gets a kill. One that falls out of
// this set is a buff the player bought and never sees fire.
func TestOnlyTheNamedEffectsAreOnKill(t *testing.T) {
	assertEffectSet(t, "WeaponBuffs_IsOnKillEffect",
		"HealOnKillEffect", "CritsOnKillEffect", "SpeedOnKillEffect",
		"MinicritsOnKillEffect", "BaseHealthOnKillEffect")
}

// assertEffectSet runs a predicate over every effect the generated table holds
// and checks it answers true for exactly the named ones.
func assertEffectSet(t *testing.T, predicate string, members ...string) {
	t.Helper()
	want := map[int]bool{}
	for _, name := range members {
		want[intDefine(t, mathSource, name)] = true
	}

	count := intDefine(t, dataSource, "WeaponEffectCount")
	var calls strings.Builder
	for effect := range count {
		fmt.Fprintf(&calls, "    printnum(%s(%d) ? 1 : 0);\n", predicate, effect)
	}

	got := driver{body: calls.String()}.run(t)
	if len(got) != count {
		t.Fatalf("%d effects went in and %d answers came out", count, len(got))
	}
	for effect := range count {
		if (got[effect] == 1) != want[effect] {
			t.Errorf("%s(%d) is %v, wanted %v", predicate, effect, got[effect] == 1, want[effect])
		}
	}
}

/*
	Every weapon the tables know is found by its definition, and only it.

WeaponBuffs_ForDefinition is the lookup the plugin does on every hit: the item
definition the game gives it, back to the row of the buff tables. A definition
it cannot find returns -1 and the hit pays nothing, which is a buff that works
on some weapons and not others with no error anywhere.

Every definition in the table is asked for, and so are three that are not in it,
because a miss has to be a miss rather than the first row.
*/
func TestEveryWeaponIsFoundByItsDefinition(t *testing.T) {
	definitions := weaponDefinitions(t)
	if len(definitions) == 0 {
		t.Fatal("the generated table has no weapon definitions")
	}

	var calls strings.Builder
	for _, definition := range definitions {
		fmt.Fprintf(&calls, "    printnum(WeaponBuffs_ForDefinition(%d));\n", definition)
	}
	/* Definitions the table does not hold. Not 0: that is the Bat, the Scout's
	   stock melee, and asking for it as a miss is how this test first failed.
	   -1 is not an item and 65535 is past every definition Valve has issued. */
	misses := []int{-1, 65535}
	for _, definition := range misses {
		fmt.Fprintf(&calls, "    printnum(WeaponBuffs_ForDefinition(%d));\n", definition)
	}

	got := driver{body: calls.String()}.run(t)
	if len(got) != len(definitions)+len(misses) {
		t.Fatalf("%d lookups went in and %d answers came out", len(definitions)+len(misses), len(got))
	}

	weapons := intDefine(t, dataSource, "WeaponCount")
	for i, definition := range definitions {
		switch {
		case got[i] < 0:
			t.Errorf("definition %d is in the table and the lookup missed it", definition)
		case int(got[i]) >= weapons:
			t.Errorf("definition %d resolved to weapon %d, past the %d in the table", definition, got[i], weapons)
		}
	}
	for i, definition := range misses {
		if answer := got[len(definitions)+i]; answer != -1 {
			t.Errorf("definition %d is not in the table and the lookup returned %d", definition, answer)
		}
	}
}

func TestBuilderToolboxResolvesToTheConstructionPDA(t *testing.T) {
	want := int32(weaponNamed(t, "Construction PDA").ID - 1)
	got := driver{body: "    printnum(WeaponBuffs_ForDefinition(28));\n"}.run(t)
	if len(got) != 1 || got[0] != want {
		t.Fatalf("builder toolbox definition 28 resolved to %v, want Construction PDA index %d", got, want)
	}
}

func TestSelfBlastHealthStateHandlesOverlappingHits(t *testing.T) {
	got := driver{body: "    printnum(WeaponBuffs_SelfBlastBaseline(0, 100));\n" +
		"    printnum(WeaponBuffs_SelfBlastBaseline(100, 180));\n" +
		"    printnum(view_as<int>(WeaponBuffs_SelfBlastGuardHealth(100, 125.5)));\n" +
		"    printnum(WeaponBuffs_SelfBlastBaseline(0, 25));\n" +
		"    printnum(view_as<int>(WeaponBuffs_SelfBlastGuardHealth(25, 120.0)));\n"}.run(t)
	if len(got) != 5 || got[0] != 100 || got[1] != 100 || bitsToFloat(got[2]) != 225.5 ||
		got[3] != 25 || bitsToFloat(got[4]) != 145.0 {
		t.Fatalf("self-blast health state returned %v", got)
	}
}

// max32 is the plugin's own floor, written out in Go rather than reached for,
// because the point of the test above is that the two agree.
func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
