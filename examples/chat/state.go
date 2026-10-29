package main

// state is the program's state: plain data, which the service reads and
// changes on the event loop.
type state struct {
	// messages is the transcript, oldest first.
	messages []message
	// running reports whether a reply is streaming. It stays set after Esc
	// until the reply's replyEnded arrives, so that no prompt starts while
	// the stopped reply's last words are still on their way.
	running bool
}

// message is one message of the transcript, identified by its index.
type message struct {
	role role
	text string
}

// role is who wrote a message.
type role int

const (
	roleUser      role = iota // a prompt
	roleAssistant             // a reply
	roleNotice                // a notice from the program, such as that a reply stopped
)
