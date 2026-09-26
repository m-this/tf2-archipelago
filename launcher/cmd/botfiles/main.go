/*
Command botfiles writes the files the defender mod reads off disk, for a server
whose game tree this process is the only one allowed to touch.

The Compose stack is that server. Its admin container mounts the game
read-only on purpose, so the admin can write .env and nothing else, and the
loadouts a player picked there never reached the bots: the variables were
exported and no one read them. This runs inside the game's own container, off
the same environment, through the same Go the native launcher uses, so a
loadout means the same thing in both.

It prints 1 when it wrote a file the mod should read and 0 when it did not,
which is what sm_redbots_manager_use_custom_loadouts wants.

	Usage: botfiles -root /opt/tf2-archipelago
*/
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/m-this/tf2-archipelago/launcher/internal/botfiles"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("botfiles", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", "", "the tree to write into, holding addons/sourcemod/configs")
	live := flags.Bool("live", false, "stage an empty loadout file when clearing a live team")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *root == "" {
		say(stderr, "botfiles: -root is required, and is the tree holding addons/sourcemod/configs")
		return 2
	}

	state := settings.ApplyEnv(settings.Defaults())
	custom, err := botfiles.Install(*root, state)
	if err != nil {
		say(stderr, "botfiles: "+err.Error())
		return 1
	}
	if *live {
		if err := botfiles.StageForLive(*root, state); err != nil {
			say(stderr, "botfiles: "+err.Error())
			return 1
		}
	}

	// The last line is the answer, so a caller reads it with $(...) and any
	// future warning above it still reaches the log.
	say(stdout, strconv.Itoa(boolToInt(custom)))
	return 0
}

func say(w io.Writer, line string) { _, _ = fmt.Fprintln(w, line) }

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
