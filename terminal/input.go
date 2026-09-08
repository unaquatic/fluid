package terminal

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/render"
)

// ---------------------------------------------------------------------------
// Input Loop
// ---------------------------------------------------------------------------

// inputLoop reads raw input from the terminal and enqueues dispatch
// closures onto the event loop. Runs as a background goroutine.
func (r *Renderer) inputLoop() {
	buf := make([]byte, 4096)
	for {
		n, err := r.backend.stdin.Read(buf)
		if err != nil || r.destroyed {
			return
		}
		if n == 0 {
			continue
		}

		events := r.parser.Parse(buf[:n])
		for _, evt := range events {
			select {
			case r.pendingWork <- func() { r.dispatchEvent(evt) }:
			case <-r.stopChan:
				return
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Event Dispatch
// ---------------------------------------------------------------------------

// dispatchEvent routes a terminal event through the unified dispatch
// pipeline. All event types follow the same contract: emit on the
// EventBus first (handlers may observe, or veto further handling via
// PreventDefault), then run framework default actions, then dispatch to
// the target widget. Synthetic hover events (MouseOver/MouseOut) are
// bookkeeping and exempt from the veto.
func (r *Renderer) dispatchEvent(evt input.TerminalEvent) {
	switch evt.Type {
	case input.EventKeyboard:
		r.dispatchKeyEvent(evt)
	case input.EventMouse:
		r.dispatchMouseEvent(evt)
	case input.EventPaste:
		r.dispatchPasteEvent(evt)
	}
}

// dispatchKeyEvent handles keyboard event routing.
func (r *Renderer) dispatchKeyEvent(evt input.TerminalEvent) {
	if evt.Key == nil {
		return
	}

	// Run input pre-processors in order. If any returns true, stop.
	r.mu.Lock()
	handlers := make([]func(string) bool, len(r.inputHandlers))
	copy(handlers, r.inputHandlers)
	r.mu.Unlock()

	for _, handler := range handlers {
		if handler(evt.Key.Raw) {
			return
		}
	}

	// Emit on EventBus — handlers may veto default actions and target
	// dispatch via PreventDefault.
	r.eventBus.Emit(evt)
	if evt.Key.DefaultPrevented() {
		return
	}

	// Ctrl+C exit — a default action, so bus handlers can override it.
	// Destroy closes the Done channel, unblocking Wait/Run.
	if r.config.ExitOnCtrlC && evt.Key.Ctrl && evt.Key.Name == "c" {
		r.Destroy()
		return
	}

	// Dispatch along the focus chain: the focused renderable first, then
	// its ancestors, until a handler consumes the event or stops its
	// propagation.

	r.mu.Lock()
	target := r.focused
	r.mu.Unlock()

	for target != nil && !target.IsDestroyed() {
		if target.HandleKeyPress(evt.Key) || evt.Key.PropagationStopped() {
			return
		}
		target = outerRenderable(target.Parent())
	}
}

// outerRenderable resolves a parent link — which points at an embedded
// *render.BaseRenderable — to the outer widget registered under the same
// num, so overridden methods dispatch correctly. Mirrors the resolution
// done by FocusRenderable.
func outerRenderable(p render.Renderable) render.Renderable {
	if p == nil {
		return nil
	}
	if actual := render.LookupRenderable(p.Num()); actual != nil {
		return actual
	}
	return p
}

// dispatchMouseEvent handles mouse event routing with hit testing,
// over/out tracking, drag detection, and auto-focus.
func (r *Renderer) dispatchMouseEvent(evt input.TerminalEvent) {
	if evt.Mouse == nil {
		return
	}

	me := evt.Mouse
	x, y := me.X, me.Y

	// Hit test: look up renderable at cursor position
	r.mu.Lock()
	var target render.Renderable
	if r.hitGrid != nil {
		id := r.hitGrid.Check(x, y)
		if id != 0 {
			target = render.LookupRenderable(id)
		}
	}
	r.mu.Unlock()

	me.Target = target

	// Over/Out tracking
	r.mu.Lock()
	lastHovered := r.lastHovered
	r.mu.Unlock()

	if target != lastHovered {
		// Mouse out on old target
		if lastHovered != nil && !lastHovered.IsDestroyed() {
			outEvent := &input.MouseEvent{
				Type:      input.MouseOut,
				Button:    me.Button,
				X:         x,
				Y:         y,
				Modifiers: me.Modifiers,
				Target:    lastHovered,
			}
			lastHovered.ProcessMouseEvent(outEvent)

			// Emit MouseOut on bus
			r.eventBus.Emit(input.TerminalEvent{
				Type:  input.EventMouse,
				Mouse: outEvent,
			})
		}

		// Mouse over on new target
		if target != nil && !target.IsDestroyed() {
			overEvent := &input.MouseEvent{
				Type:      input.MouseOver,
				Button:    me.Button,
				X:         x,
				Y:         y,
				Modifiers: me.Modifiers,
				Target:    target,
			}
			target.ProcessMouseEvent(overEvent)

			// Emit MouseOver on bus
			r.eventBus.Emit(input.TerminalEvent{
				Type:  input.EventMouse,
				Mouse: overEvent,
			})
		}

		r.mu.Lock()
		r.lastHovered = target
		r.mu.Unlock()
	}

	// Emit on EventBus — handlers may veto drag tracking, auto-focus,
	// and target dispatch via PreventDefault. The synthetic hover events
	// above are exempt.
	r.eventBus.Emit(evt)
	if me.DefaultPrevented() {
		r.RequestRender()
		return
	}

	// Drag tracking
	switch me.Type {
	case input.MouseDown:
		r.mu.Lock()
		r.dragSource = target
		r.dragButton = me.Button
		r.dragging = false
		r.mu.Unlock()

	case input.MouseDrag:
		r.mu.Lock()
		if !r.dragging {
			r.dragging = true
		}
		dragSrc := r.dragSource
		r.mu.Unlock()

		me.Source = dragSrc

	case input.MouseUp:
		r.mu.Lock()
		wasDragging := r.dragging
		dragSrc := r.dragSource
		r.dragging = false
		r.dragSource = nil
		r.mu.Unlock()

		if wasDragging {
			// Emit MouseDragEnd on the source
			if dragSrc != nil && !dragSrc.IsDestroyed() {
				dragEndEvent := &input.MouseEvent{
					Type:      input.MouseDragEnd,
					Button:    me.Button,
					X:         x,
					Y:         y,
					Modifiers: me.Modifiers,
					Source:    dragSrc,
					Target:    dragSrc,
				}
				dragSrc.ProcessMouseEvent(dragEndEvent)
				r.eventBus.Emit(input.TerminalEvent{
					Type:  input.EventMouse,
					Mouse: dragEndEvent,
				})
			}

			// Emit MouseDrop on the target
			if target != nil && !target.IsDestroyed() && target != dragSrc {
				dropEvent := &input.MouseEvent{
					Type:      input.MouseDrop,
					Button:    me.Button,
					X:         x,
					Y:         y,
					Modifiers: me.Modifiers,
					Source:    dragSrc,
					Target:    target,
				}
				target.ProcessMouseEvent(dropEvent)
				r.eventBus.Emit(input.TerminalEvent{
					Type:  input.EventMouse,
					Mouse: dropEvent,
				})
			}
		}
	}

	// Auto-focus on MouseDown
	if me.Type == input.MouseDown && r.config.AutoFocus && target != nil {
		if target.Focusable() {
			r.FocusRenderable(target)
		}
	}

	// Dispatch to target renderable
	if target != nil && !target.IsDestroyed() {
		target.ProcessMouseEvent(me)
	}

	// Request render for mouse activity
	r.RequestRender()
}

// dispatchPasteEvent handles paste event routing.
func (r *Renderer) dispatchPasteEvent(evt input.TerminalEvent) {
	if evt.Paste == nil {
		return
	}

	// Emit on EventBus — handlers may veto target dispatch via
	// PreventDefault.
	r.eventBus.Emit(evt)
	if evt.Paste.DefaultPrevented() {
		return
	}

	// Dispatch to focused renderable
	r.mu.Lock()
	focused := r.focused
	r.mu.Unlock()

	if focused != nil && !focused.IsDestroyed() {
		focused.HandlePaste(evt.Paste)
	}
}

// ---------------------------------------------------------------------------
// SIGWINCH Handling
// ---------------------------------------------------------------------------

// watchResize listens for terminal resize signals.
// Runs as a goroutine.
func (r *Renderer) watchResize() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	defer signal.Stop(sigCh)

	for {
		select {
		case <-sigCh:
			w, h, err := r.backend.Size()
			if err != nil {
				continue
			}
			r.mu.Lock()
			changed := w != r.width || h != r.height
			r.mu.Unlock()

			if changed {
				r.Enqueue(func() { r.processResize(w, h) })
			}
		case <-r.stopChan:
			return
		}
	}

	// Request render so the loop wakes up to process the resize.
	r.RequestRender()
}
