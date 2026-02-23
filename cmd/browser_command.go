package cmd

import (
	"context"
	"errors"
	"os/exec"
	"time"
)

const openBrowserTimeout = 5 * time.Second

// runBrowserCommand executes browser launcher commands. This is swapped in tests.
var runBrowserCommand = func(command string, args ...string) error { //nolint:gochecknoglobals
	ctx, cancel := context.WithTimeout(context.Background(), openBrowserTimeout)
	defer cancel()

	// Command names are hard-coded in openBrowser and URL is internally generated.
	return browserCommandError(exec.CommandContext(ctx, command, args...).Run()) //nolint:gosec
}

func browserCommandError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		// open/xdg-open may keep running for a while after successfully launching.
		return nil
	}
	return err
}
