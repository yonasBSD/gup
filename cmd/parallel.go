package cmd

import (
	"context"
	"sync"

	"github.com/nao1215/gup/internal/goutil"
)

// forEachPackage runs fn for each package with a fixed-size worker pool.
// It returns a channel that receives exactly len(pkgs) results.
func forEachPackage(ctx context.Context, pkgs []goutil.Package, cpus int, fn func(context.Context, goutil.Package) updateResult) <-chan updateResult {
	ch := make(chan updateResult, len(pkgs))

	if len(pkgs) == 0 {
		close(ch)
		return ch
	}
	if cpus < 1 {
		cpus = 1
	}
	if cpus > len(pkgs) {
		cpus = len(pkgs)
	}

	jobs := make(chan goutil.Package)
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()

		for p := range jobs {
			select {
			case <-ctx.Done():
				ch <- updateResult{pkg: p, err: ctx.Err()}
			default:
				ch <- fn(ctx, p)
			}
		}
	}

	wg.Add(cpus)
	for i := 0; i < cpus; i++ {
		go worker()
	}

	go func() {
		defer close(jobs)

		for _, p := range pkgs {
			select {
			case <-ctx.Done():
				ch <- updateResult{pkg: p, err: ctx.Err()}
			case jobs <- p:
			}
		}
	}()

	go func() {
		wg.Wait()
		close(ch)
	}()

	return ch
}
