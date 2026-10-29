package terminal_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/terminal"
	"github.com/unaquatic/fluid/widget"
)

// This example quits when Escape is pressed. A handler on the event bus sees
// every key before the focused widget does.
func ExampleRenderer_EventBus() {
	r, err := terminal.NewRenderer(terminal.Config{})
	if err != nil {
		log.Fatal(err)
	}
	r.Root().Add(widget.NewText("press Escape to quit"))

	r.EventBus().On(input.EventKeyboard, func(e input.TerminalEvent) {
		if e.Key.Matches(input.KeyEscape, 0) {
			r.Destroy()
		}
	})
	if err := r.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

// This example adds a line of text every second from another goroutine.
// Widgets belong to the event loop, so the goroutine changes them through
// Enqueue.
func ExampleRenderer_Enqueue() {
	r, err := terminal.NewRenderer(terminal.Config{InlineMode: true})
	if err != nil {
		log.Fatal(err)
	}
	messages := widget.NewBox()
	r.Root().Add(messages)

	go func() {
		for i := range 5 {
			time.Sleep(time.Second)
			r.Enqueue(func() {
				messages.Add(widget.NewText(fmt.Sprintf("message %d", i+1)))
			})
		}
	}()
	if err := r.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
