package input

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestEventInterface(t *testing.T) {
	// Each concrete input event supports the Event propagation controls.
	var e Event

	e = &KeyEvent{}
	e.StopPropagation()
	if !e.PropagationStopped() {
		t.Error("KeyEvent should implement Event")
	}

	e = &MouseEvent{}
	e.PreventDefault()
	if !e.DefaultPrevented() {
		t.Error("MouseEvent should implement Event")
	}

	e = &PasteEvent{}
	_ = e.PropagationStopped()
}

func TestEventBus_OnEmit(t *testing.T) {
	bus := NewEventBus()
	var received TerminalEvent

	bus.On(EventKeyboard, func(ev TerminalEvent) {
		received = ev
	})

	sent := TerminalEvent{
		Type: EventKeyboard,
		Key:  &KeyEvent{Name: "a"},
	}
	bus.Emit(sent)

	if received.Key == nil || received.Key.Name != "a" {
		t.Error("handler should have received the event")
	}
}

func TestEventBus_MultipleHandlers(t *testing.T) {
	bus := NewEventBus()
	count := 0

	bus.On(EventKeyboard, func(ev TerminalEvent) { count++ })
	bus.On(EventKeyboard, func(ev TerminalEvent) { count++ })
	bus.On(EventKeyboard, func(ev TerminalEvent) { count++ })

	bus.Emit(TerminalEvent{Type: EventKeyboard, Key: &KeyEvent{Name: "x"}})

	if count != 3 {
		t.Errorf("expected 3 handler calls, got %d", count)
	}
}

func TestEventBus_DifferentTypes(t *testing.T) {
	bus := NewEventBus()
	keyCount := 0
	mouseCount := 0

	bus.On(EventKeyboard, func(ev TerminalEvent) { keyCount++ })
	bus.On(EventMouse, func(ev TerminalEvent) { mouseCount++ })

	bus.Emit(TerminalEvent{Type: EventKeyboard, Key: &KeyEvent{Name: "a"}})
	bus.Emit(TerminalEvent{Type: EventMouse, Mouse: &MouseEvent{Type: MouseDown}})

	if keyCount != 1 {
		t.Errorf("key handler called %d times, want 1", keyCount)
	}
	if mouseCount != 1 {
		t.Errorf("mouse handler called %d times, want 1", mouseCount)
	}
}

func TestEventBus_Off(t *testing.T) {
	bus := NewEventBus()
	count := 0

	id := bus.On(EventKeyboard, func(ev TerminalEvent) { count++ })

	bus.Emit(TerminalEvent{Type: EventKeyboard, Key: &KeyEvent{Name: "a"}})
	if count != 1 {
		t.Fatal("handler should have fired once")
	}

	bus.Off(id)
	bus.Emit(TerminalEvent{Type: EventKeyboard, Key: &KeyEvent{Name: "b"}})
	if count != 1 {
		t.Errorf("after Off, handler fired %d times total, want 1", count)
	}
}

func TestEventBus_OffZeroID(t *testing.T) {
	bus := NewEventBus()
	// Zero does not identify a subscription and is a safe no-op.
	bus.Off(0)
}

func TestEventBus_OffIdempotent(t *testing.T) {
	bus := NewEventBus()
	id := bus.On(EventKeyboard, func(ev TerminalEvent) {})
	bus.Off(id)
	// Removing the same subscription again is a safe no-op.
	bus.Off(id)
}

func TestEventBus_Once(t *testing.T) {
	bus := NewEventBus()
	count := 0

	bus.Once(EventKeyboard, func(ev TerminalEvent) { count++ })

	bus.Emit(TerminalEvent{Type: EventKeyboard, Key: &KeyEvent{Name: "a"}})
	bus.Emit(TerminalEvent{Type: EventKeyboard, Key: &KeyEvent{Name: "b"}})
	bus.Emit(TerminalEvent{Type: EventKeyboard, Key: &KeyEvent{Name: "c"}})

	if count != 1 {
		t.Errorf("Once handler fired %d times, want 1", count)
	}
}

func TestEventBus_Clear(t *testing.T) {
	bus := NewEventBus()
	count := 0

	bus.On(EventKeyboard, func(ev TerminalEvent) { count++ })
	bus.On(EventMouse, func(ev TerminalEvent) { count++ })

	bus.Clear()

	bus.Emit(TerminalEvent{Type: EventKeyboard, Key: &KeyEvent{Name: "a"}})
	bus.Emit(TerminalEvent{Type: EventMouse, Mouse: &MouseEvent{}})

	if count != 0 {
		t.Errorf("after Clear, handlers fired %d times, want 0", count)
	}
}

func TestEventBus_EmitNoHandlers(t *testing.T) {
	bus := NewEventBus()
	// Emitting without subscribers is a safe no-op.
	bus.Emit(TerminalEvent{Type: EventKeyboard, Key: &KeyEvent{Name: "a"}})
}

func TestEventBus_GoroutineSafety(t *testing.T) {
	bus := NewEventBus()
	var count atomic.Int64

	handler := func(ev TerminalEvent) { count.Add(1) }

	var wg sync.WaitGroup
	// Register handlers concurrently before dispatch begins.
	for range 10 {
		wg.Go(func() {
			_ = bus.On(EventKeyboard, handler)
		})
	}
	wg.Wait()

	// Dispatch events concurrently to the completed subscription set.
	for range 100 {
		wg.Go(func() {
			bus.Emit(TerminalEvent{Type: EventKeyboard, Key: &KeyEvent{Name: "a"}})
		})
	}
	wg.Wait()

	// The assertion confirms dispatch occurred; the race detector checks synchronization.
	if count.Load() == 0 {
		t.Error("expected some handler calls")
	}
}

func TestEventBus_GoroutineSafetyWithOff(t *testing.T) {
	bus := NewEventBus()
	var count atomic.Int64

	var wg sync.WaitGroup
	// Modify subscriptions while callbacks execute on other goroutines.
	for range 50 {
		wg.Go(func() {
			id := bus.On(EventKeyboard, func(ev TerminalEvent) { count.Add(1) })
			bus.Emit(TerminalEvent{Type: EventKeyboard, Key: &KeyEvent{Name: "a"}})
			bus.Off(id)
		})
	}
	wg.Wait()

	// The assertion confirms dispatch occurred; the race detector checks synchronization.
	if count.Load() == 0 {
		t.Error("expected some handler calls")
	}
}

func TestResizeEvent(t *testing.T) {
	ev := TerminalEvent{
		Type:   EventResize,
		Resize: &ResizeEvent{Width: 80, Height: 24},
	}
	if ev.Resize.Width != 80 || ev.Resize.Height != 24 {
		t.Errorf("ResizeEvent = %+v, want 80x24", ev.Resize)
	}
}
