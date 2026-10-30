// Command communitycatalog appends every mission whose map is present in the
// official Potato and Moonlight archives to gamedata/community.json. Existing
// ids and hand-written names are never changed; new identities are assigned in
// popfile order after the greatest committed id.
package main

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/m-this/tf2-archipelago/gamedata"
)

type manifest struct {
	FormatVersion int       `json:"format_version"`
	Maps          []mapRow  `json:"maps"`
	Missions      []mission `json:"missions"`
}

type mapRow struct {
	ID   uint8  `json:"id"`
	Name string `json:"name"`
}

type mission struct {
	ID         uint16 `json:"id"`
	PopFile    string `json:"pop_file"`
	Name       string `json:"name"`
	MapID      uint8  `json:"map_id"`
	Difficulty string `json:"difficulty"`
	Waves      uint8  `json:"waves"`
	HasTank    bool   `json:"has_tank"`
	HasGiant   bool   `json:"has_giant"`
	Requires   string `json:"requires,omitempty"`
	Pack       string `json:"pack,omitempty"`
	Loadout    string `json:"loadout,omitempty"`
}

type population struct {
	body []byte
	pack string
}

func main() {
	output := flag.String("out", "gamedata/community.json", "manifest to update")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: communitycatalog [-out gamedata/community.json] ARCHIVE.zip [ARCHIVE.zip ...]")
		os.Exit(2)
	}
	if err := run(*output, flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(output string, sources []string) error {
	body, err := os.ReadFile(output)
	if err != nil {
		return err
	}
	var catalog manifest
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&catalog); err != nil {
		return err
	}

	bsp := make(map[string]bool)
	nav := make(map[string]bool)
	populations := make(map[string]population)
	allPopulations := make(map[string]map[string][][]byte)
	for _, source := range sources {
		if err := readArchive(source, bsp, nav, populations, allPopulations); err != nil {
			return err
		}
	}

	mapIDs, addedMaps, err := addMaps(&catalog, bsp)
	if err != nil {
		return err
	}
	added, marked, err := addMissions(&catalog, mapIDs, nav, populations, allPopulations)
	if err != nil {
		return err
	}

	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(catalog); err != nil {
		return err
	}
	temporary := output + ".new"
	if err := os.WriteFile(temporary, encoded.Bytes(), 0o644); err != nil {
		return err
	}
	if err := os.Rename(temporary, output); err != nil {
		return err
	}
	fmt.Printf("added %d maps and %d missions; updated %d existing mission requirements\n", addedMaps, added, marked)
	return nil
}

func addMaps(catalog *manifest, bsp map[string]bool) (map[string]uint8, int, error) {
	mapIDs := make(map[string]uint8)
	var nextMapID uint8
	for _, played := range gamedata.Maps {
		mapIDs[played.Name] = uint8(played.ID)
		nextMapID = max(nextMapID, uint8(played.ID))
	}
	var newMaps []string
	for name := range bsp {
		if _, known := mapIDs[name]; !known {
			newMaps = append(newMaps, name)
		}
	}
	sort.Strings(newMaps)
	for _, name := range newMaps {
		if nextMapID == 255 {
			return nil, 0, errors.New("community map ids exhausted uint8")
		}
		nextMapID++
		mapIDs[name] = nextMapID
		catalog.Maps = append(catalog.Maps, mapRow{ID: nextMapID, Name: name})
	}
	return mapIDs, len(newMaps), nil
}

func addMissions(catalog *manifest, mapIDs map[string]uint8, nav map[string]bool, populations map[string]population, allPopulations map[string]map[string][][]byte) (int, int, error) {
	existing := make(map[string]int, len(catalog.Missions))
	names := make(map[string]bool, len(catalog.Missions))
	var nextID uint16
	for i := range catalog.Missions {
		existing[catalog.Missions[i].PopFile] = i
		names[catalog.Missions[i].Name] = true
		nextID = max(nextID, catalog.Missions[i].ID)
	}
	popFiles := make([]string, 0, len(populations))
	for popFile := range populations {
		popFiles = append(popFiles, popFile)
	}
	sort.Strings(popFiles)
	added, marked := 0, 0
	for _, popFile := range popFiles {
		row, requirement, ok, err := missionRow(popFile, mapIDs, nav, populations[popFile], allPopulations, nextID, names)
		if err != nil {
			return 0, 0, err
		}
		if !ok {
			continue
		}
		if at, found := existing[popFile]; found {
			previous := catalog.Missions[at].Requires
			if previous != requirement && (previous == "" || previous == "sigsegv-mvm" || previous == "no_nav") {
				catalog.Missions[at].Requires, marked = requirement, marked+1
			}
			if row.Loadout == "medic_only" || catalog.Missions[at].Loadout == "medic_only" {
				catalog.Missions[at].Loadout = row.Loadout
			}
			continue
		}
		nextID++
		row.ID = nextID
		catalog.Missions = append(catalog.Missions, row)
		added++
	}
	return added, marked, nil
}

func missionRow(popFile string, mapIDs map[string]uint8, nav map[string]bool, pop population, allPopulations map[string]map[string][][]byte, nextID uint16, names map[string]bool) (mission, string, bool, error) {
	played, ok := mapFor(popFile, mapIDs)
	if !ok {
		return mission{}, "", false, nil
	}
	requirement := ""
	if mapIDs[played] >= gamedata.CommunityIDMin && !nav[played] {
		requirement = "no_nav"
	} else if gamedata.CommunityPopulationRequiresSigModWithIncludes(pop.body, allPopulations[pop.pack]) {
		requirement = "sigsegv-mvm"
	}
	difficulty, title, ok := missionIdentity(popFile, played)
	if !ok {
		return mission{}, "", false, fmt.Errorf("cannot infer a difficulty from %s", popFile)
	}
	waves, hasTank, hasGiant := gamedata.InspectCommunityPopulation(pop.body)
	if waves < 1 || waves > 255 {
		return mission{}, "", false, fmt.Errorf("%s reports %d waves", popFile, waves)
	}
	if names[title] {
		title += " [" + strings.TrimPrefix(played, "mvm_") + "]"
	}
	for names[title] {
		title += "*"
	}
	names[title] = true
	row := mission{
		ID: nextID + 1, PopFile: popFile, Name: title, MapID: mapIDs[played], Difficulty: difficulty,
		Waves: uint8(waves), HasTank: hasTank, HasGiant: hasGiant, Requires: requirement,
	}
	if pop.pack == "mlarchive-assets.zip" {
		row.Pack = pop.pack
	}
	if gamedata.CommunityMissionMedicOnlyForCatalog(popFile, pop.body) {
		row.Loadout = "medic_only"
	} else if strings.Contains(strings.ToLower(popFile), "medieval") {
		row.Loadout = "medieval"
	}
	return row, requirement, true, nil
}

func readArchive(path string, bsp, nav map[string]bool, populations map[string]population, allPopulations map[string]map[string][][]byte) error {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()
	pack := filepath.Base(path)
	if allPopulations[pack] == nil {
		allPopulations[pack] = make(map[string][][]byte)
	}
	for _, entry := range reader.File {
		name := filepath.ToSlash(entry.Name)
		name = strings.TrimPrefix(name, "tf/download/")
		name = strings.TrimPrefix(name, "tf/")
		switch {
		case strings.HasPrefix(name, "maps/mvm_") && strings.HasSuffix(name, ".bsp"):
			mapName := strings.TrimSuffix(filepath.Base(name), ".bsp")
			bsp[mapName] = true
			embedded, err := archiveBSPNavigation(entry, mapName)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			if embedded {
				nav[mapName] = true
			}
		case strings.HasPrefix(name, "maps/mvm_") && strings.HasSuffix(name, ".nav"):
			nav[strings.TrimSuffix(filepath.Base(name), ".nav")] = true
		case strings.HasPrefix(name, "scripts/population/") && strings.HasSuffix(strings.ToLower(name), ".pop"):
			opened, err := entry.Open()
			if err != nil {
				return err
			}
			body, readErr := io.ReadAll(opened)
			_ = opened.Close()
			if readErr != nil {
				return readErr
			}
			base := strings.ToLower(filepath.Base(name))
			allPopulations[pack][base] = append(allPopulations[pack][base], body)
			if !strings.HasPrefix(filepath.Base(name), "mvm_") {
				continue
			}
			popFile := strings.TrimSuffix(filepath.Base(name), ".pop")
			// Moonlight is the smaller, specific source when both packs carry a
			// mission. This matches the existing manifest's source labels.
			if _, exists := populations[popFile]; !exists || pack == "mlarchive-assets.zip" {
				populations[popFile] = population{body: body, pack: pack}
			}
		}
	}
	return nil
}

func mapFor(popFile string, maps map[string]uint8) (string, bool) {
	best := ""
	for name := range maps {
		if (popFile == name || strings.HasPrefix(popFile, name+"_")) && len(name) > len(best) {
			best = name
		}
	}
	return best, best != ""
}

func missionIdentity(popFile, mapName string) (difficulty, title string, ok bool) {
	rest := strings.TrimPrefix(popFile, mapName+"_")
	parts := strings.Split(rest, "_")
	markers := map[string]string{
		"666": "haunted", "int": "intermediate", "adv": "advanced", "exp": "expert", "nor": "normal",
	}
	// A few haunted files say adv_666. The 666 marker is the more useful
	// player-facing tier wherever it appears.
	for at, part := range parts {
		if part == "666" {
			return markers[part], humanName(strings.Join(parts[at+1:], "_")), true
		}
	}
	for at, part := range parts {
		if found, exists := markers[part]; exists {
			return found, humanName(strings.Join(parts[at+1:], "_")), true
		}
	}
	if len(parts) > 1 && parts[0] == "rev" {
		return "advanced", humanName(strings.Join(parts[1:], "_")), true
	}
	return "", "", false
}

func humanName(value string) string {
	words := strings.Fields(strings.ReplaceAll(strings.Trim(value, "_"), "_", " "))
	for i, word := range words {
		runes := []rune(word)
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}

// Source checks the map-named nav and maps/embed.nav inside the BSP pak lump
// after looking for a loose file. Inspect that ZIP without extracting assets.
func archiveBSPNavigation(entry *zip.File, mapName string) (bool, error) {
	stream, err := entry.Open()
	if err != nil {
		return false, err
	}
	defer func() { _ = stream.Close() }()
	const headerSize = 8 + 64*16 + 4
	var header [headerSize]byte
	if _, err := io.ReadFull(stream, header[:]); err != nil {
		return false, err
	}
	if string(header[:4]) != "VBSP" {
		return false, errors.New("invalid BSP header")
	}
	const pakLump = 8 + 40*16
	offset := uint64(binary.LittleEndian.Uint32(header[pakLump:]))
	size := uint64(binary.LittleEndian.Uint32(header[pakLump+4:]))
	if size == 0 {
		return false, nil
	}
	if offset < headerSize || offset+size > entry.UncompressedSize64 || size > 512<<20 {
		return false, errors.New("invalid or oversized BSP pak lump")
	}
	if _, err := io.CopyN(io.Discard, stream, int64(offset)-headerSize); err != nil {
		return false, err
	}
	body := make([]byte, int(size))
	if _, err := io.ReadFull(stream, body); err != nil {
		return false, err
	}
	pak, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return false, err
	}
	for _, file := range pak.File {
		name := strings.ToLower(strings.ReplaceAll(file.Name, `\`, "/"))
		if file.UncompressedSize64 > 0 && (name == "maps/"+mapName+".nav" || name == "maps/embed.nav") {
			return true, nil
		}
	}
	return false, nil
}
