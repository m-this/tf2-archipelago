package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/m-this/tf2-archipelago/gamedata"
)

func openTemp(t *testing.T) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "nested", "bridge.json"))
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func firstMission(t *testing.T) gamedata.Mission {
	t.Helper()
	mission, ok := gamedata.MissionByPopFile("mvm_decoy")
	if !ok {
		t.Fatal("mvm_decoy is not in the tables")
	}
	return mission
}

func TestAddCheckIsIdempotent(t *testing.T) {
	store := openTemp(t)
	mission := firstMission(t)

	fresh, err := store.AddCheck(mission.WaveLocationID(1))
	if err != nil || !fresh {
		t.Fatalf("first report: fresh=%v err=%v", fresh, err)
	}
	fresh, err = store.AddCheck(mission.WaveLocationID(1))
	if err != nil || fresh {
		t.Fatalf("repeat report: fresh=%v err=%v", fresh, err)
	}
	if got := store.Checks(); len(got) != 1 {
		t.Fatalf("%d checks held, want 1", len(got))
	}
}

func TestAddCheckRejectsIDsOutsideTheTables(t *testing.T) {
	store := openTemp(t)
	if _, err := store.AddCheck(1); err == nil {
		t.Fatal("an id from nowhere was accepted")
	}
}

func TestChecksSurviveAReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bridge.json")
	mission := firstMission(t)

	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddCheck(mission.ClearLocationID()); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	held := reopened.Checks()
	if len(held) != 1 || held[0] != mission.ClearLocationID() {
		t.Fatalf("reopened with %v", held)
	}
}

func TestProgressiveWeaponSlotsGrantInTableOrder(t *testing.T) {
	store := openTemp(t)
	slotItem := progressiveSlotID(t)

	if err := store.ApplyItems(0, []int64{slotItem, slotItem, slotItem, slotItem}); err != nil {
		t.Fatal(err)
	}
	grants, _ := store.GrantsSince(0)
	if len(grants) != len(gamedata.WeaponSlots) {
		t.Fatalf("%d grants for %d slots plus a spare copy", len(grants), len(gamedata.WeaponSlots))
	}
	for i, grant := range grants {
		if grant.Key != gamedata.WeaponSlots[i].Key {
			t.Errorf("grant %d is %q, want %q", i+1, grant.Key, gamedata.WeaponSlots[i].Key)
		}
		if grant.Seq != i+1 {
			t.Errorf("grant %d carries seq %d", i+1, grant.Seq)
		}
	}
}

func TestApplyItemsContinuesAndResets(t *testing.T) {
	store := openTemp(t)
	mission := firstMission(t)
	ticket := mission.TicketItemID()
	class := gamedata.Classes[0].ItemID()

	if err := store.ApplyItems(0, []int64{ticket}); err != nil {
		t.Fatal(err)
	}
	if err := store.ApplyItems(1, []int64{class}); err != nil {
		t.Fatal(err)
	}
	if got := store.Unlocks(); len(got.Of(gamedata.ItemMissionTicket)) != 1 || len(got.Of(gamedata.ItemClass)) != 1 {
		t.Fatalf("after two items: %+v", got)
	}

	// Index zero is Archipelago restating the whole inventory.
	if err := store.ApplyItems(0, []int64{class}); err != nil {
		t.Fatal(err)
	}
	got := store.Unlocks()
	if len(got.Of(gamedata.ItemMissionTicket)) != 0 || len(got.Of(gamedata.ItemClass)) != 1 {
		t.Fatalf("after a full resend: %+v", got)
	}
}

func TestApplyItemsReportsADesync(t *testing.T) {
	store := openTemp(t)
	if err := store.ApplyItems(3, []int64{gamedata.Classes[0].ItemID()}); !errors.Is(err, ErrDesync) {
		t.Fatalf("err = %v, want ErrDesync", err)
	}
}

func TestUnknownItemIDsAreSkipped(t *testing.T) {
	store := openTemp(t)
	class := gamedata.Classes[0].ItemID()

	if err := store.ApplyItems(0, []int64{424242, class}); err != nil {
		t.Fatal(err)
	}
	grants, _ := store.GrantsSince(0)
	if len(grants) != 1 || grants[0].Key != gamedata.Classes[0].Key {
		t.Fatalf("grants = %+v", grants)
	}
}

func TestASkippedItemLeavesAGapRatherThanShifting(t *testing.T) {
	store := openTemp(t)
	first := gamedata.Classes[0].ItemID()
	second := gamedata.Classes[1].ItemID()

	// The middle id is one this binary cannot read, which is what a seed from a
	// newer gamedata looks like. The item after it must keep the sequence it
	// would have had, or a later binary that can read the id renumbers it and
	// the plugin reapplies grants it already has.
	if err := store.ApplyItems(0, []int64{first, 424242, second}); err != nil {
		t.Fatal(err)
	}
	grants, _ := store.GrantsSince(0)
	if len(grants) != 2 {
		t.Fatalf("grants = %+v", grants)
	}
	if grants[0].Seq != 1 || grants[1].Seq != 3 {
		t.Fatalf("sequences are %d and %d, want 1 and 3", grants[0].Seq, grants[1].Seq)
	}
	if _, reach := store.GrantsSince(0); reach != 3 {
		t.Fatalf("the item list reaches %d, want 3", reach)
	}
	if got, _ := store.GrantsSince(1); len(got) != 1 || got[0].Seq != 3 {
		t.Fatalf("since 1: %+v", got)
	}
	if got, _ := store.GrantsSince(3); got != nil {
		t.Fatalf("since 3: %+v", got)
	}
}

func TestAFailedWriteLeavesNothingHalfApplied(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(filepath.Join(dir, "bridge.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ApplyItems(0, []int64{gamedata.Classes[0].ItemID()}); err != nil {
		t.Fatal(err)
	}

	// Nothing can be written once the directory is gone. The item list and the
	// grants derived from it have to move together or not at all.
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := store.ApplyItems(1, []int64{gamedata.Classes[1].ItemID()}); err == nil {
		t.Fatal("a write into a missing directory was reported as a success")
	}
	unlocks := store.Unlocks()
	if len(unlocks.Of(gamedata.ItemClass)) != 1 {
		t.Fatalf("the failed write left %+v", unlocks)
	}
	if got, _ := store.GrantsSince(0); len(got) != 1 {
		t.Fatalf("the failed write left %d grant(s)", len(got))
	}
}

func TestUnlocksHoldEachKeyOnce(t *testing.T) {
	store := openTemp(t)
	class := gamedata.Classes[0].ItemID()

	if err := store.ApplyItems(0, []int64{class, class}); err != nil {
		t.Fatal(err)
	}
	if got := store.Unlocks(); len(got.Of(gamedata.ItemClass)) != 1 {
		t.Fatalf("classes = %v", got.Of(gamedata.ItemClass))
	}
}

func TestGrantsSinceReturnsOnlyWhatIsNew(t *testing.T) {
	store := openTemp(t)
	if err := store.ApplyItems(0, []int64{
		gamedata.Classes[0].ItemID(),
		gamedata.Classes[1].ItemID(),
	}); err != nil {
		t.Fatal(err)
	}
	if got, _ := store.GrantsSince(1); len(got) != 1 || got[0].Seq != 2 {
		t.Fatalf("since 1: %+v", got)
	}
	if got, _ := store.GrantsSince(2); got != nil {
		t.Fatalf("since 2: %+v", got)
	}
	if got, _ := store.GrantsSince(99); got != nil {
		t.Fatalf("since a sequence past the end: %+v", got)
	}
}

func TestLearningTheSeedKeepsChecksTakenBeforeIt(t *testing.T) {
	store := openTemp(t)
	mission := firstMission(t)

	if _, err := store.AddCheck(mission.WaveLocationID(1)); err != nil {
		t.Fatal(err)
	}
	archive, err := store.BindSeed("first")
	if err != nil || archive != "" {
		t.Fatalf("first bind: archive=%q err=%v", archive, err)
	}
	if len(store.Checks()) != 1 {
		t.Fatal("the queued check did not survive learning the seed")
	}
}

func TestANewSeedDropsTheOldRun(t *testing.T) {
	store := openTemp(t)
	mission := firstMission(t)
	if _, err := store.BindSeed("first"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddCheck(mission.WaveLocationID(1)); err != nil {
		t.Fatal(err)
	}

	archive, err := store.BindSeed("first")
	if err != nil || archive != "" {
		t.Fatalf("rebinding the same seed: archive=%q err=%v", archive, err)
	}
	if len(store.Checks()) != 1 {
		t.Fatal("rebinding the same seed dropped the run")
	}

	archive, err = store.BindSeed("second")
	if err != nil || archive == "" {
		t.Fatalf("rebinding a new seed: archive=%q err=%v", archive, err)
	}
	if len(store.Checks()) != 0 {
		t.Fatal("the previous run's checks survived a new seed")
	}
}

func TestWatchWakesOnAChange(t *testing.T) {
	store := openTemp(t)
	mission := firstMission(t)
	changed := store.Watch()

	select {
	case <-changed:
		t.Fatal("woken before anything changed")
	default:
	}

	if _, err := store.AddCheck(mission.WaveLocationID(2)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-changed:
	default:
		t.Fatal("a new check did not wake the watcher")
	}
}

func TestGoalIsRecordedOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bridge.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if store.GoalSent() {
		t.Fatal("a fresh run has already won")
	}
	if err := store.MarkGoalSent(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reopened.GoalSent() {
		t.Fatal("the win did not survive a restart")
	}
}

func TestAnAcknowledgedEffectIsNotSentAgain(t *testing.T) {
	store := openTemp(t)
	class := gamedata.Classes[0].ItemID()
	cash := cashBundleID(t)

	if err := store.ApplyItems(0, []int64{class, cash}); err != nil {
		t.Fatal(err)
	}
	grants, _ := store.GrantsSince(0)
	if len(grants) != 2 {
		t.Fatalf("grants = %+v", grants)
	}
	if err := store.Ack(2); err != nil {
		t.Fatal(err)
	}

	// This is a plugin that reloaded: it asks from zero because it remembers
	// nothing. The class comes back because applying it twice changes nothing;
	// the cash must not, or the run is paid a second time.
	after, _ := store.GrantsSince(0)
	if len(after) != 1 || after[0].Kind != gamedata.ItemClass.Key() {
		t.Fatalf("after the acknowledgement: %+v", after)
	}
}

// A trap is an effect, so it reaches the plugin by key and never joins the
// unlock set. A trap in the unlock set would fire again on every map change.
func TestATrapIsAnEffectAndNotAnUnlock(t *testing.T) {
	store := openTemp(t)
	trap := gamedata.Traps[0]

	if err := store.ApplyItems(0, []int64{trap.ItemID()}); err != nil {
		t.Fatal(err)
	}
	grants, _ := store.GrantsSince(0)
	if len(grants) != 1 {
		t.Fatalf("grants = %+v", grants)
	}
	if grants[0].Kind != gamedata.ItemTrap.Key() || grants[0].Key != trap.Key {
		t.Fatalf("the plugin was told %q/%q, wanted trap/%s", grants[0].Kind, grants[0].Key, trap.Key)
	}
	if held := store.Unlocks().Of(gamedata.ItemTrap); len(held) != 0 {
		t.Fatalf("the unlock set holds %v", held)
	}

	if err := store.Ack(1); err != nil {
		t.Fatal(err)
	}
	if after, _ := store.GrantsSince(0); len(after) != 0 {
		t.Fatalf("the trap was sent again after the acknowledgement: %+v", after)
	}
}

func TestAnAcknowledgementSurvivesARestartAndOnlyMovesForward(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bridge.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ApplyItems(0, []int64{cashBundleID(t), cashBundleID(t)}); err != nil {
		t.Fatal(err)
	}
	if err := store.Ack(2); err != nil {
		t.Fatal(err)
	}
	if err := store.Ack(1); err != nil {
		t.Fatal(err)
	}
	if got := store.Stats().AckedSeq; got != 2 {
		t.Fatalf("an older acknowledgement moved the cursor back to %d", got)
	}
	if err := store.Ack(3); err == nil {
		t.Fatal("acknowledging past the items that exist was accepted")
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := reopened.GrantsSince(0); got != nil {
		t.Fatalf("a restart re-sent acknowledged effects: %+v", got)
	}
}

// The bug this pass exists to prevent, in the shape it actually takes: a cash
// bundle arrives while no plugin is listening, then a map change makes the
// plugin ask for the unlock set and resume from what it is told.
//
// The unlock set carries state only, so the bundle is not in it. If the cursor
// it hands back were the length of the item list, it would sit above the bundle
// and nothing would ever deliver it.
func TestAnEffectSurvivesAPluginThatWasNotListening(t *testing.T) {
	store := openTemp(t)
	class, cash := gamedata.Classes[0].ItemID(), cashBundleID(t)

	if err := store.ApplyItems(0, []int64{class, cash}); err != nil {
		t.Fatal(err)
	}
	unlocks := store.Unlocks()
	if len(unlocks.Of(gamedata.ItemCredits)) != 0 {
		t.Fatal("the unlock set carries an effect")
	}

	resumed, _ := store.GrantsSince(unlocks.ResumeFrom)
	paid := false
	for _, grant := range resumed {
		if grant.Kind == gamedata.ItemCredits.Key() {
			paid = true
		}
	}
	if !paid {
		t.Fatalf("resuming from %d skipped the cash nobody has been paid: %+v",
			unlocks.ResumeFrom, resumed)
	}

	// Once the plugin says it applied it, the same resume must not pay again.
	if err := store.Ack(2); err != nil {
		t.Fatal(err)
	}
	after, _ := store.GrantsSince(store.Unlocks().ResumeFrom)
	for _, grant := range after {
		if grant.Kind == gamedata.ItemCredits.Key() {
			t.Fatalf("the cash came back after being acknowledged: %+v", after)
		}
	}
}

func TestAnAcknowledgementNeverOutlivesTheItemsItCounts(t *testing.T) {
	store := openTemp(t)
	if err := store.ApplyItems(0, []int64{cashBundleID(t), cashBundleID(t), cashBundleID(t)}); err != nil {
		t.Fatal(err)
	}
	if err := store.Ack(3); err != nil {
		t.Fatal(err)
	}

	// Archipelago answering a Sync with a shorter list. An acknowledgement left
	// past the end would suppress the fresh effects landing in those slots.
	if err := store.ApplyItems(0, []int64{cashBundleID(t)}); err != nil {
		t.Fatal(err)
	}
	if got := store.Stats().AckedSeq; got != 1 {
		t.Fatalf("the acknowledgement stayed at %d against 1 item", got)
	}
	if err := store.ApplyItems(1, []int64{cashBundleID(t), cashBundleID(t)}); err != nil {
		t.Fatal(err)
	}
	got, _ := store.GrantsSince(1)
	if len(got) != 2 {
		t.Fatalf("the fresh effects were suppressed by a stale acknowledgement: %+v", got)
	}
}

func TestANewRunDoesNotInheritTheOldRunsAcknowledgement(t *testing.T) {
	store := openTemp(t)
	if _, err := store.BindSeed("first"); err != nil {
		t.Fatal(err)
	}
	if err := store.ApplyItems(0, []int64{cashBundleID(t), cashBundleID(t)}); err != nil {
		t.Fatal(err)
	}
	if err := store.Ack(2); err != nil {
		t.Fatal(err)
	}
	if _, err := store.BindSeed("second"); err != nil {
		t.Fatal(err)
	}
	if got := store.Stats().AckedSeq; got != 0 {
		t.Fatalf("the new run inherited an acknowledgement of %d", got)
	}

	// Cash arriving in the new run is cash the team has not been paid. A
	// cursor carried over from the run before would swallow it.
	if err := store.ApplyItems(0, []int64{cashBundleID(t)}); err != nil {
		t.Fatal(err)
	}
	if got, _ := store.GrantsSince(0); len(got) != 1 {
		t.Fatalf("the new run's cash was held back: %+v", got)
	}
	if err := store.Ack(5); err == nil {
		t.Fatal("a cursor past the new run's items was accepted")
	}
}

func TestChecksComeBackFromTheServer(t *testing.T) {
	store := openTemp(t)
	mission := firstMission(t)

	adopted, err := store.AdoptChecks([]int64{
		mission.WaveLocationID(1),
		mission.WaveLocationID(2),
		424242,
	})
	if err != nil {
		t.Fatal(err)
	}
	if adopted != 2 {
		t.Fatalf("adopted %d, want the two the tables know", adopted)
	}
	if got := store.Checks(); len(got) != 2 {
		t.Fatalf("holding %v", got)
	}

	// The server resends the whole list on every connect.
	adopted, err = store.AdoptChecks([]int64{mission.WaveLocationID(1)})
	if err != nil || adopted != 0 {
		t.Fatalf("a repeat adopted %d, err=%v", adopted, err)
	}
}

func TestANewSeedKeepsTheOldRunOnDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bridge.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.BindSeed("first"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddCheck(firstMission(t).WaveLocationID(1)); err != nil {
		t.Fatal(err)
	}

	archive, err := store.BindSeed("second")
	if err != nil {
		t.Fatal(err)
	}
	if archive == "" {
		t.Fatal("the dropped run was not set aside anywhere")
	}
	body, err := os.ReadFile(archive)
	if err != nil {
		t.Fatalf("the dropped run left nothing behind: %v", err)
	}
	if !strings.Contains(string(body), `"seed": "first"`) {
		t.Fatalf("the archive holds %s", body)
	}
}

func TestASeedNameIsNeverAPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bridge.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.BindSeed("../../etc/passwd"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.BindSeed("next"); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("the state directory holds %v", entries)
	}
}

func TestAStateFileFromTheOlderFormatIsStillRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bridge.json")
	mission := firstMission(t)
	older := fmt.Sprintf(
		`{"format_version":1,"seed":"first","checks":[%d],"items":[],"goal_sent":false}`,
		mission.WaveLocationID(1),
	)
	if err := os.WriteFile(path, []byte(older), 0o600); err != nil {
		t.Fatal(err)
	}

	store, err := Open(path)
	if err != nil {
		t.Fatalf("a file this bridge can read was refused: %v", err)
	}
	if got := store.Checks(); len(got) != 1 {
		t.Fatalf("the older file came back as %v", got)
	}

	// Read as version 1, written back as the current one.
	if _, err := store.AddCheck(mission.WaveLocationID(2)); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), fmt.Sprintf(`"format_version": %d`, FormatVersion)) {
		t.Fatalf("the file was not written back as the current format: %s", body)
	}
}

func TestAStateFileFromTheFutureIsRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bridge.json")
	newer := fmt.Sprintf(`{"format_version":%d,"seed":"first"}`, FormatVersion+1)
	if err := os.WriteFile(path, []byte(newer), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(path); err == nil {
		t.Fatal("a file from a format this bridge does not know was accepted")
	}
}

func cashBundleID(t *testing.T) int64 {
	t.Helper()
	for _, item := range gamedata.Items {
		if item.Kind == gamedata.ItemCredits {
			return item.ID
		}
	}
	t.Fatal("no credits item in the tables")
	return 0
}

func TestWeaponBuffIsPersistentUnlockState(t *testing.T) {
	var item gamedata.Item
	for _, candidate := range gamedata.Items {
		if candidate.Kind == gamedata.ItemWeaponBuff {
			item = candidate
			break
		}
	}
	if item.ID == 0 {
		t.Fatal("no weapon buff item in gamedata")
	}
	store := openTemp(t)
	if err := store.ApplyItems(0, []int64{item.ID}); err != nil {
		t.Fatal(err)
	}
	buff, ok := gamedata.WeaponBuffByID(item.WeaponBuff)
	if !ok {
		t.Fatalf("unknown weapon buff id %d", item.WeaponBuff)
	}
	held := store.Unlocks().Of(gamedata.ItemWeaponBuff)
	if len(held) != 1 || held[0] != buff.Key {
		t.Fatalf("weapon buff unlocks = %v, want %q", held, buff.Key)
	}
}

func TestRepeatedWeaponBuffCopiesSurviveUnlockResync(t *testing.T) {
	var item gamedata.Item
	for _, candidate := range gamedata.Items {
		if candidate.Kind == gamedata.ItemWeaponBuff {
			item = candidate
			break
		}
	}
	store := openTemp(t)
	if err := store.ApplyItems(0, []int64{item.ID, item.ID, item.ID}); err != nil {
		t.Fatal(err)
	}
	held := store.Unlocks().Of(gamedata.ItemWeaponBuff)
	if len(held) != 3 || held[0] != held[1] || held[1] != held[2] {
		t.Fatalf("repeated weapon buff unlocks = %v, want three copies", held)
	}
}

func progressiveSlotID(t *testing.T) int64 {
	t.Helper()
	for _, item := range gamedata.Items {
		if item.Kind == gamedata.ItemWeaponSlot {
			return item.ID
		}
	}
	t.Fatal("no weapon slot item in the tables")
	return 0
}

func TestAnAdoptedCheckIsNotOneThisServerPlayed(t *testing.T) {
	store := openTemp(t)
	mission := firstMission(t)

	// Another player collecting their items out of this world's locations is
	// what checks a mission's clear without anybody beating it.
	if _, err := store.AdoptChecks([]int64{mission.ClearLocationID()}); err != nil {
		t.Fatal(err)
	}
	if got := store.Checks(); len(got) != 1 {
		t.Fatalf("%d checks held, want the adopted one", len(got))
	}
	if got := store.Played(); len(got) != 0 {
		t.Fatalf("played %v, want nothing: the run has not been played", got)
	}

	if _, err := store.AddCheck(mission.WaveLocationID(1)); err != nil {
		t.Fatal(err)
	}
	played := store.Played()
	if len(played) != 1 || played[0] != mission.WaveLocationID(1) {
		t.Fatalf("played %v, want the wave this server reported", played)
	}
}

func TestPlayedSurvivesAReopen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bridge.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	mission := firstMission(t)
	if _, err := store.AddCheck(mission.ClearLocationID()); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	played := reopened.Played()
	if len(played) != 1 || played[0] != mission.ClearLocationID() {
		t.Fatalf("played %v after a reopen, want the clear", played)
	}
}

/*
Played arrived without a format bump, so a version 3 file can be from either
side of that day. kelly-cs's run crossed it: every mission the team had cleared
read as collected afterwards and the missionsanity goal counted none of them
(gh-16). A version 3 file with no played key is from before, and everything it
checked was played here.
*/
func TestAVersionThreeFileWithoutPlayedTakesItsChecksAsPlayed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bridge.json")
	mission := firstMission(t)
	before := fmt.Sprintf(
		`{"format_version":3,"seed":"first","checks":[%d,%d],"items":[],"goal_sent":false}`,
		mission.WaveLocationID(1), mission.ClearLocationID(),
	)
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := store.Played(); len(got) != 2 {
		t.Fatalf("played = %v, want both checks the file held", got)
	}

	// A version 3 file that wrote the key is from after, and an empty list
	// there means nothing was played, whatever the room had adopted.
	after := fmt.Sprintf(
		`{"format_version":3,"seed":"first","checks":[%d],"played":[],"items":[],"goal_sent":false}`,
		mission.WaveLocationID(1),
	)
	other := filepath.Join(t.TempDir(), "bridge.json")
	if err := os.WriteFile(other, []byte(after), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err = Open(other)
	if err != nil {
		t.Fatal(err)
	}
	if got := store.Played(); len(got) != 0 {
		t.Errorf("played = %v, want none: the file said so", got)
	}
}
