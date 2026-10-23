package terminal

import (
	"fmt"
	"sync"
	"testing"

	"github.com/unaquatic/fluid/style"
)

func TestLinkRegistry_New(t *testing.T) {
	lr := NewLinkRegistry()
	if lr.Count() != 0 {
		t.Errorf("expected 0 links initially, got %d", lr.Count())
	}
}

func TestLinkRegistry_Alloc(t *testing.T) {
	lr := NewLinkRegistry()

	id := lr.Alloc("https://example.com")
	if id != 1 {
		t.Errorf("expected first link ID=1, got %d", id)
	}
	if lr.Count() != 1 {
		t.Errorf("expected count=1, got %d", lr.Count())
	}
}

func TestLinkRegistry_AllocMultiple(t *testing.T) {
	lr := NewLinkRegistry()

	id1 := lr.Alloc("https://example.com")
	id2 := lr.Alloc("https://other.com")
	id3 := lr.Alloc("https://third.com")

	if id1 != 1 || id2 != 2 || id3 != 3 {
		t.Errorf("expected IDs 1,2,3, got %d,%d,%d", id1, id2, id3)
	}
	if lr.Count() != 3 {
		t.Errorf("expected count=3, got %d", lr.Count())
	}
}

func TestLinkRegistry_Dedup(t *testing.T) {
	lr := NewLinkRegistry()

	id1 := lr.Alloc("https://example.com")
	id2 := lr.Alloc("https://example.com")

	if id1 != id2 {
		t.Errorf("expected same ID for duplicate URL, got %d vs %d", id1, id2)
	}
	if lr.Count() != 1 {
		t.Errorf("expected count=1 after dedup, got %d", lr.Count())
	}
}

func TestLinkRegistry_GetURL(t *testing.T) {
	lr := NewLinkRegistry()

	lr.Alloc("https://example.com")
	lr.Alloc("https://other.com")

	got := lr.URL(1)
	if got != "https://example.com" {
		t.Errorf("expected 'https://example.com', got %q", got)
	}

	got = lr.URL(2)
	if got != "https://other.com" {
		t.Errorf("expected 'https://other.com', got %q", got)
	}
}

func TestLinkRegistry_GetURL_Invalid(t *testing.T) {
	lr := NewLinkRegistry()

	// ID 0 is reserved
	if lr.URL(0) != "" {
		t.Error("expected empty for ID 0")
	}
	// Out of range
	if lr.URL(99) != "" {
		t.Error("expected empty for out-of-range ID")
	}
}

func TestLinkRegistry_IntegrationWithAttributes(t *testing.T) {
	lr := NewLinkRegistry()

	linkID := lr.Alloc("https://example.com")
	attrs := style.AttributesWithLink(style.AttrBold|style.AttrUnderline, linkID)

	// Extract and verify
	gotBase := style.GetBaseAttributes(attrs)
	if gotBase != style.AttrBold|style.AttrUnderline {
		t.Errorf("expected base attrs Bold|Underline, got %d", gotBase)
	}

	gotLinkID := style.GetLinkID(attrs)
	if gotLinkID != linkID {
		t.Errorf("expected linkID %d, got %d", linkID, gotLinkID)
	}

	gotURL := lr.URL(gotLinkID)
	if gotURL != "https://example.com" {
		t.Errorf("expected 'https://example.com', got %q", gotURL)
	}
}

func TestLinkRegistry_ConcurrentAlloc(t *testing.T) {
	lr := NewLinkRegistry()

	var wg sync.WaitGroup
	const n = 100

	ids := make([]uint32, n)
	for i := range n {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			url := fmt.Sprintf("https://example.com/%d", idx)
			ids[idx] = lr.Alloc(url)
		}(i)
	}
	wg.Wait()

	if lr.Count() != n {
		t.Errorf("expected %d links, got %d", n, lr.Count())
	}

	// All IDs should be unique and non-zero
	seen := make(map[uint32]bool)
	for i, id := range ids {
		if id == 0 {
			t.Errorf("link %d has ID 0", i)
		}
		if seen[id] {
			t.Errorf("duplicate ID %d", id)
		}
		seen[id] = true
	}
}

func TestLinkRegistry_ConcurrentDedupAlloc(t *testing.T) {
	lr := NewLinkRegistry()

	var wg sync.WaitGroup
	const n = 50

	ids := make([]uint32, n)
	for i := range n {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			// All goroutines alloc the same URL
			ids[idx] = lr.Alloc("https://example.com")
		}(i)
	}
	wg.Wait()

	// All IDs should be the same
	for i := 1; i < n; i++ {
		if ids[i] != ids[0] {
			t.Errorf("expected all IDs to be %d, got %d at index %d", ids[0], ids[i], i)
		}
	}
	if lr.Count() != 1 {
		t.Errorf("expected count=1 after concurrent dedup, got %d", lr.Count())
	}
}
