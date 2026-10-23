package input

import (
	"sync"
)

// Event controls propagation and default handling for an input event.
type Event interface {
	StopPropagation()
	PreventDefault()
	PropagationStopped() bool
	DefaultPrevented() bool
}

// Verify that each input event implements Event.
var (
	_ Event = (*KeyEvent)(nil)
	_ Event = (*MouseEvent)(nil)
	_ Event = (*PasteEvent)(nil)
)

// EventType identifies a terminal event category.
type EventType int

const (
	EventKeyboard EventType = iota
	EventMouse
	EventPaste
	EventResize
)

// ResizeEvent reports new terminal dimensions in cells.
type ResizeEvent struct {
	// Width is the terminal's new width in cells.
	Width int
	// Height is the terminal's new height in cells.
	Height int
}

// TerminalEvent contains one event payload identified by Type.
type TerminalEvent struct {
	// Type names the payload that is set.
	Type EventType
	// Key is the payload of an EventKeyboard event.
	Key *KeyEvent
	// Mouse is the payload of an EventMouse event.
	Mouse *MouseEvent
	// Paste is the payload of an EventPaste event.
	Paste *PasteEvent
	// Resize is the payload of an EventResize event.
	Resize *ResizeEvent
}

// HandlerID identifies a subscription created by On or Once. Its zero value
// does not identify a subscription.
type HandlerID uint64

// handlerEntry associates a subscription ID with its callback.
type handlerEntry struct {
	id uint64
	fn func(TerminalEvent)
}

// EventBus dispatches events to subscribers. Its methods are safe for
// concurrent use.
type EventBus struct {
	mu       sync.RWMutex
	handlers map[EventType][]handlerEntry
	nextID   uint64
}

// NewEventBus returns an EventBus with no subscriptions.
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[EventType][]handlerEntry),
	}
}

// On subscribes handler to eventType and returns its subscription ID.
func (b *EventBus) On(eventType EventType, handler func(TerminalEvent)) HandlerID {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nextID++
	id := b.nextID
	b.handlers[eventType] = append(b.handlers[eventType], handlerEntry{id: id, fn: handler})
	return HandlerID(id)
}

// Off removes the subscription identified by id. It does nothing if id is zero
// or no longer registered.
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

// Once subscribes handler and removes the subscription after the callback
// returns. The returned ID can cancel the subscription before it runs.
func (b *EventBus) Once(eventType EventType, handler func(TerminalEvent)) HandlerID {
	var id HandlerID
	id = b.On(eventType, func(ev TerminalEvent) {
		handler(ev)
		b.Off(id)
	})
	return id
}

// Emit calls a snapshot of the matching subscribers in subscription order.
func (b *EventBus) Emit(event TerminalEvent) {
	b.mu.RLock()
	// Release the lock before callbacks so they can modify subscriptions.
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

// Clear removes every subscription.
func (b *EventBus) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = make(map[EventType][]handlerEntry)
	// Do not reuse IDs that callers may still retain.
}
