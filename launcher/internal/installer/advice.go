package installer

import (
	"errors"
	"fmt"
)

// HostError is a download the remote site did not serve: it answered with
// something other than 200, or could not be reached at all. The URL is kept
// because it is what tells a player the failure is not on their machine.
type HostError struct {
	URL string
	// Status is the HTTP status the site answered, 0 when it never answered.
	Status int
	Err    error
}

func (e *HostError) Error() string {
	if e.Status != 0 {
		return fmt.Sprintf("%s answered %d", e.URL, e.Status)
	}
	return e.Err.Error()
}

func (e *HostError) Unwrap() error { return e.Err }

/*
Advice is what to tell a player after err stopped an install.

Repair throws away SteamCMD and the mods and downloads them again from the same
places, which helps with a broken install and does nothing for a site that is
down: alliedmods.net answered 404 for the Metamod tarball for an evening, and
Repair would have asked it again.
*/
func Advice(err error) string {
	if errors.Is(err, ErrMissingLibrary) {
		return "Repair cannot fix this: the library comes from the system, so install the package named above"
	}
	host, ok := errors.AsType[*HostError](err)
	if !ok {
		return RepairAdvice
	}
	if host.Status != 0 {
		return fmt.Sprintf("The site serving %s is failing (HTTP %d), not this install. "+
			"Nothing here can fix that: try again later", host.URL, host.Status)
	}
	return fmt.Sprintf("%s could not be reached: the site is down or this machine is offline, "+
		"and reinstalling cannot fix either. Try again later", host.URL)
}
