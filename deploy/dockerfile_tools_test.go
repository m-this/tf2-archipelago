package deploy_test

import (
	"maps"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"testing"
)

/*
The tools stage copies every package its commands need.

That stage lists the tree it copies rather than taking the repository whole,
because the launcher's embedded assets are a third of a gigabyte and are
.dockerignored: a stage that copied everything could not compile at all. The
cost of the list is that adding an import to botfiles breaks the image build
and nothing else, and the image is built by the nightly and the release rather
than by the gate. This is the gate asking.
*/
func TestTheToolsStageCopiesWhatItsCommandsImport(t *testing.T) {
	t.Parallel()

	dockerfile, err := os.ReadFile("Dockerfile.srcds")
	if err != nil {
		t.Fatal(err)
	}
	stage, _, found := strings.Cut(string(dockerfile), "FROM cm2network")
	if !found {
		t.Fatal("Dockerfile.srcds has no runtime stage to read up to")
	}

	for _, command := range []string{"./launcher/cmd/rcon", "./launcher/cmd/botfiles", "./deploy/stockpop"} {
		// From the repository root: the package paths are relative to it, and
		// this test lives a directory down.
		list := exec.Command("go", "list", "-deps", command)
		list.Dir = ".."
		out, err := list.Output()
		if err != nil {
			t.Fatalf("go list %s: %v", command, err)
		}
		for dep := range strings.FieldsSeq(string(out)) {
			path, inRepo := strings.CutPrefix(dep, "github.com/m-this/tf2-archipelago/")
			if !inRepo {
				continue
			}
			if !strings.Contains(stage, "COPY "+path+" "+path) {
				t.Errorf("%s needs %s and the tools stage does not copy it", command, path)
			}
		}
	}
}

/*
The bots stage copies every repository file its build script reads.

Same failure as the one above and a slower one to find: build.sh reaches for a
path with $root in front of it, the stage copies a handful of directories, and
the two only meet in an image build. Moving bot_names.txt into the launcher
broke this and nothing said so until six minutes of compiling had gone by.
*/
func TestTheBotsStageCopiesWhatItsScriptReads(t *testing.T) {
	t.Parallel()

	dockerfile, err := os.ReadFile("Dockerfile.srcds")
	if err != nil {
		t.Fatal(err)
	}
	stage := between(t, string(dockerfile), "FROM golang:1.27-bookworm AS bots", "\nFROM ")

	script, err := os.ReadFile("bots/build.sh")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range slices.Sorted(maps.Keys(rootPaths(string(script)))) {
		if !strings.Contains(stage, "COPY "+path) {
			t.Errorf("bots/build.sh reads %s and the bots stage does not copy it", path)
		}
	}
}

// rootPaths is every repository path the script names with $root in front of
// it, as the directory the Dockerfile would copy.
func rootPaths(script string) map[string]bool {
	out := map[string]bool{}
	for _, match := range regexp.MustCompile(`\$root/([A-Za-z0-9_./-]+)`).FindAllStringSubmatch(script, -1) {
		parts := strings.Split(match[1], "/")
		if len(parts) < 2 {
			continue
		}
		// The two directories deep the stage lists, which is as fine-grained
		// as any COPY here gets.
		out[strings.Join(parts[:2], "/")] = true
	}
	return out
}

func between(t *testing.T, body, from, to string) string {
	t.Helper()
	_, after, found := strings.Cut(body, from)
	if !found {
		t.Fatalf("Dockerfile.srcds has no %q", from)
	}
	stage, _, found := strings.Cut(after, to)
	if !found {
		return after
	}
	return stage
}
