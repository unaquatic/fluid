package terminal

import "sync"

// LinkRegistry assigns numeric IDs to URLs used by OSC 8 hyperlinks. Its methods
// are safe for concurrent use.
type LinkRegistry struct {
	mu    sync.RWMutex
	urls  []string          // Index zero is reserved to mean no link.
	dedup map[string]uint32 // URL to existing link ID.
}

// NewLinkRegistry returns an empty link registry.
func NewLinkRegistry() *LinkRegistry {
	return &LinkRegistry{
		urls:  []string{""}, // Reserve ID zero.
		dedup: make(map[string]uint32),
	}
}

// Alloc returns a positive ID for url, reusing its existing ID when present.
func (lr *LinkRegistry) Alloc(url string) uint32 {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	if id, ok := lr.dedup[url]; ok {
		return id
	}

	id := uint32(len(lr.urls))
	lr.urls = append(lr.urls, url)
	lr.dedup[url] = id
	return id
}

// URL returns the URL identified by linkID, or an empty string for an invalid
// or reserved ID.
func (lr *LinkRegistry) URL(linkID uint32) string {
	lr.mu.RLock()
	defer lr.mu.RUnlock()

	if linkID == 0 || int(linkID) >= len(lr.urls) {
		return ""
	}
	return lr.urls[linkID]
}

// Count returns the number of allocated URLs.
func (lr *LinkRegistry) Count() int {
	lr.mu.RLock()
	defer lr.mu.RUnlock()
	return len(lr.urls) - 1
}
