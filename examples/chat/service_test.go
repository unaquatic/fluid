package main

import (
	"slices"
	"testing"
)

// The service touches no widget, so these tests drive it as the event loop
// does, with no terminal, and check the state and the events it returns.

// TestSubmit checks that a prompt starts a reply, and that submitting does
// nothing while one is streaming.
func TestSubmit(t *testing.T) {
	s := &service{state: &state{}}
	events := s.submit("hi")
	if !s.state.running || !slices.Contains(events, uiEvent(eventReplyStart{prompt: "hi"})) {
		t.Fatalf("submit(%q) = %+v, want a reply started", "hi", events)
	}
	if events := s.submit("again"); events != nil {
		t.Errorf("submit while streaming = %+v, want nothing", events)
	}
}

// TestStream checks that the reply's events build its text in the order they
// arrive, that Esc stops the reply, which runs until its end arrives, and
// that Esc quits once no reply is streaming.
func TestStream(t *testing.T) {
	s := &service{state: &state{}}
	s.submit("hi")
	s.applyAgentEvent(replyDelta{text: "Hel"})
	s.applyAgentEvent(replyDelta{text: "lo"})
	if got := s.state.messages[1].text; got != "Hello" {
		t.Errorf("reply = %q, want %q", got, "Hello")
	}

	if events := s.escape(); !slices.Equal(events, []uiEvent{eventReplyStop{}}) || !s.state.running {
		t.Fatalf("escape() = %+v with running = %t, want a stop and still running", events, s.state.running)
	}
	s.applyAgentEvent(replyEnded{stopped: true})
	if s.state.running || s.state.messages[2].text != "stopped" {
		t.Errorf("after the end, running = %t and messages = %+v, want a stopped notice", s.state.running, s.state.messages)
	}
	if events := s.escape(); !slices.Equal(events, []uiEvent{eventQuit{}}) {
		t.Errorf("escape() with no reply streaming = %+v, want a quit", events)
	}
}
