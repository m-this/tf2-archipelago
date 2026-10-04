package installer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// gzipMagic is the first two bytes of every gzip stream. Valve and
// AlliedModders both serve a zip for Windows and a tarball for Linux, and the
// URL already says which: reading the bytes rather than the name means one
// unpacker and no way for the two to disagree.
var gzipMagic = []byte{0x1f, 0x8b}

// unpackTo writes an archive into dir, whether it is a zip or a gzipped tar.
func unpackTo(data []byte, dir string) error {
	return unpackKeeping(data, dir, nil)
}

// unpackKeeping is unpackTo for an upgrade over an install already there: an
// entry for which keep answers true is left alone when its file exists. keep
// sees the entry's slash-separated name inside the archive.
func unpackKeeping(data []byte, dir string, keep func(name string) bool) error {
	if bytes.HasPrefix(data, gzipMagic) {
		return untarTo(data, dir, keep)
	}
	return unzipKeeping(data, dir, keep)
}

// kept says whether an upgrade leaves this entry's existing file in place.
func kept(keep func(name string) bool, name, target string) bool {
	if keep == nil || !keep(name) {
		return false
	}
	_, err := os.Lstat(target)
	return err == nil
}

// untarTo unpacks a .tar.gz into dir.
//
// Symlinks are kept: the SteamCMD tarball ships linux32/steamclient.so as one,
// and a copy of the target would be a second copy of a 100 MB library. Hard
// links become a copy, which nothing in these archives relies on.
func untarTo(data []byte, dir string, keep func(name string) bool) error {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("cannot read the archive: %w", err)
	}
	defer func() { _ = gz.Close() }()

	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("cannot read the archive: %w", err)
		}
		target, err := safeJoin(dir, header.Name)
		if err != nil {
			return err
		}
		if header.Typeflag != tar.TypeDir && kept(keep, header.Name, target) {
			continue
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			// An install run twice finds its own symlink already there.
			_ = os.Remove(target)
			if err := os.Symlink(header.Linkname, target); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := writeTarFile(reader, target, header.FileInfo().Mode()); err != nil {
				return err
			}
		}
	}
}

// writeTarFile writes one entry, keeping the mode: srcds_run and steamcmd.sh
// are useless without their execute bit.
//
// The parent directory is made here rather than trusted to a TypeDir entry
// ahead of it. Valve's SteamCMD tarball carries no entry for linux32/ and
// every file under it, so waiting for one loses the whole directory.
func writeTarFile(reader io.Reader, target string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	file, err := openForWrite(target, mode.Perm())
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	// The archives here are the game server's mods, tens of megabytes at most.
	if _, err := io.Copy(file, reader); err != nil { //nolint:gosec // trusted upstream archives
		return err
	}
	return file.Close()
}

// safeJoin refuses an entry that would land outside dir. An archive from
// AlliedModders never tries, and an archive that does is the one worth
// stopping.
func safeJoin(dir, name string) (string, error) {
	target := filepath.Join(dir, filepath.Clean("/"+name))
	if !strings.HasPrefix(target, filepath.Clean(dir)+string(os.PathSeparator)) {
		return "", fmt.Errorf("archive entry %q points outside the install directory", name)
	}
	return target, nil
}
