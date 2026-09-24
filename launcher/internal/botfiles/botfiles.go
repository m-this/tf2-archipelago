/*
Package botfiles writes the files the defender mod reads off disk, for a server
whose game tree the launcher does not own.

The Compose stack is that server: its admin container mounts the game read-only
on purpose, so the only process allowed to write there is the one inside the
game's own container. That container runs a command built on this package, and
gets the same answer the native launcher would, because both are asking
botloadout the same question.

Its own package rather than a function in srcdsconfig, which does the same job
for the native launcher: srcdsconfig embeds the plugin, the apworld and the two
mod archives, and none of them can be in a game image's build context. A
package that renders a text file should not cost three hundred megabytes to
link.
*/
package botfiles

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/m-this/tf2-archipelago/launcher/internal/botlive"
	"github.com/m-this/tf2-archipelago/launcher/internal/botloadout"
	"github.com/m-this/tf2-archipelago/launcher/internal/botnames"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

/*
Install writes the bot files under root, and reports whether the mod should be
told to read the loadouts.

The names are written every time, because a pool is always a pool: the file is
the list, and a run that changed nothing gets the shipped one back. The loadouts
are not. The mod ships an example loadout file and reads it only when
sm_redbots_manager_use_custom_loadouts is 1, so the convar is the whole of
"nothing was chosen", and deleting what the image shipped to say so would lose
the one copy of the format anybody reads before editing it by hand.
*/
func Install(root string, s settings.Settings) (bool, error) {
	if err := write(NamesPath(root), []byte(botnames.Render(s.SrcdsBotNamesExcluded, s.SrcdsBotNamesAdded))); err != nil {
		return false, err
	}

	library := botlive.LibraryOf(s)
	seats := botlive.SeatsOf(s)

	// Weapons decide the convar; a name only decides that the file has to be
	// there, because the mod reads a seat's name out of it whatever the convar
	// says.
	weapons := library.Anything(s.SrcdsBotLoadouts, seats)
	if !weapons && !botloadout.Named(seats) {
		return false, nil
	}

	if err := write(LoadoutPath(root), []byte(library.Render(s.SrcdsBotLoadouts, seats))); err != nil {
		return false, err
	}
	return weapons, nil
}

// StageForLive always writes the managed loadout file, including the empty
// version when the last card is removed. The game copies this overlay file
// over its previous live file; leaving an old file in place would resurrect
// cards after clearing the team.
func StageForLive(root string, s settings.Settings) error {
	if _, err := Install(root, s); err != nil {
		return err
	}
	return write(LoadoutPath(root), []byte(botlive.LibraryOf(s).Render(s.SrcdsBotLoadouts, botlive.SeatsOf(s))))
}

// LoadoutPath and NamesPath are where the mod looks for them, under a game tree
// or a tree staged to become one.
func LoadoutPath(root string) string { return configPath(root, "loadout.cfg") }

func NamesPath(root string) string { return configPath(root, "bot_names.txt") }

func configPath(root, name string) string {
	return filepath.Join(root, "addons", "sourcemod", "configs", "defenderbots", name)
}

// write leaves a file that already says this alone, because this runs on a
// loop: the server's own supervisor calls it every thirty seconds, and a file
// rewritten every pass is a file whose modification time says nothing.
func write(target string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("cannot create %s: %w", filepath.Dir(target), err)
	}
	if existing, err := os.ReadFile(target); err == nil && bytes.Equal(existing, content) {
		return nil
	}
	if err := os.WriteFile(target, content, 0o644); err != nil {
		return fmt.Errorf("cannot write %s: %w", target, err)
	}
	return nil
}
