package input

// PasteEvent represents pasted text from the terminal (bracketed paste mode).
type PasteEvent struct {
	Text string // The pasted text (ANSI stripped)

	propagationStopped bool
	defaultPrevented   bool
}

// StopPropagation stops propagation.
func (e *PasteEvent) StopPropagation() { e.propagationStopped = true }

// PreventDefault prevents default handling.
func (e *PasteEvent) PreventDefault() { e.defaultPrevented = true }

// PropagationStopped returns true if propagation was stopped.
func (e *PasteEvent) PropagationStopped() bool { return e.propagationStopped }

// DefaultPrevented returns true if default was prevented.
func (e *PasteEvent) DefaultPrevented() bool { return e.defaultPrevented }
