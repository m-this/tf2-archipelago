package webapi

import (
	"fmt"
	"strings"
	"time"

	"github.com/m-this/tf2-archipelago/launcher/internal/botlive"
	apruntime "github.com/m-this/tf2-archipelago/launcher/internal/runtime"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// lineupRefusals are the replies with which the game turns a lineup command
// down while the RCON call itself succeeds.
var lineupRefusals = []string{
	"Unknown command",
	"cards unchanged",
	"Cannot read the new lineup",
	"No card loadout file to reload",
}

// applyLineup sends the lineup commands in order and stops at the first one
// the game fails or refuses, since the ones after it assume it held.
func applyLineup(exec func(command string) (string, error), commands []string) error {
	for _, command := range commands {
		reply, err := exec(command)
		if err != nil {
			return fmt.Errorf("%s: %w", command, err)
		}
		for _, refusal := range lineupRefusals {
			if strings.Contains(reply, refusal) {
				return fmt.Errorf("%s: %s", command, strings.TrimSpace(reply))
			}
		}
	}
	return nil
}

// applyLocalTeam is applyAttachedTeam for a server this launcher runs: the bot
// files are already on disk, so only the lineup commands go, on one connection,
// each with its reply in the log.
func (a *App) applyLocalTeam(before, after settings.Settings) error {
	client, err := dialRCON(after)
	if err != nil {
		return fmt.Errorf("connect to the game: %w", err)
	}
	defer func() { _ = client.Close() }()
	return applyLineup(func(command string) (string, error) {
		a.append(apruntime.Line{At: time.Now(), Source: "rcon", Text: "> " + command})
		reply, err := client.Exec(command)
		for line := range strings.SplitSeq(reply, "\n") {
			if strings.TrimSpace(line) != "" {
				a.append(apruntime.Line{At: time.Now(), Source: "rcon", Text: line})
			}
		}
		return reply, err
	}, botlive.Commands(before, after))
}
