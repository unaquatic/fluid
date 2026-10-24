package terminal

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/render"
)

// inputLoop reads terminal bytes in a background goroutine and enqueues parsed
// events for dispatch on the event-loop goroutine.
func (r *Renderer) inputLoop() {
	buf := make([]byte, 4096)
	for {
		n, err := r.backend.stdin.Read(buf)
		if err != nil || r.destroyed.Load() {
			return
		}
		if n == 0 {
			continue
		}

		for _, evt := range r.parser.Parse(buf[:n]) {
			r.Enqueue(func() { r.dispatchEvent(evt) })
		}
	}
}

// dispatchEvent routes keyboard, mouse, and paste events. Each original event
// reaches EventBus before framework defaults or target dispatch, allowing bus
// handlers to call PreventDefault. Synthetic hover events are dispatched
// separately.
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

// dispatchKeyEvent runs raw-input preprocessors, bus handlers, default actions,
// and focused-widget propagation in order.
func (r *Renderer) dispatchKeyEvent(evt input.TerminalEvent) {
	if evt.Key == nil {
		return
	}

	// A preprocessor returning true consumes the raw sequence.
	r.mu.Lock()
	handlers := make([]func(string) bool, len(r.inputHandlers))
	copy(handlers, r.inputHandlers)
	r.mu.Unlock()

	for _, handler := range handlers {
		if handler(evt.Key.Raw) {
			return
		}
	}

	// Bus handlers can prevent framework and widget handling.
	r.eventBus.Emit(evt)
	if evt.Key.DefaultPrevented() {
		return
	}

	// Unprevented Control-C is a configurable framework shutdown action.
	if r.config.ExitOnCtrlC && evt.Key.Mods.Has(input.ModCtrl) && evt.Key.Name == "c" {
		r.Destroy()
		return
	}

	// Visit the focused renderable and then its ancestors until handled.

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

// outerRenderable replaces an embedded BaseRenderable parent link with the
// concrete widget registered under the same numeric ID.
func outerRenderable(p render.Renderable) render.Renderable {
	if p == nil {
		return nil
	}
	if actual := render.LookupRenderable(p.Num()); actual != nil {
		return actual
	}
	return p
}

// dispatchMouseEvent performs hit testing, hover transitions, bus dispatch,
// drag tracking, automatic focus, and target dispatch.
func (r *Renderer) dispatchMouseEvent(evt input.TerminalEvent) {
	if evt.Mouse == nil {
		return
	}

	me := evt.Mouse
	x, y := me.X, me.Y

	// Resolve the topmost renderable at the pointer coordinates.
	var target render.Renderable
	if r.HitTesting() {
		r.mu.Lock()
		id := r.hitGrid.Check(x, y)
		r.mu.Unlock()
		if id != 0 {
			target = render.LookupRenderable(id)
		}
	}

	me.Target = target

	// Compare the current target with the target from the previous mouse event.
	r.mu.Lock()
	lastHovered := r.lastHovered
	r.mu.Unlock()

	if target != lastHovered {
		// Notify the previous target before publishing its MouseOut event.
		if lastHovered != nil && !lastHovered.IsDestroyed() {
			outEvent := &input.MouseEvent{
				Type:   input.MouseOut,
				Button: me.Button,
				X:      x,
				Y:      y,
				Mods:   me.Mods,
				Target: lastHovered,
			}
			lastHovered.ProcessMouseEvent(outEvent)

			r.eventBus.Emit(input.TerminalEvent{
				Type:  input.EventMouse,
				Mouse: outEvent,
			})
		}

		// Notify the new target before publishing its MouseOver event.
		if target != nil && !target.IsDestroyed() {
			overEvent := &input.MouseEvent{
				Type:   input.MouseOver,
				Button: me.Button,
				X:      x,
				Y:      y,
				Mods:   me.Mods,
				Target: target,
			}
			target.ProcessMouseEvent(overEvent)

			r.eventBus.Emit(input.TerminalEvent{
				Type:  input.EventMouse,
				Mouse: overEvent,
			})
		}

		r.mu.Lock()
		r.lastHovered = target
		r.mu.Unlock()
	}

	// Bus handlers can prevent drag tracking, focus changes, and target dispatch.
	r.eventBus.Emit(evt)
	if me.DefaultPrevented() {
		r.RequestRender()
		return
	}

	// Preserve the press target as the drag source until button release.
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
			// Send MouseDragEnd to the original press target.
			if dragSrc != nil && !dragSrc.IsDestroyed() {
				dragEndEvent := &input.MouseEvent{
					Type:   input.MouseDragEnd,
					Button: me.Button,
					X:      x,
					Y:      y,
					Mods:   me.Mods,
					Source: dragSrc,
					Target: dragSrc,
				}
				dragSrc.ProcessMouseEvent(dragEndEvent)
				r.eventBus.Emit(input.TerminalEvent{
					Type:  input.EventMouse,
					Mouse: dragEndEvent,
				})
			}

			// Send MouseDrop to a distinct release target.
			if target != nil && !target.IsDestroyed() && target != dragSrc {
				dropEvent := &input.MouseEvent{
					Type:   input.MouseDrop,
					Button: me.Button,
					X:      x,
					Y:      y,
					Mods:   me.Mods,
					Source: dragSrc,
					Target: target,
				}
				target.ProcessMouseEvent(dropEvent)
				r.eventBus.Emit(input.TerminalEvent{
					Type:  input.EventMouse,
					Mouse: dropEvent,
				})
			}
		}
	}

	// Mouse down can transfer focus to a focusable target.
	if me.Type == input.MouseDown && r.config.AutoFocus && target != nil {
		if target.Focusable() {
			r.FocusRenderable(target)
		}
	}

	// Deliver the original event after framework state is updated.
	if target != nil && !target.IsDestroyed() {
		target.ProcessMouseEvent(me)
	}

	// Mouse activity can change hover, drag, focus, or widget state.
	r.RequestRender()
}

// dispatchPasteEvent publishes a paste and then delivers unprevented content to
// the focused renderable.
func (r *Renderer) dispatchPasteEvent(evt input.TerminalEvent) {
	if evt.Paste == nil {
		return
	}

	// Bus handlers can prevent focused-widget delivery.
	r.eventBus.Emit(evt)
	if evt.Paste.DefaultPrevented() {
		return
	}

	r.mu.Lock()
	focused := r.focused
	r.mu.Unlock()

	if focused != nil && !focused.IsDestroyed() {
		focused.HandlePaste(evt.Paste)
	}
}

// watchResize receives SIGWINCH in a background goroutine and enqueues changed
// terminal dimensions on the event loop.
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
				// Inline layout only marks itself dirty, so the frame is
				// requested here.
				r.Enqueue(func() {
					r.processResize(w, h)
					r.RequestRender()
				})
			}
		case <-r.stopChan:
			return
		}
	}
}
