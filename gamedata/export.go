package gamedata

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// The exported JSON is what the Python apworld reads at import time. It is
// committed so the apworld is standalone, and CI regenerates it to catch a
// stale copy. Every file repeats format_version so the apworld can refuse a
// shape it does not know. See ADR 0001.

const (
	FileMeta     = "meta.json"
	FileMissions = "missions.json"
	FileItems    = "items.json"
)

type metaFile struct {
	FormatVersion int              `json:"format_version"`
	Game          string           `json:"game"`
	BaseID        int64            `json:"base_id"`
	Difficulties  []difficultyJSON `json:"difficulties"`
	Maps          []mapJSON        `json:"maps"`
	Classes       []classJSON      `json:"classes"`
	WeaponSlots   []weaponSlotJSON `json:"weapon_slots"`
}

type difficultyJSON struct {
	ID   Difficulty `json:"id"`
	Key  string     `json:"key"`
	Name string     `json:"name"`
}

type mapJSON struct {
	ID   MapID  `json:"id"`
	Name string `json:"name"`
}

type classJSON struct {
	ID   ClassID `json:"id"`
	Key  string  `json:"key"`
	Name string  `json:"name"`
}

type weaponSlotJSON struct {
	ID   WeaponSlotID `json:"id"`
	Key  string       `json:"key"`
	Name string       `json:"name"`
}

type missionsFile struct {
	FormatVersion int           `json:"format_version"`
	Missions      []missionJSON `json:"missions"`
}

type missionJSON struct {
	ID         MissionID      `json:"id"`
	PopFile    string         `json:"pop_file"`
	Name       string         `json:"name"`
	MapID      MapID          `json:"map_id"`
	Difficulty string         `json:"difficulty"`
	Waves      uint8          `json:"waves"`
	Locations  []locationJSON `json:"locations"`
}

type locationJSON struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
	Wave uint8  `json:"wave,omitempty"`
}

type itemsFile struct {
	FormatVersion int        `json:"format_version"`
	Items         []itemJSON `json:"items"`
}

type itemJSON struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	Kind           string    `json:"kind"`
	Classification string    `json:"classification"`
	Count          uint8     `json:"count"`
	MissionID      MissionID `json:"mission_id,omitempty"`
	ClassID        ClassID   `json:"class_id,omitempty"`
	Credits        uint16    `json:"credits,omitempty"`
}

// Export writes the three data files into dir, replacing what is there.
func Export(dir string) error {
	if err := Validate(); err != nil {
		return fmt.Errorf("gamedata is not valid, refusing to export: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	files := map[string]any{
		FileMeta:     buildMetaFile(),
		FileMissions: buildMissionsFile(),
		FileItems:    buildItemsFile(),
	}
	for name, content := range files {
		body, err := json.MarshalIndent(content, "", "  ")
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), append(body, '\n'), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func buildMetaFile() metaFile {
	meta := metaFile{
		FormatVersion: FormatVersion,
		Game:          GameName,
		BaseID:        BaseID,
		Difficulties:  make([]difficultyJSON, 0, len(Difficulties)),
		Maps:          make([]mapJSON, 0, len(Maps)),
		Classes:       make([]classJSON, 0, len(Classes)),
		WeaponSlots:   make([]weaponSlotJSON, 0, len(WeaponSlots)),
	}
	for _, d := range Difficulties {
		meta.Difficulties = append(meta.Difficulties, difficultyJSON{d, d.Key(), d.String()})
	}
	for _, m := range Maps {
		meta.Maps = append(meta.Maps, mapJSON(m))
	}
	for _, c := range Classes {
		meta.Classes = append(meta.Classes, classJSON(c))
	}
	for _, s := range WeaponSlots {
		meta.WeaponSlots = append(meta.WeaponSlots, weaponSlotJSON(s))
	}
	return meta
}

func buildMissionsFile() missionsFile {
	byMission := make(map[MissionID][]locationJSON, len(Missions))
	for _, l := range Locations {
		byMission[l.Mission] = append(byMission[l.Mission], locationJSON{l.ID, l.Name, l.Kind.Key(), l.Wave})
	}
	file := missionsFile{
		FormatVersion: FormatVersion,
		Missions:      make([]missionJSON, 0, len(Missions)),
	}
	for _, m := range Missions {
		file.Missions = append(file.Missions, missionJSON{
			ID:         m.ID,
			PopFile:    m.PopFile,
			Name:       m.Name,
			MapID:      m.Map,
			Difficulty: m.Difficulty.Key(),
			Waves:      m.Waves,
			Locations:  byMission[m.ID],
		})
	}
	return file
}

func buildItemsFile() itemsFile {
	file := itemsFile{
		FormatVersion: FormatVersion,
		Items:         make([]itemJSON, 0, len(Items)),
	}
	for _, it := range Items {
		file.Items = append(file.Items, itemJSON{
			ID:             it.ID,
			Name:           it.Name,
			Kind:           it.Kind.Key(),
			Classification: it.Classification.Key(),
			Count:          it.Count,
			MissionID:      it.Mission,
			ClassID:        it.Class,
			Credits:        it.Credits,
		})
	}
	return file
}
