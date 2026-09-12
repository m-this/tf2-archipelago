package gamedata

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// The exported JSON is what the Python apworld reads at import time. It is
// committed so the apworld is standalone, and CI regenerates it to catch a
// stale copy. Every file repeats format_version so the apworld can refuse a
// shape it does not know. See ADR 0001.

const (
	FileMeta          = "meta.json"
	FileMissions      = "missions.json"
	FileItems         = "items.json"
	FileWeaponClasses = "weapon_classes.json"
)

type metaFile struct {
	FormatVersion int              `json:"format_version"`
	Game          string           `json:"game"`
	BaseID        int64            `json:"base_id"`
	Difficulties  []difficultyJSON `json:"difficulties"`
	Maps          []mapJSON        `json:"maps"`
	Classes       []classJSON      `json:"classes"`
	WeaponSlots   []weaponSlotJSON `json:"weapon_slots"`
	ServerMods    []serverModJSON  `json:"server_mods"`
}

type serverModJSON struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type difficultyJSON struct {
	ID   Difficulty `json:"id"`
	Key  string     `json:"key"`
	Name string     `json:"name"`
}

type mapJSON struct {
	ID        MapID  `json:"id"`
	Name      string `json:"name"`
	Community bool   `json:"community"`
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
	HasTank    bool           `json:"has_tank"`
	HasGiant   bool           `json:"has_giant"`
	Community  bool           `json:"community"`
	Playable   bool           `json:"playable"`
	Requires   string         `json:"requires,omitempty"`
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
	WeaponBuffID   uint16    `json:"weapon_buff_id,omitempty"`
	Stackable      bool      `json:"stackable,omitempty"`
	Eligible       bool      `json:"eligible,omitempty"`
}

type weaponClassesFile struct {
	FormatVersion int                 `json:"format_version"`
	Weapons       []weaponClassesJSON `json:"weapons"`
}

type weaponClassesJSON struct {
	Name    string   `json:"name"`
	Classes []string `json:"classes"`
	Icon    string   `json:"icon"`
	Aliases []string `json:"aliases,omitempty"`
}

// Export writes the generated data files into dir, replacing what is there.
func Export(dir string) error {
	if err := Validate(); err != nil {
		return fmt.Errorf("gamedata is not valid, refusing to export: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	files := map[string]any{
		FileMeta:          buildMetaFile(),
		FileMissions:      buildMissionsFile(),
		FileItems:         buildItemsFile(),
		FileWeaponClasses: buildWeaponClassesFile(),
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
		meta.Maps = append(meta.Maps, mapJSON{m.ID, m.Name, IsCommunityMap(m.ID)})
	}
	// Named field by field rather than converted: Class carries a slot order
	// the apworld has no use for, and the export must not grow one silently.
	for _, c := range Classes {
		meta.Classes = append(meta.Classes, classJSON{c.ID, c.Key, c.Name})
	}
	for _, s := range WeaponSlots {
		meta.WeaponSlots = append(meta.WeaponSlots, weaponSlotJSON(s))
	}
	for _, mod := range ServerMods {
		meta.ServerMods = append(meta.ServerMods, serverModJSON{mod.Key, mod.Name})
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
			HasTank:    m.HasTank,
			HasGiant:   m.HasGiant,
			Community:  IsCommunityMission(m.ID),
			Playable:   IsPlayableMission(m.ID),
			Requires:   MissionRequirement(m.ID),
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
		stackable := false
		eligible := false
		if it.Kind == ItemWeaponBuff {
			if buff, ok := WeaponBuffByID(it.WeaponBuff); ok {
				stackable = buff.Mode != BuffToggle
				eligible = buff.Eligible
			}
		}
		file.Items = append(file.Items, itemJSON{
			ID:             it.ID,
			Name:           it.Name,
			Kind:           it.Kind.Key(),
			Classification: it.Classification.Key(),
			Count:          it.Count,
			MissionID:      it.Mission,
			ClassID:        it.Class,
			Credits:        it.Credits,
			WeaponBuffID:   it.WeaponBuff,
			Stackable:      stackable,
			Eligible:       eligible,
		})
	}
	return file
}

func buildWeaponClassesFile() weaponClassesFile {
	file := weaponClassesFile{
		FormatVersion: FormatVersion,
		Weapons:       make([]weaponClassesJSON, 0, len(BuffWeapons)),
	}
	for _, weapon := range BuffWeapons {
		if weapon.ID != weapon.ApplyID {
			continue
		}
		classes := weaponClassNames(weapon.Name, weapon.DefIndexes)
		if len(classes) != 0 {
			displayName := weapon.DisplayName()
			file.Weapons = append(file.Weapons, weaponClassesJSON{
				Name: displayName, Classes: classes, Icon: trackerItemIconPath(displayName),
				Aliases: weaponFamilyAliases(weapon.Name),
			})
		}
	}
	return file
}

func weaponFamilyAliases(canonical string) []string {
	for _, family := range weaponFamilies {
		if family[0] != canonical {
			continue
		}
		// A renamed family needs its internal canonical name exposed as an alias;
		// normally only the remaining, equivalent item definitions are aliases.
		if canonical == allClassMeleeName {
			return slices.Clone(family)
		}
		return slices.Clone(family[1:])
	}
	return nil
}

var tfWikiItemIconNames = map[string]string{
	"All-Class Melee":  "Conscientious Objector",
	"Construction PDA": "PDA Build",
	"Destruction PDA":  "PDA Destroy",
	"Force-a-Nature":   "Force-A-Nature",
	"Invis Watch":      "Invisibility Watch",
	"Sapper":           "Electro Sapper",
	"SMG":              "Submachine Gun",
	"Übersaw":          "Ubersaw",
}

func trackerItemIconPath(weapon string) string {
	if filename, ok := tfWikiItemIconNames[weapon]; ok {
		weapon = filename
	}
	filename := "Item_icon_" + strings.ReplaceAll(weapon, " ", "_") + ".png"
	hash := fmt.Sprintf("%x", md5.Sum([]byte(filename)))
	return "assets/tf2/items/" + hash + ".png"
}
