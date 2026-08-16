package state

import (
	"errors"
	"path/filepath"
	"testing"

	"git-ssh.croque.top/mathis/tf2-archipelago/gamedata"
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
	grants := store.GrantsSince(0)
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
	if got := store.Unlocks(); len(got.Missions) != 1 || len(got.Classes) != 1 {
		t.Fatalf("after two items: %+v", got)
	}

	// Index zero is Archipelago restating the whole inventory.
	if err := store.ApplyItems(0, []int64{class}); err != nil {
		t.Fatal(err)
	}
	got := store.Unlocks()
	if len(got.Missions) != 0 || len(got.Classes) != 1 || got.Seq != 1 {
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
	grants := store.GrantsSince(0)
	if len(grants) != 1 || grants[0].Key != gamedata.Classes[0].Key {
		t.Fatalf("grants = %+v", grants)
	}
}

func TestUnlocksHoldEachKeyOnce(t *testing.T) {
	store := openTemp(t)
	class := gamedata.Classes[0].ItemID()

	if err := store.ApplyItems(0, []int64{class, class}); err != nil {
		t.Fatal(err)
	}
	if got := store.Unlocks(); len(got.Classes) != 1 {
		t.Fatalf("classes = %v", got.Classes)
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
	if got := store.GrantsSince(1); len(got) != 1 || got[0].Seq != 2 {
		t.Fatalf("since 1: %+v", got)
	}
	if got := store.GrantsSince(2); got != nil {
		t.Fatalf("since 2: %+v", got)
	}
	if got := store.GrantsSince(99); got != nil {
		t.Fatalf("since a sequence past the end: %+v", got)
	}
}

func TestLearningTheSeedKeepsChecksTakenBeforeIt(t *testing.T) {
	store := openTemp(t)
	mission := firstMission(t)

	if _, err := store.AddCheck(mission.WaveLocationID(1)); err != nil {
		t.Fatal(err)
	}
	wiped, err := store.BindSeed("first")
	if err != nil || wiped {
		t.Fatalf("first bind: wiped=%v err=%v", wiped, err)
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

	wiped, err := store.BindSeed("first")
	if err != nil || wiped {
		t.Fatalf("rebinding the same seed: wiped=%v err=%v", wiped, err)
	}
	if len(store.Checks()) != 1 {
		t.Fatal("rebinding the same seed dropped the run")
	}

	wiped, err = store.BindSeed("second")
	if err != nil || !wiped {
		t.Fatalf("rebinding a new seed: wiped=%v err=%v", wiped, err)
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
