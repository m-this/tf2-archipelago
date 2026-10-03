package deploy_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

/*
versions.env names one SourceMod drop, and says which TF2 build it was checked
on.

There used to be two: SOURCEMOD_VERSION for our own plugin and the Windows
launcher, and DEFENDERBOTS_SOURCEMOD_VERSION for the defender mod, which needs
an older spcomp. The second one is gone and deploy/bots/build.sh reads the mod's
own pin instead, so this is the check that it does not come back: a copy of a
pin is a thing that drifts, and that pair had drifted ninety builds apart.

The TF2 build is here because the KeyValues layout a SourceMod drop reads is
decided by the game, not by SourceMod. When the layout moves, a client's
MvM_UpgradesBegin segfaults the server and a plugin's MVM_Upgrade is refused in
silence. Which build a pin was checked on is therefore load bearing, and it
lived in somebody's memory until it was written down here.
*/
func TestVersionsEnvNamesOneSourcemodDropAndTheTF2BuildItWasCheckedOn(t *testing.T) {
	t.Parallel()

	pins := readEnvFile(t, filepath.Join("env", "versions.env"))

	// A SourceMod drop by the shape of what it names, not by the name somebody
	// gave it: the next copy of this pin will be called something else.
	drop := regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+-git[0-9]+$`)
	var named []string
	for key, value := range pins {
		if strings.Contains(key, "SOURCEMOD") && drop.MatchString(value) {
			named = append(named, key)
		}
	}
	if len(named) != 1 || named[0] != "SOURCEMOD_VERSION" {
		t.Errorf("versions.env names these SourceMod drops: %q, want only SOURCEMOD_VERSION. A second pin for the same thing is a second thing to forget", named)
	}

	build := pins["SOURCEMOD_TF2_SERVER_VERSION"]
	if !regexp.MustCompile(`^[0-9]+$`).MatchString(build) {
		t.Errorf("SOURCEMOD_TF2_SERVER_VERSION = %q, want the bare TF2 ServerVersion the SourceMod pin was checked on", build)
	}
}

/*
The SourceMod pin is not keyed to a TF2 build older than the one the stack is on.

TF2_SERVER_VERSION is the build somebody last checked the game-keyed pins
against, and versions.env says bumping it is the last step of dealing with an
update, not the first: .github/workflows/tf2-build-watch.yml keeps filing the
issue until it moves. So the two fields are only allowed to drift one way. A
SOURCEMOD_TF2_SERVER_VERSION ahead of TF2_SERVER_VERSION is somebody who checked
SourceMod against a new build while another pin is still being fixed, which is
the normal middle of that work. Behind means the game has moved past the last
build this drop was run on and nothing says so, which is a server that segfaults
when a client joins while the watcher is quiet, because TF2_SERVER_VERSION says
the pins are fine.

That is the state this repository was in between the 2026-10-02 update and the
git7255 pin, and it was found by reading the file rather than by a gate.

Numeric, not string: a ServerVersion is an increasing integer and will change
digit count.
*/
func TestTheSourcemodPinIsNotBehindTheTF2BuildTheStackIsOn(t *testing.T) {
	t.Parallel()

	pins := readEnvFile(t, filepath.Join("env", "versions.env"))

	checked, err := strconv.ParseInt(pins["SOURCEMOD_TF2_SERVER_VERSION"], 10, 64)
	if err != nil {
		t.Fatalf("SOURCEMOD_TF2_SERVER_VERSION = %q, want a bare TF2 ServerVersion: %v", pins["SOURCEMOD_TF2_SERVER_VERSION"], err)
	}
	stack, err := strconv.ParseInt(pins["TF2_SERVER_VERSION"], 10, 64)
	if err != nil {
		t.Fatalf("TF2_SERVER_VERSION = %q, want a bare TF2 ServerVersion: %v", pins["TF2_SERVER_VERSION"], err)
	}

	if checked < stack {
		t.Errorf("SOURCEMOD_TF2_SERVER_VERSION = %d, behind TF2_SERVER_VERSION = %d: SOURCEMOD_VERSION=%s has not been run on the build the rest of the pins are checked against, so bump it once it has, or hold TF2_SERVER_VERSION back until the pins are right", checked, stack, pins["SOURCEMOD_VERSION"])
	}
}

/*
The defender mod's compiler pin resolves out of the pinned module.

deploy/bots/build.sh reads the drop the mod compiles with from the mod's own
plugin/testbed/versions.env, which is the only place it is written down. That
derivation is a sed over a file in another repository, so it breaks quietly:
rename the key there, or write it as anything other than KEY=value on one line,
and the script exits with a message about a missing pin six minutes into an
image build. This asks the same question in the gate.

The keys come out of the script rather than being listed here, so the two cannot
disagree about which ones matter.
*/
func TestTheDefenderModNamesTheCompilerTheBotsBuildReads(t *testing.T) {
	t.Parallel()

	script, err := os.ReadFile(filepath.Join("bots", "build.sh"))
	if err != nil {
		t.Fatal(err)
	}
	read := regexp.MustCompile(`defenderbots_pin ([A-Z0-9_]+)`).FindAllStringSubmatch(string(script), -1)
	if len(read) == 0 {
		t.Fatal("bots/build.sh reads no pin out of the mod's versions.env; this test is checking something that is no longer there")
	}

	module := "github.com/m-this/tf2-mvm-bots-go"
	// Downloaded first: the module cache is where the pin is read from, and a
	// checkout that has only ever built has no Dir for it yet.
	if out, err := run(t, "go", "mod", "download", module); err != nil {
		t.Fatalf("go mod download %s: %v\n%s", module, err, out)
	}
	dir, err := run(t, "go", "list", "-m", "-f", "{{.Dir}}", module)
	if err != nil {
		t.Fatalf("go list -m %s: %v\n%s", module, err, dir)
	}

	versions := filepath.Join(strings.TrimSpace(dir), "plugin", "testbed", "versions.env")
	pins := readEnvFile(t, versions)
	for _, key := range read {
		if pins[key[1]] == "" {
			t.Errorf("bots/build.sh reads %s out of %s and that file does not name it", key[1], versions)
		}
	}
}

// readEnvFile is the KEY=value lines of one env file, read the way the shell
// scripts and `docker compose --env-file` read it: one assignment per line,
// nothing continued, comments and blanks skipped.
func readEnvFile(t *testing.T, path string) map[string]string {
	t.Helper()

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	pins := map[string]string{}
	for line := range strings.SplitSeq(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		pins[key] = value
	}
	return pins
}

// run is one command from the repository root, which is a directory up from
// this test.
func run(t *testing.T, name string, args ...string) (string, error) {
	t.Helper()

	command := exec.Command(name, args...)
	command.Dir = ".."
	out, err := command.CombinedOutput()
	return string(out), err
}
