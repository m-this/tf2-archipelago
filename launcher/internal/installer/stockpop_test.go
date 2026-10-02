package installer

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/m-this/tf2-archipelago/launcher/internal/stockpop"
)

// writeStockVPK writes a tf2_misc_dir.vpk holding only Valve's _666 Ghost Town
// mission, stored inline after the tree, as TF2 ships small files.
func writeStockVPK(t *testing.T, modDir string) {
	t.Helper()
	body := []byte("WaveSchedule\n{\n\tCanBotsAttackWhileInSpawnRoom no\n\tWave { Tank { StartingPathTrackNode \"boss_path_a1\" } }\n}\n")
	var tree bytes.Buffer
	tree.WriteString("pop\x00scripts/population\x00mvm_ghost_town_666\x00")
	for _, field := range []any{crc32.ChecksumIEEE(body), uint16(0), uint16(0x7fff), uint32(0), uint32(len(body)), uint16(0xffff)} {
		if err := binary.Write(&tree, binary.LittleEndian, field); err != nil {
			t.Fatal(err)
		}
	}
	tree.Write([]byte{0, 0, 0})
	var index bytes.Buffer
	for _, field := range []uint32{0x55aa1234, 1, uint32(tree.Len())} {
		if err := binary.Write(&index, binary.LittleEndian, field); err != nil {
			t.Fatal(err)
		}
	}
	index.Write(tree.Bytes())
	index.Write(body)
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "tf2_misc_dir.vpk"), index.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func recordLog() (logf func(string, ...any), lines func() []string) {
	var got []string
	return func(format string, args ...any) { got = append(got, fmt.Sprintf(format, args...)) },
		func() []string { return got }
}

// The plugin refuses Caliginous Caper without this file, and only the srcds
// image used to write it: every launcher install refused the mission.
func TestUpdateGameWritesCaliginousCapersMission(t *testing.T) {
	tests := []struct {
		name     string
		outcomes []string
		wantErr  error
	}{
		{"updated", []string{"ok"}, nil},
		{"Steam out of reach, installed build starts", []string{"lost", "lost"}, ErrGameNotUpdated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, gameDir, _ := fakeSteamcmd(t, tt.outcomes...)
			modDir := filepath.Join(gameDir, "tf")
			writeStockVPK(t, modDir)

			err := UpdateGame(context.Background(), filepath.Dir(gameDir), discard)
			if !errors.Is(err, tt.wantErr) || (tt.wantErr == nil && err != nil) {
				t.Fatalf("UpdateGame = %v, want %v", err, tt.wantErr)
			}
			got, err := os.ReadFile(filepath.Join(modDir, "scripts", "population", stockpop.RuntimeName))
			if err != nil {
				t.Fatalf("the runtime mission is not in the mod dir: %v", err)
			}
			if !bytes.Contains(got, []byte("boss_path_1")) || !bytes.Contains(got, []byte("EventPopfile Halloween")) {
				t.Errorf("the runtime mission is not repaired:\n%s", got)
			}
		})
	}
}

func TestUpdateGameWithoutTheStockMissionWarnsAndStarts(t *testing.T) {
	_, gameDir, _ := fakeSteamcmd(t, "ok")
	logf, lines := recordLog()

	if err := UpdateGame(context.Background(), filepath.Dir(gameDir), logf); err != nil {
		t.Fatalf("a missing VPK failed the start: %v", err)
	}
	var warnings []string
	for _, line := range lines() {
		if strings.Contains(line, "Caliginous Caper") {
			warnings = append(warnings, line)
		}
	}
	if len(warnings) != 1 {
		t.Errorf("want one warning about Caliginous Caper, got %q", warnings)
	}
	if _, err := os.Stat(filepath.Join(gameDir, "tf", "scripts", "population", stockpop.RuntimeName)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a mission was written with no stock mission to make it from: %v", err)
	}
}
