package runshape

import (
	"slices"
	"testing"

	"github.com/m-this/tf2-archipelago/gamedata"
)

func TestCommunityMissionsStayHiddenUntilTheirArchiveExists(t *testing.T) {
	for _, mission := range VisibleMissions(nil) {
		if gamedata.IsCommunityMission(mission.ID) {
			t.Fatalf("community mission %s is visible without an archive", mission.PopFile)
		}
	}

	visible := VisibleMissions([]string{"archive-assets.zip"})
	if !slices.ContainsFunc(visible, func(m gamedata.Mission) bool { return m.PopFile == "mvm_kelly_rc1b_adv_homestead_happenings" }) {
		t.Fatal("a portable Potato mission did not appear with its archive")
	}
	if !slices.ContainsFunc(visible, func(m gamedata.Mission) bool { return m.PopFile == "mvm_bogland_rc12_adv_swamp_fever" }) {
		t.Fatal("the missing-NAV compatibility row did not appear with its archive")
	}
	if slices.ContainsFunc(visible, func(m gamedata.Mission) bool { return m.PopFile == "mvm_area_52_rc3_int_anomalous_materials" }) {
		t.Fatal("a Moonlight mission appeared without the Moonlight archive")
	}
}

func TestLockedCommunityMissionsNeverEnterStartChoices(t *testing.T) {
	for _, choice := range StartMissionChoicesForPacks([]string{"archive-assets.zip", "mlarchive-assets.zip"}) {
		if choice.PopFile == "mvm_bogland_rc12_adv_swamp_fever" || choice.PopFile == "mvm_cyberia_rc6a_adv_silver_snow" {
			t.Fatalf("locked mission entered the start menu: %s", choice.PopFile)
		}
	}
}
