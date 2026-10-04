package debugbundle

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

/* pins compares what the bundle already knew.

The summary listed every version this launcher pins and the TF2 build the
server was running, and on 2026-10-02 that was the whole diagnosis sitting in
one file with nothing to put the numbers side by side. The evening went:
players reporting that the server died when a bot bought an upgrade, then a
bundle, then somebody reading Breakpad output by hand to find that SigMod was
keyed to the build before. Every input for that was in the first bundle.

Each line below is two facts and what the pair rules out. "These two cannot
work together" is a fact; "this is your bug" is not, and nothing here says it.
*/

// What the game moved, and the versions that absorbed it.
const (
	/* The TF2 update of 2026-10-02 that moved the KeyValues layout SourceMod
	   reads. Below the fix, a real client opening the upgrade station segfaults
	   the server and an MVM_Upgrade a plugin sends is refused silently:
	   alliedmodders/sourcemod#2587, fixed by #2588. */
	tf2KeyValuesBuild = 11076587
	// The SourceMod snapshot that carries #2588.
	sourcemodKeyValuesFix = 7255
)

/* refusalFloor is how many refused purchases make "none of them worked" worth
 * saying.
 *
 * A bot that asks for one upgrade the game will not apply and takes the next
 * is ordinary, and a wave or two of that is a handful of lines. Below this the
 * pair of counts is not evidence of anything.
 */
const refusalFloor = 10

// gitSnapshot pulls the snapshot number out of a SourceMod version string:
// 1.12.0-git7253 is 7253. A release without one (1.12.0) has no number.
var gitSnapshot = regexp.MustCompile(`-git(\d+)$`)

/* crossCheck is the summary's second section: what the versions and the counts
 * above rule out, in plain words.
 *
 * running is the TF2 build srcds printed at start, pinnedFor is the build
 * deploy/env/versions.env says these pins were last checked against, and got
 * carries the two purchase counts from this run. Any of them may be missing,
 * and a comparison that cannot be made is simply not made.
 */
func crossCheck(running, pinnedFor string, versions map[string]string, got scan) string {
	var found []string

	if running != "" && pinnedFor != "" && running != pinnedFor {
		found = append(found, fmt.Sprintf(
			"TF2 is on build %s, and the versions above were checked against %s.\n"+
				"      SigMod knows one TF2 ServerVersion. On any other, IAddr_FixedAddr\n"+
				"      refuses every fixed address, so SigMod resolves nothing and the server\n"+
				"      faults: at extension load, when anybody buys an upgrade, or when a\n"+
				"      player inspects a weapon. A launcher release with a SigMod build for\n"+
				"      %s is what fixes it.", running, pinnedFor, running))
	}

	if snapshot, ok := snapshotOf(versions["sourcemod"]); ok && snapshot < sourcemodKeyValuesFix {
		if build, err := strconv.Atoi(running); err == nil && build >= tf2KeyValuesBuild {
			found = append(found, fmt.Sprintf(
				"SourceMod is %s and TF2 is on build %d.\n"+
					"      The 2026-10-02 TF2 update moved the KeyValues layout SourceMod reads,\n"+
					"      and git%d is the snapshot that absorbed it. Below that, a player\n"+
					"      opening the upgrade station segfaults the server, and an upgrade the\n"+
					"      plugin buys for a bot is refused without saying so:\n"+
					"      alliedmodders/sourcemod#2587, fixed by #2588.",
				versions["sourcemod"], build, sourcemodKeyValuesFix))
		}
	}

	unresolved := got.found[sigmodUnresolvedRule] != nil
	if line, ok := halfUpdatedInstall(running, pinnedFor, unresolved); ok {
		found = append(found, line)
	}
	if line, ok := everyPurchaseRefused(got, versions["sourcemod"], unresolved); ok {
		found = append(found, line)
	}

	var b strings.Builder
	b.WriteString("\nWhat does not add up\n")
	b.WriteString("Two things this bundle already knew, put side by side. A comparison, not\n")
	b.WriteString("a diagnosis.\n\n")
	if len(found) == 0 {
		if running == "" {
			b.WriteString("  The TF2 build is not in the logs, so nothing here could be compared\n" +
				"  against it. srcds prints it at every start, so the server did not get\n" +
				"  that far.\n")
			return b.String()
		}
		fmt.Fprintf(&b, "  Nothing above is a pairing this launcher knows cannot work on TF2\n"+
			"  build %s. That is not the same as nothing being wrong.\n", running)
		return b.String()
	}
	for _, line := range found {
		fmt.Fprintf(&b, "  %s\n", line)
	}
	return b.String()
}

// snapshotOf reads the git snapshot number out of a SourceMod version.
func snapshotOf(version string) (int, bool) {
	m := gitSnapshot.FindStringSubmatch(version)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false
	}
	return n, true
}

/*
	halfUpdatedInstall is SigMod made for the running build, and still resolving

nothing.

The build line is the engine's own Version, which says what TF2 reports, not
that every file on disk is that build. An install updated halfway reports the
new build while SigMod reads a server binary from the old one. Celsius's bundle
was exactly this: build 11076587, SigMod 20261005 made for it, 8760 failed
lookups, then a SigMod fault; resetting the game files fixed it. Telling them to
compare the build with the pins sent them to two numbers that already agreed.
*/
func halfUpdatedInstall(running, pinnedFor string, unresolved bool) (string, bool) {
	if !unresolved || running == "" || running != pinnedFor {
		return "", false
	}
	return fmt.Sprintf(
		"SigMod could not find the game's addresses, yet TF2 reports build %s,\n"+
			"      the build this launcher's SigMod was made for. That number comes from\n"+
			"      the engine, not from the server binary SigMod reads, so an install\n"+
			"      updated halfway reports the new build and still holds old files. Press\n"+
			"      Repair in Settings: the next start verifies every game file.", running), true
}

/*
	everyPurchaseRefused says so when the game turned down every bot purchase.

Only a SourceMod below the fix earns the KeyValues explanation. At or past it,
naming SourceMod points away from the cause: Celsius's bundle had git7255 and
SigMod faulting under every purchase.
*/
func everyPurchaseRefused(got scan, sourcemod string, unresolved bool) (string, bool) {
	if got.refused < refusalFloor || got.taken != 0 {
		return "", false
	}
	line := fmt.Sprintf(
		"Every bot purchase this run was refused: %d refused, none applied.\n"+
			"      One refusal is ordinary. All of them means the game turned down every\n"+
			"      MVM_Upgrade the plugin sent. The bots shop, spend nothing, and the wave\n"+
			"      arrives against an unupgraded team.\n", got.refused)
	snapshot, known := snapshotOf(sourcemod)
	switch {
	case known && snapshot >= sourcemodKeyValuesFix && unresolved:
		line += fmt.Sprintf(
			"      SourceMod here is %s, which builds the KeyValues this game\n"+
				"      reads, so it is not that. SigMod could not find the game's addresses on\n"+
				"      this run, and a SigMod that cannot is what\n"+
				"      faults when an upgrade is bought.",
			sourcemod)
	case known && snapshot >= sourcemodKeyValuesFix:
		line += fmt.Sprintf(
			"      SourceMod here is %s, which builds the KeyValues this game\n"+
				"      reads, so it is not that, and nothing else in these logs says what is.",
			sourcemod)
	default:
		line += "      That is what a SourceMod that cannot build the KeyValues this game\n" +
			"      reads looks like from the outside."
	}
	return line, true
}
