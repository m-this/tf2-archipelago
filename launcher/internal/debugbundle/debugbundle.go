// Package debugbundle collects everything somebody would ask a play-tester for
// into one zip: the launcher's log, SourceMod's error logs, the game server's
// console log, the player file, and the settings with the passwords taken out.
//
// It exists because the alternative is asking a player to find five files in
// three folders, and because a zip posted in a chat channel is the shortest
// path from "it broke" to a diagnosis.
package debugbundle

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

// fileBytesMax caps one file inside the zip. A console log from a long evening
// runs to hundreds of megabytes, and the end of it is the part that matters.
const fileBytesMax = 8 << 20

// Write builds the zip next to the game files and returns its path. stamp names
// it, so two bundles from one evening do not overwrite each other.
func Write(s settings.Settings, versions map[string]string, stamp time.Time) (string, error) {
	name := fmt.Sprintf("debug-logs-%s.zip", stamp.Format("2006-01-02-150405"))
	path := filepath.Join(s.InstallRoot, name)

	file, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("cannot create %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	archive := zip.NewWriter(file)
	game := filepath.Join(s.InstallRoot, "tf-dedicated", "tf")

	add(archive, "summary.txt", strings.NewReader(summary(s, versions, stamp)))
	add(archive, "config.json", strings.NewReader(redactedSettings(s)))
	copyIn(archive, "launcher.log", filepath.Join(s.InstallRoot, "launcher.log"))
	copyIn(archive, "tf2.yaml", filepath.Join(s.InstallRoot, settings.PlayerFileName))
	copyIn(archive, "console.log", filepath.Join(game, "console.log"))
	copyIn(archive, "server.cfg", filepath.Join(game, "cfg", "server.cfg"))

	// Every SourceMod log, because the error log names the plugin and the plain
	// log holds what happened around it.
	logs := filepath.Join(game, "addons", "sourcemod", "logs")
	for _, entry := range newestLogs(logs, 6) {
		copyIn(archive, filepath.Join("sourcemod", entry), filepath.Join(logs, entry))
	}

	if err := archive.Close(); err != nil {
		return "", fmt.Errorf("cannot finish %s: %w", path, err)
	}
	return path, nil
}

// summary is the first thing to read: what was installed, what the run is, and
// where it was pointed.
func summary(s settings.Settings, versions map[string]string, stamp time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, "collected     %s\n", stamp.Format(time.RFC3339))
	fmt.Fprintf(&b, "install root  %s\n", s.InstallRoot)
	fmt.Fprintf(&b, "room          %s (tls=%v), slot %s\n",
		settings.Room{Host: s.APHost, Port: s.APPort}, s.APTls, s.APSlotName)
	fmt.Fprintf(&b, "server        %q on port %d, lan=%v\n", s.SrcdsHostname, s.SrcdsPort, s.SrcdsLan)
	fmt.Fprintf(&b, "start map     %s\n", s.SrcdsStartMap)
	fmt.Fprintf(&b, "bots          on=%v, fill RED to %d\n", s.SrcdsBots, s.SrcdsBotTeamSize)
	fmt.Fprintf(&b, "run           %d missions, %s and harder, goal %s, missionsanity %d%%, death link %v\n",
		s.MvmMissionCount, s.MvmDifficulty, s.MvmGoal, s.MvmMissionsanityPct, s.MvmDeathLink)

	names := make([]string, 0, len(versions))
	for name := range versions {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(&b, "%-13s %s\n", name, versions[name])
	}
	b.WriteString("\nPasswords are not in this file, and not in config.json either.\n")
	return b.String()
}

// redactedSettings renders the settings with every secret replaced.
//
// This zip is made to be posted in a chat channel. An RCON password in it is a
// stranger's admin console, and a room password is somebody else's multiworld.
func redactedSettings(s settings.Settings) string {
	const hidden = "(removed from the bundle)"
	if s.SrcdsRconPw != "" {
		s.SrcdsRconPw = hidden
	}
	if s.SrcdsPw != "" {
		s.SrcdsPw = hidden
	}
	if s.APPassword != "" {
		s.APPassword = hidden
	}
	if s.SrcdsToken != "" && s.SrcdsToken != "0" {
		s.SrcdsToken = hidden
	}
	body, err := settings.Render(s)
	if err != nil {
		return fmt.Sprintf("cannot render the settings: %v\n", err)
	}
	return body
}

// newestLogs returns up to limit file names from dir, newest last.
func newestLogs(dir string, limit int) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".log") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	if len(names) > limit {
		names = names[len(names)-limit:]
	}
	return names
}

func add(archive *zip.Writer, name string, body io.Reader) {
	writer, err := archive.Create(filepath.ToSlash(name))
	if err != nil {
		return
	}
	_, _ = io.Copy(writer, body)
}

// copyIn adds a file if it is there, keeping the last fileBytesMax of it. A
// missing file is normal: not every run has a console log or a SourceMod error.
func copyIn(archive *zip.Writer, name, path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return
	}
	if info.Size() > fileBytesMax {
		if _, err := file.Seek(info.Size()-fileBytesMax, 0); err != nil {
			return
		}
		name = strings.TrimSuffix(name, filepath.Ext(name)) + "-tail" + filepath.Ext(name)
	}
	add(archive, name, file)
}
