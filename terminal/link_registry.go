package terminal

import "sync"

// LinkRegistry maps numeric link IDs to URLs for OSC 8 hyperlink support.
// Link IDs are packed into the upper 24 bits of a cell's uint32 attribute
// word via AttributesWithLink. The renderer looks up URLs from this registry
// when flushing cells that carry a link ID, emitting the appropriate OSC 8
// escape sequences.
//
// The registry is safe for concurrent use.
type LinkRegistry struct {
	mu    sync.RWMutex
	urls  []string          // index 0 is reserved (linkID 0 = "no link")
	dedup map[string]uint32 // url → linkID (optional dedup)
}

// NewLinkRegistry creates a new empty LinkRegistry.
func NewLinkRegistry() *LinkRegistry {
	return &LinkRegistry{
		urls:  []string{""}, // slot 0 reserved
		dedup: make(map[string]uint32),
	}
}

// Alloc allocates a link ID for the given URL. If the same URL was previously
// allocated, the existing ID is returned (deduplication).
// Returns the link ID (>= 1).
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

// URL returns the URL for the given link ID, or "" if invalid.
func (lr *LinkRegistry) URL(linkID uint32) string {
	lr.mu.RLock()
	defer lr.mu.RUnlock()

	if linkID == 0 || int(linkID) >= len(lr.urls) {
		return ""
	}
	return lr.urls[linkID]
}

// Count returns the number of allocated links (excluding the reserved slot 0).
func (lr *LinkRegistry) Count() int {
	lr.mu.RLock()
	defer lr.mu.RUnlock()
	return len(lr.urls) - 1
}
