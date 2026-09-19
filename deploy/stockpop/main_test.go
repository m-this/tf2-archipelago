package main

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallExtractsAndChecksStockPopfile(t *testing.T) {
	dir := t.TempDir()
	content := []byte("WaveSchedule\r\n{\r\n\tCanBotsAttackWhileInSpawnRoom no\r\n\tWave { Tank { StartingPathTrackNode \"boss_path_a1\" } }\r\n}")
	var tree bytes.Buffer
	tree.WriteString("pop\x00" + targetPath + "\x00" + targetName + "\x00")
	entry := vpkEntry{
		CRC: crc32.ChecksumIEEE(content), Archive: 22, Offset: 3,
		Length: uint32(len(content)), Sentinel: 0xffff,
	}
	if err := binary.Write(&tree, binary.LittleEndian, entry); err != nil {
		t.Fatal(err)
	}
	tree.Write([]byte{0, 0, 0}) // end of filename, path, and extension lists
	var index bytes.Buffer
	for _, value := range []uint32{magic, 2, uint32(tree.Len()), 0, 0, 0, 0} {
		if err := binary.Write(&index, binary.LittleEndian, value); err != nil {
			t.Fatal(err)
		}
	}
	index.Write(tree.Bytes())
	if err := os.WriteFile(filepath.Join(dir, "tf2_misc_dir.vpk"), index.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	archive := append([]byte{0, 0, 0}, content...)
	if err := os.WriteFile(filepath.Join(dir, "tf2_misc_022.vpk"), archive, 0o644); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(dir, "scripts", "population", "mvm_ghost_town_ap_caliginous_caper.pop")
	if err := install(dir, destination); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(destination)
	want := []byte("WaveSchedule\r\n{\r\n\tCanBotsAttackWhileInSpawnRoom no\r\n\tEventPopfile Halloween\r\n\tWave { Tank { StartingPathTrackNode \"boss_path_1\" } }\r\n}")
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("installed popfile = %q, %v", got, err)
	}
	archive[len(archive)-1] ^= 1
	if err := os.WriteFile(filepath.Join(dir, "tf2_misc_022.vpk"), archive, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := install(dir, destination); err == nil {
		t.Fatal("corrupted VPK content passed its checksum")
	}
	got, err = os.ReadFile(destination)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("checksum failure replaced a good popfile: %q, %v", got, err)
	}
}

func TestRepairRequiresRecognizedTankPath(t *testing.T) {
	if _, err := repair([]byte("WaveSchedule { Tank { StartingPathTrackNode elsewhere } }")); err == nil {
		t.Fatal("unrecognized tank path was accepted")
	}
}
