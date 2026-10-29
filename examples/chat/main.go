// Command chat shows the lifecycle of a fluid program as flow's terminal UI
// runs it: the program's state, the events that change it, and how those
// changes reach the widgets.
//
//   - state (state.go) is plain data. It holds no widgets.
//   - service (service.go) changes the state and returns uiEvent values
//     (events.go) that say what the screen must show. It never touches a
//     widget, so its tests (service_test.go) need no terminal.
//   - ui (ui.go) owns the widgets, applies the events to them, and routes
//     keys.
//
// The state and the widgets change only on the renderer's event loop. The
// input's submit callback runs there, Esc reaches ui.handleKey there through
// the event bus, and the reply, which streams from a goroutine of its own
// (agent.go), reaches it through Renderer.Enqueue.
//
// Type a prompt and press Enter to send it. Esc stops the reply, or exits when
// none is streaming.
//
//	go run ./examples/chat
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/terminal"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "chat:", err)
		os.Exit(1)
	}
}

// run runs the program inline, below the shell's prompt, until the user
// quits.
func run(ctx context.Context) error {
	r, err := terminal.NewRenderer(terminal.Config{InlineMode: true})
	if err != nil {
		return err
	}
	u := newUI(r, &service{state: &state{}})

	// Every key goes through handleKey before the focused input. The input
	// handles Enter itself and calls its OnSubmit.
	r.EventBus().On(input.EventKeyboard, func(e input.TerminalEvent) {
		if e.Key.Type != input.KeyRelease {
			u.handleKey(e.Key)
		}
	})
	r.FocusRenderable(u.input)
	return r.Run(ctx)
}
