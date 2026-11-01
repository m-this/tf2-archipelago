// Package sigmodload recognises the line SigMod's Windows extension prints
// when it refuses to load, and says what it means to a player.
//
// The extension is the only judge of whether it loads: it compares the PE
// headers of the binaries in memory with the address table it shipped with.
// The launcher does not repeat that test; it reads the verdict from the
// server's own output. Two wordings exist, because players run either build:
// a table without Keys refuses on ServerVersion alone, one with Keys names the
// DLL whose TimeDateStamp or SizeOfImage differs.
package sigmodload

import (
	"fmt"
	"regexp"
	"strings"
)

// Key is the catalog key of the mod whose load this package reports on.
const Key = "sigsegv-mvm"

// Refusal is one "SigMod did not load" verdict.
type Refusal struct {
	// Lib is the DLL the extension named, without ".dll". Empty for the older
	// wording, which names no DLL.
	Lib string
	// Table and Server are the two ServerVersions of the older wording.
	Table, Server string
}

var (
	// With Keys: "<lib>.dll is TimeDateStamp … the address table for ServerVersion …",
	// and the neighbours that also name a DLL.
	named = []*regexp.Regexp{
		regexp.MustCompile(`(\w+)\.dll is TimeDateStamp .*the address table for ServerVersion`),
		regexp.MustCompile(`the address table has addresses in (\w+)\.dll`),
		regexp.MustCompile(`the address table's identity of (\w+)\.dll`),
		regexp.MustCompile(`(\w+)\.dll at \S+ has no (?:MZ|PE32) header`),
	}
	// Without Keys: "this SigMod build is for TF2 ServerVersion A, and the server is B".
	versionOnly = regexp.MustCompile(`this SigMod build is for TF2 ServerVersion (\d+), and the server is (\d+)`)
)

// Parse reports whether line is the extension refusing to load.
func Parse(line string) (Refusal, bool) {
	for _, pattern := range named {
		if match := pattern.FindStringSubmatch(line); match != nil {
			return Refusal{Lib: strings.ToLower(match[1])}, true
		}
	}
	if match := versionOnly.FindStringSubmatch(line); match != nil {
		return Refusal{Table: match[1], Server: match[2]}, true
	}
	return Refusal{}, false
}

// Message is what the launcher shows: which part does not fit, and what to do.
func (r Refusal) Message() string {
	const fix = "Update the launcher: a release with a new SigMod build fixes it. Until then SigMod missions are left out of new seeds."
	if r.Lib != "" {
		return fmt.Sprintf("SigMod did not load: %s.dll in this TF2 install is not the one SigMod's address table was made from, so TF2 was updated. %s", r.Lib, fix)
	}
	return fmt.Sprintf("SigMod did not load: this SigMod build is for TF2 build %s and the server is on %s. %s", r.Table, r.Server, fix)
}
