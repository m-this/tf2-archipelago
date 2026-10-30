// Command communityfiles installs existing, validated community packs into a
// disposable wave fixture using the launcher's normal archive installer.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/m-this/tf2-archipelago/launcher/internal/installer"
	"github.com/m-this/tf2-archipelago/launcher/internal/settings"
)

func main() {
	dir := flag.String("archives", "", "directory containing both supported full community ZIPs")
	target := flag.String("tf", "", "disposable TF2 content directory")
	check := flag.Bool("check", false, "validate archives without extracting")
	mods := flag.String("mods", "sigsegv-mvm", "installed server mods, comma separated")
	flag.Parse()
	if *dir == "" || (!*check && *target == "") {
		fmt.Fprintln(os.Stderr, "communityfiles: -archives and either -tf or -check are required")
		os.Exit(2)
	}
	logf := func(format string, args ...any) { fmt.Fprintf(os.Stderr, format+"\n", args...) }
	archives := settings.KnownCommunityArchives(*dir)
	var err error
	if *check {
		err = installer.ValidateCommunityArchives(archives, logf)
	} else {
		err = installer.InstallCommunityArchives(archives, *target, strings.Split(*mods, ","), logf)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "communityfiles:", err)
		os.Exit(1)
	}
}
