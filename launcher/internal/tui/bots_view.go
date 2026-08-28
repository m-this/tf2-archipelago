package tui

import (
	"fmt"
	"strings"

	"github.com/m-this/tf2-archipelago/launcher/internal/botlive"
)

/*
	bots is the Bot Switcher: the team RED is playing, and how to change it.

The three columns are one width apart down the page rather than sized to their
contents, because a table that moves when a seat changes class is harder to read
than a wide one that does not.
*/
func (m *model) bots() string {
	height := m.bodyHeight()
	rows := make([]string, 0, height)

	if !m.supervisor.Running() {
		rows = append(rows, styleMuted.Render("The server is not running. Press s to start it."))
	}

	rows = append(rows, fmt.Sprintf("RED holds %d, humans included. Seats fill from the top.", m.settings.SrcdsBotTeamSize))
	rows = append(rows, "")
	rows = append(rows, styleMuted.Render(fmt.Sprintf("  %-6s %-16s %s", "Seat", "Class", "Weapons")))

	for _, seat := range botlive.Team(m.settings) {
		row := fmt.Sprintf("  %-6d %-16s %s", seat.Number, seat.Class, seat.Weapons)
		if seat.Class == botlive.DrawnClass {
			rows = append(rows, styleMuted.Render(truncate(row, m.width)))
			continue
		}
		rows = append(rows, truncate(row, m.width))
	}

	if drawn := botlive.Drawn(m.settings); drawn != "" {
		rows = append(rows, "")
		rows = append(rows, styleMuted.Render(truncate("  The mod draws the rest from: "+drawn, m.width)))
	}

	rows = append(rows, "")
	rows = append(rows, styleMuted.Render(
		styleKey.Render(",")+" change the team, on the Bots page of the settings. Saving hands it to the running server"))

	for len(rows) < height {
		rows = append(rows, "")
	}
	return strings.Join(rows[:height], "\n")
}
