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

	if got.refused >= refusalFloor && got.taken == 0 {
		found = append(found, fmt.Sprintf(
			"Every bot purchase this run was refused: %d refused, none applied.\n"+
				"      One refusal is ordinary. All of them means the game turned down every\n"+
				"      MVM_Upgrade the plugin sent, which is what a SourceMod that cannot build\n"+
				"      the KeyValues this game reads looks like from the outside. The bots shop,\n"+
				"      spend nothing, and the wave arrives against an unupgraded team.", got.refused))
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
