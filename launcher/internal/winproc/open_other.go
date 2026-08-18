//go:build !windows

package winproc

import (
	"context"
	"fmt"
	"os/exec"
)

// Open shows a file or a folder to the player, through the desktop's own
// handler.
func Open(path string) error {
	if err := exec.CommandContext(context.Background(), "xdg-open", path).Start(); err != nil {
		return fmt.Errorf("cannot open %s: %w", path, err)
	}
	return nil
}
