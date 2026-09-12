package installer

import (
	"fmt"
	"runtime"

	"github.com/m-this/tf2-archipelago/launcher/internal/assets"
)

/*
Where the launcher-managed downloads come from, per platform.

Valve and AlliedModders both publish one archive per platform, and they do not
agree on the format: SteamCMD is a zip on Windows and a tarball on Linux, and
the mod drops are a zip on Windows and a tarball on Linux too. unpackTo reads
whichever arrives, so nothing above this cares which it was.

The versions are the ones deploy/env/versions.env pins and -ldflags injects.
Only the platform suffix and the extension change here.
*/

// steamcmdURL is Valve's own bootstrap archive.
func steamcmdURL() string {
	if runtime.GOOS == "windows" {
		return "https://steamcdn-a.akamaihd.net/client/installer/steamcmd.zip"
	}
	return "https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz"
}

// metamodURL is the Metamod:Source drop that matches the pinned version.
func metamodURL() string {
	return fmt.Sprintf("https://mms.alliedmods.net/mmsdrop/%s/mmsource-%s-%s",
		assets.MetamodBranch, assets.MetamodVersion, platformArchive())
}

// sourcemodURL is the SourceMod drop that matches the pinned version.
func sourcemodURL() string {
	return fmt.Sprintf("https://sm.alliedmods.net/smdrop/%s/sourcemod-%s-%s",
		assets.SourcemodBranch, assets.SourcemodVersion, platformArchive())
}

// sigmodURL is upstream's verified Linux-only package. ServerMod's platform
// flags keep Windows from reaching this URL.
func sigmodURL() string {
	return fmt.Sprintf("https://github.com/rafradek/sigsegv-mvm/releases/download/%s/package-linux.zip",
		assets.SigsegvMVMVersion)
}

// platformArchive is the tail both AlliedModders drops share: the platform and
// the format it ships in.
func platformArchive() string {
	if runtime.GOOS == "windows" {
		return "windows.zip"
	}
	return "linux.tar.gz"
}

// steamcmdNames is what the bootstrap unpacks to, in the order to look. The
// Windows name comes first on Windows and the shell script first everywhere
// else, but both are checked: a directory left over from the other platform is
// a clearer error than "not found".
func steamcmdNames() []string {
	if runtime.GOOS == "windows" {
		return []string{"steamcmd.exe", "steamcmd.sh"}
	}
	return []string{"steamcmd.sh", "steamcmd.exe"}
}

// srcdsNames is the game server's own launcher, same idea.
func srcdsNames() []string {
	if runtime.GOOS == "windows" {
		return []string{"srcds.exe", "srcds_run"}
	}
	return []string{"srcds_run", "srcds.exe"}
}
