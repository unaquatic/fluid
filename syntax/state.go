package syntax

import "slices"

// State is where a lexer stands at the start of a line: inside a block
// comment, a multiline string, or an embedded language, say. The zero State
// is the start of source, outside of every construct.
//
// A State is a stack of Frames, which only the Lexer that made it reads.
// States compare with ==, and two lines that start in equal States lex
// alike.
type State string

// Frame is one level of a State: a construct a lexer is inside of.
type Frame struct {
	// Mode identifies the construct, in values of the lexer's own.
	Mode uint8
	// N is a count the construct needs, such as the nesting depth of a
	// block comment.
	N uint8
	// Delim is text the construct needs, such as the word that ends a
	// heredoc.
	Delim string
}

// Push returns st with f on top. A Delim longer than 255 bytes is cut
// short.
func (st State) Push(f Frame) State {
	return State(f.append([]byte(st)))
}

// append appends the encoding of f to b and returns the extended slice:
// its Delim, cut to 255 bytes, then the Delim's length, N, and Mode.
func (f Frame) append(b []byte) []byte {
	d := f.Delim[:min(len(f.Delim), 255)]
	b = append(b, d...)
	return append(b, byte(len(d)), f.N, f.Mode)
}

// Top returns the frame on top of st and the State below it. It reports
// false for the zero State and for one that Push did not make.
func (st State) Top() (f Frame, below State, ok bool) {
	n := len(st)
	if n < 3 {
		return Frame{}, "", false
	}
	d := int(st[n-3])
	if n-3 < d {
		return Frame{}, "", false
	}
	f = Frame{Mode: st[n-1], N: st[n-2], Delim: string(st[n-3-d : n-3])}
	return f, st[:n-3-d], true
}

// Frames is a stack of frames a lexer decodes a State into, innermost last.
type Frames []Frame

// AppendFrames appends the frames of st to dst, outermost first, and
// returns the extended stack. A malformed part of st is dropped.
func AppendFrames(dst Frames, st State) Frames {
	n := len(dst)
	for {
		f, below, ok := st.Top()
		if !ok {
			break
		}
		dst = append(dst, f)
		st = below
	}
	// Top yields the innermost frame first.
	slices.Reverse(dst[n:])
	return dst
}

// State returns the State that holds fs, outermost first.
func (fs Frames) State() State {
	if len(fs) == 0 {
		return ""
	}
	n := 0
	for _, f := range fs {
		n += min(len(f.Delim), 255) + 3
	}
	b := make([]byte, 0, n)
	for _, f := range fs {
		b = f.append(b)
	}
	return State(b)
}

// Top returns a pointer to the innermost frame of fs, or nil if fs is
// empty.
func (fs Frames) Top() *Frame {
	if len(fs) == 0 {
		return nil
	}
	return &fs[len(fs)-1]
}
