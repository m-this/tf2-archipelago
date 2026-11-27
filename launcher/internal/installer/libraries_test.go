//go:build !windows

package installer

import (
	"bytes"
	"context"
	"debug/elf"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// What ldd printed for the release launcher's install in a Fedora 43 container
// with glibc.i686 and nothing else: the restart loop LuigiFan45 and Lee
// Harrold sent, named.
const lddFedoraNoCurl = `/w/root/tf-dedicated/bin/replay_srv.so:
	linux-gate.so.1 (0xf7f2e000)
	libtier0_srv.so => /w/root/tf-dedicated/bin/libtier0_srv.so (0xf7dd7000)
	libvstdlib_srv.so => /w/root/tf-dedicated/bin/libvstdlib_srv.so (0xf7c9d000)
	libcurl-gnutls.so.4 => not found
	libdl.so.2 => /lib/libdl.so.2 (0xf7c96000)
	/lib/ld-linux.so.2 (0xf7f30000)
	libc.so.6 => /lib/libc.so.6 (0xf79a9000)
/w/root/tf-dedicated/bin/engine_srv.so:
	linux-gate.so.1 (0xf7f84000)
	libsteam_api.so => /w/root/tf-dedicated/bin/libsteam_api.so (0xf7a02000)
	libc.so.6 => /lib/libc.so.6 (0xf770e000)
`

// What ldd printed on Debian 13 with Debian's versioned libcurl.so.4 linked in
// under the gnutls name: the loader refuses it.
const lddDebianVersioned = `/g/bin/replay_srv.so: /x/libcurl-gnutls.so.4: version ` + "`CURL_GNUTLS_3'" + ` not found (required by /g/bin/replay_srv.so)
	linux-gate.so.1 (0xf7f20000)
	libcurl-gnutls.so.4 => /x/libcurl-gnutls.so.4 (0xf7e00000)
	/lib/ld-linux.so.2 (0xf7f22000)
`

func TestMissingFromLdd(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   []string
	}{
		{"a library not found", lddFedoraNoCurl, []string{CurlGnutls}},
		{"a library of the wrong version", lddDebianVersioned, []string{CurlGnutls}},
		{"everything found", "/g/bin/engine_srv.so:\n\tlibc.so.6 => /lib/libc.so.6 (0xf770e000)\n", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := missingFromLdd(tt.output); !slices.Equal(got, tt.want) {
				t.Errorf("missingFromLdd = %q, want %q", got, tt.want)
			}
		})
	}
}

func writeELF(t *testing.T, path string, class elf.Class, machine elf.Machine) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	ident := [elf.EI_NIDENT]byte{0x7f, 'E', 'L', 'F', byte(class), byte(elf.ELFDATA2LSB), byte(elf.EV_CURRENT)}
	var header any
	switch class {
	case elf.ELFCLASS32:
		header = elf.Header32{Ident: ident, Type: uint16(elf.ET_DYN), Machine: uint16(machine), Version: uint32(elf.EV_CURRENT), Ehsize: 52}
	default:
		header = elf.Header64{Ident: ident, Type: uint16(elf.ET_DYN), Machine: uint16(machine), Version: uint32(elf.EV_CURRENT), Ehsize: 64}
	}
	if err := binary.Write(&b, binary.LittleEndian, header); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Arch keeps the 64-bit curl in /usr/lib, where Fedora keeps the 32-bit one.
func TestCurlStandInTakesOnlyA32BitCurl(t *testing.T) {
	lib64, lib32 := t.TempDir(), t.TempDir()
	writeELF(t, filepath.Join(lib64, "libcurl.so.4"), elf.ELFCLASS64, elf.EM_X86_64)
	writeELF(t, filepath.Join(lib32, "libcurl.so.4"), elf.ELFCLASS32, elf.EM_386)

	if got := curlStandIn([]string{lib64}); got != "" {
		t.Errorf("curlStandIn took the 64-bit %s", got)
	}
	want := filepath.Join(lib32, "libcurl.so.4")
	if got := curlStandIn([]string{lib64, t.TempDir(), lib32}); got != want {
		t.Errorf("curlStandIn = %q, want %q", got, want)
	}
}

/*
fakeLdd puts an ldd first on PATH that answers like the real one: the gnutls
curl is missing until bin/ holds it, and once it does, refused says whether
the loader turns the stand-in down for its symbol versions.
*/
func fakeLdd(t *testing.T, refused bool) {
	t.Helper()
	dir := t.TempDir()
	verdict := `echo "	libcurl-gnutls.so.4 => $PWD/bin/libcurl-gnutls.so.4 (0xf7e00000)"`
	if refused {
		verdict = "echo \"$1: $PWD/bin/libcurl-gnutls.so.4: version \\`CURL_GNUTLS_3' not found (required by $1)\""
	}
	script := "#!/bin/sh\n" +
		"case \":$LD_LIBRARY_PATH:\" in *\":$PWD/bin:\"*) ;; *) echo \"bin/ is not on the library path\"; exit 1;; esac\n" +
		"echo \"$1:\"\n" +
		"if [ -e bin/libcurl-gnutls.so.4 ]; then " + verdict + "; else echo \"	libcurl-gnutls.so.4 => not found\"; fi\n"
	if err := os.WriteFile(filepath.Join(dir, "ldd"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func gameWithReplay(t *testing.T) string {
	t.Helper()
	gameDir := t.TempDir()
	writeELF(t, filepath.Join(gameDir, "bin", "replay_srv.so"), elf.ELFCLASS32, elf.EM_386)
	return gameDir
}

/*
The release launcher on Fedora 43 with glibc.i686 and libcurl.i686 crash-looped
at replay_srv.so: Fedora has no libcurl-gnutls.so.4, and its own unversioned
libcurl.so.4 is accepted under that name.
*/
func TestEnsureLibrariesLinksAnUnversionedCurl(t *testing.T) {
	fakeLdd(t, false)
	gameDir := gameWithReplay(t)
	lib32 := t.TempDir()
	standIn := filepath.Join(lib32, "libcurl.so.4")
	writeELF(t, standIn, elf.ELFCLASS32, elf.EM_386)

	if err := ensureLibraries(context.Background(), gameDir, []string{lib32}, t.Logf); err != nil {
		t.Fatalf("ensureLibraries = %v", err)
	}
	got, err := os.Readlink(filepath.Join(gameDir, "bin", CurlGnutls))
	if err != nil || got != standIn {
		t.Fatalf("bin/%s -> %q (%v), want %q", CurlGnutls, got, err, standIn)
	}
	if err := ensureLibraries(context.Background(), gameDir, []string{lib32}, t.Logf); err != nil {
		t.Fatalf("a second start refused its own link: %v", err)
	}
}

func TestEnsureLibrariesTakesBackARefusedCurl(t *testing.T) {
	fakeLdd(t, true)
	gameDir := gameWithReplay(t)
	lib32 := t.TempDir()
	writeELF(t, filepath.Join(lib32, "libcurl.so.4"), elf.ELFCLASS32, elf.EM_386)

	err := ensureLibraries(context.Background(), gameDir, []string{lib32}, t.Logf)
	if !errors.Is(err, ErrMissingLibrary) {
		t.Fatalf("ensureLibraries = %v, want ErrMissingLibrary", err)
	}
	if _, statErr := os.Lstat(filepath.Join(gameDir, "bin", CurlGnutls)); !os.IsNotExist(statErr) {
		t.Errorf("the refused link was left in bin/: %v", statErr)
	}
}

func TestEnsureLibrariesNamesThePackage(t *testing.T) {
	fakeLdd(t, false)
	gameDir := gameWithReplay(t)

	err := ensureLibraries(context.Background(), gameDir, []string{t.TempDir()}, t.Logf)
	if !errors.Is(err, ErrMissingLibrary) {
		t.Fatalf("ensureLibraries = %v, want ErrMissingLibrary", err)
	}
	for _, want := range []string{CurlGnutls, "libcurl.i686", "rpm-ostree", "lib32-libcurl-gnutls", "libcurl3t64-gnutls:i386"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not say %q: %v", want, err)
		}
	}
}
