package stockpop

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

// ErrNotInVPK is a readable VPK without the stock mission in it.
var ErrNotInVPK = errors.New("stockpop: mvm_ghost_town_666.pop is not in tf2_misc_dir.vpk")

const (
	vpkMagic      = 0x55aa1234
	inlineArchive = 0x7fff
	maxPopBytes   = 1 << 20
	stockExt      = "pop"
	stockPath     = "scripts/population"
	stockName     = "mvm_ghost_town_666"
)

// entry is a VPK directory entry as it sits in the tree, after the file name.
type entry struct {
	CRC      uint32
	Preload  uint16
	Archive  uint16
	Offset   uint32
	Length   uint32
	Sentinel uint16
}

// extract returns the stock mission's bytes. It reads the directory index and
// one entry, and checks the entry's CRC.
func extract(tfDir string) ([]byte, error) {
	indexPath := filepath.Join(tfDir, "tf2_misc_dir.vpk")
	index, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("reading the TF2 VPK index: %w", err)
	}
	if len(index) < 12 || binary.LittleEndian.Uint32(index) != vpkMagic {
		return nil, fmt.Errorf("%s is not a VPK index", indexPath)
	}
	var headerSize int
	switch version := binary.LittleEndian.Uint32(index[4:]); version {
	case 1:
		headerSize = 12
	case 2:
		headerSize = 28
	default:
		return nil, fmt.Errorf("%s has unsupported VPK version %d", indexPath, version)
	}
	treeSize := int(binary.LittleEndian.Uint32(index[8:]))
	if treeSize < 1 || headerSize+treeSize > len(index) {
		return nil, fmt.Errorf("%s has an invalid tree size", indexPath)
	}
	tree := bufio.NewReader(bytes.NewReader(index[headerSize : headerSize+treeSize]))
	e, found, err := find(tree)
	if err != nil {
		return nil, fmt.Errorf("reading the tree of %s: %w", indexPath, err)
	}
	if !found {
		return nil, ErrNotInVPK
	}
	return entryBytes(tfDir, index[headerSize+treeSize:], tree, e)
}

// find walks the tree to the stock mission's entry and leaves tree at its
// preload bytes. The tree is extensions, then paths, then file names, each
// list ended by an empty string.
func find(tree *bufio.Reader) (entry, bool, error) {
	for {
		ext, err := cstring(tree)
		if err != nil || ext == "" {
			return entry{}, false, err
		}
		for {
			path, err := cstring(tree)
			if err != nil {
				return entry{}, false, err
			}
			if path == "" {
				break
			}
			for {
				name, err := cstring(tree)
				if err != nil {
					return entry{}, false, err
				}
				if name == "" {
					break
				}
				var e entry
				if err := binary.Read(tree, binary.LittleEndian, &e); err != nil {
					return entry{}, false, err
				}
				if e.Sentinel != 0xffff {
					return entry{}, false, errors.New("invalid entry terminator")
				}
				if ext == stockExt && path == stockPath && name == stockName {
					return e, true, nil
				}
				if _, err := tree.Discard(int(e.Preload)); err != nil {
					return entry{}, false, err
				}
			}
		}
	}
}

func cstring(r *bufio.Reader) (string, error) {
	s, err := r.ReadString(0)
	if err != nil {
		return "", err
	}
	return s[:len(s)-1], nil
}

func entryBytes(tfDir string, inline []byte, tree io.Reader, e entry) ([]byte, error) {
	if int64(e.Preload)+int64(e.Length) > maxPopBytes {
		return nil, errors.New("the stock mission exceeds its size limit")
	}
	body := make([]byte, int(e.Preload)+int(e.Length))
	if _, err := io.ReadFull(tree, body[:e.Preload]); err != nil {
		return nil, fmt.Errorf("reading the stock mission's preload: %w", err)
	}
	rest := body[e.Preload:]
	if e.Archive == inlineArchive {
		end := int64(e.Offset) + int64(e.Length)
		if end > int64(len(inline)) {
			return nil, errors.New("the stock mission runs past the end of the VPK index")
		}
		copy(rest, inline[e.Offset:end])
	} else if err := readArchive(filepath.Join(tfDir, fmt.Sprintf("tf2_misc_%03d.vpk", e.Archive)), rest, e.Offset); err != nil {
		return nil, err
	}
	if crc32.ChecksumIEEE(body) != e.CRC {
		return nil, errors.New("the stock mission fails its VPK checksum")
	}
	return body, nil
}

func readArchive(path string, into []byte, offset uint32) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening the VPK archive: %w", err)
	}
	defer func() { _ = file.Close() }() // read-only
	if _, err := file.ReadAt(into, int64(offset)); err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	return nil
}
