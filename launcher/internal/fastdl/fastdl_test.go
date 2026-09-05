package fastdl

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func gameTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range map[string]string{
		"maps/mvm_test.bsp":               "bsp",
		"maps/mvm_test.bsp.bz2":           "bz2",
		"materials/models/x.vtf":          "vtf",
		"cfg/server.cfg":                  `rcon_password "secret"`,
		"addons/sourcemod/plugins/ap.smx": "smx",
	} {
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestHandlerServesContentAndNothingElse(t *testing.T) {
	server := httptest.NewServer(Handler(gameTree(t)))
	defer server.Close()
	for _, c := range []struct {
		path string
		want int
		body string
	}{
		{"/tf/maps/mvm_test.bsp", http.StatusOK, "bsp"},
		{"/tf/maps/mvm_test.bsp.bz2", http.StatusOK, "bz2"},
		{"/tf/materials/models/x.vtf", http.StatusOK, "vtf"},
		{"/tf/maps/missing.bsp", http.StatusNotFound, ""},
		{"/tf/maps/", http.StatusNotFound, ""},
		{"/tf/maps", http.StatusNotFound, ""},
		{"/tf", http.StatusOK, statusText},
		{"/tf/", http.StatusOK, statusText},
		{"/tf/cfg/server.cfg", http.StatusNotFound, ""},
		{"/tf/addons/sourcemod/plugins/ap.smx", http.StatusNotFound, ""},
		{"/tf/maps/../cfg/server.cfg", http.StatusNotFound, ""},
		{"/tf/maps/%2e%2e/cfg/server.cfg", http.StatusNotFound, ""},
		{"/tf/maps%5c..%5ccfg%5cserver.cfg", http.StatusNotFound, ""},
		{"/tf/maps/dir%5c..%5c..%5ccfg%5cserver.cfg", http.StatusNotFound, ""},
		{"/maps/mvm_test.bsp", http.StatusNotFound, ""},
		{"/", http.StatusNotFound, ""},
	} {
		resp, err := http.Get(server.URL + c.path)
		if err != nil {
			t.Fatalf("%s: %v", c.path, err)
		}
		body := make([]byte, 64)
		n, _ := resp.Body.Read(body)
		_ = resp.Body.Close()
		if resp.StatusCode != c.want {
			t.Errorf("%s: status %d, want %d", c.path, resp.StatusCode, c.want)
		}
		if c.body != "" && string(body[:n]) != c.body {
			t.Errorf("%s: body %q, want %q", c.path, body[:n], c.body)
		}
	}
}

func TestHandlerDoesNotFollowLinksOutOfAssetDirectory(t *testing.T) {
	game := gameTree(t)
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("outside secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	for link, target := range map[string]string{
		filepath.Join(game, "maps", "outside.bsp"): outside,
		filepath.Join(game, "maps", "server.cfg"):  filepath.Join(game, "cfg", "server.cfg"),
	} {
		if err := os.Symlink(target, link); err != nil {
			t.Skipf("symlinks are unavailable: %v", err)
		}
	}

	server := httptest.NewServer(Handler(game))
	defer server.Close()
	for _, name := range []string{"outside.bsp", "server.cfg"} {
		resp, err := http.Get(server.URL + "/tf/maps/" + name)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s: status %d, want %d", name, resp.StatusCode, http.StatusNotFound)
		}
	}
}

func TestContentPathRequiresCanonicalAssetPath(t *testing.T) {
	for _, test := range []struct {
		path     string
		wantDir  string
		wantFile string
	}{
		{"/tf/maps/mvm_test.bsp", "maps", "mvm_test.bsp"},
		{"/tf/materials/models/x.vtf", "materials", "models/x.vtf"},
		{"/tf/maps/../cfg/server.cfg", "", ""},
		{"/tf/maps/./mvm_test.bsp", "", ""},
		{"/tf/maps//mvm_test.bsp", "", ""},
		{`/tf/maps\..\cfg\server.cfg`, "", ""},
		{"/tf/cfg/server.cfg", "", ""},
		{"/tf/maps/", "", ""},
	} {
		dir, file, ok := contentPath(test.path)
		if ok != (test.wantDir != "") || dir != test.wantDir || file != test.wantFile {
			t.Errorf("contentPath(%q) = (%q, %q, %t), want (%q, %q)", test.path, dir, file, ok, test.wantDir, test.wantFile)
		}
	}
}

func TestHandlerRefusesWrites(t *testing.T) {
	server := httptest.NewServer(Handler(gameTree(t)))
	defer server.Close()
	resp, err := http.Post(server.URL+"/tf/maps/mvm_test.bsp", "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("POST: status %d, want %d", resp.StatusCode, http.StatusMethodNotAllowed)
	}
}

func TestURL(t *testing.T) {
	if got, want := URL("192.168.1.10", DefaultPort), "http://192.168.1.10:27080/tf"; got != want {
		t.Errorf("URL = %q, want %q", got, want)
	}
}
