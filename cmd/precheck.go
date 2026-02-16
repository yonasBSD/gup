package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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

func newSignalCancelContext() (context.Context, context.CancelFunc, chan os.Signal) {
	ctx, cancel := context.WithCancel(context.Background())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP,
		syscall.SIGQUIT, syscall.SIGABRT)
	go catchSignal(signals, cancel)
	return ctx, cancel, signals
}

func stopSignalCancelContext(cancel context.CancelFunc, signals chan os.Signal) {
	signal.Stop(signals)
	close(signals)
	cancel()
}

func catchSignal(c <-chan os.Signal, cancel context.CancelFunc) {
	if _, ok := <-c; ok {
		cancel()
	}
}
