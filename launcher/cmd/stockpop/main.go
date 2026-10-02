// Command stockpop writes Caliginous Caper's mission from the game's own
// files. The srcds image runs it as tf2ap-stockpop; the launcher calls the
// package on every start.
package main

import (
	"fmt"
	"os"

	"github.com/m-this/tf2-archipelago/launcher/internal/stockpop"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: stockpop <tf directory> <destination popfile>")
		os.Exit(2)
	}
	if _, err := stockpop.Install(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, "stockpop:", err)
		os.Exit(1)
	}
}
