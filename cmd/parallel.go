package cmd

import (
	"context"

	"github.com/nao1215/gup/internal/goutil"
	"golang.org/x/sync/semaphore"
)

// forEachPackage runs fn for each package concurrently, limiting parallelism
// to cpus goroutines via a weighted semaphore. It returns a channel that
// receives exactly len(pkgs) results.
func forEachPackage(ctx context.Context, pkgs []goutil.Package, cpus int, fn func(context.Context, goutil.Package) updateResult) <-chan updateResult {
	ch := make(chan updateResult, len(pkgs))
	sem := semaphore.NewWeighted(int64(cpus))

	for _, p := range pkgs {
		go func(p goutil.Package) {
			if err := sem.Acquire(ctx, 1); err != nil {
				ch <- updateResult{pkg: p, err: err}
				return
			}
			defer sem.Release(1)
			ch <- fn(ctx, p)
		}(p)
	}

	return ch
}
