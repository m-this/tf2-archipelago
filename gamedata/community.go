package gamedata

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// community.json is deliberately part of gamedata rather than runtime
// configuration. Archipelago seeds record numeric ids, so every process that
// handles a seed must be built from the same immutable community manifest.
//
//go:embed community.json
var communityJSON []byte

const CommunityFormatVersion = 1

// CommunityIDMin leaves room for Valve to add maps and missions without
// colliding with an operator's manifest. IDs at and above this value belong to
// community content and, like every Archipelago-facing ID, must never be
// reused for something else.
const CommunityIDMin = 100

type communityFile struct {
	FormatVersion int                `json:"format_version"`
	Maps          []communityMap     `json:"maps"`
	Missions      []communityMission `json:"missions"`
}

type communityMap struct {
	ID   MapID  `json:"id"`
	Name string `json:"name"`
}

type communityMission struct {
	ID         MissionID `json:"id"`
	PopFile    string    `json:"pop_file"`
	Name       string    `json:"name"`
	MapID      MapID     `json:"map_id"`
	Difficulty string    `json:"difficulty"`
	Waves      uint8     `json:"waves"`
	HasTank    bool      `json:"has_tank"`
	HasGiant   bool      `json:"has_giant"`
	Requires   string    `json:"requires,omitempty"`
	Pack       string    `json:"pack,omitempty"`
}

type loadedCommunity struct {
	Maps         []Map
	Missions     []Mission
	Requirements map[MissionID]string
	Packs        map[MissionID]string
}

func loadCommunity(body []byte) (loadedCommunity, error) {
	var file communityFile
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return loadedCommunity{}, fmt.Errorf("decode community.json: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return loadedCommunity{}, fmt.Errorf("decode community.json: trailing content")
	}
	if file.FormatVersion != CommunityFormatVersion {
		return loadedCommunity{}, fmt.Errorf(
			"community.json format version is %d, expected %d",
			file.FormatVersion, CommunityFormatVersion,
		)
	}

	content := loadedCommunity{
		Maps:         make([]Map, 0, len(file.Maps)),
		Missions:     make([]Mission, 0, len(file.Missions)),
		Requirements: make(map[MissionID]string, len(file.Missions)),
		Packs:        make(map[MissionID]string, len(file.Missions)),
	}
	for _, entry := range file.Maps {
		if entry.ID < CommunityIDMin {
			return loadedCommunity{}, fmt.Errorf("community map %q: id %d must be at least %d", entry.Name, entry.ID, CommunityIDMin)
		}
		content.Maps = append(content.Maps, Map(entry))
	}
	for _, entry := range file.Missions {
		if entry.ID < CommunityIDMin {
			return loadedCommunity{}, fmt.Errorf("community mission %q: id %d must be at least %d", entry.PopFile, entry.ID, CommunityIDMin)
		}
		difficulty, known := DifficultyByKey(entry.Difficulty)
		if !known {
			return loadedCommunity{}, fmt.Errorf("community mission %q: unknown difficulty %q", entry.PopFile, entry.Difficulty)
		}
		if entry.Requires != "" && entry.Requires != "no_nav" {
			return loadedCommunity{}, fmt.Errorf("community mission %q: unknown requirement %q", entry.PopFile, entry.Requires)
		}
		if entry.Pack != "" && entry.Pack != "mlarchive-assets.zip" {
			return loadedCommunity{}, fmt.Errorf("community mission %q: unknown pack %q", entry.PopFile, entry.Pack)
		}
		content.Missions = append(content.Missions, Mission{
			ID:         entry.ID,
			PopFile:    entry.PopFile,
			Name:       entry.Name,
			Map:        entry.MapID,
			Difficulty: difficulty,
			Waves:      entry.Waves,
			HasTank:    entry.HasTank,
			HasGiant:   entry.HasGiant,
		})
		content.Requirements[entry.ID] = entry.Requires
		if entry.Pack == "" {
			entry.Pack = "archive-assets.zip"
		}
		content.Packs[entry.ID] = entry.Pack
	}
	return content, nil
}

// ValidateCommunityFiles checks that the immutable manifest and the TF2
// content tree describe the same pack. It runs before export/build so a typo
// cannot become an Archipelago seed whose locations are impossible to reach.
func ValidateCommunityFiles(tfRoot string) error {
	return ValidateCommunitySources(tfRoot)
}

// ValidateCommunitySources accepts extracted TF roots and Potato-style ZIPs,
// combining them before reporting anything missing.
func ValidateCommunitySources(sources ...string) error {
	required := make(map[string]string)
	populations := make(map[string][][]byte)
	for _, m := range communityMaps {
		required[filepath.ToSlash(filepath.Join("maps", m.Name+".bsp"))] = "map " + m.Name
	}
	for _, m := range communityMissions {
		if IsPlayableMission(m.ID) {
			if played, ok := MapByID(m.Map); ok {
				required[filepath.ToSlash(filepath.Join("maps", played.Name+".nav"))] = "navigation mesh " + played.Name
			}
		}
		required[filepath.ToSlash(filepath.Join("scripts", "population", m.PopFile+".pop"))] = "mission " + m.PopFile
	}
	for _, source := range sources {
		info, err := os.Stat(source)
		if err != nil {
			return err
		}
		if info.IsDir() {
			for relative := range required {
				path := filepath.Join(source, filepath.FromSlash(relative))
				if file, err := os.Stat(path); err == nil && file.Mode().IsRegular() {
					delete(required, relative)
					if strings.HasSuffix(relative, ".pop") {
						if body, err := os.ReadFile(path); err == nil {
							populations[relative] = append(populations[relative], body)
						}
					}
				}
			}
			continue
		}
		reader, err := zip.OpenReader(source)
		if err != nil {
			return fmt.Errorf("cannot read community archive %s: %w", source, err)
		}
		for _, file := range reader.File {
			name := filepath.ToSlash(file.Name)
			name = strings.TrimPrefix(name, "tf/download/")
			name = strings.TrimPrefix(name, "tf/")
			if _, wanted := required[name]; wanted && strings.HasSuffix(name, ".pop") {
				if opened, err := file.Open(); err == nil {
					if body, err := io.ReadAll(opened); err == nil {
						populations[name] = append(populations[name], body)
					}
					_ = opened.Close()
				}
			}
			delete(required, name)
		}
		_ = reader.Close()
	}
	for relative, description := range required {
		return fmt.Errorf("community %s is missing: %s", description, relative)
	}
	for _, mission := range communityMissions {
		relative := filepath.ToSlash(filepath.Join("scripts", "population", mission.PopFile+".pop"))
		var found []populationFacts
		matches := false
		for _, body := range populations[relative] {
			facts := inspectPopulation(body)
			found = append(found, facts)
			if facts.Waves == int(mission.Waves) && facts.HasTank == mission.HasTank && facts.HasGiant == mission.HasGiant {
				matches = true
				break
			}
		}
		if !matches {
			return fmt.Errorf(
				"community mission %s metadata is waves=%d tank=%t giant=%t, population file reports %v",
				mission.PopFile, mission.Waves, mission.HasTank, mission.HasGiant, found,
			)
		}
	}
	return nil
}

type populationFacts struct {
	Waves    int
	HasTank  bool
	HasGiant bool
}

func (f populationFacts) String() string {
	return fmt.Sprintf("waves=%d tank=%t giant=%t", f.Waves, f.HasTank, f.HasGiant)
}

// inspectPopulation reads only the stock population syntax needed to decide
// which checks a mission can satisfy. Template names count as giant evidence:
// community missions commonly put Attributes MiniBoss in a #base file but
// refer to the active spawner as T_TFBot_Giant_* or another *_Giant_* name.
func inspectPopulation(body []byte) populationFacts {
	tokens := populationTokens(body)
	var facts populationFacts
	for i, token := range tokens {
		nextIsBlock := i+1 < len(tokens) && tokens[i+1] == "{"
		switch {
		case strings.EqualFold(token, "Wave") && nextIsBlock:
			facts.Waves++
		case strings.EqualFold(token, "Tank") && nextIsBlock:
			facts.HasTank = true
		case strings.EqualFold(token, "MiniBoss"):
			facts.HasGiant = true
		case strings.EqualFold(token, "Template") && i+1 < len(tokens):
			template := strings.ToLower(tokens[i+1])
			if strings.Contains(template, "giant") && !strings.Contains(template, "sentrybuster") {
				facts.HasGiant = true
			}
		}
	}
	return facts
}

// populationTokens removes comments and preserves quoted template names. It
// is intentionally smaller than a full KeyValues parser, since population
// files also contain event-specific extensions that stock KeyValues rejects.
func populationTokens(body []byte) []string {
	tokens := make([]string, 0, len(body)/8)
	for i := 0; i < len(body); {
		switch {
		case body[i] == '/' && i+1 < len(body) && body[i+1] == '/':
			i += 2
			for i < len(body) && body[i] != '\n' {
				i++
			}
		case body[i] == '{' || body[i] == '}':
			tokens = append(tokens, string(body[i]))
			i++
		case body[i] == '"':
			i++
			start := i
			for i < len(body) && body[i] != '"' {
				if body[i] == '\\' && i+1 < len(body) {
					i += 2
					continue
				}
				i++
			}
			tokens = append(tokens, string(body[start:i]))
			if i < len(body) {
				i++
			}
		case body[i] == ' ' || body[i] == '\t' || body[i] == '\r' || body[i] == '\n':
			i++
		default:
			start := i
			for i < len(body) && body[i] != ' ' && body[i] != '\t' && body[i] != '\r' && body[i] != '\n' && body[i] != '{' && body[i] != '}' && body[i] != '"' {
				if body[i] == '/' && i+1 < len(body) && body[i+1] == '/' {
					break
				}
				i++
			}
			if start != i {
				tokens = append(tokens, string(body[start:i]))
			}
		}
	}
	return tokens
}

func requireCommunityFile(root, relative, kind, name string) error {
	path := filepath.Join(root, relative)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("community %s %q is missing: %s", kind, name, path)
		}
		return fmt.Errorf("community %s %q: %w", kind, name, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("community %s %q is not a regular file: %s", kind, name, path)
	}
	return nil
}

func mustLoadCommunity() loadedCommunity {
	content, err := loadCommunity(communityJSON)
	if err != nil {
		panic("gamedata: " + err.Error())
	}
	return content
}

var (
	communityContent  = mustLoadCommunity()
	communityMaps     = communityContent.Maps
	communityMissions = communityContent.Missions
)

// IsCommunityMap reports whether the map came from community.json.
func IsCommunityMap(id MapID) bool {
	for _, m := range communityMaps {
		if m.ID == id {
			return true
		}
	}
	return false
}

// IsCommunityMission reports whether the mission came from community.json.
func IsCommunityMission(id MissionID) bool {
	_, ok := communityContent.Packs[id]
	return ok
}

// IsPlayableMission reports whether this build can safely put a mission in a
// seed. Community entries with a known compatibility requirement remain
// visible to launchers but are never offered or drawn.
func IsPlayableMission(id MissionID) bool {
	if !IsCommunityMission(id) {
		return true
	}
	return communityContent.Requirements[id] == ""
}

// PlayableMissions is the pool launchers and test rooms may offer.
func PlayableMissions() []Mission {
	out := make([]Mission, 0, len(Missions))
	for _, mission := range Missions {
		if IsPlayableMission(mission.ID) {
			out = append(out, mission)
		}
	}
	return out
}

// MissionPack is the archive that owns a community mission. Valve missions
// return blank. The names deliberately match the launcher's persisted pack
// names without importing the launcher from gamedata.
func MissionPack(id MissionID) string {
	return communityContent.Packs[id]
}

// MissionRequirement reports why a cataloged community mission is not a
// portable choice. Blank means the mission is supported by the stock server.
// Launchers use this to show unavailable content without making it seedable.
func MissionRequirement(id MissionID) string {
	return communityContent.Requirements[id]
}
