package input

import "fmt"

// ---------------------------------------------------------------------------
// MouseEventType
// ---------------------------------------------------------------------------

// MouseEventType enumerates the 9 mouse event subtypes.
type MouseEventType int

const (
	MouseDown MouseEventType = iota
	MouseUp
	MouseMove
	MouseDrag
	MouseDragEnd
	MouseDrop
	MouseOver
	MouseOut
	MouseScroll
)

// String returns the human-readable name of a mouse event type.
func (t MouseEventType) String() string {
	switch t {
	case MouseDown:
		return "mousedown"
	case MouseUp:
		return "mouseup"
	case MouseMove:
		return "mousemove"
	case MouseDrag:
		return "mousedrag"
	case MouseDragEnd:
		return "mousedragend"
	case MouseDrop:
		return "mousedrop"
	case MouseOver:
		return "mouseover"
	case MouseOut:
		return "mouseout"
	case MouseScroll:
		return "mousescroll"
	default:
		return "unknown"
	}
}

// ---------------------------------------------------------------------------
// MouseButton
// ---------------------------------------------------------------------------

// MouseButton identifies which mouse button was involved.
type MouseButton int

const (
	ButtonNone   MouseButton = -1
	ButtonLeft   MouseButton = 0
	ButtonMiddle MouseButton = 1
	ButtonRight  MouseButton = 2
	WheelUp      MouseButton = 4
	WheelDown    MouseButton = 5
)

// ---------------------------------------------------------------------------
// ScrollInfo
// ---------------------------------------------------------------------------

// ScrollInfo carries scroll delta information.
type ScrollInfo struct {
	DeltaX int // Horizontal scroll delta
	DeltaY int // Vertical scroll delta (-1 = up, 1 = down)
}

// ---------------------------------------------------------------------------
// MouseModifiers
// ---------------------------------------------------------------------------

// MouseModifiers tracks which modifier keys were held during a mouse event.
type MouseModifiers struct {
	Shift bool
	Alt   bool
	Ctrl  bool
}

// ---------------------------------------------------------------------------
// MouseEvent
// ---------------------------------------------------------------------------

// MouseEvent represents a mouse input event.
type MouseEvent struct {
	Type       MouseEventType
	Button     MouseButton
	X, Y       int // Absolute terminal coordinates
	Modifiers  MouseModifiers
	Scroll     *ScrollInfo // Non-nil only for MouseScroll events
	IsDragging bool

	// Source and target are set by the renderer during dispatch.
	// These use interface{} here to avoid circular imports;
	// the render package provides typed accessors.
	Source any // Renderable that initiated the event
	Target any // Renderable under the cursor (from hit grid)

	propagationStopped bool
	defaultPrevented   bool
}

// StopPropagation stops the event from bubbling.
func (e *MouseEvent) StopPropagation() { e.propagationStopped = true }

// PreventDefault prevents default handling.
func (e *MouseEvent) PreventDefault() { e.defaultPrevented = true }

// PropagationStopped returns true if propagation was stopped.
func (e *MouseEvent) PropagationStopped() bool { return e.propagationStopped }

// DefaultPrevented returns true if default was prevented.
func (e *MouseEvent) DefaultPrevented() bool { return e.defaultPrevented }

// String returns a human-readable description of the mouse event.
func (e *MouseEvent) String() string {
	s := fmt.Sprintf("%s btn=%d (%d,%d)", e.Type, e.Button, e.X, e.Y)
	if e.Modifiers.Ctrl {
		s += " ctrl"
	}
	if e.Modifiers.Alt {
		s += " alt"
	}
	if e.Modifiers.Shift {
		s += " shift"
	}
	if e.Scroll != nil {
		s += fmt.Sprintf(" scroll(%d,%d)", e.Scroll.DeltaX, e.Scroll.DeltaY)
	}
	return s
}
