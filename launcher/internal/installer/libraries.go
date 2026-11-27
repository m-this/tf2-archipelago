package installer

import (
	"bytes"
	"context"
	"debug/elf"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"
)

/*
The system libraries the Linux game server loads, checked before it starts.

srcds dlopens replay_srv.so right after the game's own server_srv.so. When that
fails it prints "Could not load: replay_srv.so", segfaults, and srcds_run starts
it again into the same failure every 15 seconds. The engine never names the
library it lacked. ldd does, so the installer asks it, with the library path
srcds_run sets, before the server is allowed to start.

replay_srv.so needs libcurl-gnutls.so.4, a name only Debian and Arch package.
Fedora, and so Bazzite, has one 32-bit curl, libcurl.so.4, built without symbol
versions, and the loader accepts an unversioned library for the two
curl_global_* calls replay_srv.so makes. So where the gnutls build is missing
and that one is installed, a link in tf-dedicated/bin, which srcds_run puts on
LD_LIBRARY_PATH, stands in for it. ldd then says whether the loader took it: a
versioned curl, Debian's or Arch's, is refused with "version `CURL_GNUTLS_3'
not found", and the link comes back out.
*/

// CurlGnutls is the curl replay_srv.so is linked against.
const CurlGnutls = "libcurl-gnutls.so.4"

// serverLibraries are the libraries srcds loads on its way up, relative to the
// game directory.
var serverLibraries = []string{"bin/replay_srv.so", "bin/engine_srv.so", "bin/dedicated_srv.so", "tf/bin/server_srv.so"}

// libraryDirs32 is where Arch, Debian and Fedora keep 32-bit x86 libraries.
var libraryDirs32 = []string{"/usr/lib32", "/usr/lib/i386-linux-gnu", "/lib/i386-linux-gnu", "/usr/lib", "/lib"}

const lddTimeout = 30 * time.Second

// ErrMissingLibrary is a game server that would crash at replay_srv.so, or at
// another library it loads, because the system lacks a library it needs.
var ErrMissingLibrary = errors.New("the game server is missing a system library")

// curlAdvice names the package on each distribution, because nothing in the
// engine's output names anything a package manager knows.
const curlAdvice = "Install the 32-bit curl library, then start again: " +
	"`sudo dnf install libcurl.i686` on Fedora, " +
	"`rpm-ostree install libcurl.i686` and a reboot on Bazzite or another atomic Fedora " +
	"(or dnf inside the distrobox or toolbox the launcher runs in), " +
	"lib32-libcurl-gnutls on Arch, " +
	"libcurl3t64-gnutls:i386 on Debian 13 and Ubuntu 24.04 or later (libcurl3-gnutls:i386 before them)."

func ensureLibraries(ctx context.Context, gameDir string, standInDirs []string, logf func(string, ...any)) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	missing, err := MissingLibraries(ctx, gameDir)
	if errors.Is(err, exec.ErrNotFound) {
		logf("ldd is not installed, so the game server's libraries were not checked")
		return nil
	}
	if err != nil || !slices.Contains(missing, CurlGnutls) {
		return errors.Join(err, libraryError(missing))
	}
	standIn := curlStandIn(standInDirs)
	if standIn == "" {
		return libraryError(missing)
	}
	link := filepath.Join(gameDir, "bin", CurlGnutls)
	if err := os.Remove(link); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot replace %s: %w", link, err)
	}
	if err := os.Symlink(standIn, link); err != nil {
		return fmt.Errorf("cannot link %s into %s: %w", standIn, link, err)
	}
	missing, err = MissingLibraries(ctx, gameDir)
	if err != nil {
		return err
	}
	if slices.Contains(missing, CurlGnutls) {
		if err := os.Remove(link); err != nil {
			return fmt.Errorf("cannot remove %s, which the game server refuses: %w", link, err)
		}
		return libraryError(missing)
	}
	logf("linked %s to %s for replay_srv.so", link, standIn)
	return libraryError(missing)
}

// MissingLibraries asks ldd which libraries the game server's own cannot find,
// with the library path srcds_run gives them.
func MissingLibraries(ctx context.Context, gameDir string) ([]string, error) {
	var files []string
	for _, name := range serverLibraries {
		if path := filepath.Join(gameDir, name); exists(path) {
			files = append(files, path)
		}
	}
	if len(files) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, lddTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ldd", files...)
	cmd.Dir = gameDir
	cmd.Env = libraryEnv(gameDir)
	out, err := cmd.CombinedOutput()
	missing := missingFromLdd(string(out))
	if err != nil && len(missing) == 0 {
		return nil, fmt.Errorf("cannot check the game server's libraries: ldd: %w: %s", err, bytes.TrimSpace(out))
	}
	return missing, nil
}

// libraryEnv is this process's environment with the LD_LIBRARY_PATH srcds_run
// exports: ".:bin:" ahead of whatever was there.
func libraryEnv(gameDir string) []string {
	dirs := []string{gameDir, filepath.Join(gameDir, "bin")}
	if old := os.Getenv("LD_LIBRARY_PATH"); old != "" { //nolint:forbidigo // the loader's own path, which srcds inherits
		dirs = append(dirs, old)
	}
	env := slices.DeleteFunc(os.Environ(), func(entry string) bool {
		return strings.HasPrefix(entry, "LD_LIBRARY_PATH=")
	})
	return append(env, "LD_LIBRARY_PATH="+strings.Join(dirs, ":"))
}

// missingFromLdd reads the libraries ldd could not find, or found in a version
// that lacks the symbols asked for, once each.
func missingFromLdd(output string) []string {
	var missing []string
	for line := range strings.Lines(output) {
		line = strings.TrimSpace(line)
		var name string
		if before, ok := strings.CutSuffix(line, "=> not found"); ok {
			name = strings.TrimSpace(before)
		} else if before, _, ok := strings.Cut(line, ": version `"); ok {
			name = filepath.Base(before[strings.LastIndex(before, ": ")+2:])
		}
		if name != "" && !slices.Contains(missing, name) {
			missing = append(missing, name)
		}
	}
	return missing
}

// curlStandIn is the first 32-bit x86 libcurl.so.4 in dirs, or "" for none.
func curlStandIn(dirs []string) string {
	for _, dir := range dirs {
		path := filepath.Join(dir, "libcurl.so.4")
		f, err := elf.Open(path)
		if err != nil {
			continue
		}
		fits := f.Class == elf.ELFCLASS32 && f.Machine == elf.EM_386
		_ = f.Close()
		if fits {
			return path
		}
	}
	return ""
}

func libraryError(missing []string) error {
	if len(missing) == 0 {
		return nil
	}
	advice := "Install the 32-bit package that provides each of them, then start again."
	if slices.Contains(missing, CurlGnutls) {
		advice = curlAdvice
	}
	return fmt.Errorf("%w: the game server would crash at replay_srv.so or the next library it loads, without %s. %s",
		ErrMissingLibrary, strings.Join(missing, ", "), advice)
}
