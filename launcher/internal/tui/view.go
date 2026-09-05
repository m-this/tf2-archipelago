package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/m-this/tf2-archipelago/launcher/internal/assets"
	apruntime "github.com/m-this/tf2-archipelago/launcher/internal/runtime"
	"github.com/m-this/tf2-archipelago/launcher/internal/session"
)

// The status light, in the colours every other status light uses, and the grey
// everything that is not the point is written in.
var (
	styleStopped = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	styleRunning = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	styleMuted   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleTab     = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleTabOn   = lipgloss.NewStyle().Bold(true).Underline(true)
	styleKey     = lipgloss.NewStyle().Bold(true)
	styleCleared = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	styleLocked  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

// chrome is what the body has to share the screen with: the two header lines,
// the tab line, the RCON line and the keys.
const chrome = 6

func (m *model) bodyHeight() int {
	return max(m.height-chrome, 3)
}

func (m *model) View() string {
	if !m.ready {
		return "starting...\n"
	}
	if m.form != nil {
		return m.form.view(m.width, m.height)
	}

	var out strings.Builder
	out.WriteString(m.header())
	out.WriteString("\n")
	out.WriteString(m.joinLine())
	out.WriteString("\n")
	out.WriteString(m.tabs())
	out.WriteString("\n")

	switch m.view {
	case viewBots:
		out.WriteString(m.bots())
	case viewLog:
		out.WriteString(m.log())
	case viewSession:
		out.WriteString(m.session())
	case viewUnlocks:
		out.WriteString(m.unlocks())
	}

	out.WriteString("\n")
	out.WriteString(m.prompt())
	out.WriteString("\n")
	out.WriteString(m.keys())
	return out.String()
}

func (m *model) header() string {
	light, status := styleStopped.Render("●"), m.status
	if m.supervisor.Running() {
		light = styleRunning.Render("●")
	}

	parts := []string{light, status}
	if room := m.roomLine(); room != "" {
		parts = append(parts, styleMuted.Render(room))
	}
	if m.mission != "" {
		parts = append(parts, styleMuted.Render(m.mission))
	}
	parts = append(parts, styleMuted.Render(m.summary()))
	// Which build this is, for the same reason the window puts it in its title:
	// several builds carry one version and the question is always which.
	if version := assets.LauncherVersion; version != "" {
		parts = append(parts, styleMuted.Render(version))
	}
	return truncate(strings.Join(parts, "  "), m.width)
}

func (m *model) roomLine() string {
	if m.settings.TestMode {
		return "test mode, no room"
	}
	if m.settings.APPort == 0 {
		return "no room set"
	}
	return fmt.Sprintf("room %s:%d", m.settings.APHost, m.settings.APPort)
}

// joinLine is what a friend types after connect. Every address of this machine,
// because nothing here knows which network the friends are on, and the relayed
// one once Steam has handed it over.
func (m *model) joinLine() string {
	// Truncated, because a machine with a container runtime on it has an
	// address per bridge and they do not all fit. Copy takes the whole list.
	return truncate(styleMuted.Render("join  ")+strings.Join(m.joinAddresses(), "   "), m.width)
}

// joinAddresses is the same list as plain text, which is what the clipboard
// takes and what a friend is sent.
func (m *model) joinAddresses() []string {
	addresses := []string{}
	// Named, and first. Over Steam this is the address: the ones under it are
	// this network's and mean nothing to the friend being sent them.
	if m.steamURL != "" {
		addresses = append(addresses, "Steam public IP: "+m.steamURL)
	}
	for _, address := range apruntime.LocalAddresses() {
		addresses = append(addresses, fmt.Sprintf("%s:%d", address, m.settings.SrcdsPort))
	}
	if len(addresses) == 0 {
		addresses = append(addresses, fmt.Sprintf("127.0.0.1:%d", m.settings.SrcdsPort))
	}
	return addresses
}

func (m *model) tabs() string {
	names := []string{"Session", "Unlocks", "Bot Switcher", "Log"}
	rendered := make([]string, 0, len(names))
	for i, name := range names {
		if view(i) == m.view {
			rendered = append(rendered, styleTabOn.Render(name))
			continue
		}
		rendered = append(rendered, styleTab.Render(name))
	}
	return strings.Join(rendered, "  ")
}

// log is the last screenful, or the screenful the player scrolled back to.
func (m *model) log() string {
	height := m.bodyHeight()
	end := max(len(m.lines)-m.offset, 0)
	start := max(end-height, 0)

	shown := make([]string, 0, height)
	for _, line := range m.lines[start:end] {
		shown = append(shown, truncate(line, m.width))
	}
	for len(shown) < height {
		shown = append(shown, "")
	}
	return strings.Join(shown, "\n")
}

// session is the run: what the multiworld says, and the missions the seed drew
// with what has been done to them.
func (m *model) session() string {
	height := m.bodyHeight()
	rows := make([]string, 0, height)

	switch {
	case !m.supervisor.Running():
		rows = append(rows, styleMuted.Render("The server is not running."))
	case m.fetchErr != nil:
		rows = append(rows, styleMuted.Render("Bridge: "+m.fetchErr.Error()))
	default:
		rows = append(rows, "Multiworld: "+m.snapshot.Health.Summary())
		if line := m.runLine(); line != "" {
			rows = append(rows, styleMuted.Render(line))
		}
		/* Where to connect, on the tab a player looks at
		 *
		 * The window has carried these since the Session tab existed. The
		 * terminal never did, so on Linux the only way to the address was the
		 * C key on the log tab, which is a keystroke nobody finds without
		 * being told. Reported from play as "no clear display of where the IP
		 * is to connect". */
		if m.itemServer != "" {
			rows = append(rows, styleMuted.Render(m.itemServer))
		}
		rows = append(rows, "")
		for _, line := range apruntime.ConnectLines(m.settings) {
			rows = append(rows, styleMuted.Render(line))
		}
		rows = append(rows, "")
		rows = append(rows, m.missionRows(height-len(rows))...)
	}

	for len(rows) < height {
		rows = append(rows, "")
	}
	return strings.Join(rows[:height], "\n")
}

// unlocks is everything the multiworld has handed this run, named for a person:
// the classes, the weapon slots, the missions and the weapon buffs with the
// level a repeated buff reached.
func (m *model) unlocks() string {
	height := m.bodyHeight()
	rows := make([]string, 0, height)

	switch {
	case !m.supervisor.Running():
		rows = append(rows, styleMuted.Render("The server is not running."))
	case m.fetchErr != nil:
		rows = append(rows, styleMuted.Render("Bridge: "+m.fetchErr.Error()))
	case len(m.snapshot.Unlocks) == 0:
		rows = append(rows, styleMuted.Render("Nothing unlocked yet."))
	default:
		rows = append(rows, styleMuted.Render("In the game, !ap buffs shows the buffs on the loadout you hold."))
		rows = append(rows, "")
		for _, unlock := range m.snapshot.Unlocks {
			level := ""
			if unlock.Level > 1 {
				level = fmt.Sprintf("  x%d", unlock.Level)
			}
			rows = append(rows, truncate(fmt.Sprintf("  %-12s %s%s", unlock.Kind, unlock.Name, level), m.width))
		}
	}

	for len(rows) < height {
		rows = append(rows, "")
	}
	return strings.Join(rows[:height], "\n")
}

func (m *model) runLine() string {
	health := m.snapshot.Health
	var parts []string
	if health.Seed != "" {
		parts = append(parts, "seed "+health.Seed)
	}
	if health.LastCheck != "" {
		parts = append(parts, "last check: "+health.LastCheck)
	}
	if health.DeathLink {
		parts = append(parts, "Death Link on")
	}
	if health.GoalSent {
		parts = append(parts, "goal reached")
	}
	return strings.Join(parts, "   ")
}

// missionRows lists the run in seed order, scrolled to keep the selection on
// screen.
func (m *model) missionRows(height int) []string {
	if height < 1 || len(m.snapshot.Missions) == 0 {
		return nil
	}
	start := max(m.selected-height+1, 0)
	end := min(start+height, len(m.snapshot.Missions))

	rows := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		mission := m.snapshot.Missions[i]
		marker := "  "
		if i == m.selected {
			marker = "> "
		}
		row := fmt.Sprintf("%s%2d  %-28s %-14s %2d waves  %s",
			marker, i+1, mission.Name, mission.Map, mission.Waves, missionState(mission))
		rows = append(rows, style(mission).Render(truncate(row, m.width)))
	}
	return rows
}

func style(mission session.Mission) lipgloss.Style {
	switch {
	case mission.Played:
		return styleCleared
	case !mission.Unlocked:
		return styleLocked
	default:
		return lipgloss.NewStyle()
	}
}

/* missionState is what this player did, not what the room holds.
 *
 * A mission another world's !collect touched has its check on the disk without
 * anybody here playing it. Drawn as "cleared" it made the run list stop saying
 * what this player had done, which is what Peppy reported. The two are shown
 * apart: cleared is yours, collected is the room's.
 */
func missionState(mission session.Mission) string {
	switch {
	case mission.Played:
		return "cleared"
	case mission.Cleared:
		return "collected"
	case mission.Unlocked:
		return "unlocked"
	default:
		return "locked"
	}
}

func (m *model) prompt() string {
	if m.typing {
		return "rcon> " + m.command + "█"
	}
	return styleMuted.Render("rcon> press i")
}

// keys is the only place the bindings are written down, so it is what the
// player reads rather than a manual.
func (m *model) keys() string {
	if m.typing {
		return styleMuted.Render(styleKey.Render("enter") + " send   " + styleKey.Render("esc") + " back")
	}

	pairs := [][2]string{
		{"s", startStopLabel(m.supervisor.Running())},
		{"r", "restart"},
		{"j", "join"},
		{"c", "copy"},
		{",", "settings"},
		{"tab", "view"},
		{"i", "rcon"},
	}
	switch m.view {
	case viewSession:
		pairs = append(pairs, [2]string{"p", "play mission"})
	case viewUnlocks:
		// Read-only: the list is the whole of it.
	case viewBots:
		pairs = append(pairs, [2]string{"a", "apply team"})
	case viewLog:
		// The log has no key of its own: everything it offers is in the row
		// above, and scrolling is the arrows.
	}
	pairs = append(pairs, [2]string{"q", "quit"})

	parts := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		parts = append(parts, styleKey.Render(pair[0])+" "+pair[1])
	}
	return styleMuted.Render(strings.Join(parts, "   "))
}

func startStopLabel(running bool) string {
	if running {
		return "stop"
	}
	return "start"
}

// truncate cuts a line to the width of the screen. Wrapping a log line that is
// three screens wide, which srcds prints, pushes everything else off.
func truncate(line string, width int) string {
	if width <= 0 || lipgloss.Width(line) <= width {
		return line
	}
	runes := []rune(line)
	if len(runes) <= width {
		return line
	}
	return string(runes[:max(width-1, 0)]) + "…"
}
