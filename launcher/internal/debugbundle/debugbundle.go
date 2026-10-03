// Package debugbundle collects everything somebody would ask a play-tester for
// into one zip: the launcher's log and the one before it, SourceMod's error
// logs, the game server's console log, what the bridge says about the run, the
// player file, and the settings with the passwords taken out. collected.txt
// says which of those made it in, so a file that is not here is a fact rather
// than a silence.
//
// It exists because the alternative is asking a player to find five files in
// three folders, and because a zip posted in a chat channel is the shortest
// path from "it broke" to a diagnosis.
package debugbundle

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	apruntime "github.com/m-this/tf2-archipelago/launcher/internal/runtime"
	"github.com/m-this/tf2-archipelago/launcher/internal/session"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
	"github.com/m-this/tf2-archipelago/launcher/internal/srcdsconfig"
)

// fileBytesMax caps one file inside the zip. A console log from a long evening
// runs to hundreds of megabytes, and the end of it is the part that matters.
const fileBytesMax = 8 << 20

// bridgeTimeout bounds the one request this package makes. The bridge is on
// loopback, and a bundle must not hang on a bridge that is not running.
const bridgeTimeout = 3 * time.Second

// Write builds the zip next to the game files and returns its path. stamp names
// it, so two bundles from one evening do not overwrite each other.
//
// pinnedTF2Build is the TF2 build deploy/env/versions.env says these versions
// were last checked against, which is what the summary compares the build the
// server actually started against.
func Write(s settings.Settings, versions map[string]string, pinnedTF2Build string, stamp time.Time) (string, error) {
	name := fmt.Sprintf("debug-logs-%s.zip", stamp.Format("2006-01-02-150405"))
	path := filepath.Join(s.InstallRoot, name)

	file, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("cannot create %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	archive := zip.NewWriter(file)
	game := filepath.Join(s.InstallRoot, "tf-dedicated", "tf")
	var held collected
	copyIn := func(archive *zip.Writer, name, path string) {
		held.note(name, path, copyFile(archive, name, path, secrets(s)))
	}

	add(archive, "summary.txt", strings.NewReader(summary(s, versions, pinnedTF2Build, stamp)))
	add(archive, "config.json", strings.NewReader(redactedSettings(s)))
	add(archive, "bridge.json", strings.NewReader(bridgeState()))
	held.note("summary.txt", "written from the settings and the logs", nil)
	held.note("config.json", "written from the settings, passwords taken out", nil)
	held.note("bridge.json", "asked of the bridge on loopback", nil)
	// The run before this one as well: a player who hits a bug restarts the
	// server and then goes looking for the button, so the run that broke is
	// usually not the run the bundle is made from.
	copyIn(archive, apruntime.LogFileName, filepath.Join(s.InstallRoot, apruntime.LogFileName))
	copyIn(archive, apruntime.LogPreviousName, filepath.Join(s.InstallRoot, apruntime.LogPreviousName))
	copyIn(archive, "tf2.yaml", filepath.Join(s.InstallRoot, settings.PlayerFileName))
	copyIn(archive, apruntime.ConsoleLogName, filepath.Join(game, apruntime.ConsoleLogName))
	copyIn(archive, apruntime.ConsolePreviousName, filepath.Join(game, apruntime.ConsolePreviousName))
	/* debug.log is what -debug leaves behind when the server dies, and it names
	   the function it died in. A bundle arrived with two access violations and
	   nothing that could say where, which is why the flag and this line exist. */
	copyIn(archive, "debug.log", filepath.Join(game, "debug.log"))
	copyIn(archive, "debug.log", filepath.Join(filepath.Dir(game), "debug.log"))
	copyIn(archive, "server.cfg", filepath.Join(game, "cfg", "server.cfg"))
	// The plugin's own config, which is written once and then belongs to the
	// server. Two bundles were read without knowing that this file still said
	// tf2ap_debug 0, which is why they held nothing worth reading.
	copyIn(archive, "tf2_archipelago.cfg", filepath.Join(game, "cfg", "sourcemod", "tf2_archipelago.cfg"))

	// The crash dumps, newest last. srcds runs under Breakpad and writes one
	// per crash, and a crash that leaves no line in any log leaves one of
	// these: it is the only file that names the function the server died in.
	dumps := newestCrashDumps(game, systemCrashDumpDir(), 3)
	for _, dump := range dumps {
		name := filepath.Join("crashes", filepath.Base(dump))
		held.note(name, dump, copyRaw(archive, name, dump))
	}
	if len(dumps) == 0 {
		// Which directories, in summary.txt, and only when there is a crash to
		// go looking for. Here it is the fact that none were found.
		held.note("crashes/", "", fmt.Errorf("no .mdmp or .dmp in the %d directories srcds has been seen to use",
			len(crashDumpDirs(game, systemCrashDumpDir()))))
	}

	// Every SourceMod log, because the error log names the plugin and the plain
	// log holds what happened around it.
	logs := filepath.Join(game, "addons", "sourcemod", "logs")
	entries, err := newestLogs(logs, 6)
	if err != nil {
		held.note("sourcemod/", "", err)
	}
	if err == nil && len(entries) == 0 {
		held.note("sourcemod/", "", fmt.Errorf("no .log in %s", logs))
	}
	for _, entry := range entries {
		copyIn(archive, filepath.Join("sourcemod", entry), filepath.Join(logs, entry))
	}

	add(archive, "collected.txt", strings.NewReader(held.render()))

	if err := archive.Close(); err != nil {
		return "", fmt.Errorf("cannot finish %s: %w", path, err)
	}
	return path, nil
}

// bridgeURL is a variable so a test can point it at a bridge of its own.
var bridgeURL = session.BridgeURL

// bridgeState is what the bridge says about the run: whether it is connected,
// what it has checked, and which missions it believes are unlocked.
//
// It is fetched rather than read off disk, so it only answers while the server
// is up. A bundle made after everything was stopped carries the refusal, which
// is itself worth reading: it says the state came from nowhere rather than
// leaving a reader to assume the run was empty.
func bridgeState() string {
	ctx, cancel := context.WithTimeout(context.Background(), bridgeTimeout)
	defer cancel()

	snapshot, err := session.Fetch(ctx, bridgeURL)
	if err != nil {
		return fmt.Sprintf("{\n  \"error\": %q\n}\n", err.Error())
	}
	body, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Sprintf("{\n  \"error\": %q\n}\n", err.Error())
	}
	return string(body) + "\n"
}

// summary is the first thing to read: what was installed, what the run is, and
// where it was pointed.
func summary(s settings.Settings, versions map[string]string, pinnedTF2Build string, stamp time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, "collected     %s\n", stamp.Format(time.RFC3339))
	fmt.Fprintf(&b, "install root  %s\n", s.InstallRoot)
	fmt.Fprintf(&b, "room          %s (tls=%v), slot %s\n",
		settings.Room{Host: s.APHost, Port: s.APPort}, s.APTls, s.APSlotName)
	fmt.Fprintf(&b, "server        %q on port %d, reach=%s\n", s.SrcdsHostname, s.SrcdsPort, s.SrcdsReach)
	fmt.Fprintf(&b, "start mission %s\n", s.SrcdsStartMission)
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

	got := scanLogs(
		currentRun(filepath.Join(s.InstallRoot, apruntime.LogFileName)),
		previousRun(filepath.Join(s.InstallRoot, apruntime.LogPreviousName)),
		currentRun(filepath.Join(s.InstallRoot, "tf-dedicated", "tf", apruntime.ConsoleLogName)),
	)
	/* The build every pin above is pinned against, which the summary used to
	   leave out. srcds prints it at every start and the logs in this bundle
	   already carried it; reading it meant knowing to grep Breakpad's banner. */
	fmt.Fprintf(&b, "tf2 build     %s\n", tf2BuildNote(got.tf2Build, pinnedTF2Build))

	b.WriteString(configDrift(s))
	b.WriteString(crossCheck(got.tf2Build, pinnedTF2Build, versions, got))
	b.WriteString(got.report())
	b.WriteString(crashDumpNote(s, got))

	b.WriteString("\nPasswords are taken out of every file in this bundle.\n")
	b.WriteString("collected.txt names every file this bundle went for, and why one is missing.\n")
	return b.String()
}

// tf2BuildNote is the build the server started on, and the build the pins
// beside it were checked against when the two differ.
func tf2BuildNote(running, pinnedFor string) string {
	if running == "" {
		return "not in the logs: srcds prints it at start, so the server did not get that far"
	}
	if pinnedFor == "" || running == pinnedFor {
		return running
	}
	return fmt.Sprintf("%s, and the versions above were checked against %s", running, pinnedFor)
}

/* configDrift says whether the server.cfg on disk is the one these settings
 * render.
 *
 * A bundle once held a config.json that blacklisted the Spy beside a server.cfg
 * that blacklisted nothing, and the disagreement was the whole bug. Reading it
 * meant knowing to compare the two files by eye. The summary should say it.
 */
func configDrift(s settings.Settings) string {
	target := filepath.Join(s.InstallRoot, "tf-dedicated", "tf", "cfg", "server.cfg")
	onDisk, err := os.ReadFile(target)
	if err != nil {
		return "\nserver.cfg     not readable, so the server is running on something unknown\n"
	}
	wanted, err := srcdsconfig.RenderServerCfg(s)
	if err != nil {
		return ""
	}
	if string(onDisk) == wanted {
		return "\nserver.cfg     matches these settings\n"
	}
	return "\nserver.cfg     DIFFERS from these settings: the running server is not\n" +
		"               playing what config.json in this bundle says. Compare the two.\n"
}

/* crashDumpNote says when the logs hold a crash and the bundle holds no dump.
 *
 * The minidump is the only file that names the function the server died in, and
 * a bundle without one looks exactly like a bundle from a run that never
 * crashed. One arrived with two access violations in it and an empty crashes/
 * folder, and the absence had to be noticed rather than read.
 */
func crashDumpNote(s settings.Settings, got scan) string {
	if !got.sawCrash() {
		return ""
	}
	game := filepath.Join(s.InstallRoot, "tf-dedicated", "tf")
	if len(newestCrashDumps(game, systemCrashDumpDir(), 1)) > 0 {
		return "\n  A crash dump is in crashes/. That is the file worth reading first.\n"
	}
	/* Why Shysoul's bundle had none, and why hunting for one was wasted time:
	   SigMod installs its own fault handler. It prints the stack it faulted on
	   and the stack it exits from, then calls ExitProcess, which Windows sees
	   as a program choosing to quit. Breakpad never runs, so there is no dump
	   to find, and the stacks are in the logs that are already here. */
	if got.sigmodQuit() {
		return "\n  No crash dump, and there is none to find. SigMod caught this fault\n" +
			"  itself and called ExitProcess, which Breakpad does not see as a crash.\n" +
			"  What a dump would have said is in this bundle already: search\n" +
			"  launcher.log for \"SigMod: fault\" and for \"SigMod: ExitProcess\".\n"
	}
	return "\n  NO CRASH DUMP was found, though the logs hold a crash. srcds runs under\n" +
		"  Breakpad and should write one, and these are the directories that were\n" +
		"  looked in:\n" +
		"      " + strings.Join(crashDumpDirs(game, systemCrashDumpDir()), "\n      ") + "\n" +
		"  If a .mdmp or .dmp is anywhere else under the install root, say where.\n"
}

// redactedSettings renders the settings with every secret replaced.
//
// This zip is made to be posted in a chat channel. An RCON password in it is a
// stranger's admin console, and a room password is somebody else's multiworld.
func redactedSettings(s settings.Settings) string {
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

/* newestCrashDumps looks where srcds actually drops one.
 *
 * A bundle arrived with two access violations in its logs and no crashes/
 * folder in it, which is the one file that names the function the server died
 * in. Two guesses were wrong: only `.mdmp` was matched, and only two
 * directories were searched.
 *
 * Breakpad writes beside the binary, so the game directory and its parent are
 * the likely places, but the launcher's own working directory is the install
 * root and a dump can land there instead. Some builds write `.dmp`, and some
 * drop the file into a CrashDumps folder rather than beside themselves.
 *
 * Reading a directory that does not exist is the normal case here, not an
 * error: most installs have none of the optional ones.
 */
func newestCrashDumps(gameDir, systemDir string, limit int) []string {
	dirs := crashDumpDirs(gameDir, systemDir)

	seen := map[string]bool{}
	var found []string
	collect := func(dir string, wanted func(string) bool) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, entry := range entries {
			if entry.IsDir() || !wanted(entry.Name()) {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			if seen[path] {
				continue
			}
			seen[path] = true
			found = append(found, path)
		}
	}
	for _, dir := range dirs {
		if dir == systemDir {
			continue
		}
		collect(dir, isCrashDump)
	}
	/* Windows Error Reporting writes somewhere else entirely, and for every
	   program on the machine. Two bundles carried GameBar, Refunct and THPS12
	   dumps under crashes/ with the note saying to read them first, while the
	   server's own crash had left nothing. Only the game server's dumps count
	   there. Breakpad names the ones beside the binary by a GUID, so the game
	   directories above keep every dump they hold. */
	if systemDir != "" {
		collect(systemDir, isGameServerDump)
	}
	sort.Slice(found, func(i, j int) bool {
		return modTime(found[i]).Before(modTime(found[j]))
	})
	if len(found) > limit {
		found = found[len(found)-limit:]
	}
	return found
}

/* crashDumpDirs is everywhere a dump has been found, in the order they are
 * read. The summary prints this list when it has no dump to show, because
 * "look under the install root" is not an instruction anybody can follow.
 *
 * Breakpad writes beside the binary, so the game directory and its parent are
 * the likely places, but the launcher's own working directory is the install
 * root and a dump can land there instead. Some builds drop the file into a
 * CrashDumps folder rather than beside themselves.
 */
func crashDumpDirs(gameDir, systemDir string) []string {
	root := filepath.Dir(filepath.Dir(gameDir))
	bases := []string{gameDir, filepath.Dir(gameDir), root}
	dirs := append([]string{}, bases...)
	for _, base := range bases {
		for _, name := range []string{"crashdumps", "CrashDumps"} {
			dirs = append(dirs, filepath.Join(base, name))
		}
	}
	if systemDir != "" {
		dirs = append(dirs, systemDir)
	}
	return dirs
}

// isCrashDump covers both suffixes srcds has been seen to write.
func isCrashDump(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".mdmp") || strings.HasSuffix(lower, ".dmp")
}

// isGameServerDump is a Windows Error Reporting dump of the game server or the
// launcher, which WER names after the process: srcds.exe.1234.dmp.
func isGameServerDump(name string) bool {
	if !isCrashDump(name) {
		return false
	}
	lower := strings.ToLower(name)
	for _, process := range []string{"srcds", "hl2", "tf2ap"} {
		if strings.HasPrefix(lower, process) {
			return true
		}
	}
	return false
}

func modTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

// newestLogs returns up to limit file names from dir, newest last. The error
// goes to the manifest: a bundle with no sourcemod/ in it is either a server
// that logged nothing or a directory that could not be read, and those two
// read the same from the outside.
func newestLogs(dir string, limit int) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
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
	return names, nil
}

func add(archive *zip.Writer, name string, body io.Reader) {
	writer, err := archive.Create(filepath.ToSlash(name))
	if err != nil {
		return
	}
	_, _ = io.Copy(writer, body)
}

// hidden is what a secret reads as inside the bundle.
const hidden = "(removed from the bundle)"

// secretSetting matches a password given to srcds, in server.cfg or on the
// command line that debug.log repeats on every crash. A Cowser bundle carried
// his RCON password 27 times that way, under a summary saying it did not.
var secretSetting = regexp.MustCompile(`(?i)(\+?(?:rcon_password|sv_password|sv_setsteamaccount)["\s]+)("[^"\r\n]*"|[^\s"]+)`)

// secrets are the values the settings hold that nobody else should read. A
// value shorter than four characters is left out: "0" is the token meaning
// none, and replacing every 0 in a log would ruin it without hiding anything.
func secrets(s settings.Settings) []string {
	var out []string
	for _, v := range []string{s.SrcdsRconPw, s.SrcdsPw, s.APPassword, s.SrcdsToken} {
		if len(v) >= 4 {
			out = append(out, v)
		}
	}
	return out
}

// redact takes every secret out of a text file before it goes in the zip.
func redact(body []byte, secrets []string) []byte {
	body = secretSetting.ReplaceAll(body, []byte("${1}"+hidden))
	for _, v := range secrets {
		body = bytes.ReplaceAll(body, []byte(v), []byte(hidden))
	}
	return body
}

// copyFile adds a text file if it is there, keeping the last fileBytesMax of
// it, with every secret taken out. A missing file is normal: not every run has
// a console log or a SourceMod error. It is returned rather than dropped so
// the manifest can say which file was not there, and why.
func copyFile(archive *zip.Writer, name, path string, secrets []string) error {
	body, name, err := readTail(name, path)
	if err != nil {
		return err
	}
	add(archive, name, bytes.NewReader(redact(body, secrets)))
	return nil
}

// copyRaw adds a binary file as it is, keeping the last fileBytesMax of it.
// A crash dump is not text, and a replacement inside it would corrupt it.
func copyRaw(archive *zip.Writer, name, path string) error {
	body, name, err := readTail(name, path)
	if err != nil {
		return err
	}
	add(archive, name, bytes.NewReader(body))
	return nil
}

func readTail(name, path string) ([]byte, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, name, err
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return nil, name, err
	}
	if info.Size() > fileBytesMax {
		if _, err := file.Seek(info.Size()-fileBytesMax, 0); err != nil {
			return nil, name, err
		}
		name = strings.TrimSuffix(name, filepath.Ext(name)) + "-tail" + filepath.Ext(name)
	}
	body, err := io.ReadAll(file)
	if err != nil {
		return nil, name, err
	}
	return body, name, nil
}
