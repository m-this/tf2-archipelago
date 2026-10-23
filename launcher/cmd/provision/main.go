/*
Command provision is the Docker image's way to ask provision.For what the game
server starts with, from the environment Compose hands the srcds container.

The native launcher asks the same package from its own settings, so server.cfg,
the command line and the mods loaded cannot mean two things.

	Usage:
	  provision install <tf directory>   write server.cfg and SigMod's convars;
	                                     print "changed" when server.cfg did
	  provision mods                     the server mods this run loads, one a line
	  provision args                     the srcds command line, NUL-separated
*/
package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/m-this/tf2-archipelago/launcher/internal/fastdl"
	"github.com/m-this/tf2-archipelago/launcher/internal/provision"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		say(stderr, "usage: provision install <tf directory> | mods | args")
		return 2
	}
	s := fromEnv()
	host, err := imageHost()
	if err != nil {
		say(stderr, "provision: "+err.Error())
		return 2
	}
	server, err := provision.For(s, host)
	if err != nil {
		say(stderr, "provision: "+err.Error())
		return 1
	}
	switch {
	case args[0] == "install" && len(args) == 2:
		changed, err := server.Install(args[1])
		if err != nil {
			say(stderr, "provision: "+err.Error())
			return 1
		}
		if changed {
			say(stdout, "changed")
		}
	case args[0] == "mods" && len(args) == 1:
		for _, key := range server.Mods {
			say(stdout, key)
		}
	case args[0] == "args" && len(args) == 1:
		if s.SrcdsReach != settings.ReachLan && s.EffectiveReach().Lan() {
			say(stderr, "[AP] no SRCDS_TOKEN, so the server stays on the local network: get one at steamcommunity.com/dev/managegameservers for app id 440")
		}
		_, _ = io.WriteString(stdout, strings.Join(server.Args, "\x00")+"\x00")
	default:
		say(stderr, "usage: provision install <tf directory> | mods | args")
		return 2
	}
	return 0
}

/*
fromEnv is the settings the container was given.

FASTDL_HOST is the stack's own spelling of a download host: the fastdl service
on the address the operator named. It becomes the download URL here rather
than in provision, because a launcher never has one.
*/
func fromEnv() settings.Settings {
	s := settings.ApplyEnv(settings.Defaults())
	if host := os.Getenv("FASTDL_HOST"); s.SrcdsDownloadURL == "" && host != "" && s.FastDLPort > 0 {
		s.SrcdsDownloadURL = fastdl.URL(host, s.FastDLPort)
	}
	return s
}

func imageHost() (provision.Host, error) {
	host := provision.Host{GOOS: "linux", Exe: "srcds_run"}
	if value := strings.TrimSpace(os.Getenv("TF2AP_NEXT_MISSION_DELAY")); value != "" {
		seconds, err := strconv.Atoi(value)
		if err != nil || seconds < 0 {
			return host, fmt.Errorf("TF2AP_NEXT_MISSION_DELAY=%q is not a number of seconds", value)
		}
		host.NextMissionDelaySeconds = &seconds
	}
	return host, nil
}

func say(w io.Writer, line string) { _, _ = fmt.Fprintln(w, line) }
