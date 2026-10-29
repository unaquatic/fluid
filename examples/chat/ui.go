package main

import (
	"sync"

	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/terminal"
	"github.com/unaquatic/fluid/textbuf"
	"github.com/unaquatic/fluid/widget"
)

// ui owns the widgets and applies the service's events to them. Its methods
// run on the event loop.
type ui struct {
	renderer   *terminal.Renderer
	service    *service
	transcript *widget.Box
	rows       []*widget.Text // a row for each message, by index
	input      *widget.Input
	// stopReply stops the reply that is streaming.
	stopReply func()
}

// newUI builds the widgets under r's root. The input keeps its default keys:
// Enter submits, which calls the service from its OnSubmit.
func newUI(r *terminal.Renderer, s *service) *ui {
	u := &ui{renderer: r, service: s, transcript: widget.NewBox()}
	u.input = widget.NewInput(
		widget.InputWithPlaceholder("Type a message..."),
		widget.InputWithOnSubmit(func(text string) { u.apply(u.service.submit(text)) }),
	)
	r.Root().Add(u.transcript)
	r.Root().Add(widget.NewBox(
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderSingle),
		widget.Children(u.input),
	))
	r.Root().Add(widget.NewText(" Esc: Stop/Exit | Enter: Send", widget.TextWithFg(style.ColorGray)))
	return u
}

// handleKey handles Esc before the focused input sees it, and marks it with
// PreventDefault so that the input never does. Every other key, such as a
// typed letter or Enter, goes on to the input.
func (u *ui) handleKey(k *input.KeyEvent) {
	if k.Matches(input.KeyEscape, 0) {
		u.apply(u.service.escape())
		k.PreventDefault()
	}
}

// apply carries out events on the widgets, in order.
func (u *ui) apply(events []uiEvent) {
	for _, evt := range events {
		switch e := evt.(type) {
		case eventMessageAdded:
			u.addRow(e.msg)
		case eventMessageUpdated:
			u.rows[e.i].SetContent(e.msg.text)
		case eventInputClear:
			u.input.SetValue("")
		case eventReplyStart:
			u.startReply(e.prompt)
		case eventReplyStop:
			u.stopReply()
		case eventQuit:
			u.renderer.Destroy()
		}
	}
}

// startReply streams the reply to prompt on a goroutine of its own. The
// goroutine must not touch the state or the widgets, so it hands each event
// to the event loop through Enqueue, which runs them in the order it sends
// them.
func (u *ui) startReply(prompt string) {
	stop := make(chan struct{})
	// Esc can stop a reply again before its end arrives.
	u.stopReply = sync.OnceFunc(func() { close(stop) })
	go stream(prompt, stop, func(ev agentEvent) {
		u.renderer.Enqueue(func() { u.apply(u.service.applyAgentEvent(ev)) })
	})
}

// addRow appends a row for msg to the transcript.
func (u *ui) addRow(msg message) {
	text, fg := msg.text, style.ColorWhite
	switch msg.role {
	case roleUser:
		text = "› " + text
	case roleNotice:
		text, fg = "· "+text, style.ColorGray
	}
	row := widget.NewText(text, widget.TextWithFg(fg), widget.TextWithWrapMode(textbuf.WrapWord))
	u.rows = append(u.rows, row)
	u.transcript.Add(row)
}
