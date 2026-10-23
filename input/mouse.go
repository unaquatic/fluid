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
	// DeltaX is the horizontal scroll delta.
	DeltaX int
	// DeltaY is the vertical scroll delta; negative is up and positive is down.
	DeltaY int
}

// MouseModifiers reports the modifier keys held during a mouse event.
type MouseModifiers struct {
	// Shift reports whether Shift was held.
	Shift bool
	// Alt reports whether Alt or Option was held.
	Alt bool
	// Ctrl reports whether Ctrl was held.
	Ctrl bool
}

// MouseEvent describes one mouse input event.
type MouseEvent struct {
	// Type is the kind of event, such as a press, a drag, or a scroll.
	Type MouseEventType
	// Button is the button pressed or held, or ButtonNone.
	Button MouseButton
	// X and Y are the zero-based terminal coordinates.
	X, Y int
	// Modifiers are the modifier keys held during the event.
	Modifiers MouseModifiers
	// Scroll is non-nil for MouseScroll events.
	Scroll *ScrollInfo
	// IsDragging reports whether a button was held while the mouse moved.
	IsDragging bool

	// Source is the renderable that initiated the event. The renderer sets it
	// during dispatch. It is any to avoid an import cycle; the render package
	// provides typed accessors.
	Source any
	// Target is the renderable under the cursor, set and typed as Source is.
	Target any

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
