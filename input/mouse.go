package input

import "fmt"

// MouseEventType identifies a mouse event subtype.
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

// String returns the mouse event type's name.
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

// MouseButton identifies a mouse button or scroll wheel direction.
type MouseButton int

const (
	ButtonNone   MouseButton = -1
	ButtonLeft   MouseButton = 0
	ButtonMiddle MouseButton = 1
	ButtonRight  MouseButton = 2
	WheelUp      MouseButton = 4
	WheelDown    MouseButton = 5
)

// ScrollInfo contains horizontal and vertical scroll deltas.
type ScrollInfo struct {
	DeltaX int // The horizontal scroll delta.
	DeltaY int // The vertical scroll delta; negative is up and positive is down.
}

// MouseModifiers reports the modifier keys held during a mouse event.
type MouseModifiers struct {
	Shift bool
	Alt   bool
	Ctrl  bool
}

// MouseEvent describes one mouse input event.
type MouseEvent struct {
	Type       MouseEventType
	Button     MouseButton
	X, Y       int // Zero-based terminal coordinates.
	Modifiers  MouseModifiers
	Scroll     *ScrollInfo // Non-nil for MouseScroll events.
	IsDragging bool

	// The renderer sets Source and Target during dispatch. They use any to avoid
	// an import cycle; the render package provides typed accessors.
	Source any // The renderable that initiated the event.
	Target any // The renderable under the cursor.

	propagationStopped bool
	defaultPrevented   bool
}

// StopPropagation prevents the event from reaching later propagation targets.
func (e *MouseEvent) StopPropagation() { e.propagationStopped = true }

// PreventDefault prevents the event's default behavior.
func (e *MouseEvent) PreventDefault() { e.defaultPrevented = true }

// PropagationStopped reports whether StopPropagation was called.
func (e *MouseEvent) PropagationStopped() bool { return e.propagationStopped }

// DefaultPrevented reports whether PreventDefault was called.
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
