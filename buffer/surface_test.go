package buffer

import (
	"strings"
	"testing"

	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/textbuf"
)

func TestDrawTextBufferView_Basic(t *testing.T) {
	tb := textbuf.NewTextBuffer()
	tb.SetText("hello")

	surface := textbuf.NewTextSurface(tb)
	buf := New(10, 3)
	buf.DrawTextSurface(surface, 0, 0)

	text := ""
	for x := range 5 {
		c := buf.Cell(x, 0)
		text += string(c.Char)
	}
	if text != "hello" {
		t.Errorf("expected %q, got %q", "hello", text)
	}
}

func TestDrawTextBufferView_MultiLine(t *testing.T) {
	tb := textbuf.NewTextBuffer()
	tb.SetText("abc\ndef")

	surface := textbuf.NewTextSurface(tb)
	buf := New(10, 3)
	buf.DrawTextSurface(surface, 0, 0)

	for x, expected := range "abc" {
		c := buf.Cell(x, 0)
		if c.Char != expected {
			t.Errorf("cell (%d,0): expected %q, got %q", x, string(expected), string(c.Char))
		}
	}
	for x, expected := range "def" {
		c := buf.Cell(x, 1)
		if c.Char != expected {
			t.Errorf("cell (%d,1): expected %q, got %q", x, string(expected), string(c.Char))
		}
	}
}

func TestDrawTextBufferView_Offset(t *testing.T) {
	tb := textbuf.NewTextBuffer()
	tb.SetText("hi")

	surface := textbuf.NewTextSurface(tb)
	buf := New(10, 5)
	buf.DrawTextSurface(surface, 3, 2)

	c := buf.Cell(3, 2)
	if c.Char != 'h' {
		t.Errorf("expected 'h' at (3,2), got %q", string(c.Char))
	}
	c = buf.Cell(4, 2)
	if c.Char != 'i' {
		t.Errorf("expected 'i' at (4,2), got %q", string(c.Char))
	}

	// Drawing at an offset leaves the origin unchanged.
	c = buf.Cell(0, 0)
	if c.Char != ' ' {
		t.Errorf("expected space at (0,0), got %q", string(c.Char))
	}
}

func TestDrawTextBufferView_StyledText(t *testing.T) {
	tb := textbuf.NewTextBuffer()
	red := style.ColorFromInts(255, 0, 0, 255)
	blue := style.ColorFromInts(0, 0, 255, 255)
	tb.SetSpans([]textbuf.Span{
		{Text: "RR", Fg: red},
		{Text: "BB", Fg: blue},
	})

	surface := textbuf.NewTextSurface(tb)
	buf := New(10, 3)
	buf.DrawTextSurface(surface, 0, 0)

	c0 := buf.Cell(0, 0)
	if c0.Fg.R != 255 || c0.Fg.A != 255 {
		t.Errorf("cell (0,0) fg should be red, got %+v", c0.Fg)
	}

	c2 := buf.Cell(2, 0)
	if c2.Fg.B != 255 || c2.Fg.A != 255 {
		t.Errorf("cell (2,0) fg should be blue, got %+v", c2.Fg)
	}
}

func TestDrawTextBufferView_Selection(t *testing.T) {
	tb := textbuf.NewTextBuffer()
	tb.SetText("hello")

	surface := textbuf.NewTextSurface(tb)
	selBg := style.ColorFromInts(0, 0, 255, 255)
	surface.SetSelection(&textbuf.TextSelection{Start: 1, End: 4, BgColor: &selBg})

	buf := New(10, 3)
	buf.DrawTextSurface(surface, 0, 0)

	// Selection [1,4) highlights only the middle three cells.
	c0 := buf.Cell(0, 0)
	if c0.Bg == selBg {
		t.Error("cell 0 should not have selection bg")
	}

	c1 := buf.Cell(1, 0)
	if c1.Bg != selBg {
		t.Errorf("cell 1 should have selection bg, got %+v", c1.Bg)
	}

	c3 := buf.Cell(3, 0)
	if c3.Bg != selBg {
		t.Errorf("cell 3 should have selection bg, got %+v", c3.Bg)
	}

	c4 := buf.Cell(4, 0)
	if c4.Bg == selBg {
		t.Error("cell 4 should not have selection bg")
	}
}

func TestDrawEditBufferView_Basic(t *testing.T) {
	tb := textbuf.NewTextBuffer()
	eb := textbuf.NewEditBuffer(tb)
	eb.SetText("test line")

	ev := textbuf.NewEditSurface(eb, 80, 24)

	buf := New(20, 5)
	buf.DrawEditSurface(ev, 0, 0)

	text := ""
	for x := range 9 {
		c := buf.Cell(x, 0)
		text += string(c.Char)
	}
	if text != "test line" {
		t.Errorf("expected %q, got %q", "test line", text)
	}
}

func TestDrawEditBufferView_WithViewport(t *testing.T) {
	tb := textbuf.NewTextBuffer()
	eb := textbuf.NewEditBuffer(tb)
	var text strings.Builder
	for i := range 10 {
		if i > 0 {
			text.WriteString("\n")
		}
		text.WriteString(string(rune('a' + i)))
	}
	eb.SetText(text.String()) // One letter per line, from "a" through "j".

	ev := textbuf.NewEditSurface(eb, 80, 3)

	buf := New(10, 3)
	buf.DrawEditSurface(ev, 0, 0)

	// The three-row viewport exposes the first three lines.
	c0 := buf.Cell(0, 0)
	if c0.Char != 'a' {
		t.Errorf("expected 'a' at (0,0), got %q", string(c0.Char))
	}
	c2 := buf.Cell(0, 2)
	if c2.Char != 'c' {
		t.Errorf("expected 'c' at (0,2), got %q", string(c2.Char))
	}
}

func TestDrawTextBufferView_Clipping(t *testing.T) {
	tb := textbuf.NewTextBuffer()
	tb.SetText("abcdefghij") // Ten single-column characters.

	surface := textbuf.NewTextSurface(tb)
	buf := New(5, 3) // Only the first five characters fit.
	buf.DrawTextSurface(surface, 0, 0)

	for x := range 5 {
		expected := rune('a' + x)
		c := buf.Cell(x, 0)
		if c.Char != expected {
			t.Errorf("cell (%d,0): expected %q, got %q", x, string(expected), string(c.Char))
		}
	}
}

func TestDrawTextBufferView_Empty(t *testing.T) {
	tb := textbuf.NewTextBuffer()
	tb.SetText("")

	surface := textbuf.NewTextSurface(tb)
	buf := New(10, 3)
	buf.DrawTextSurface(surface, 0, 0)

	// Drawing an empty surface leaves the buffer unchanged.
	c := buf.Cell(0, 0)
	if c.Char != ' ' {
		t.Errorf("expected space at (0,0) for empty buffer, got %q", string(c.Char))
	}
}

func TestDrawTextBufferView_NegativeY(t *testing.T) {
	tb := textbuf.NewTextBuffer()
	tb.SetText("aaa\nbbb\nccc")

	surface := textbuf.NewTextSurface(tb)
	buf := New(10, 3)
	buf.DrawTextSurface(surface, 0, -1)

	// A y-offset of -1 places the first line above the buffer and the second at row 0.
	c := buf.Cell(0, 0)
	if c.Char != 'b' {
		t.Errorf("expected 'b' at (0,0), got %q", string(c.Char))
	}
}
