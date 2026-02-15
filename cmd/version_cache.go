package cmd

import "sync"

// latestVerCache deduplicates concurrent getLatestVer calls for the same module path.
// When multiple goroutines request the latest version of the same module,
// only one network call is made; others wait and share the result.
type latestVerCache struct {
	mu      sync.Mutex
	entries map[string]*latestVerEntry
}

type latestVerEntry struct {
	once    sync.Once
	version string
	err     error
}

func newLatestVerCache() *latestVerCache {
	return &latestVerCache{entries: make(map[string]*latestVerEntry)}
}

// get returns the latest version for the given module path,
// calling getLatestVer at most once per unique module path.
func (c *latestVerCache) get(modulePath string) (string, error) {
	c.mu.Lock()
	entry, ok := c.entries[modulePath]
	if !ok {
		entry = &latestVerEntry{}
		c.entries[modulePath] = entry
	}
	c.mu.Unlock()

	entry.once.Do(func() {
		entry.version, entry.err = getLatestVer(modulePath)
	})
	return entry.version, entry.err
}
