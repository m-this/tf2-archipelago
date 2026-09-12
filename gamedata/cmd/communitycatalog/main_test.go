package main

import "testing"

func TestMissionIdentityUsesWholeFilenameSegments(t *testing.T) {
	tests := []struct {
		popFile, mapName, difficulty, title string
	}{
		{"mvm_bronx_rc2_adv_point_of_impact", "mvm_bronx_rc2", "advanced", "Point Of Impact"},
		{"mvm_downpour_rc3a_adv_666_last_stand", "mvm_downpour_rc3a", "haunted", "Last Stand"},
		{"mvm_autumnull_rc2_rev_exp_codename_omega", "mvm_autumnull_rc2", "expert", "Codename Omega"},
		{"mvm_cyberia_rc6a_rev_arctic_arrangement", "mvm_cyberia_rc6a", "advanced", "Arctic Arrangement"},
	}
	for _, test := range tests {
		difficulty, title, ok := missionIdentity(test.popFile, test.mapName)
		if !ok || difficulty != test.difficulty || title != test.title {
			t.Errorf("missionIdentity(%q) = %q, %q, %t; want %q, %q, true",
				test.popFile, difficulty, title, ok, test.difficulty, test.title)
		}
	}
}
