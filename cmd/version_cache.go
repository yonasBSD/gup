package cmd

import (
	"context"
	"errors"
	"sync"
)

// latestVerCache deduplicates concurrent getLatestVer calls for the same module path.
// When multiple goroutines request the latest version of the same module,
// only one network call is made; others wait and share the result.
type latestVerCache struct {
	mu      sync.Mutex
	entries map[string]*latestVerEntry
}

type latestVerEntry struct {
	mu       sync.Mutex
	fetching bool
	waitCh   chan struct{}
	done     bool
	version  string
	err      error
}

func newLatestVerCache() *latestVerCache {
	return &latestVerCache{entries: make(map[string]*latestVerEntry)}
}

// get returns the latest version for the given module path,
// calling getLatestVer at most once per unique module path.
func (c *latestVerCache) get(ctx context.Context, modulePath string) (string, error) {
	c.mu.Lock()
	entry, ok := c.entries[modulePath]
	if !ok {
		entry = &latestVerEntry{}
		c.entries[modulePath] = entry
	}
	c.mu.Unlock()

	for {
		entry.mu.Lock()
		if entry.done {
			version, err := entry.version, entry.err
			entry.mu.Unlock()
			return version, err
		}

		if entry.fetching {
			waitCh := entry.waitCh
			entry.mu.Unlock()
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-waitCh:
				continue
			}
		}

		entry.fetching = true
		entry.waitCh = make(chan struct{})
		entry.mu.Unlock()

		version, err := getLatestVerCtx(ctx, modulePath)

		entry.mu.Lock()
		entry.fetching = false
		waitCh := entry.waitCh
		entry.waitCh = nil

		// Do not cache context-related failures; allow a fresh retry.
		switch {
		case err == nil:
			entry.version = version
			entry.err = nil
			entry.done = true
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			entry.version = ""
			entry.err = nil
			entry.done = false
		default:
			entry.version = ""
			entry.err = err
			entry.done = true
		}
		entry.mu.Unlock()
		close(waitCh)

		return version, err
	}
}
