package terminal

// InlineMode reports whether the renderer draws into the normal screen and
// scrollback instead of the alternate screen.
func (r *Renderer) InlineMode() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.config.InlineMode
}

// SetInlineMode switches rendering between the alternate screen and inline
// output in the normal screen and scrollback. The next frame is drawn in full.
//
// The alternate screen covers inline output without erasing it, and leaving it
// shows that output again, with the cursor where it was. The last inline frame
// and its anchor are therefore kept across a stay on the alternate screen, and
// the first inline frame after it is compared against them. Only rows that
// changed are rewritten, and the flush falls back to erasing the screen and
// scrollback when the terminal was resized or a changed row is in the
// scrollback.
//
// Call it from the event-loop goroutine, for example through Enqueue. Before
// setup it only records the mode.
func (r *Renderer) SetInlineMode(inline bool) {
	r.mu.Lock()
	if r.config.InlineMode == inline {
		r.mu.Unlock()
		return
	}
	r.config.InlineMode = inline
	r.config.Fullscreen = !inline
	w, h := r.width, r.height
	setupDone := r.controlState != StateIdle && r.controlState != StateExplicitStopped
	r.mu.Unlock()

	// The layout height describes the mode being left.
	r.lastInlineLayoutWidth = 0
	r.invalidated = true

	if !setupDone {
		// Nothing ties the frame to the screen, so the next one is drawn afresh.
		r.inlineCurrentBuf = nil
		r.inlineViewportTop = 0
		r.inlineCursorRow = 0
		r.inlineTermHeight = 0
		return
	}

	r.applyScreenMode(inline)
	r.processResize(w, h)
	r.RequestRender()
}

// applyScreenMode switches the terminal between the alternate and normal screen
// for the requested mode, and hands the mouse to whichever should own it.
func (r *Renderer) applyScreenMode(inline bool) {
	if inline {
		if r.backend.AlternateScreen() {
			r.backend.LeaveAlternateScreen()
		}
		// Inline mode does not hit test, so the terminal keeps the mouse for its
		// own selection and scrolling, and the cursor is drawn by inverting a cell.
		r.backend.DisableMouse()
		r.backend.HideCursor()
		return
	}

	if !r.backend.AlternateScreen() {
		r.backend.EnterAlternateScreen()
	}
	if r.config.UseMouse {
		r.backend.EnableMouse(r.config.EnableMouseMovement)
	}
	r.backend.Clear()
}
