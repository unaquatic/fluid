package main

// uiEvent is a change that the screen must show. The service returns
// uiEvent values, and ui.apply carries them out.
type uiEvent interface{ uiEvent() }

// eventMessageAdded appends a row for message i to the transcript.
type eventMessageAdded struct {
	i   int
	msg message
}

// eventMessageUpdated shows the new text of message i.
type eventMessageUpdated struct {
	i   int
	msg message
}

// eventInputClear empties the input.
type eventInputClear struct{}

// eventReplyStart starts streaming the reply to prompt.
type eventReplyStart struct{ prompt string }

// eventReplyStop stops the reply that is streaming.
type eventReplyStop struct{}

// eventQuit closes the program.
type eventQuit struct{}

func (eventMessageAdded) uiEvent()   {}
func (eventMessageUpdated) uiEvent() {}
func (eventInputClear) uiEvent()     {}
func (eventReplyStart) uiEvent()     {}
func (eventReplyStop) uiEvent()      {}
func (eventQuit) uiEvent()           {}
