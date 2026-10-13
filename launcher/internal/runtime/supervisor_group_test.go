//go:build !windows

package runtime

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// srcds_run is a shell script that runs the game server as a child. Killing
// the script alone leaves that child holding the game port and the rcon port,
// and the next Start binds neither.
func TestStopTakesDownTheServerTheScriptStarted(t *testing.T) {
	root := t.TempDir()
	s := fakeServer(t, root)
	pidFile := filepath.Join(root, "child.pid")
	script := "#!/bin/sh\nsleep 600 &\necho $! > " + pidFile + "\necho fake server up\nwait\n"
	if err := os.WriteFile(filepath.Join(root, "tf-dedicated", "srcds_run"), []byte(script), 0o755); err != nil {
		t.Fatalf("cannot write the fake server: %v", err)
	}

	spoke := make(chan struct{})
	var once sync.Once
	sup := NewSupervisor(s, nil, func(line Line) {
		if line.Source == "srcds" && line.Text == "fake server up" {
			once.Do(func() { close(spoke) })
		}
	})
	if err := sup.Start(func(error) {}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer sup.Stop()
	select {
	case <-spoke:
	case <-time.After(10 * time.Second):
		t.Fatal("the fake server never spoke")
	}

	raw, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("the fake server wrote no pid: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatalf("the pid file holds %q: %v", raw, err)
	}

	sup.Stop()
	if alive(pid) {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		t.Fatalf("process %d outlived the Stop", pid)
	}
}

// A game server that sits on the SIGTERM has to go too. srcds_run passes the
// signal to srcds_linux, and what it does with it is the game's business, so
// the launcher cannot be the thing that gives up: WaitDelay kills the script
// and the script is not what holds the ports.
func TestStopKillsAServerThatIgnoresTheSignal(t *testing.T) {
	root := t.TempDir()
	s := fakeServer(t, root)
	pidFile := filepath.Join(root, "child.pid")
	// trap '' TERM is a shell that cannot be asked to leave, only killed, and
	// its child inherits the ignored signal with it.
	script := "#!/bin/sh\ntrap '' TERM\nsleep 600 &\necho $! > " + pidFile +
		"\necho fake server up\nwait\n"
	if err := os.WriteFile(filepath.Join(root, "tf-dedicated", "srcds_run"), []byte(script), 0o755); err != nil {
		t.Fatalf("cannot write the fake server: %v", err)
	}

	spoke := make(chan struct{})
	var once sync.Once
	sup := NewSupervisor(s, nil, func(line Line) {
		if line.Source == "srcds" && line.Text == "fake server up" {
			once.Do(func() { close(spoke) })
		}
	})
	if err := sup.Start(func(error) {}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer sup.Stop()
	select {
	case <-spoke:
	case <-time.After(10 * time.Second):
		t.Fatal("the fake server never spoke")
	}

	raw, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("the fake server wrote no pid: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatalf("the pid file holds %q: %v", raw, err)
	}

	sup.Stop()
	// Longer than the grace the group gets before it is killed: Stop returns
	// when the wait for it gave up, and the kill lands after that.
	if aliveFor(pid, 10*time.Second) {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		t.Fatalf("process %d survived a Stop it ignored", pid)
	}
}

// alive reports whether the process is still there, with a moment's grace for
// the signal to land.
func alive(pid int) bool {
	return aliveFor(pid, 2*time.Second)
}

// aliveFor reports whether the process is still there after waiting up to this
// long for it to go.
func aliveFor(pid int, grace time.Duration) bool {
	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		if !running(pid) {
			return false
		}
		time.Sleep(100 * time.Millisecond)
	}
	return true
}

// running reports whether the process is a live one.
//
// kill(pid, 0) alone says yes for a zombie, and a killed orphan stays one
// until something reaps it. Under systemd or launchd that is immediate, so the
// tests pass on a laptop; in a container whose pid 1 is `tail -f /dev/null`,
// which is what the CI job runs in, nothing ever reaps it and every Stop test
// fails on a process that is already dead. Read the state where the kernel
// publishes it, and fall back to the signal where it does not.
func running(pid int) bool {
	stat, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err == nil {
		return !zombie(stat)
	}
	return syscall.Kill(pid, 0) == nil
}

// zombie reads the state field out of /proc/<pid>/stat. It comes after the
// program name, which is parenthesised and may itself hold spaces or brackets,
// so the split is on the last ')' rather than on the second field.
func zombie(stat []byte) bool {
	_, rest, ok := bytes.CutLast(stat, []byte(")"))
	if !ok {
		return false
	}
	fields := strings.Fields(string(rest))
	return len(fields) > 0 && fields[0] == "Z"
}
