package main

import "strings"

// service changes the state and returns the uiEvent values that show what
// changed. It never touches a widget. Its methods run on the event loop.
type service struct {
	state *state
}

// submit sends text as a prompt and starts its reply. It does nothing while a
// reply is streaming.
func (s *service) submit(text string) []uiEvent {
	text = strings.TrimSpace(text)
	if text == "" || s.state.running {
		return nil
	}
	s.state.running = true
	events := []uiEvent{eventInputClear{}}
	events = append(events, s.add(roleUser, text)...)
	events = append(events, s.add(roleAssistant, "")...)
	return append(events, eventReplyStart{prompt: text})
}

// escape stops the reply that is streaming, or quits when none is.
func (s *service) escape() []uiEvent {
	if !s.state.running {
		return []uiEvent{eventQuit{}}
	}
	return []uiEvent{eventReplyStop{}}
}

// applyAgentEvent applies an event of the streaming reply, which is the last
// message.
func (s *service) applyAgentEvent(ev agentEvent) []uiEvent {
	switch e := ev.(type) {
	case replyDelta:
		i := len(s.state.messages) - 1
		s.state.messages[i].text += e.text
		return []uiEvent{eventMessageUpdated{i: i, msg: s.state.messages[i]}}
	case replyEnded:
		s.state.running = false
		if e.stopped {
			return s.add(roleNotice, "stopped")
		}
	}
	return nil
}

// add appends a message to the transcript.
func (s *service) add(r role, text string) []uiEvent {
	msg := message{role: r, text: text}
	s.state.messages = append(s.state.messages, msg)
	return []uiEvent{eventMessageAdded{i: len(s.state.messages) - 1, msg: msg}}
}
