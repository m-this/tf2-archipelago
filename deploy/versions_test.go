package deploy_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
