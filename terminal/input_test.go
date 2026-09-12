package terminal

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
)

// ---------------------------------------------------------------------------
// Helper: create a renderer for input testing (no real terminal)
// ---------------------------------------------------------------------------

func newInputTestRenderer() *Renderer {
	var outBuf bytes.Buffer
	w, h := 40, 20

	root := NewRootRenderable(w, h)
	hitGrid := NewHitGrid(w, h)

	r := &Renderer{
		root:          root,
		hitGrid:       hitGrid,
		backend:       NewBackend(strings.NewReader(""), &outBuf),
		config:        DefaultConfig(),
		currentBuffer: buffer.New(w, h),
		nextBuffer:    buffer.New(w, h),
		width:         w,
		height:        h,
		controlState:  StateIdle,
		parser:        input.NewParser(),
		eventBus:      input.NewEventBus(),
		pendingWork:   make(chan func()),
		wake:          make(chan struct{}, 1),
		stopChan:      make(chan struct{}, 1),
		done:          make(chan struct{}),
		sel:           render.NewSelection(),
		linkRegistry:  NewLinkRegistry(),
		stats:         NewStats(60),
		lastFrameTime: time.Now(),
	}

	root.SetContext(r)

	return r
}

// ---------------------------------------------------------------------------
// dispatchKeyEvent tests
// ---------------------------------------------------------------------------

func TestDispatchKeyEvent_EmitsOnBus(t *testing.T) {
	r := newInputTestRenderer()

	var received bool
	r.eventBus.On(input.EventKeyboard, func(evt input.TerminalEvent) {
		received = true
	})

	r.dispatchEvent(input.TerminalEvent{
		Type: input.EventKeyboard,
		Key: &input.KeyEvent{
			Name:   "a",
			Type:   input.KeyPress,
			Raw:    "a",
			Source: "raw",
		},
	})

	if !received {
		t.Error("expected keyboard event to be emitted on EventBus")
	}
}

func TestDispatchKeyEvent_DeliveredToFocused(t *testing.T) {
	r := newInputTestRenderer()

	child := render.NewBaseRenderable(render.WithID("focusable"), render.WithFocusable(true))
	r.root.Add(child)

	var keyReceived string
	child.OnKeyDownHandler = func(e *input.KeyEvent) {
		keyReceived = e.Name
	}

	r.focused = child

	r.dispatchEvent(input.TerminalEvent{
		Type: input.EventKeyboard,
		Key: &input.KeyEvent{
			Name:   "x",
			Type:   input.KeyPress,
			Raw:    "x",
			Source: "raw",
		},
	})

	if keyReceived != "x" {
		t.Errorf("expected focused renderable to receive key 'x', got %q", keyReceived)
	}
}

func TestDispatchKeyEvent_InputHandlerStops(t *testing.T) {
	r := newInputTestRenderer()

	var busReceived bool
	r.eventBus.On(input.EventKeyboard, func(evt input.TerminalEvent) {
		busReceived = true
	})

	r.AddInputHandler(func(seq string) bool {
		return true // consume all input
	})

	r.dispatchEvent(input.TerminalEvent{
		Type: input.EventKeyboard,
		Key: &input.KeyEvent{
			Name:   "a",
			Type:   input.KeyPress,
			Raw:    "a",
			Source: "raw",
		},
	})

	if busReceived {
		t.Error("expected input handler to stop event before reaching EventBus")
	}
}

// keyPressEvent builds a keyboard TerminalEvent for dispatch tests.
func keyPressEvent(name string) input.TerminalEvent {
	return input.TerminalEvent{
		Type: input.EventKeyboard,
		Key: &input.KeyEvent{
			Name:   name,
			Type:   input.KeyPress,
			Raw:    name,
			Source: "raw",
		},
	}
}

func TestDispatchKeyEvent_BubblesToParent(t *testing.T) {
	r := newInputTestRenderer()

	parent := render.NewBaseRenderable(render.WithID("parent"))
	child := render.NewBaseRenderable(render.WithID("child"), render.WithFocusable(true))
	r.root.Add(parent)
	parent.Add(child)

	var parentKey string
	parent.OnKeyDownHandler = func(e *input.KeyEvent) { parentKey = e.Name }

	r.focused = child
	r.dispatchEvent(keyPressEvent("x"))

	if parentKey != "x" {
		t.Errorf("expected unhandled key to bubble to parent, got %q", parentKey)
	}
}

func TestDispatchKeyEvent_ConsumedStopsBubbling(t *testing.T) {
	r := newInputTestRenderer()

	parent := render.NewBaseRenderable(render.WithID("parent"))
	child := render.NewBaseRenderable(render.WithID("child"), render.WithFocusable(true))
	r.root.Add(parent)
	parent.Add(child)

	var parentCalled bool
	child.OnKeyDownHandler = func(e *input.KeyEvent) { e.StopPropagation() }
	parent.OnKeyDownHandler = func(e *input.KeyEvent) { parentCalled = true }

	r.focused = child
	r.dispatchEvent(keyPressEvent("x"))

	if parentCalled {
		t.Error("expected consumed key not to bubble to parent")
	}
}

// stopPropRenderable stops propagation without consuming the event,
// exercising the PropagationStopped branch of the focus-chain walk.
type stopPropRenderable struct {
	*render.BaseRenderable
}

func (s *stopPropRenderable) HandleKeyPress(e *input.KeyEvent) bool {
	e.StopPropagation()
	return false
}

func TestDispatchKeyEvent_StopPropagationStopsBubbling(t *testing.T) {
	r := newInputTestRenderer()

	parent := render.NewBaseRenderable(render.WithID("parent"))
	child := &stopPropRenderable{
		BaseRenderable: render.NewBaseRenderable(render.WithID("child"), render.WithFocusable(true)),
	}
	r.root.Add(parent)
	parent.Add(child)

	var parentCalled bool
	parent.OnKeyDownHandler = func(e *input.KeyEvent) { parentCalled = true }

	r.focused = child
	r.dispatchEvent(keyPressEvent("x"))

	if parentCalled {
		t.Error("expected StopPropagation to halt bubbling even when unconsumed")
	}
}

func TestDispatchKeyEvent_UnhandledReachesRoot(t *testing.T) {
	r := newInputTestRenderer()

	parent := render.NewBaseRenderable(render.WithID("parent"))
	child := render.NewBaseRenderable(render.WithID("child"), render.WithFocusable(true))
	r.root.Add(parent)
	parent.Add(child)

	var rootKey string
	r.root.OnKeyDownHandler = func(e *input.KeyEvent) { rootKey = e.Name }

	r.focused = child
	r.dispatchEvent(keyPressEvent("z"))

	if rootKey != "z" {
		t.Errorf("expected unhandled key to reach root, got %q", rootKey)
	}
}

func TestDispatchKeyEvent_ExitOnCtrlC(t *testing.T) {
	r := newInputTestRenderer()
	r.config.ExitOnCtrlC = true

	evt := keyPressEvent("c")
	evt.Key.Ctrl = true
	r.dispatchEvent(evt)

	if !r.destroyed {
		t.Error("expected ctrl+c to destroy the renderer")
	}
}

func TestDispatchKeyEvent_BusVetoesExitOnCtrlC(t *testing.T) {
	r := newInputTestRenderer()
	r.config.ExitOnCtrlC = true

	r.eventBus.On(input.EventKeyboard, func(evt input.TerminalEvent) {
		evt.Key.PreventDefault()
	})

	evt := keyPressEvent("c")
	evt.Key.Ctrl = true
	r.dispatchEvent(evt)

	if r.destroyed {
		t.Error("expected PreventDefault to veto the ExitOnCtrlC default action")
	}
}

func TestDispatchMouseEvent_BusVetoesTargetDispatch(t *testing.T) {
	r := newInputTestRenderer()

	child := render.NewBaseRenderable(render.WithID("veto-click"), render.WithFocusable(true))
	child.SetBounds(0, 0, 10, 10)
	r.root.Add(child)
	r.hitGrid.Add(0, 0, 10, 10, child.Num())

	var clicked bool
	child.OnMouseDownHandler = func(e *input.MouseEvent) { clicked = true }

	r.eventBus.On(input.EventMouse, func(evt input.TerminalEvent) {
		evt.Mouse.PreventDefault()
	})

	r.dispatchEvent(input.TerminalEvent{
		Type: input.EventMouse,
		Mouse: &input.MouseEvent{
			Type:   input.MouseDown,
			Button: input.ButtonLeft,
			X:      1,
			Y:      1,
		},
	})

	if clicked {
		t.Error("expected PreventDefault to veto target dispatch")
	}
	if r.focused == child {
		t.Error("expected PreventDefault to veto auto-focus")
	}
}

func TestDispatchPasteEvent_BusVetoesFocusedDispatch(t *testing.T) {
	r := newInputTestRenderer()

	child := render.NewBaseRenderable(render.WithID("paste-veto"), render.WithFocusable(true))
	r.root.Add(child)

	var pasted bool
	child.OnPasteHandler = func(e *input.PasteEvent) { pasted = true }
	r.focused = child

	r.eventBus.On(input.EventPaste, func(evt input.TerminalEvent) {
		evt.Paste.PreventDefault()
	})

	r.dispatchEvent(input.TerminalEvent{
		Type:  input.EventPaste,
		Paste: &input.PasteEvent{Text: "blocked"},
	})

	if pasted {
		t.Error("expected PreventDefault to veto focused paste dispatch")
	}
}

func TestPrependInputHandler(t *testing.T) {
	r := newInputTestRenderer()

	var order []int
	r.AddInputHandler(func(seq string) bool {
		order = append(order, 1)
		return false
	})
	r.PrependInputHandler(func(seq string) bool {
		order = append(order, 0)
		return false
	})

	r.dispatchEvent(input.TerminalEvent{
		Type: input.EventKeyboard,
		Key: &input.KeyEvent{
			Name:   "a",
			Type:   input.KeyPress,
			Raw:    "a",
			Source: "raw",
		},
	})

	if len(order) != 2 || order[0] != 0 || order[1] != 1 {
		t.Errorf("expected prepended handler first, got order: %v", order)
	}
}

// ---------------------------------------------------------------------------
// dispatchMouseEvent tests
// ---------------------------------------------------------------------------

func TestDispatchMouseEvent_HitTest(t *testing.T) {
	r := newInputTestRenderer()

	child := render.NewBaseRenderable(render.WithID("clickable"))
	child.SetBounds(5, 5, 10, 10)
	r.root.Add(child)

	// Register in hit grid
	r.hitGrid.Add(5, 5, 10, 10, child.Num())

	var clicked bool
	child.OnMouseDownHandler = func(e *input.MouseEvent) {
		clicked = true
	}

	r.dispatchEvent(input.TerminalEvent{
		Type: input.EventMouse,
		Mouse: &input.MouseEvent{
			Type:   input.MouseDown,
			Button: input.ButtonLeft,
			X:      7,
			Y:      7,
		},
	})

	if !clicked {
		t.Error("expected mouse down to be dispatched to hit renderable")
	}
}

func TestDispatchMouseEvent_AutoFocus(t *testing.T) {
	r := newInputTestRenderer()

	child := render.NewBaseRenderable(render.WithID("focusable-click"), render.WithFocusable(true))
	child.SetBounds(0, 0, 10, 10)
	r.root.Add(child)

	r.hitGrid.Add(0, 0, 10, 10, child.Num())

	r.dispatchEvent(input.TerminalEvent{
		Type: input.EventMouse,
		Mouse: &input.MouseEvent{
			Type:   input.MouseDown,
			Button: input.ButtonLeft,
			X:      1,
			Y:      1,
		},
	})

	if r.focused != child {
		t.Error("expected auto-focus to set focus on clicked focusable renderable")
	}
}

func TestDispatchMouseEvent_OverOut(t *testing.T) {
	r := newInputTestRenderer()

	childA := render.NewBaseRenderable(render.WithID("a"))
	childA.SetBounds(0, 0, 5, 5)

	childB := render.NewBaseRenderable(render.WithID("b"))
	childB.SetBounds(10, 0, 5, 5)

	r.root.Add(childA)
	r.root.Add(childB)

	r.hitGrid.Add(0, 0, 5, 5, childA.Num())
	r.hitGrid.Add(10, 0, 5, 5, childB.Num())

	var mu sync.Mutex
	var gotOverA, gotOutA, gotOverB bool
	childA.OnMouseOverHandler = func(e *input.MouseEvent) {
		mu.Lock()
		gotOverA = true
		mu.Unlock()
	}
	childA.OnMouseOutHandler = func(e *input.MouseEvent) {
		mu.Lock()
		gotOutA = true
		mu.Unlock()
	}
	childB.OnMouseOverHandler = func(e *input.MouseEvent) {
		mu.Lock()
		gotOverB = true
		mu.Unlock()
	}

	// Move into A
	r.dispatchEvent(input.TerminalEvent{
		Type: input.EventMouse,
		Mouse: &input.MouseEvent{
			Type: input.MouseMove,
			X:    2,
			Y:    2,
		},
	})

	mu.Lock()
	if !gotOverA {
		t.Error("expected MouseOver on A")
	}
	mu.Unlock()

	// Move into B (should trigger Out on A, Over on B)
	r.dispatchEvent(input.TerminalEvent{
		Type: input.EventMouse,
		Mouse: &input.MouseEvent{
			Type: input.MouseMove,
			X:    12,
			Y:    2,
		},
	})

	mu.Lock()
	if !gotOutA {
		t.Error("expected MouseOut on A")
	}
	if !gotOverB {
		t.Error("expected MouseOver on B")
	}
	mu.Unlock()
}

func TestDispatchMouseEvent_EmitsOnBus(t *testing.T) {
	r := newInputTestRenderer()

	var received bool
	r.eventBus.On(input.EventMouse, func(evt input.TerminalEvent) {
		if evt.Mouse != nil && evt.Mouse.Type == input.MouseDown {
			received = true
		}
	})

	r.dispatchEvent(input.TerminalEvent{
		Type: input.EventMouse,
		Mouse: &input.MouseEvent{
			Type:   input.MouseDown,
			Button: input.ButtonLeft,
			X:      0,
			Y:      0,
		},
	})

	if !received {
		t.Error("expected mouse event to be emitted on EventBus")
	}
}

// ---------------------------------------------------------------------------
// dispatchPasteEvent tests
// ---------------------------------------------------------------------------

func TestDispatchPasteEvent_DeliveredToFocused(t *testing.T) {
	r := newInputTestRenderer()

	child := render.NewBaseRenderable(render.WithID("paste-target"), render.WithFocusable(true))
	r.root.Add(child)

	var pastedText string
	child.OnPasteHandler = func(e *input.PasteEvent) {
		pastedText = e.Text
	}

	r.focused = child

	r.dispatchEvent(input.TerminalEvent{
		Type: input.EventPaste,
		Paste: &input.PasteEvent{
			Text: "hello paste",
		},
	})

	if pastedText != "hello paste" {
		t.Errorf("expected paste text 'hello paste', got %q", pastedText)
	}
}

func TestDispatchPasteEvent_EmitsOnBus(t *testing.T) {
	r := newInputTestRenderer()

	var received bool
	r.eventBus.On(input.EventPaste, func(evt input.TerminalEvent) {
		received = true
	})

	r.dispatchEvent(input.TerminalEvent{
		Type: input.EventPaste,
		Paste: &input.PasteEvent{
			Text: "pasted",
		},
	})

	if !received {
		t.Error("expected paste event to be emitted on EventBus")
	}
}

// ---------------------------------------------------------------------------
// handleResize test
// ---------------------------------------------------------------------------

func TestHandleResize(t *testing.T) {
	r := newInputTestRenderer()

	var resizeReceived bool
	r.eventBus.On(input.EventResize, func(evt input.TerminalEvent) {
		resizeReceived = true
		if evt.Resize.Width != 60 || evt.Resize.Height != 30 {
			t.Errorf("expected resize 60x30, got %dx%d", evt.Resize.Width, evt.Resize.Height)
		}
	})

	// Call processResize directly — tests don't run an event loop.
	r.processResize(60, 30)

	if r.width != 60 || r.height != 30 {
		t.Errorf("expected width=60, height=30, got %d, %d", r.width, r.height)
	}
	if r.currentBuffer.Width() != 60 || r.currentBuffer.Height() != 30 {
		t.Error("expected currentBuffer resized")
	}
	if r.nextBuffer.Width() != 60 || r.nextBuffer.Height() != 30 {
		t.Error("expected nextBuffer resized")
	}
	if !resizeReceived {
		t.Error("expected resize event emitted on bus")
	}
}

// ---------------------------------------------------------------------------
// Drag tracking tests
// ---------------------------------------------------------------------------

func TestDispatchMouseEvent_DragAndDrop(t *testing.T) {
	r := newInputTestRenderer()
	_ = style.ColorBlack // suppress unused import

	source := render.NewBaseRenderable(render.WithID("drag-source"))
	source.SetBounds(0, 0, 5, 5)

	target := render.NewBaseRenderable(render.WithID("drag-target"))
	target.SetBounds(10, 0, 5, 5)

	r.root.Add(source)
	r.root.Add(target)

	r.hitGrid.Add(0, 0, 5, 5, source.Num())
	r.hitGrid.Add(10, 0, 5, 5, target.Num())

	var gotDragEnd, gotDrop bool

	source.OnMouseDragEndHandler = func(e *input.MouseEvent) {
		gotDragEnd = true
	}
	target.OnMouseDropHandler = func(e *input.MouseEvent) {
		gotDrop = true
	}

	// Mouse down on source
	r.dispatchEvent(input.TerminalEvent{
		Type: input.EventMouse,
		Mouse: &input.MouseEvent{
			Type:   input.MouseDown,
			Button: input.ButtonLeft,
			X:      2,
			Y:      2,
		},
	})

	// Drag (motion with button held)
	r.dispatchEvent(input.TerminalEvent{
		Type: input.EventMouse,
		Mouse: &input.MouseEvent{
			Type:       input.MouseDrag,
			Button:     input.ButtonLeft,
			X:          5,
			Y:          2,
			IsDragging: true,
		},
	})

	// Mouse up on target
	r.dispatchEvent(input.TerminalEvent{
		Type: input.EventMouse,
		Mouse: &input.MouseEvent{
			Type:   input.MouseUp,
			Button: input.ButtonLeft,
			X:      12,
			Y:      2,
		},
	})

	if !gotDragEnd {
		t.Error("expected MouseDragEnd on source")
	}
	if !gotDrop {
		t.Error("expected MouseDrop on target")
	}
}
