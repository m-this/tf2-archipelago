// Package fastdl serves the game's content tree over HTTP, so a client that
// joins without a community map fetches it from here rather than through the
// game server's in-band transfer. That transfer can reach the end of a large
// packed BSP without the client accepting it, and the map restarts forever.
//
// Everything is on the same machine as the game server. The caller chooses
// whether the listener is available on the LAN or only on loopback behind a
// public tunnel; this package owns only the HTTP paths and files.
package fastdl

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// DefaultPort is one above the game port's neighbourhood so it never collides
// with srcds's own TCP listener on the game port.
const DefaultPort = 27080

// urlPrefix is what sv_downloadurl ends in. The client appends the file's
// game-relative path: <url>/maps/mvm_x.bsp.bz2, then <url>/maps/mvm_x.bsp.
const urlPrefix = "/tf/"

// contentDirs are the only directories a client is served from. The game tree
// also holds cfg/, with the rcon password in it, and addons/, with the
// plugins: neither is anything a client downloads, so neither is reachable.
var contentDirs = map[string]bool{
	"maps": true, "materials": true, "models": true, "sound": true,
	"particles": true, "resource": true,
}

// Timeouts bound one slow client without cutting off a slow one. A packed BSP
// is tens of megabytes, and a friend on a poor line can spend longer than five
// minutes on it, so the write gets a quarter of an hour; the idle timeout is
// what a blip mid-download has to fit in before the client asks again.
const (
	readHeaderTimeout = 10 * time.Second
	writeTimeout      = 15 * time.Minute
	idleTimeout       = 2 * time.Minute
	statusText        = "TF2 Archipelago FastDL is ready.\n"
)

// URL is the sv_downloadurl value for a server on host, serving on port.
func URL(host string, port int) string {
	return "http://" + net.JoinHostPort(host, strconv.Itoa(port)) + strings.TrimSuffix(urlPrefix, "/")
}

// Handler serves files under gameDir, the game's tf/ directory. Only regular
// files inside contentDirs answer; a directory, a path that climbs out, or
// anything in another directory is a 404, indistinguishable from a file that
// is not there.
func Handler(gameDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// This is a health check, not a directory listing. It makes the URL
		// shown by the launcher useful to an operator without exposing names.
		if r.URL.Path == strings.TrimSuffix(urlPrefix, "/") || r.URL.Path == urlPrefix {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte(statusText))
			return
		}
		dir, relative, ok := contentPath(r.URL.Path)
		if !ok {
			http.NotFound(w, r)
			return
		}
		// Root confines resolution even through symlinks and Windows junctions.
		// Rooting again at the allowlisted directory also prevents a link in
		// maps/ from reaching a sensitive sibling such as cfg/ or addons/.
		gameRoot, err := os.OpenRoot(gameDir)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer func() { _ = gameRoot.Close() }()
		assetRoot, err := gameRoot.OpenRoot(dir)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer func() { _ = assetRoot.Close() }()
		file, err := assetRoot.Open(filepath.FromSlash(relative))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer func() { _ = file.Close() }()
		info, err := file.Stat()
		if err != nil || !info.Mode().IsRegular() {
			http.NotFound(w, r)
			return
		}
		http.ServeContent(w, r, filepath.Base(relative), info.ModTime(), file)
	})
}

// contentPath separates a request into an allowlisted asset directory and a
// path within it. Backslashes are rejected before filepath can interpret them
// as separators on Windows.
func contentPath(requestPath string) (dir, relative string, ok bool) {
	if !strings.HasPrefix(requestPath, urlPrefix) {
		return "", "", false
	}
	untrusted := strings.TrimPrefix(requestPath, urlPrefix)
	if strings.ContainsRune(untrusted, '\\') || !fs.ValidPath(untrusted) {
		return "", "", false
	}
	dir, relative, found := strings.Cut(untrusted, "/")
	if !found || relative == "" || !contentDirs[dir] {
		return "", "", false
	}
	return dir, relative, true
}

// Serve listens on listen until ctx ends. It returns nil on a clean stop and
// the listen error when the port is taken, which is worth a line in the log
// and not worth stopping the game server over.
func Serve(ctx context.Context, gameDir, listen string) error {
	server := &http.Server{
		Addr:              listen,
		Handler:           Handler(gameDir),
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
	done := make(chan error, 1)
	go func() { done <- server.ListenAndServe() }()
	select {
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.WithoutCancel(ctx), idleTimeout)
		defer cancel()
		_ = server.Shutdown(shutdown)
		return nil
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("fastdl on %s: %w", listen, err)
	}
}
