package debugbundle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeLog(t *testing.T, lines ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "launcher.log")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("cannot write the log: %v", err)
	}
	return path
}

// readLog scans one log as the run this bundle was made from.
func readLog(t *testing.T, lines ...string) scan {
	t.Helper()
	return scanLogs(currentRun(writeLog(t, lines...)))
}

// The crash is the line that matters most, and it is the one that was read as
// "bridge stopping" for weeks.
func TestTheCrashIsFound(t *testing.T) {
	got := readLog(t,
		`21:09:25  srcds    something ordinary`,
		`21:09:26  bridge   time=2026-08-26T21:09:26.548+09:00 level=INFO msg="bridge stopping"`,
		`21:09:26  launcher game server stopped: exit status 0xc0000005`,
	).report()
	if !strings.Contains(got, "the game server crashed") {
		t.Fatalf("the crash was not reported:\n%s", got)
	}
	if !strings.Contains(got, "0xc0000005") {
		t.Errorf("the status was not quoted:\n%s", got)
	}
}

/*
One message logged four ways is one message.

The launcher prefixes a clock and a source, SourceMod prefixes its own date,
the plugin tags itself, and the console carries the plain text. Counting those
apart turned a 282-line loop into four entries of about 70 and buried it under
lines that repeat for good reason.
*/
func TestOneMessageCountsOnce(t *testing.T) {
	var lines []string
	for range 30 {
		lines = append(lines,
			`20:28:48  srcds    [AP] debug: The grant poll had stopped. The plugin starts it again.`,
			`20:28:48  srcds    L 08/26/2026 - 20:28:48: [tf2_archipelago.smx] The grant poll had stopped. The plugin starts it again.`,
			`[AP] debug: The grant poll had stopped. The plugin starts it again.`,
		)
	}
	got := readLog(t, lines...).report()
	if !strings.Contains(got, "90 x") {
		t.Fatalf("the ninety copies did not count as one message:\n%s", got)
	}
}

// A retry backoff differs only by the seconds inside the message, and "in=1s"
// keeps its digit under a word-boundary match.
func TestABackoffCollapses(t *testing.T) {
	a := shapeOf(`20:25:43  bridge   msg="session ended, will retry" in=1s`)
	b := shapeOf(`20:25:44  bridge   msg="session ended, will retry" in=2s`)
	if a != b {
		t.Errorf("a backoff did not collapse:\n%q\n%q", a, b)
	}
}

// A run with nothing wrong says so, and says what that is worth.
func TestAQuietRunSaysSo(t *testing.T) {
	got := readLog(t, `20:00:00  srcds    Server is hibernating`).report()
	if !strings.Contains(got, "nothing matched") {
		t.Errorf("a quiet run did not say so:\n%s", got)
	}
	if !strings.Contains(got, "not a diagnosis") {
		t.Errorf("the caveat is missing:\n%s", got)
	}
}

// A missing file is the normal case: the previous-run log does not exist on a
// first run, and a bundle must still be written.
func TestMissingLogsAreNotAnError(t *testing.T) {
	if got := scanLogs(currentRun(filepath.Join(t.TempDir(), "absent.log"))).report(); got != "" {
		t.Errorf("a missing log produced %q", got)
	}
}

func TestTheStuckBotAndTheThrowAreFound(t *testing.T) {
	got := readLog(t,
		`20:35:28  srcds    [defenderbots] stuck: SomeDude (engineer) at 289 571 544 for 12s, DefenderEngineerIdle`,
		`20:25:45  sourcemod [SM] Exception reported: Assertion failed - wearable entity 565 not attached to player`,
		`18:24:12  srcds    [AP] error: The run restarted. The plugin asks for the unlock set again.`,
	).report()
	for _, want := range []string{"a bot got stuck", "a plugin threw", "the plugin reported an error"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q missing from:\n%s", want, got)
		}
	}
}

// The crash flag is what decides whether a missing minidump is worth saying
// anything about, so it has to be false on a quiet run.
func TestTheCrashFlagFollowsTheCrash(t *testing.T) {
	if readLog(t, `20:00:00  srcds    Server is hibernating`).sawCrash() {
		t.Error("a quiet run reported a crash")
	}
	if !readLog(t, `21:09:26  launcher game server stopped: exit status 0xc0000005`).sawCrash() {
		t.Error("an access violation did not set the crash flag")
	}
}

/*
Cowser's bundle: eighty-eight console lines, every tf2ap_ and sm_redbots_
convar in server.cfg answering "Unknown command", and a summary that said
nothing matched. Metamod and SourceMod had not loaded and the server was stock
Mann vs Machine. The line is the diagnosis, and the summary says so.
*/
func TestAServerWithoutThePluginIsNamed(t *testing.T) {
	got := readLog(t,
		`21:19:01  srcds    Executing dedicated server config file server.cfg`,
		`21:19:01  srcds    Unknown command "sm_redbots_manager_mode"`,
		`21:19:01  srcds    Unknown command "tf2ap_start_mission"`,
	).report()
	if !strings.Contains(got, pluginMissingRule) {
		t.Fatalf("the missing plugin was not reported:\n%s", got)
	}
	if !strings.Contains(got, "stock Mann vs") {
		t.Errorf("the summary does not say what the lines mean:\n%s", got)
	}
}

// SigMod for Windows refuses a TF2 build it was not made for, and a bundle
// should say so rather than leave the refused missions unexplained.
func TestASigModForAnotherBuildIsNamed(t *testing.T) {
	got := readLog(t,
		`22:23:31  srcds    [SM] Unable to load extension "sigsegv.ext": this SigMod build is for TF2 ServerVersion 11068238, and the server is 11076587`,
	).report()
	if !strings.Contains(got, "SigMod did not load") || !strings.Contains(got, "TF2 updated") {
		t.Fatalf("the refused SigMod was not reported:\n%s", got)
	}
}

/*
The number the whole of 2026-10-02 turned on, and the one the summary did not
print. srcds writes it at every start and every bundle from that evening
already carried it; finding it meant knowing to grep Breakpad's banner.
*/
func TestTheTF2BuildComesOutOfTheLog(t *testing.T) {
	got := readLog(t,
		`21:48:15  srcds    Using Breakpad minidump system. Version: 11076587 AppID: 232250`,
		`21:48:15  srcds    Setting breakpad minidump AppID = 232250`,
	)
	if got.tf2Build != "11076587" {
		t.Errorf("the TF2 build read as %q", got.tf2Build)
	}
}

// A server restarted onto a new TF2 build has the old one in the previous
// run's log. The build that matters is the one this run started on.
func TestTheBuildFromThisRunWins(t *testing.T) {
	before := writeLog(t, `20:12:21  srcds    Using Breakpad minidump system. Version: 10828683 AppID: 232250`)
	now := writeLog(t, `18:27:02  srcds    Using Breakpad minidump system. Version: 11076587 AppID: 232250`)

	if got := scanLogs(currentRun(now), previousRun(before)); got.tf2Build != "11076587" {
		t.Errorf("the previous run's build won: %q", got.tf2Build)
	}
	// With nothing from this run, the one before it is still better than none.
	if got := scanLogs(currentRun(writeLog(t, `nothing`)), previousRun(before)); got.tf2Build != "10828683" {
		t.Errorf("the only build in the bundle was dropped: %q", got.tf2Build)
	}
}

/*
Shysoul's bundle, 2026-10-02: a SigMod build with no ServerVersion check loads
on the wrong TF2 build, resolves nothing, and faults on the first address it
uses. It was reported as a line that repeats a lot, 96 times for
CBaseEntity::m_DataMap alone, under a crash with no dump.
*/
func TestSigModResolvingNothingIsNamed(t *testing.T) {
	got := readLog(t,
		`23:00:24  srcds    AddrManager::GetAddr FAIL: cannot resolve addr for name "CBaseEntity::m_DataMap"`,
		`23:00:24  srcds    AddrManager::GetAddr FAIL: cannot resolve addr for name "TE_TFParticleEffect"`,
		`23:00:24  srcds    AddrManager::GetAddr FAIL: cannot resolve addr for name "GetParticleSystemIndex"`,
	).report()
	if !strings.Contains(got, sigmodUnresolvedRule) {
		t.Fatalf("the unresolved addresses were not reported:\n%s", got)
	}
	if !strings.Contains(got, "made for another TF2 build") {
		t.Errorf("the summary does not say what the lines mean:\n%s", got)
	}
}

/*
Why Shysoul's bundle had no minidump: SigMod handles the fault itself, prints
two stacks, and calls ExitProcess, which Windows does not treat as a crash. The
summary said Breakpad should have written one and sent the player looking.
*/
func TestSigModQuittingIsNotAMissingDump(t *testing.T) {
	got := readLog(t,
		`21:48:22  srcds    SigMod: fault 0xc0000005 at ?+0xacf1200 touching 0x0acf1200 (esp 0x00f7ce8c), EBP chain:`,
		`21:48:22  srcds      sigsegv.ext.2.tf2.dll+0x149033`,
		`21:48:22  srcds    SigMod: ExitProcess(4294967295) called from:`,
		`21:48:22  launcher game server CRASHED: exit status 0xffffffff (an unhandled exception, so this is a crash and not a stop)`,
	)
	if !got.sigmodQuit() {
		t.Fatal("SigMod exiting the process went unnoticed")
	}
	if !got.sawCrash() {
		t.Error("the crash flag is not set, so no dump note is printed at all")
	}
	if !strings.Contains(got.report(), sigmodFaultRule) {
		t.Errorf("the fault was not reported:\n%s", got.report())
	}
}

// The build the pins were checked against, beside the build the server is on.
// Nothing in the bundle compared the two.
func TestAPinForAnotherBuildIsStated(t *testing.T) {
	got := crossCheck("11076587", "11068238", map[string]string{"sourcemod": "1.12.0-git7255"}, scan{})
	if !strings.Contains(got, "11076587") || !strings.Contains(got, "11068238") {
		t.Fatalf("neither build is named:\n%s", got)
	}
	if !strings.Contains(got, "IAddr_FixedAddr") {
		t.Errorf("the summary does not say what the mismatch does:\n%s", got)
	}
}

// Cowser is still crashing on this pair: git7253 reads the KeyValues layout
// that TF2 11076587 moved. alliedmodders/sourcemod#2587.
func TestSourceModBelowTheKeyValuesFixIsStated(t *testing.T) {
	got := crossCheck("11076587", "11076587", map[string]string{"sourcemod": "1.12.0-git7253"}, scan{})
	if !strings.Contains(got, "git7255") {
		t.Fatalf("the fix snapshot is not named:\n%s", got)
	}
	if !strings.Contains(got, "upgrade station") {
		t.Errorf("the summary does not say what breaks:\n%s", got)
	}
	// The snapshot that carries the fix is not a pairing to report.
	clean := crossCheck("11076587", "11076587", map[string]string{"sourcemod": "1.12.0-git7255"}, scan{})
	if strings.Contains(clean, "git7255 is the snapshot") {
		t.Errorf("the fixed SourceMod was reported anyway:\n%s", clean)
	}
}

/*
Every bot purchase refused and none applied, which is what the KeyValues bug
looks like from outside the server. One refusal is ordinary and a run that
mixes refusals with purchases is an ordinary evening, so the pair of counts is
the finding and neither count is.
*/
func TestEveryPurchaseRefusedIsStated(t *testing.T) {
	got := crossCheck("11076587", "11076587", nil, scan{refused: 43})
	if !strings.Contains(got, "43 refused, none applied") {
		t.Fatalf("the refusals were not reported:\n%s", got)
	}
	mixed := crossCheck("11076587", "11076587", nil, scan{refused: 93, taken: 186})
	if strings.Contains(mixed, "refused, none applied") {
		t.Errorf("a run that bought things was reported as refused:\n%s", mixed)
	}
	// A bot that asks for one upgrade the game will not apply is not a finding.
	quiet := crossCheck("11076587", "11076587", nil, scan{refused: 2})
	if strings.Contains(quiet, "refused, none applied") {
		t.Errorf("two refusals were reported as a finding:\n%s", quiet)
	}
}

// The counts are about one run. Cowser's bundle held a working previous run
// with 329 purchases beside a current run with none, and summed they look
// like an ordinary evening.
func TestThePurchaseCountsAreThisRunOnly(t *testing.T) {
	before := writeLog(t, `18:30:00  srcds    [AP] debug: Bot One bought damage bonus [0] (primary) x1, and holds 400 credits.`)
	now := writeLog(t, `11:33:55  srcds    L 10/03/2026 - 11:33:55: [tf2_defenderbots.smx] Shopping: Bot One stopped, the game refused one, trying the next, 400 credits left`)

	got := scanLogs(currentRun(now), previousRun(before))
	if got.taken != 0 {
		t.Errorf("the previous run's purchases counted: taken=%d", got.taken)
	}
	if got.refused != 1 {
		t.Errorf("this run's refusal did not count: refused=%d", got.refused)
	}
}

// A comparison that cannot be made is not made, and the section says which.
func TestCrossCheckWithoutABuildSaysSo(t *testing.T) {
	got := crossCheck("", "11076587", map[string]string{"sourcemod": "1.12.0-git7253"}, scan{})
	if !strings.Contains(got, "not in the logs") {
		t.Errorf("a missing build was not stated:\n%s", got)
	}
	quiet := crossCheck("11076587", "11076587", map[string]string{"sourcemod": "1.12.0-git7255"}, scan{})
	if !strings.Contains(quiet, "not the same as nothing being wrong") {
		t.Errorf("a quiet check over-claimed:\n%s", quiet)
	}
}

/*
Celsius's bundle on 2026-10-04: TF2 build 11076587, SigMod 20261005 made for

	it, thousands of failed address lookups, and a reset of the game files fixed
	it. The summary told them to compare the build with the pins, two numbers
	that already agreed. When they agree and SigMod still resolves nothing, the
	game files are the thing to look at.
*/
func TestSigModFailingOnItsOwnBuildPointsAtTheGameFiles(t *testing.T) {
	unresolved := scan{found: map[string]*hit{sigmodUnresolvedRule: {count: 8760}}}

	same := crossCheck("11076587", "11076587", map[string]string{"sourcemod": "1.12.0-git7255"}, unresolved)
	if !strings.Contains(same, "updated halfway") || !strings.Contains(same, "Repair") {
		t.Fatalf("a SigMod on its own build resolving nothing did not point at the game files:\n%s", same)
	}

	// On another build the mismatch is the explanation, and this one would
	// send the player to verify files that are fine.
	other := crossCheck("11076587", "11068238", map[string]string{"sourcemod": "1.12.0-git7255"}, unresolved)
	if strings.Contains(other, "updated halfway") {
		t.Errorf("a build mismatch was reported as a half-updated install:\n%s", other)
	}

	// Without failed lookups there is nothing to say about the files.
	clean := crossCheck("11076587", "11076587", map[string]string{"sourcemod": "1.12.0-git7255"}, scan{})
	if strings.Contains(clean, "updated halfway") {
		t.Errorf("a clean run was told to repair its game files:\n%s", clean)
	}
}

/*
The refusal explanation named SourceMod's KeyValues bug on a bundle whose

	SourceMod already carried the fix, while SigMod was faulting under every
	purchase. SourceMod is the cause only below git7255.
*/
func TestRefusedPurchasesOnlyBlameAnOldSourceMod(t *testing.T) {
	const keyValues = "SourceMod that cannot build the KeyValues"

	old := crossCheck("11076587", "11076587", map[string]string{"sourcemod": "1.12.0-git7253"}, scan{refused: 34})
	if !strings.Contains(old, keyValues) {
		t.Errorf("refusals on git7253 did not name the KeyValues bug:\n%s", old)
	}

	unknown := crossCheck("11076587", "11076587", nil, scan{refused: 34})
	if !strings.Contains(unknown, keyValues) {
		t.Errorf("refusals with no SourceMod version lost the KeyValues explanation:\n%s", unknown)
	}

	withSigMod := crossCheck("11076587", "11076587", map[string]string{"sourcemod": "1.12.0-git7255"},
		scan{refused: 34, found: map[string]*hit{sigmodUnresolvedRule: {count: 8760}}})
	if strings.Contains(withSigMod, keyValues) {
		t.Errorf("refusals on git7255 still blamed SourceMod:\n%s", withSigMod)
	}
	if !strings.Contains(withSigMod, "faults when an upgrade is bought") {
		t.Errorf("refusals beside a SigMod that resolves nothing did not point at SigMod:\n%s", withSigMod)
	}

	fixed := crossCheck("11076587", "11076587", map[string]string{"sourcemod": "1.12.0-git7255"}, scan{refused: 34})
	if strings.Contains(fixed, keyValues) || !strings.Contains(fixed, "nothing else in these logs") {
		t.Errorf("refusals on git7255 without SigMod trouble were misexplained:\n%s", fixed)
	}
}
