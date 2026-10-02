package sigmodpatch

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// SigMod passes an empty phrase name to the final %t while drawing an
// unselected extra loadout item. SourceMod then returns the entire format
// string literally. The Windows package can be corrected in place by
// changing that one placeholder to %s; both strings have the same length.
// The Linux package is built from m-this/sigsegv-mvm-win's master, which
// carries the fix in source, so it arrives already corrected.
var (
	sigmodLoadoutMenuBroken = []byte("%t: %s %s %t\x00")
	sigmodLoadoutMenuFixed  = []byte("%t: %s %s %s\x00")
)

// ExtensionPaths lists the native files that must carry the patch.
func ExtensionPaths(goos string) []string {
	base := "addons/sourcemod/extensions/"
	if goos == "windows" {
		return []string{base + "sigsegv.ext.2.tf2.dll"}
	}
	return []string{base + "sigsegv.ext.2.tf2.so", base + "x64/sigsegv.ext.2.tf2.so"}
}

// Ready reports whether all installed extensions carry this menu correction.
func Ready(modDir, goos string) bool {
	for _, relative := range ExtensionPaths(goos) {
		body, err := os.ReadFile(filepath.Join(modDir, filepath.FromSlash(relative)))
		if err != nil || bytes.Count(body, sigmodLoadoutMenuFixed) != 1 || bytes.Contains(body, sigmodLoadoutMenuBroken) {
			return false
		}
	}
	return true
}

// Patch updates the SHA-verified SigMod package before its install stamp is written.
func Patch(modDir, goos string) error {
	for _, relative := range ExtensionPaths(goos) {
		path := filepath.Join(modDir, filepath.FromSlash(relative))
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		switch {
		case bytes.Count(body, sigmodLoadoutMenuBroken) == 1 && !bytes.Contains(body, sigmodLoadoutMenuFixed):
		case bytes.Count(body, sigmodLoadoutMenuFixed) == 1 && !bytes.Contains(body, sigmodLoadoutMenuBroken):
			continue
		default:
			return fmt.Errorf("SigMod extra loadout menu marker is missing or ambiguous in %s", relative)
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		fixed := bytes.Replace(body, sigmodLoadoutMenuBroken, sigmodLoadoutMenuFixed, 1)
		if err := os.WriteFile(path, fixed, info.Mode().Perm()); err != nil {
			return fmt.Errorf("cannot patch SigMod extra loadout menu in %s: %w", relative, err)
		}
	}
	return nil
}
