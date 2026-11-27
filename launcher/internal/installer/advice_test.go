package installer

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The game server is 32-bit on every platform. A 64-bit Linux without the
// 32-bit C library runs neither it nor SteamCMD, and the shell names no
// library when it says so. Repair cannot help, so the installer must not
// suggest it.
func TestMissing32BitIsRecognised(t *testing.T) {
	for _, line := range []string{
		"/home/me/tf2-archipelago/steamcmd/steamcmd.sh: line 37: /home/me/tf2-archipelago/steamcmd/linux32/steamcmd: cannot execute: required file not found",
		"./srcds_linux: /lib/ld-linux.so.2: No such file or directory",
	} {
		if !missing32Bit(line) {
			t.Errorf("missed a 32-bit failure: %q", line)
		}
		advice := steamcmdStateAdvice(line)
		if !strings.Contains(advice, "32-bit C library") {
			t.Errorf("advice for %q was %q", line, advice)
		}
	}
}

// Every other line SteamCMD prints, including the two states that already had
// advice of their own. None of them is a missing library.
func TestMissing32BitLeavesOtherLinesAlone(t *testing.T) {
	for _, line := range []string{
		" Update state (0x61) downloading, progress: 1.36 (202092440 / 14908968403)",
		"Error! app '232250' state is 0x202 after update job.",
		"Error! app '232250' state is 0x602 after update job.",
		"Success! App '232250' fully installed.",
		"",
	} {
		if missing32Bit(line) {
			t.Errorf("read a 32-bit failure out of %q", line)
		}
	}
	// The two that had advice before keep it.
	if advice := steamcmdStateAdvice("app state is 0x202 after update job"); !strings.Contains(advice, "disk is full") {
		t.Errorf("0x202 advice = %q", advice)
	}
	if advice := steamcmdStateAdvice("app state is 0x602 after update job"); !strings.Contains(advice, "lost the download") {
		t.Errorf("0x602 advice = %q", advice)
	}
}

// The sighting has to reach the caller, not just the log: the window prints
// the error, and the error is where the operator looks first.
func TestLineSplitterRemembersThe32BitFailure(t *testing.T) {
	writer := lineWriter(func(string, ...any) {})
	if _, err := writer.Write([]byte("steamcmd.sh: line 37: linux32/steamcmd: cannot execute: required file not found\n")); err != nil {
		t.Fatal(err)
	}
	if !writer.saw32BitFailure {
		t.Error("the splitter did not remember the 32-bit failure")
	}

	quiet := lineWriter(func(string, ...any) {})
	if _, err := quiet.Write([]byte("Success! App '232250' fully installed.\n")); err != nil {
		t.Fatal(err)
	}
	if quiet.saw32BitFailure {
		t.Error("an ordinary line was read as a 32-bit failure")
	}
}

// Lee's install stopped on a 404 from mms.alliedmods.net while the site was
// down, and the launcher told him to press Repair, which fetches the same URL
// from the same site again. A host that fails is a host to wait for, and the
// player needs its address to see that it is not their machine.
func TestADownloadHostThatFailsIsNotARepair(t *testing.T) {
	failing := func(status int) string {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
		}))
		t.Cleanup(server.Close)
		return server.URL + "/mmsdrop/1.12/mmsource-1.12.0-git1228-linux.tar.gz"
	}
	closed := httptest.NewServer(http.NotFoundHandler())
	closed.Close()
	unreachable := closed.URL + "/sourcemod.tar.gz"

	for _, url := range []string{failing(http.StatusNotFound), failing(http.StatusBadGateway), unreachable} {
		_, err := fetch(t.Context(), url)
		if err == nil {
			t.Fatalf("%s: fetch succeeded", url)
		}
		advice := Advice(fmt.Errorf("cannot download Metamod:Source: %w", err))
		if strings.Contains(advice, "Repair") {
			t.Errorf("%s: advises Repair: %q", url, advice)
		}
		if !strings.Contains(advice, url) {
			t.Errorf("%s: does not name the URL: %q", url, advice)
		}
		if !strings.Contains(advice, "later") {
			t.Errorf("%s: does not say to try again later: %q", url, advice)
		}
	}
}

// Everything else an install can trip on is still worth a Repair.
func TestAnyOtherInstallFailureStillAdvisesRepair(t *testing.T) {
	if advice := Advice(errors.New("cannot unpack Metamod:Source: unexpected EOF")); advice != RepairAdvice {
		t.Errorf("advice = %q", advice)
	}
}

// A missing system library is the system's to install: Repair downloads the
// same game files and crashes on the same replay_srv.so.
func TestAMissingSystemLibraryIsNotARepair(t *testing.T) {
	advice := Advice(fmt.Errorf("install: %w", libraryError([]string{CurlGnutls})))
	if advice == RepairAdvice || !strings.Contains(advice, "package") {
		t.Errorf("advice = %q", advice)
	}
}
