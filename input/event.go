package input

import (
	"sync"
)

// ---------------------------------------------------------------------------
// Event interface
// ---------------------------------------------------------------------------

// Event is the common interface for all event types.
type Event interface {
	StopPropagation()
	PreventDefault()
	PropagationStopped() bool
	DefaultPrevented() bool
}

// Compile-time interface checks.
var (
	_ Event = (*KeyEvent)(nil)
	_ Event = (*MouseEvent)(nil)
	_ Event = (*PasteEvent)(nil)
)

// ---------------------------------------------------------------------------
// EventType
// ---------------------------------------------------------------------------

// EventType tags event categories for the bus.
type EventType int

const (
	EventKeyboard EventType = iota
	EventMouse
	EventPaste
	EventResize
)

// ---------------------------------------------------------------------------
// ResizeEvent
// ---------------------------------------------------------------------------

// ResizeEvent signals a terminal size change.
type ResizeEvent struct {
	Width  int
	Height int
}

// ---------------------------------------------------------------------------
// TerminalEvent
// ---------------------------------------------------------------------------

// TerminalEvent wraps any event type with its EventType tag.
// Used for channel-based delivery via the EventBus.
type TerminalEvent struct {
	Type   EventType
	Key    *KeyEvent
	Mouse  *MouseEvent
	Paste  *PasteEvent
	Resize *ResizeEvent
}

// ---------------------------------------------------------------------------
// HandlerID
// ---------------------------------------------------------------------------

// HandlerID uniquely identifies a subscription registered via On or Once.
// Pass it to Off to unsubscribe. Zero is never a valid ID.
type HandlerID uint64

// ---------------------------------------------------------------------------
// EventBus
// ---------------------------------------------------------------------------

// handlerEntry pairs a unique ID with its callback.
type handlerEntry struct {
	id uint64
	fn func(TerminalEvent)
}

// EventBus provides goroutine-safe pub/sub for events.
// Subscriptions are identified by HandlerID — no reflection needed for Off.
type EventBus struct {
	mu       sync.RWMutex
	handlers map[EventType][]handlerEntry
	nextID   uint64
}

// NewEventBus creates a new EventBus.
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[EventType][]handlerEntry),
	}
}

// On subscribes a handler to an event type and returns a HandlerID
// that can be passed to Off to unsubscribe.
func (b *EventBus) On(eventType EventType, handler func(TerminalEvent)) HandlerID {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nextID++
	id := b.nextID
	b.handlers[eventType] = append(b.handlers[eventType], handlerEntry{id: id, fn: handler})
	return HandlerID(id)
}

// Off unsubscribes the handler identified by the given HandlerID.
// It is safe to call Off with an already-removed or zero ID (no-op).
func (b *EventBus) Off(id HandlerID) {
	if id == 0 {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	target := uint64(id)
	for et, entries := range b.handlers {
		for i, e := range entries {
			if e.id == target {
				b.handlers[et] = append(entries[:i], entries[i+1:]...)
				return
			}
		}
	}
}

// Once subscribes a handler that fires only once, then removes itself.
// Returns the HandlerID so the caller can cancel before it fires if needed.
func (b *EventBus) Once(eventType EventType, handler func(TerminalEvent)) HandlerID {
	var id HandlerID
	id = b.On(eventType, func(ev TerminalEvent) {
		handler(ev)
		b.Off(id)
	})
	return id
}

// Emit dispatches an event to all handlers of the matching type.
// Handlers are called in subscription order.
func (b *EventBus) Emit(event TerminalEvent) {
	b.mu.RLock()
	// Copy the slice so handlers can call On/Off without deadlocking.
	entries := b.handlers[event.Type]
	handlers := make([]func(TerminalEvent), len(entries))
	for i, e := range entries {
		handlers[i] = e.fn
	}
	b.mu.RUnlock()

	for _, h := range handlers {
		h(event)
	}
}

// Clear removes all handlers from all event types.
func (b *EventBus) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = make(map[EventType][]handlerEntry)
	// Note: nextID is intentionally not reset so IDs remain globally unique.
}
