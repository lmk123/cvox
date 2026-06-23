package cli

import (
	"os"

	"github.com/lmk123/cvox/internal/notify"
)

// Notify processes a single Claude Code hook event from stdin and fires the
// configured voice and/or desktop notification.
func Notify(args []string) error {
	// stdinIsTTY: if stdin is a terminal, short-circuit to a no-op to avoid
	// hanging when someone manually runs `cvox notify` in a shell.
	var stdinIsTTY bool
	fi, err := os.Stdin.Stat()
	if err == nil {
		stdinIsTTY = fi.Mode()&os.ModeCharDevice != 0
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return notify.Run(os.Stdin, cwd, stdinIsTTY)
}
