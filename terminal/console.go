package terminal

import (
	"io"
	"unicode/utf16"
	"unicode/utf8"
)

// Kinds of event that a ConsoleRecord holds.
const (
	// ConsoleKeyEvent is a key press or release.
	ConsoleKeyEvent = 0x0001
	// ConsoleBufferSizeEvent tells that the console's screen buffer
	// changed size, as when its window is resized.
	ConsoleBufferSizeEvent = 0x0004
)

// ConsoleRecord is an input record of a Windows console, laid out as the
// INPUT_RECORD that ReadConsoleInputW fills.
type ConsoleRecord struct {
	// EventType is the kind of event the record holds, such as
	// ConsoleKeyEvent or ConsoleBufferSizeEvent.
	EventType uint16
	_         uint16
	// Event holds the event, laid out as the record of its kind.
	Event [4]uint32
}

// KeyDown reports whether the key event that r holds is a key press.
func (r ConsoleRecord) KeyDown() bool {
	return r.Event[0] != 0
}

// Char returns the UTF-16 code unit that the key event r holds stands for,
// which may be half of a surrogate pair.
func (r ConsoleRecord) Char() uint16 {
	return uint16(r.Event[2] >> 16)
}

// ConsoleDecoder turns the input records of a Windows console in virtual
// terminal input mode into the bytes that a terminal sends: the text of its
// key presses, which hold the escape sequences of special keys, in UTF-8.
// A character beyond the Basic Multilingual Plane comes in two records, one
// for each half of its surrogate pair, which the decoder joins, even across
// calls. The zero ConsoleDecoder is ready to use.
type ConsoleDecoder struct {
	// high holds a first half of a surrogate pair, by whether its record
	// was a key release (0) or press (1), until its second half comes.
	high [2]uint16
}

// Decode appends to dst the bytes that records stand for, and reports
// whether one of them tells that the console changed size. Every key press
// counts, Ctrl-Z and the NUL of Ctrl-Space too. A key release counts only
// for a surrogate pair, as the console may send one on a release alone. A
// lone half of a pair decodes as utf8.RuneError.
func (d *ConsoleDecoder) Decode(dst []byte, records []ConsoleRecord) (_ []byte, resized bool) {
	for _, r := range records {
		switch r.EventType {
		case ConsoleKeyEvent:
			dst = d.decodeKey(dst, r)
		case ConsoleBufferSizeEvent:
			resized = true
		}
	}
	return dst, resized
}

// decodeKey appends to dst what the key event r stands for.
func (d *ConsoleDecoder) decodeKey(dst []byte, r ConsoleRecord) []byte {
	down := 0
	if r.KeyDown() {
		down = 1
	}
	c := rune(r.Char())
	high := rune(d.high[down])
	d.high[down] = 0
	switch {
	case 0xD800 <= c && c < 0xDC00:
		if high != 0 {
			dst = utf8.AppendRune(dst, utf8.RuneError)
		}
		d.high[down] = uint16(c)
		return dst
	case 0xDC00 <= c && c < 0xE000:
		if high == 0 {
			return utf8.AppendRune(dst, utf8.RuneError)
		}
		return utf8.AppendRune(dst, utf16.DecodeRune(high, c))
	}
	if high != 0 {
		dst = utf8.AppendRune(dst, utf8.RuneError)
	}
	if down == 1 {
		dst = utf8.AppendRune(dst, c)
	}
	return dst
}

// consoleInput reads the input of a Windows console as the bytes that a
// terminal sends, and tells when the console changes size, which a Windows
// console reports among its input rather than with a signal.
type consoleInput interface {
	io.ReadCloser
	// Resized receives when the console's size may have changed.
	Resized() <-chan struct{}
}
