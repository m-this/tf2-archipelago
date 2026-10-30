package main

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func TestMissionRowResolvesIncludesOnlyFromSelectedArchive(t *testing.T) {
	popFile := "mvm_bronx_rc2_adv_point_of_impact"
	mapIDs := map[string]uint8{"mvm_bronx_rc2": 150}
	nav := map[string]bool{"mvm_bronx_rc2": true}
	body := []byte("#base shared.pop\nWaveSchedule { Wave { } }")
	all := map[string]map[string][][]byte{
		"archive-assets.zip":   {"shared.pop": {[]byte("LuaScriptFile example.lua")}},
		"mlarchive-assets.zip": {"shared.pop": {[]byte("WaveSchedule { }")}},
	}
	_, requirement, ok, err := missionRow(popFile, mapIDs, nav, population{body: body, pack: "mlarchive-assets.zip"}, all, 200, map[string]bool{})
	if err != nil || !ok || requirement != "" {
		t.Fatalf("Moonlight row inherited Potato requirement: %q, ok=%t, err=%v", requirement, ok, err)
	}
	_, requirement, ok, err = missionRow(popFile, mapIDs, nav, population{body: body, pack: "archive-assets.zip"}, all, 200, map[string]bool{})
	if err != nil || !ok || requirement != "sigsegv-mvm" {
		t.Fatalf("Potato row lost its requirement: %q, ok=%t, err=%v", requirement, ok, err)
	}
}

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

func TestArchiveBSPNavigationFindsOnlyEngineSupportedEmbeddedNames(t *testing.T) {
	for _, test := range []struct {
		name string
		want bool
	}{
		{"maps/mvm_embedded.nav", true},
		{"maps/embed.nav", true},
		{"maps/mvm_other.nav", false},
		{"materials/embed.nav", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var pak bytes.Buffer
			writer := zip.NewWriter(&pak)
			file, err := writer.Create(test.name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := file.Write([]byte("navigation mesh")); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			body := make([]byte, 1036+pak.Len())
			copy(body, "VBSP")
			binary.LittleEndian.PutUint32(body[8+40*16:], 1036)
			binary.LittleEndian.PutUint32(body[12+40*16:], uint32(pak.Len()))
			copy(body[1036:], pak.Bytes())
			entry := bspTestEntry(t, body)
			got, err := archiveBSPNavigation(entry, "mvm_embedded")
			if err != nil || got != test.want {
				t.Fatalf("navigation=%t, err=%v; want %t", got, err, test.want)
			}
		})
	}
	for _, body := range [][]byte{[]byte("truncated"), make([]byte, 1036)} {
		if _, err := archiveBSPNavigation(bspTestEntry(t, body), "mvm_embedded"); err == nil {
			t.Fatal("accepted malformed BSP")
		}
	}
	body := make([]byte, 1036)
	copy(body, "VBSP")
	binary.LittleEndian.PutUint32(body[8+40*16:], 1036)
	binary.LittleEndian.PutUint32(body[12+40*16:], 100)
	if _, err := archiveBSPNavigation(bspTestEntry(t, body), "mvm_embedded"); err == nil {
		t.Fatal("accepted out-of-bounds pak lump")
	}
	body = make([]byte, 1036)
	copy(body, "VBSP")
	if got, err := archiveBSPNavigation(bspTestEntry(t, body), "mvm_embedded"); err != nil || got {
		t.Fatalf("empty pak: %t, %v", got, err)
	}
}

func bspTestEntry(t *testing.T, body []byte) *zip.File {
	t.Helper()
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	file, err := writer.Create("maps/mvm_embedded.bsp")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(archive.Bytes()), int64(archive.Len()))
	if err != nil {
		t.Fatal(err)
	}
	return reader.File[0]
}

func TestRecoveredNavigationUpdatesRequirementWithoutChangingIdentity(t *testing.T) {
	for _, body := range []string{"WaveSchedule { Wave {} }", "WaveSchedule\n{\nLuaScriptFile example.lua\nWave {}\n}"} {
		catalog := manifest{Missions: []mission{{ID: 107, PopFile: "mvm_embedded_adv_test", Name: "Handwritten title", MapID: 106, Requires: "no_nav"}}}
		added, marked, err := addMissions(&catalog, map[string]uint8{"mvm_embedded": 106}, map[string]bool{"mvm_embedded": true}, map[string]population{"mvm_embedded_adv_test": {body: []byte(body)}}, nil)
		want := ""
		if strings.Contains(body, "LuaScriptFile") {
			want = "sigsegv-mvm"
		}
		if err != nil || added != 0 || marked != 1 {
			t.Fatalf("added=%d, marked=%d, err=%v", added, marked, err)
		}
		row := catalog.Missions[0]
		if row.ID != 107 || row.Name != "Handwritten title" || row.Requires != want {
			t.Fatalf("updated row: %+v; want requirement %q", row, want)
		}
	}
}
