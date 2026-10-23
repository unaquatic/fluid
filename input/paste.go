package input

// PasteEvent contains text received through terminal bracketed paste mode.
type PasteEvent struct {
	// Text is the pasted text after ANSI escape sequences are removed.
	Text string

	propagationStopped bool
	defaultPrevented   bool
}

// StopPropagation prevents the event from reaching later propagation targets.
func (e *PasteEvent) StopPropagation() { e.propagationStopped = true }

// PreventDefault prevents the event's default behavior.
func (e *PasteEvent) PreventDefault() { e.defaultPrevented = true }

// PropagationStopped reports whether StopPropagation was called.
func (e *PasteEvent) PropagationStopped() bool { return e.propagationStopped }

// DefaultPrevented reports whether PreventDefault was called.
func (e *PasteEvent) DefaultPrevented() bool { return e.defaultPrevented }
