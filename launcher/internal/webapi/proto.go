package webapi

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/m-this/tf2-archipelago/gamedata"

	"github.com/m-this/tf2-archipelago/launcher/internal/botlive"
	"github.com/m-this/tf2-archipelago/launcher/internal/form"
	launcherv1 "github.com/m-this/tf2-archipelago/launcher/internal/gen/tf2ap/launcher/v1"
	apruntime "github.com/m-this/tf2-archipelago/launcher/internal/runtime"
	"github.com/m-this/tf2-archipelago/launcher/internal/session"
)

// This file is the whole of the translation between what the launcher holds and
// what goes on the wire. Nothing else in the package knows proto exists, and
// nothing the browser reads is assembled anywhere else.

// statusNames maps the word Snapshot carries to the enum. A status the launcher
// invents and nobody added here reaches the browser as unspecified, which the
// shell draws as "unknown" rather than silently as "stopped".
var statusNames = map[string]launcherv1.ServerStatus{
	"stopped":  launcherv1.ServerStatus_SERVER_STATUS_STOPPED,
	"starting": launcherv1.ServerStatus_SERVER_STATUS_STARTING,
	"running":  launcherv1.ServerStatus_SERVER_STATUS_RUNNING,
}

// Proto is the wire form of one draw.
//
// Exported because the fake launcher the browser tests run against builds a
// Snapshot by hand and has to encode it exactly as the real one does. A second
// encoder written for the tests would be a test of itself.
func (s Snapshot) Proto() *launcherv1.Snapshot {
	return &launcherv1.Snapshot{
		Title:         s.Title,
		Slot:          s.Slot,
		Status:        statusNames[s.Status],
		Running:       s.Running,
		Busy:          s.Busy,
		Room:          s.Room,
		Join:          s.Join,
		JoinUrl:       s.JoinURL,
		Mission:       s.Mission,
		Logs:          logsProto(s.Logs),
		Session:       sessionProto(s.Session),
		SessionError:  s.SessionError,
		Bots:          botsProto(s.Bots),
		DrawnBots:     s.DrawnBots,
		Form:          modelProto(s.Form),
		FormPage:      s.FormPage,
		Notice:        s.Notice,
		NoticeSeq:     s.NoticeSeq,
		ItemServer:    s.ItemServer,
		MissionPool:   poolProto(s.MissionPool),
		RestartNeeded: s.RestartNeeded,
	}
}

func logsProto(lines []apruntime.Line) []*launcherv1.LogLine {
	out := make([]*launcherv1.LogLine, 0, len(lines))
	for _, line := range lines {
		out = append(out, lineProto(line))
	}
	return out
}

func lineProto(line apruntime.Line) *launcherv1.LogLine {
	return &launcherv1.LogLine{
		At:     timestamppb.New(line.At),
		Source: line.Source,
		Text:   line.Text,
	}
}

func sessionProto(s session.Snapshot) *launcherv1.Session {
	missions := make([]*launcherv1.SessionMission, 0, len(s.Missions))
	for _, mission := range s.Missions {
		missions = append(missions, &launcherv1.SessionMission{
			PopFile: mission.PopFile, Name: mission.Name, Map: mission.Map,
			Waves: int32(mission.Waves), Source: mission.Source, Loadout: mission.Loadout,
			Unlocked: mission.Unlocked, Cleared: mission.Cleared, Played: mission.Played,
			Tier: tierOf(mission.PopFile), WaveReached: int32(mission.WaveReached),
		})
	}
	unlocks := make([]*launcherv1.SessionUnlock, 0, len(s.Unlocks))
	for _, unlock := range s.Unlocks {
		unlocks = append(unlocks, &launcherv1.SessionUnlock{
			Kind: unlock.Kind, Name: unlock.Name, Level: int32(unlock.Level),
		})
	}
	return &launcherv1.Session{
		Health: &launcherv1.SessionHealth{
			Connected: s.Health.Connected, Slot: s.Health.Slot, Seed: s.Health.Seed,
			Checks: int32(s.Health.Checks), Items: int32(s.Health.Items),
			DeathLink: s.Health.DeathLink, GoalSent: s.Health.GoalSent,
			LastCheck: s.Health.LastCheck, LastError: s.Health.LastError,
		},
		Missions: missions,
		Unlocks:  unlocks,
	}
}

func botsProto(seats []botlive.Seat) []*launcherv1.BotSeat {
	out := make([]*launcherv1.BotSeat, 0, len(seats))
	for _, seat := range seats {
		out = append(out, &launcherv1.BotSeat{
			Number: int32(seat.Number), Class: seat.Class, Weapons: seat.Weapons,
		})
	}
	return out
}

func poolProto(rows []MissionPoolRow) []*launcherv1.MissionPoolRow {
	out := make([]*launcherv1.MissionPoolRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, &launcherv1.MissionPoolRow{
			Field: row.Field, Source: row.Source, Map: row.Map, Name: row.Name,
			Waves: row.Waves, Compatibility: row.Compatibility, Mods: row.Mods, Tier: row.Tier,
			Disabled: row.Disabled,
		})
	}
	return out
}

// modelProto answers nil for a closed settings screen, which is the presence
// the browser reads: a Model with no tabs is a screen with nothing on it, and
// those are two different things.
func modelProto(model *form.Model) *launcherv1.Model {
	if model == nil {
		return nil
	}
	tabs := make([]*launcherv1.Tab, 0, len(model.Tabs))
	for _, tab := range model.Tabs {
		fields := make([]*launcherv1.Field, 0, len(tab.Fields))
		for _, field := range tab.Fields {
			fields = append(fields, fieldProto(field))
		}
		tabs = append(tabs, &launcherv1.Tab{
			Title: tab.Title, Intro: tab.Intro, Under: tab.Under, Fields: fields,
		})
	}
	return &launcherv1.Model{Tabs: tabs}
}

func fieldProto(field form.Field) *launcherv1.Field {
	options := make([]*launcherv1.Option, 0, len(field.Options))
	for _, option := range field.Options {
		options = append(options, &launcherv1.Option{Value: option.Value, Label: option.Label})
	}
	return &launcherv1.Field{
		Id:          field.ID,
		Kind:        launcherv1.Kind(field.Kind),
		Label:       field.Label,
		Help:        field.Help,
		Group:       field.Group,
		Bar:         field.Bar,
		Value:       field.Value,
		Placeholder: field.Placeholder,
		Low:         int32(field.Low),
		High:        int32(field.High),
		Options:     options,
		Hint:        field.Hint,
		Warning:     field.Warning,
		HintOff:     field.HintOff,
		Browse:      field.Browse,
		Deferred:    field.Deferred,
		Disabled:    field.Disabled,
		Reason:      field.Reason,
	}
}

// Proto is the wire form of the settings screen. Exported for the same reason
// Snapshot.Proto is.
func (screen Screen) Proto() *launcherv1.Screen {
	return &launcherv1.Screen{
		Model:         modelProto(screen.Form),
		Page:          screen.Page,
		MissionPool:   poolProto(screen.MissionPool),
		RestartNeeded: screen.RestartNeeded,
	}
}

// changeFrom is the one direction that goes the other way: an answer the player
// gave, on its way to form.Apply, which is where a value becomes an int or a
// bool and where a bad one gets the message the player reads.
func changeFrom(change *launcherv1.Change) form.Change {
	return form.Change{Field: change.GetField(), Value: change.GetValue()}
}

// tierOf reads a mission's tier out of gamedata. The bridge does not carry one,
// and a browser guessing it from the name would be guessing.
func tierOf(popFile string) string {
	mission, known := gamedata.MissionByPopFile(popFile)
	if !known {
		return ""
	}
	return mission.Difficulty.String()
}
