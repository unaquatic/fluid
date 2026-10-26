package textbuf

import (
	"testing"

	"github.com/unaquatic/fluid/style"
)

func TestTextBuffer_SetText(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello\nworld")

	if tb.LineCount() != 2 {
		t.Fatalf("expected 2 lines, got %d", tb.LineCount())
	}
	if tb.LineTextAt(0) != "hello" {
		t.Errorf("line 0: expected %q, got %q", "hello", tb.LineTextAt(0))
	}
	if tb.LineTextAt(1) != "world" {
		t.Errorf("line 1: expected %q, got %q", "world", tb.LineTextAt(1))
	}
}

func TestTextBuffer_SetText_Empty(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("")

	if tb.LineCount() != 1 {
		t.Fatalf("expected 1 line, got %d", tb.LineCount())
	}
	if tb.LineTextAt(0) != "" {
		t.Errorf("expected empty line, got %q", tb.LineTextAt(0))
	}
}

func TestTextBuffer_SetSpans(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetSpans([]Span{
		{Text: "red text", Fg: style.ColorFromInts(255, 0, 0, 255)},
		{Text: "\n"},
		{Text: "blue text", Fg: style.ColorFromInts(0, 0, 255, 255)},
	})

	if tb.LineCount() != 2 {
		t.Fatalf("expected 2 lines, got %d", tb.LineCount())
	}
	if tb.LineTextAt(0) != "red text" {
		t.Errorf("line 0: expected %q, got %q", "red text", tb.LineTextAt(0))
	}
	if tb.LineTextAt(1) != "blue text" {
		t.Errorf("line 1: expected %q, got %q", "blue text", tb.LineTextAt(1))
	}
}

func TestTextBuffer_Append(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello")
	tb.Append(" world")

	if tb.LineCount() != 1 {
		t.Fatalf("expected 1 line, got %d", tb.LineCount())
	}
	if tb.LineTextAt(0) != "hello world" {
		t.Errorf("expected %q, got %q", "hello world", tb.LineTextAt(0))
	}
}

func TestTextBuffer_Append_MultiLine(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("a")
	tb.Append("\nb\nc")

	if tb.LineCount() != 3 {
		t.Fatalf("expected 3 lines, got %d", tb.LineCount())
	}
	if tb.LineTextAt(2) != "c" {
		t.Errorf("line 2: expected %q, got %q", "c", tb.LineTextAt(2))
	}
}

func TestTextBuffer_GetPlainText(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello\nworld\nfoo")

	got := tb.PlainText()
	if got != "hello\nworld\nfoo" {
		t.Errorf("expected %q, got %q", "hello\nworld\nfoo", got)
	}
}

func TestTextBuffer_LineWidth(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("abc")

	if tb.LineWidthAt(0) != 3 {
		t.Errorf("expected width 3, got %d", tb.LineWidthAt(0))
	}
}

func TestTextBuffer_LineWidth_WideChars(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("日本語")

	if tb.LineWidthAt(0) != 6 {
		t.Errorf("expected width 6, got %d (CJK characters are 2 columns each)", tb.LineWidthAt(0))
	}
}

func TestTextBuffer_Clear(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello\nworld")
	tb.Clear()

	if tb.LineCount() != 1 {
		t.Fatalf("expected 1 line, got %d", tb.LineCount())
	}
	if tb.LineTextAt(0) != "" {
		t.Errorf("expected empty, got %q", tb.LineTextAt(0))
	}
}

func TestTextBuffer_Epoch(t *testing.T) {
	tb := NewTextBuffer()
	e0 := tb.Epoch()
	tb.SetText("x")
	e1 := tb.Epoch()
	if e1 <= e0 {
		t.Error("epoch should increase after SetText")
	}
	tb.Append("y")
	e2 := tb.Epoch()
	if e2 <= e1 {
		t.Error("epoch should increase after Append")
	}
}

func TestTextBuffer_Highlights(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello world")

	tb.AddHighlight(0, Highlight{Start: 0, End: 5, Ref: 1})
	tb.AddHighlight(0, Highlight{Start: 6, End: 11, Ref: 2})

	hls := tb.LineHighlights(0)
	if len(hls) != 2 {
		t.Fatalf("expected 2 highlights, got %d", len(hls))
	}

	tb.RemoveHighlightsByRef(1)
	hls = tb.LineHighlights(0)
	if len(hls) != 1 {
		t.Fatalf("expected 1 highlight after removal, got %d", len(hls))
	}
	if hls[0].Ref != 2 {
		t.Errorf("expected ref=2, got %d", hls[0].Ref)
	}

	tb.ClearAllHighlights()
	hls = tb.LineHighlights(0)
	if len(hls) != 0 {
		t.Errorf("expected 0 highlights after clear, got %d", len(hls))
	}
}

func TestTextBuffer_GetTextRange(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello\nworld")
	// "hello" occupies offsets 0-4, the newline 5, and "world" 6-10.

	got := tb.TextRange(0, 5)
	if got != "hello" {
		t.Errorf("expected %q, got %q", "hello", got)
	}

	got = tb.TextRange(6, 11)
	if got != "world" {
		t.Errorf("expected %q, got %q", "world", got)
	}
}

func TestTextBuffer_Defaults(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetDefaultFg(style.ColorFromInts(255, 128, 0, 255))
	if tb.DefaultFg().R != 255 || tb.DefaultFg().G != 128 {
		t.Error("SetDefaultFg not applied")
	}
}

func TestTextBuffer_Length(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello\nworld")
	// Two five-rune lines and their newline total 11 runes.
	if tb.Length() != 11 {
		t.Errorf("expected Length()=11, got %d", tb.Length())
	}
}

func TestTextBuffer_Length_WideChars(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("日本") // 2 runes
	if tb.Length() != 2 {
		t.Errorf("expected Length()=2, got %d", tb.Length())
	}
}

func TestTextBuffer_ByteSize(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello\nworld")
	if tb.ByteSize() != 11 {
		t.Errorf("expected ByteSize()=11, got %d", tb.ByteSize())
	}
}

func TestTextBuffer_ByteSize_UTF8(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("日本") // 6 bytes (3 per CJK char)
	if tb.ByteSize() != 6 {
		t.Errorf("expected ByteSize()=6, got %d", tb.ByteSize())
	}
}

func TestTextBuffer_Reset(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello\nworld")
	tb.SetDefaultFg(style.ColorFromInts(255, 0, 0, 255))
	tb.SetTabWidth(8)
	tb.AddHighlight(0, Highlight{Start: 0, End: 3, Ref: 1})

	tb.Reset()

	if tb.LineCount() != 1 || tb.LineTextAt(0) != "" {
		t.Error("content should be empty after Reset")
	}
	if tb.DefaultFg() != style.ColorWhite {
		t.Error("defaultFg should be reset to white")
	}
	if tb.TabWidth() != 4 {
		t.Errorf("tabWidth should be 4, got %d", tb.TabWidth())
	}
	if tb.HighlightCount() != 0 {
		t.Error("highlights should be cleared")
	}
}

func TestTextBuffer_ResetDefaults(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetDefaultFg(style.ColorFromInts(255, 0, 0, 255))
	tb.SetDefaultBg(style.ColorFromInts(0, 0, 255, 255))
	tb.SetDefaultAttributes(0xFF)

	tb.ResetDefaults()

	if tb.DefaultFg() != style.ColorWhite {
		t.Error("fg should be reset")
	}
	if tb.DefaultBg() != style.ColorTransparent {
		t.Error("bg should be reset")
	}
	if tb.DefaultAttributes() != 0 {
		t.Error("attributes should be reset")
	}
}

func TestTextBuffer_LoadFile(t *testing.T) {
	tb := NewTextBuffer()
	// A missing file reports failure without replacing content.
	if err := tb.LoadFile("/tmp/nonexistent_fluid_test_file_xyz"); err == nil {
		t.Error("LoadFile should return an error for a non-existent file")
	}
}

func TestTextBuffer_GetTextRangeByCoords(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello\nworld\nfoo")

	got := tb.TextRangeByCoords(0, 0, 0, 5)
	if got != "hello" {
		t.Errorf("expected %q, got %q", "hello", got)
	}

	got = tb.TextRangeByCoords(0, 2, 0, 4)
	if got != "ll" {
		t.Errorf("expected %q, got %q", "ll", got)
	}

	// Extract a range spanning a newline.
	got = tb.TextRangeByCoords(0, 3, 1, 3)
	if got != "lo\nwor" {
		t.Errorf("expected %q, got %q", "lo\nwor", got)
	}
}

func TestTextBuffer_AddHighlightByCharRange(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello\nworld")
	// "hello" occupies offsets 0-4, the newline 5, and "world" 6-10.

	tb.AddHighlightByCharRange(Highlight{Start: 3, End: 8, Ref: 42})

	// A cross-line global range creates one highlight on each line.
	h0 := tb.LineHighlights(0)
	if len(h0) != 1 {
		t.Fatalf("expected 1 highlight on line 0, got %d", len(h0))
	}
	if h0[0].Start != 3 || h0[0].End != 5 {
		t.Errorf("line 0 highlight: expected 3-5, got %d-%d", h0[0].Start, h0[0].End)
	}

	h1 := tb.LineHighlights(1)
	if len(h1) != 1 {
		t.Fatalf("expected 1 highlight on line 1, got %d", len(h1))
	}
	if h1[0].Start != 0 || h1[0].End != 2 {
		t.Errorf("line 1 highlight: expected 0-2, got %d-%d", h1[0].Start, h1[0].End)
	}
	if h1[0].Ref != 42 {
		t.Errorf("expected ref=42, got %d", h1[0].Ref)
	}
}

func TestTextBuffer_HighlightCount(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello\nworld")
	tb.AddHighlight(0, Highlight{Start: 0, End: 3, Ref: 1})
	tb.AddHighlight(0, Highlight{Start: 3, End: 5, Ref: 2})
	tb.AddHighlight(1, Highlight{Start: 0, End: 5, Ref: 3})

	if tb.HighlightCount() != 3 {
		t.Errorf("expected 3, got %d", tb.HighlightCount())
	}
}
