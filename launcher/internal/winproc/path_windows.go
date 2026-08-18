//go:build windows

package winproc

import "golang.org/x/sys/windows"

// ShortPath returns the 8.3 form of a path, or the path unchanged when Windows
// will not give one.
//
// SteamCMD parses its own command line and gets a path with a space or an
// accent in it wrong, which is one way to reach "Failed to install app
// '232250' (Missing configuration)": it ends up with an install directory that
// is not the one asked for. The short form has neither. The directory has to
// exist first; Windows derives the short name from the file system.
func ShortPath(path string) string {
	wide, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return path
	}
	size, err := windows.GetShortPathName(wide, nil, 0)
	if err != nil || size == 0 {
		return path
	}
	buf := make([]uint16, size)
	size, err = windows.GetShortPathName(wide, &buf[0], size)
	if err != nil || size == 0 {
		return path
	}
	return windows.UTF16ToString(buf[:size])
}
