package cmd

import (
	"fmt"

	"github.com/nao1215/gup/internal/goutil"
)

func ensureGoCommandAvailable() error {
	if err := goutil.CanUseGoCmd(); err != nil {
		return fmt.Errorf("%s: %w", "you didn't install golang", err)
	}
	return nil
}

func clampJobs(cpus int) int {
	if cpus < 1 {
		return 1
	}
	return cpus
}
