package main

import (
	"fmt"
	"strings"
	"time"
)

// agentEvent is an event of a streaming reply, which stream sends from its
// own goroutine.
type agentEvent interface{ agentEvent() }

// replyDelta carries the next words of the reply.
type replyDelta struct{ text string }

// replyEnded reports that the reply ended. stopped reports that Esc stopped
// it before it was complete.
type replyEnded struct{ stopped bool }

func (replyDelta) agentEvent() {}
func (replyEnded) agentEvent() {}

// stream sends the reply to prompt with send, a word at a time as a model
// streams one, until it is complete or stop is closed. It runs on a goroutine
// of its own.
func stream(prompt string, stop <-chan struct{}, send func(agentEvent)) {
	text := fmt.Sprintf("You said %q. This reply streams from a goroutine, a word at a time, "+
		"and each word reaches the event loop through Renderer.Enqueue. Press Esc to stop it.", prompt)
	for _, word := range strings.SplitAfter(text, " ") {
		select {
		case <-stop:
			send(replyEnded{stopped: true})
			return
		case <-time.After(80 * time.Millisecond):
		}
		send(replyDelta{text: word})
	}
	send(replyEnded{})
}
