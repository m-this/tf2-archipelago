// Command export writes the gamedata tables into the apworld's data directory.
// The output is committed; CI runs this and fails if the working tree differs.
//
// Usage: go run ./cmd/export ../apworld/tf2_mvm/data
package main

import (
	"fmt"
	"os"

	"git-ssh.croque.top/mathis/tf2-archipelago/gamedata"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: export <dir>")
		os.Exit(2)
	}
	if err := gamedata.Export(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "export:", err)
		os.Exit(1)
	}
}
