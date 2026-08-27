// Command communitycheck verifies that gamedata/community.json and a TF2
// community-content tree contain the same maps, missions and upgrade tables.
package main

import (
	"fmt"
	"os"

	"github.com/m-this/tf2-archipelago/gamedata"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: communitycheck TF_ROOT_OR_ZIP [TF_ROOT_OR_ZIP ...]")
		os.Exit(2)
	}
	if err := gamedata.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := gamedata.ValidateCommunitySources(os.Args[1:]...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("community manifest and TF2 content tree agree")
}
