package installer

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const manifestNormal = `"AppState"
{
	"appid"		"232250"
	"Universe"		"1"
	"name"		"Team Fortress 2 Dedicated Server"
	"StateFlags"		"4"
	"installdir"		"Team Fortress 2 Dedicated Server"
	"LastUpdated"		"1727856000"
	"SizeOnDisk"		"14523675392"
	"buildid"		"16234567"
	"TargetBuildID"		"16234567"
	"InstalledDepots"
	{
		"232251"
		{
			"manifest"		"1234567890123456789"
			"size"		"14000000000"
			"buildid"		"999"
		}
	}
	"UserConfig"
	{
	}
}
`

func installManifest(t *testing.T, text string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(gamePath(root), "steamapps")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "appmanifest_"+AppID+".acf"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestInstalledBuildReadsTheTopLevelBuildID(t *testing.T) {
	build, err := InstalledBuild(installManifest(t, manifestNormal))
	if err != nil {
		t.Fatal(err)
	}
	if build != "16234567" {
		t.Fatalf("build = %q, want 16234567", build)
	}
}

func TestInstalledBuildIgnoresADepotsBuildID(t *testing.T) {
	text := strings.Replace(manifestNormal, "\t\"buildid\"\t\t\"16234567\"\n", "", 1)
	if _, err := InstalledBuild(installManifest(t, text)); err == nil {
		t.Fatal("a buildid inside InstalledDepots was read as the app's")
	}
}

func TestInstalledBuildWithoutAManifest(t *testing.T) {
	_, err := InstalledBuild(t.TempDir())
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("err = %v, want fs.ErrNotExist", err)
	}
}

func TestInstalledBuildRefusesAMalformedManifest(t *testing.T) {
	for name, text := range map[string]string{
		"empty":          "",
		"not keyvalues":  "buildid=16234567\n",
		"unclosed quote": "\"AppState\"\n{\n\t\"buildid\"\t\t\"16234567\n}\n",
		"unclosed block": "\"AppState\"\n{\n\t\"buildid\"\t\t\"16234567\"\n",
		"stray brace":    "\"AppState\"\n{\n}\n}\n",
		"not a number":   "\"AppState\"\n{\n\t\"buildid\"\t\t\"latest\"\n}\n",
		"other root":     "\"Other\"\n{\n\t\"buildid\"\t\t\"16234567\"\n}\n",
		"key with block": "\"AppState\"\n{\n\t\"buildid\"\n\t{\n\t}\n}\n",
	} {
		t.Run(name, func(t *testing.T) {
			if build, err := InstalledBuild(installManifest(t, text)); err == nil {
				t.Fatalf("build = %q, want an error", build)
			}
		})
	}
}
