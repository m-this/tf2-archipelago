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

// The crash is the line that matters most, and it is the one that was read as
// "bridge stopping" for weeks.
func TestTheCrashIsFound(t *testing.T) {
	got, _ := scanLogs(writeLog(t,
		`21:09:25  srcds    something ordinary`,
		`21:09:26  bridge   time=2026-08-26T21:09:26.548+09:00 level=INFO msg="bridge stopping"`,
		`21:09:26  launcher game server stopped: exit status 0xc0000005`,
	))
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
	got, _ := scanLogs(writeLog(t, lines...))
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
	got, _ := scanLogs(writeLog(t, `20:00:00  srcds    Server is hibernating`))
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
	if got, _ := scanLogs(filepath.Join(t.TempDir(), "absent.log")); got != "" {
		t.Errorf("a missing log produced %q", got)
	}
}

func TestTheStuckBotAndTheThrowAreFound(t *testing.T) {
	got, _ := scanLogs(writeLog(t,
		`20:35:28  srcds    [defenderbots] stuck: SomeDude (engineer) at 289 571 544 for 12s, DefenderEngineerIdle`,
		`20:25:45  sourcemod [SM] Exception reported: Assertion failed - wearable entity 565 not attached to player`,
		`18:24:12  srcds    [AP] error: The run restarted. The plugin asks for the unlock set again.`,
	))
	for _, want := range []string{"a bot got stuck", "a plugin threw", "the plugin reported an error"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q missing from:\n%s", want, got)
		}
	}
}

// The crash flag is what decides whether a missing minidump is worth saying
// anything about, so it has to be false on a quiet run.
func TestTheCrashFlagFollowsTheCrash(t *testing.T) {
	if _, sawCrash := scanLogs(writeLog(t, `20:00:00  srcds    Server is hibernating`)); sawCrash {
		t.Error("a quiet run reported a crash")
	}
	_, sawCrash := scanLogs(writeLog(t, `21:09:26  launcher game server stopped: exit status 0xc0000005`))
	if !sawCrash {
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
	got, _ := scanLogs(writeLog(t,
		`21:19:01  srcds    Executing dedicated server config file server.cfg`,
		`21:19:01  srcds    Unknown command "sm_redbots_manager_mode"`,
		`21:19:01  srcds    Unknown command "tf2ap_start_mission"`,
	))
	if !strings.Contains(got, pluginMissingRule) {
		t.Fatalf("the missing plugin was not reported:\n%s", got)
	}
	if !strings.Contains(got, "stock Mann vs") {
		t.Errorf("the summary does not say what the lines mean:\n%s", got)
	}
}
