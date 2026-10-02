package stockpop

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	stockLF      = "WaveSchedule\n{\n\tCanBotsAttackWhileInSpawnRoom no\n\tWave { Tank { StartingPathTrackNode \"boss_path_a1\" } }\n}\n"
	repairedLF   = "WaveSchedule\n{\n\tCanBotsAttackWhileInSpawnRoom no\n\tEventPopfile Halloween\n\tWave { Tank { StartingPathTrackNode \"boss_path_1\" } }\n}\n"
	stockCRLF    = "WaveSchedule\r\n{\r\n\tCanBotsAttackWhileInSpawnRoom no\r\n\tWave { Tank { StartingPathTrackNode \"boss_path_a1\" } }\r\n}"
	repairedCRLF = "WaveSchedule\r\n{\r\n\tCanBotsAttackWhileInSpawnRoom no\r\n\tEventPopfile Halloween\r\n\tWave { Tank { StartingPathTrackNode \"boss_path_1\" } }\r\n}"
)

// vpkFile is one file in a synthetic VPK. Archive 0x7fff stores the data after
// the tree in the index itself; any other number stores it at Offset in
// tf2_misc_<archive>.vpk.
type vpkFile struct {
	ext, path, name string
	body            []byte
	preload         int
	archive         uint16
	offset          uint32
}

// writeVPK writes tf2_misc_dir.vpk, and an archive per non-inline file, into
// tfDir, in the layout extract reads.
func writeVPK(t *testing.T, tfDir string, version uint32, files ...vpkFile) {
	t.Helper()
	var tree, inline bytes.Buffer
	for _, f := range files {
		tree.WriteString(f.ext + "\x00" + f.path + "\x00" + f.name + "\x00")
		rest := f.body[f.preload:]
		e := entry{
			CRC: crc32.ChecksumIEEE(f.body), Preload: uint16(f.preload), Archive: f.archive,
			Length: uint32(len(rest)), Sentinel: 0xffff,
		}
		if f.archive == inlineArchive {
			e.Offset = uint32(inline.Len())
			inline.Write(rest)
		} else {
			e.Offset = f.offset
			archive := append(make([]byte, f.offset), rest...)
			name := filepath.Join(tfDir, fmt.Sprintf("tf2_misc_%03d.vpk", f.archive))
			if err := os.WriteFile(name, archive, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if err := binary.Write(&tree, binary.LittleEndian, e); err != nil {
			t.Fatal(err)
		}
		tree.Write(f.body[:f.preload])
		tree.Write([]byte{0, 0})
	}
	tree.WriteByte(0)

	header := []uint32{vpkMagic, version, uint32(tree.Len())}
	if version == 2 {
		header = append(header, uint32(inline.Len()), 0, 0, 0)
	}
	var index bytes.Buffer
	if err := binary.Write(&index, binary.LittleEndian, header); err != nil {
		t.Fatal(err)
	}
	index.Write(tree.Bytes())
	index.Write(inline.Bytes())
	if err := os.MkdirAll(tfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tfDir, "tf2_misc_dir.vpk"), index.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func stockFile(body string) vpkFile {
	return vpkFile{ext: "pop", path: "scripts/population", name: "mvm_ghost_town_666", body: []byte(body), archive: inlineArchive}
}

func TestInstallWritesTheRepairedMission(t *testing.T) {
	t.Parallel()
	decoy := vpkFile{ext: "pop", path: "scripts/population", name: "mvm_ghost_town", body: []byte("decoy"), preload: 2, archive: inlineArchive}
	other := vpkFile{ext: "txt", path: "scripts", name: "readme", body: []byte("other"), archive: inlineArchive}

	tests := []struct {
		name    string
		version uint32
		file    vpkFile
		want    string
	}{
		{"v1 inline", 1, stockFile(stockLF), repairedLF},
		{"v2 inline with preload", 2, func() vpkFile { f := stockFile(stockLF); f.preload = 7; return f }(), repairedLF},
		{"v2 archive CRLF", 2, func() vpkFile { f := stockFile(stockCRLF); f.archive, f.offset = 22, 3; return f }(), repairedCRLF},
		{"already repaired", 2, stockFile(repairedLF), repairedLF},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tfDir := t.TempDir()
			writeVPK(t, tfDir, tt.version, other, decoy, tt.file)
			destination := Destination(tfDir)

			changed, err := Install(tfDir, destination)
			if err != nil || !changed {
				t.Fatalf("Install = %v, %v; want true, nil", changed, err)
			}
			got, err := os.ReadFile(destination)
			if err != nil || string(got) != tt.want {
				t.Fatalf("installed mission = %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}

func TestInstallLeavesAnIdenticalMissionAlone(t *testing.T) {
	t.Parallel()
	tfDir := t.TempDir()
	writeVPK(t, tfDir, 2, stockFile(stockLF))
	destination := Destination(tfDir)
	if _, err := Install(tfDir, destination); err != nil {
		t.Fatal(err)
	}
	old := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(destination, old, old); err != nil {
		t.Fatal(err)
	}

	changed, err := Install(tfDir, destination)
	if err != nil || changed {
		t.Fatalf("second Install = %v, %v; want false, nil", changed, err)
	}
	info, err := os.Stat(destination)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(old) {
		t.Errorf("an unchanged mission was rewritten: mtime %v", info.ModTime())
	}
	leftovers, err := filepath.Glob(filepath.Join(filepath.Dir(destination), ".caliginous-*"))
	if err != nil || len(leftovers) != 0 {
		t.Errorf("temporary files left behind: %v, %v", leftovers, err)
	}
}

func TestInstallRefusesBadSources(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func(t *testing.T, tfDir string)
		is    error
	}{
		{"no VPK", func(*testing.T, string) {}, fs.ErrNotExist},
		{"no stock mission", func(t *testing.T, tfDir string) {
			t.Helper()
			f := stockFile(stockLF)
			f.name = "mvm_ghost_town"
			writeVPK(t, tfDir, 2, f)
		}, ErrNotInVPK},
		{"missing archive", func(t *testing.T, tfDir string) {
			t.Helper()
			f := stockFile(stockLF)
			f.archive = 4
			writeVPK(t, tfDir, 2, f)
			if err := os.Remove(filepath.Join(tfDir, "tf2_misc_004.vpk")); err != nil {
				t.Fatal(err)
			}
		}, fs.ErrNotExist},
		{"unsupported version", func(t *testing.T, tfDir string) {
			t.Helper()
			writeVPK(t, tfDir, 3, stockFile(stockLF))
		}, nil},
		{"not a VPK", func(t *testing.T, tfDir string) {
			t.Helper()
			if err := os.WriteFile(filepath.Join(tfDir, "tf2_misc_dir.vpk"), []byte("hello, world"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, nil},
		{"truncated tree", func(t *testing.T, tfDir string) {
			t.Helper()
			writeVPK(t, tfDir, 1, stockFile(stockLF))
			path := filepath.Join(tfDir, "tf2_misc_dir.vpk")
			index, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			binary.LittleEndian.PutUint32(index[8:], 20)
			if err := os.WriteFile(path, index, 0o644); err != nil {
				t.Fatal(err)
			}
		}, nil},
		{"unknown tank path", func(t *testing.T, tfDir string) {
			t.Helper()
			writeVPK(t, tfDir, 2, stockFile("WaveSchedule { CanBotsAttackWhileInSpawnRoom no\n Tank { StartingPathTrackNode elsewhere } }"))
		}, nil},
		{"no header to put the event after", func(t *testing.T, tfDir string) {
			t.Helper()
			writeVPK(t, tfDir, 2, stockFile("WaveSchedule { Tank { StartingPathTrackNode boss_path_a1 } }"))
		}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tfDir := t.TempDir()
			tt.setup(t, tfDir)
			destination := Destination(tfDir)

			changed, err := Install(tfDir, destination)
			if err == nil || changed {
				t.Fatalf("Install = %v, %v; want an error", changed, err)
			}
			if tt.is != nil && !errors.Is(err, tt.is) {
				t.Errorf("Install error %v is not %v", err, tt.is)
			}
			if _, err := os.Stat(destination); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("a refused source wrote the mission: %v", err)
			}
		})
	}
}

func TestInstallKeepsTheGoodMissionWhenTheChecksumFails(t *testing.T) {
	t.Parallel()
	tfDir := t.TempDir()
	f := stockFile(stockCRLF)
	f.archive, f.offset = 22, 3
	writeVPK(t, tfDir, 2, f)
	destination := Destination(tfDir)
	if _, err := Install(tfDir, destination); err != nil {
		t.Fatal(err)
	}

	archive := filepath.Join(tfDir, "tf2_misc_022.vpk")
	body, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	body[len(body)-1] ^= 1
	if err := os.WriteFile(archive, body, 0o644); err != nil {
		t.Fatal(err)
	}

	if changed, err := Install(tfDir, destination); err == nil || changed {
		t.Fatalf("corrupted VPK content passed its checksum: %v, %v", changed, err)
	}
	got, err := os.ReadFile(destination)
	if err != nil || string(got) != repairedCRLF {
		t.Fatalf("checksum failure replaced a good mission: %q, %v", got, err)
	}
}

func TestDestinationIsTheRuntimeName(t *testing.T) {
	t.Parallel()
	got := Destination(filepath.Join("srv", "tf"))
	want := filepath.Join("srv", "tf", "scripts", "population", "mvm_ghost_town_ap_caliginous_caper.pop")
	if got != want {
		t.Errorf("Destination = %q, want %q", got, want)
	}
}
