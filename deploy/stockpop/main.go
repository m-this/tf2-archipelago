// Command stockpop installs a repaired copy of Valve's _666 Ghost Town wave
// under Caliginous Caper's runtime alias. The source stays in TF2's VPK so
// the image does not ship Valve's mission.
package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
)

const (
	magic       = 0x55aa1234
	maxPopBytes = 1 << 20
	targetPath  = "scripts/population"
	targetName  = "mvm_ghost_town_666"
)

type vpkEntry struct {
	CRC      uint32
	Preload  uint16
	Archive  uint16
	Offset   uint32
	Length   uint32
	Sentinel uint16
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: stockpop <tf directory> <destination popfile>")
		os.Exit(2)
	}
	if err := install(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, "stockpop:", err)
		os.Exit(1)
	}
}

func install(tfDir, destination string) error {
	body, err := extract(tfDir)
	if err != nil {
		return err
	}
	body, err = repair(body)
	if err != nil {
		return err
	}
	if old, err := os.ReadFile(destination); err == nil && bytes.Equal(old, body) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(destination), ".caliginous-*.pop")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err = tmp.Write(body); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), destination)
}

func repair(body []byte) ([]byte, error) {
	// Ghost Town's BSP has boss_path_1, but none of the _666 file's four
	// boss_path_a1 nodes. A tank that cannot find its first path node cannot
	// start the later groups that wait for wave07 to die.
	if !bytes.Contains(body, []byte("boss_path_a1")) && !bytes.Contains(body, []byte("boss_path_1")) {
		return nil, errors.New("_666 mission has no recognized tank path")
	}
	body = bytes.ReplaceAll(body, []byte("boss_path_a1"), []byte("boss_path_1"))

	// Without the event setting, TF2 spawns the regular robot models.
	if !bytes.Contains(body, []byte("EventPopfile Halloween")) {
		marker := []byte("CanBotsAttackWhileInSpawnRoom no")
		at := bytes.Index(body, marker)
		if at < 0 {
			return nil, errors.New("_666 mission has no expected WaveSchedule header")
		}
		lineEnd := bytes.IndexByte(body[at+len(marker):], '\n')
		if lineEnd < 0 {
			return nil, errors.New("_666 mission has no header line ending")
		}
		insertAt := at + len(marker) + lineEnd + 1
		eventLine := []byte("\tEventPopfile Halloween\n")
		if insertAt > 0 && body[insertAt-1] == '\n' && insertAt > 1 && body[insertAt-2] == '\r' {
			eventLine = []byte("\tEventPopfile Halloween\r\n")
		}
		body = append(append(append([]byte(nil), body[:insertAt]...), eventLine...), body[insertAt:]...)
	}
	return body, nil
}

func extract(tfDir string) ([]byte, error) {
	index, err := os.ReadFile(filepath.Join(tfDir, "tf2_misc_dir.vpk"))
	if err != nil {
		return nil, err
	}
	if len(index) < 12 || binary.LittleEndian.Uint32(index) != magic {
		return nil, errors.New("invalid TF2 VPK index")
	}
	version := binary.LittleEndian.Uint32(index[4:])
	headerSize := 12
	if version == 2 {
		headerSize = 28
	} else if version != 1 {
		return nil, fmt.Errorf("unsupported TF2 VPK version %d", version)
	}
	treeSize := int(binary.LittleEndian.Uint32(index[8:]))
	if treeSize < 1 || headerSize+treeSize > len(index) {
		return nil, errors.New("invalid TF2 VPK tree size")
	}
	tree := bufio.NewReader(bytes.NewReader(index[headerSize : headerSize+treeSize]))
	for {
		ext, err := cstring(tree)
		if err != nil {
			return nil, err
		}
		if ext == "" {
			break
		}
		for {
			path, err := cstring(tree)
			if err != nil {
				return nil, err
			}
			if path == "" {
				break
			}
			for {
				name, err := cstring(tree)
				if err != nil {
					return nil, err
				}
				if name == "" {
					break
				}
				var entry vpkEntry
				if err := binary.Read(tree, binary.LittleEndian, &entry); err != nil {
					return nil, err
				}
				if entry.Sentinel != 0xffff {
					return nil, errors.New("invalid TF2 VPK entry terminator")
				}
				if ext == "pop" && path == targetPath && name == targetName {
					return entryBytes(tfDir, index, headerSize+treeSize, tree, entry)
				}
				if _, err := io.CopyN(io.Discard, tree, int64(entry.Preload)); err != nil {
					return nil, err
				}
			}
		}
	}
	return nil, errors.New("stock Caliginous Caper popfile missing from TF2 VPK")
}

func cstring(r *bufio.Reader) (string, error) {
	s, err := r.ReadString(0)
	if err != nil {
		return "", err
	}
	return s[:len(s)-1], nil
}

func entryBytes(tfDir string, index []byte, dataStart int, tree *bufio.Reader, entry vpkEntry) ([]byte, error) {
	if int64(entry.Preload)+int64(entry.Length) > maxPopBytes {
		return nil, errors.New("stock popfile exceeds size limit")
	}
	body := make([]byte, int(entry.Preload)+int(entry.Length))
	if _, err := io.ReadFull(tree, body[:entry.Preload]); err != nil {
		return nil, err
	}
	if entry.Archive == 0x7fff {
		start := int64(dataStart) + int64(entry.Offset)
		if start+int64(entry.Length) > int64(len(index)) {
			return nil, errors.New("stock popfile exceeds VPK index size")
		}
		copy(body[entry.Preload:], index[start:start+int64(entry.Length)])
	} else {
		archive := filepath.Join(tfDir, fmt.Sprintf("tf2_misc_%03d.vpk", entry.Archive))
		file, err := os.Open(archive)
		if err != nil {
			return nil, err
		}
		defer func() { _ = file.Close() }()
		if _, err := file.ReadAt(body[entry.Preload:], int64(entry.Offset)); err != nil {
			return nil, err
		}
	}
	if crc32.ChecksumIEEE(body) != entry.CRC {
		return nil, errors.New("stock popfile VPK checksum mismatch")
	}
	return body, nil
}
