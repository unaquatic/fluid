package textbuf

import (
	"testing"

	"github.com/unaquatic/fluid/style"
)

func TestTextBufferView_NoWrap(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello\nworld")

	v := NewTextSurface(tb)
	vlines := v.VirtualLines()

	if len(vlines) != 2 {
		t.Fatalf("expected 2 virtual lines, got %d", len(vlines))
	}
	if vlines[0].SourceLine != 0 || vlines[1].SourceLine != 1 {
		t.Errorf("source lines: got %d, %d", vlines[0].SourceLine, vlines[1].SourceLine)
	}
	if vlines[0].Width != 5 {
		t.Errorf("vline 0 width: expected 5, got %d", vlines[0].Width)
	}
	if vlines[1].Width != 5 {
		t.Errorf("vline 1 width: expected 5, got %d", vlines[1].Width)
	}
}

func TestTextBufferView_Viewport(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("line1\nline2\nline3\nline4\nline5")

	v := NewTextSurface(tb)
	v.SetViewport(&Viewport{X: 0, Y: 1, Width: 80, Height: 2})

	vlines := v.VirtualLines()
	if len(vlines) != 2 {
		t.Fatalf("expected 2 visible lines, got %d", len(vlines))
	}
	if vlines[0].SourceLine != 1 {
		t.Errorf("first visible line should be source line 1, got %d", vlines[0].SourceLine)
	}
	if vlines[1].SourceLine != 2 {
		t.Errorf("second visible line should be source line 2, got %d", vlines[1].SourceLine)
	}
}

func TestTextBufferView_WrapChar(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("abcdefghij") // 10 chars

	v := NewTextSurface(tb)
	v.SetWrapMode(WrapChar)
	v.SetViewport(&Viewport{Width: 4, Height: 10})

	vlines := v.VirtualLines()

	// 10 chars at width 4: "abcd", "efgh", "ij" = 3 virtual lines
	if len(vlines) != 3 {
		t.Fatalf("expected 3 virtual lines, got %d", len(vlines))
	}
	if vlines[0].Width != 4 {
		t.Errorf("vline 0 width: expected 4, got %d", vlines[0].Width)
	}
	if vlines[1].Width != 4 {
		t.Errorf("vline 1 width: expected 4, got %d", vlines[1].Width)
	}
	if vlines[2].Width != 2 {
		t.Errorf("vline 2 width: expected 2, got %d", vlines[2].Width)
	}

	// Check source col offsets
	if vlines[0].SourceColOffset != 0 {
		t.Errorf("vline 0 SourceColOffset: expected 0, got %d", vlines[0].SourceColOffset)
	}
	if vlines[1].SourceColOffset != 4 {
		t.Errorf("vline 1 SourceColOffset: expected 4, got %d", vlines[1].SourceColOffset)
	}
	if vlines[2].SourceColOffset != 8 {
		t.Errorf("vline 2 SourceColOffset: expected 8, got %d", vlines[2].SourceColOffset)
	}
}

func TestTextBufferView_WrapWord(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello world foo")

	v := NewTextSurface(tb)
	v.SetWrapMode(WrapWord)
	v.SetViewport(&Viewport{Width: 10, Height: 10})

	vlines := v.VirtualLines()

	// "hello world foo" at width 10:
	// "hello " (6 cols) + "worl" would be 10 but "world" is 5+space
	// Actually: "hello " fits, then "world " fits up to 11 but break at "world "
	// "hello wor" is 9, then "ld" would make 11... word wrap should break at "hello "
	// Expected: "hello " (line 1), "world foo" (line 2) or "world " + "foo"
	// With width 10: "hello " = 6 cols, "world" would make 11 > 10,
	// so break after "hello " -> "hello " then "world foo" = 9 cols fits.

	if len(vlines) < 2 {
		t.Fatalf("expected at least 2 virtual lines, got %d", len(vlines))
	}

	// All virtual lines should be from source line 0
	for i, vl := range vlines {
		if vl.SourceLine != 0 {
			t.Errorf("vline %d: expected source 0, got %d", i, vl.SourceLine)
		}
	}
}

func TestTextBufferView_EmptyLine(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("a\n\nb")

	v := NewTextSurface(tb)
	vlines := v.VirtualLines()

	if len(vlines) != 3 {
		t.Fatalf("expected 3 virtual lines, got %d", len(vlines))
	}
	if vlines[1].Width != 0 {
		t.Errorf("empty line should have width 0, got %d", vlines[1].Width)
	}
}

func TestTextBufferView_Selection(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello")

	v := NewTextSurface(tb)
	bg := style.ColorFromInts(0, 0, 255, 255)
	v.SetSelection(&TextSelection{Start: 1, End: 4, BgColor: &bg})

	sel := v.Selection()
	if sel == nil {
		t.Fatal("expected selection")
	}
	if sel.Start != 1 || sel.End != 4 {
		t.Errorf("selection range: expected 1-4, got %d-%d", sel.Start, sel.End)
	}

	v.ResetSelection()
	if v.Selection() != nil {
		t.Error("selection should be nil after reset")
	}
}

func TestTextBufferView_LineInfo(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("aaa\nbb\nc")

	v := NewTextSurface(tb)
	info := v.CachedLineInfo()

	if len(info.Widths) != 3 {
		t.Fatalf("expected 3 line widths, got %d", len(info.Widths))
	}
	if info.Widths[0] != 3 {
		t.Errorf("line 0 width: expected 3, got %d", info.Widths[0])
	}
	if info.Widths[1] != 2 {
		t.Errorf("line 1 width: expected 2, got %d", info.Widths[1])
	}
	if info.Widths[2] != 1 {
		t.Errorf("line 2 width: expected 1, got %d", info.Widths[2])
	}
	if info.MaxWidth != 3 {
		t.Errorf("max width: expected 3, got %d", info.MaxWidth)
	}
}

func TestTextBufferView_StyledText(t *testing.T) {
	tb := NewTextBuffer()
	red := style.ColorFromInts(255, 0, 0, 255)
	blue := style.ColorFromInts(0, 0, 255, 255)
	tb.SetSpans([]Span{
		{Text: "red", Fg: red},
		{Text: "blue", Fg: blue},
	})

	v := NewTextSurface(tb)
	vlines := v.VirtualLines()

	if len(vlines) != 1 {
		t.Fatalf("expected 1 virtual line, got %d", len(vlines))
	}
	if len(vlines[0].Chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(vlines[0].Chunks))
	}
	if vlines[0].Chunks[0].Fg != red {
		t.Error("chunk 0 should be red")
	}
	if vlines[0].Chunks[1].Fg != blue {
		t.Error("chunk 1 should be blue")
	}
}

func TestTextBufferView_CacheInvalidation(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello")

	v := NewTextSurface(tb)
	vlines := v.VirtualLines()
	if len(vlines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(vlines))
	}

	tb.SetText("a\nb\nc")
	vlines = v.VirtualLines()
	if len(vlines) != 3 {
		t.Errorf("expected 3 lines after buffer change, got %d", len(vlines))
	}
}

func TestTextBufferView_WrapChar_MultiLine(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("abcdef\nxy")

	v := NewTextSurface(tb)
	v.SetWrapMode(WrapChar)
	v.SetViewport(&Viewport{Width: 3, Height: 20})

	vlines := v.VirtualLines()

	// "abcdef" at width 3: "abc", "def" = 2 vlines
	// "xy" at width 3: "xy" = 1 vline
	// Total: 3 vlines
	if len(vlines) != 3 {
		t.Fatalf("expected 3 virtual lines, got %d", len(vlines))
	}
	if vlines[0].SourceLine != 0 {
		t.Errorf("vline 0 source: expected 0, got %d", vlines[0].SourceLine)
	}
	if vlines[2].SourceLine != 1 {
		t.Errorf("vline 2 source: expected 1, got %d", vlines[2].SourceLine)
	}
}

// ---------------------------------------------------------------------------
// Tests for new TextSurface methods
// ---------------------------------------------------------------------------

func TestTextBufferView_SetWrapWidth(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("abcdefghijklmnop") // 16 chars

	v := NewTextSurface(tb)
	v.SetWrapMode(WrapChar)
	v.SetViewport(&Viewport{Width: 80, Height: 20}) // viewport is wide

	// Without explicit wrapWidth, uses viewport width (80)
	vlines := v.VirtualLines()
	if len(vlines) != 1 {
		t.Fatalf("expected 1 virtual line with wide viewport, got %d", len(vlines))
	}

	// Set explicit wrap width to 4
	v.SetWrapWidth(4)
	vlines = v.VirtualLines()
	// 16 chars at width 4: 4 virtual lines
	if len(vlines) != 4 {
		t.Fatalf("expected 4 virtual lines with wrapWidth=4, got %d", len(vlines))
	}
	for _, vl := range vlines {
		if vl.Width != 4 {
			t.Errorf("expected width 4, got %d", vl.Width)
		}
	}

	// Reset wrapWidth to 0 -> fallback to viewport width
	v.SetWrapWidth(0)
	vlines = v.VirtualLines()
	if len(vlines) != 1 {
		t.Fatalf("expected 1 virtual line after resetting wrapWidth, got %d", len(vlines))
	}
}

func TestTextBufferView_SetTruncate(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello")

	v := NewTextSurface(tb)

	// Default truncation should be false
	if v.truncate {
		t.Error("expected truncation to be disabled by default")
	}

	v.SetTruncate(true)
	if !v.truncate {
		t.Error("expected truncation to be enabled after SetTruncate(true)")
	}

	v.SetTruncate(false)
	if v.truncate {
		t.Error("expected truncation to be disabled after SetTruncate(false)")
	}
}

func TestTextBufferView_SetTabIndicator(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello")

	v := NewTextSurface(tb)

	if v.tabIndicator != 0 {
		t.Errorf("expected tab indicator to be 0 by default, got %d", v.tabIndicator)
	}

	v.SetTabIndicator('→')
	if v.tabIndicator != '→' {
		t.Errorf("expected tab indicator '→', got %c", v.tabIndicator)
	}

	// Setting should mark dirty for re-layout
	v.SetTabIndicator('·')
	if v.tabIndicator != '·' {
		t.Errorf("expected tab indicator '·', got %c", v.tabIndicator)
	}

	v.SetTabIndicator(0)
	if v.tabIndicator != 0 {
		t.Error("expected tab indicator to be cleared")
	}
}

func TestTextBufferView_SetTabIndicatorColor(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello")

	v := NewTextSurface(tb)

	if v.tabIndicatorColor != nil {
		t.Error("expected tab indicator color to be nil by default")
	}

	c := style.ColorFromInts(128, 128, 128, 255)
	v.SetTabIndicatorColor(c)
	if v.tabIndicatorColor == nil {
		t.Fatal("expected tab indicator color to be set")
	}
	if *v.tabIndicatorColor != c {
		t.Errorf("expected color %v, got %v", c, *v.tabIndicatorColor)
	}
}

func TestTextBufferView_GetLogicalLineInfo(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello\nab\nworld!")

	v := NewTextSurface(tb)
	// Enable wrapping to ensure LogicalLineInfo ignores it
	v.SetWrapMode(WrapChar)
	v.SetViewport(&Viewport{Width: 3, Height: 20})

	info := v.LogicalLineInfo()

	// Should always have 3 entries regardless of wrapping
	if len(info.Widths) != 3 {
		t.Fatalf("expected 3 logical lines, got %d", len(info.Widths))
	}
	if info.Widths[0] != 5 {
		t.Errorf("line 0 width: expected 5, got %d", info.Widths[0])
	}
	if info.Widths[1] != 2 {
		t.Errorf("line 1 width: expected 2, got %d", info.Widths[1])
	}
	if info.Widths[2] != 6 {
		t.Errorf("line 2 width: expected 6, got %d", info.Widths[2])
	}
	if info.MaxWidth != 6 {
		t.Errorf("max width: expected 6, got %d", info.MaxWidth)
	}

	// Sources should be 0, 1, 2
	for i, s := range info.Sources {
		if s != i {
			t.Errorf("source[%d]: expected %d, got %d", i, i, s)
		}
	}

	// Starts: line 0 starts at 0; line 1 starts at 0+5+1=6; line 2 starts at 6+2+1=9
	expectedStarts := []int{0, 6, 9}
	for i, exp := range expectedStarts {
		if info.Starts[i] != exp {
			t.Errorf("start[%d]: expected %d, got %d", i, exp, info.Starts[i])
		}
	}
}

func TestTextBufferView_SetLocalSelection(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello world")

	v := NewTextSurface(tb)
	v.SetViewport(&Viewport{Width: 80, Height: 10})

	bg := style.ColorFromInts(0, 0, 255, 255)
	fg := style.ColorFromInts(255, 255, 255, 255)

	// Select "ello" (columns 1-5 on row 0)
	ok := v.SetLocalSelection(1, 0, 5, 0, &bg, &fg)
	if !ok {
		t.Fatal("expected SetLocalSelection to return true")
	}

	sel := v.Selection()
	if sel == nil {
		t.Fatal("expected selection to be set")
	}
	if sel.Start > sel.End {
		t.Errorf("expected Start <= End, got %d > %d", sel.Start, sel.End)
	}
	if sel.BgColor == nil || *sel.BgColor != bg {
		t.Error("expected selection BgColor to match")
	}
	if sel.FgColor == nil || *sel.FgColor != fg {
		t.Error("expected selection FgColor to match")
	}
}

func TestTextBufferView_SetLocalSelection_Invalid(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello")

	v := NewTextSurface(tb)
	v.SetViewport(&Viewport{Width: 80, Height: 10})

	// Out-of-range Y coordinate
	ok := v.SetLocalSelection(0, 99, 5, 99, nil, nil)
	if ok {
		t.Error("expected SetLocalSelection to return false for out-of-range coords")
	}
}

func TestTextBufferView_UpdateSelection(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello world")

	v := NewTextSurface(tb)

	bg := style.ColorFromInts(0, 0, 255, 255)

	// No selection yet — should create one
	v.UpdateSelection(3, &bg, nil)
	sel := v.Selection()
	if sel == nil {
		t.Fatal("expected selection to be created")
	}
	if sel.Start != 3 || sel.End != 3 {
		t.Errorf("expected start=3, end=3, got start=%d, end=%d", sel.Start, sel.End)
	}

	// Update end
	v.UpdateSelection(8, nil, nil)
	sel = v.Selection()
	if sel.End != 8 {
		t.Errorf("expected end=8, got %d", sel.End)
	}
	// BgColor should still be set from before (only nil doesn't overwrite)
	if sel.BgColor == nil || *sel.BgColor != bg {
		t.Error("expected BgColor to be preserved")
	}
}

func TestTextBufferView_UpdateLocalSelection(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello world")

	v := NewTextSurface(tb)
	v.SetViewport(&Viewport{Width: 80, Height: 10})

	ok := v.UpdateLocalSelection(2, 0, 8, 0, nil, nil)
	if !ok {
		t.Fatal("expected UpdateLocalSelection to return true")
	}
	sel := v.Selection()
	if sel == nil {
		t.Fatal("expected selection to be set")
	}
}

func TestTextBufferView_ResetLocalSelection(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello")

	v := NewTextSurface(tb)
	v.SetViewport(&Viewport{Width: 80, Height: 10})

	v.SetLocalSelection(0, 0, 3, 0, nil, nil)
	if v.Selection() == nil {
		t.Fatal("expected selection to be set")
	}

	v.ResetLocalSelection()
	if v.Selection() != nil {
		t.Error("expected selection to be nil after ResetLocalSelection")
	}
}

func TestTextBufferView_GetSelectedText(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello world")

	v := NewTextSurface(tb)

	// No selection
	if v.SelectedText() != "" {
		t.Error("expected empty string with no selection")
	}

	// Select "world" (offsets 6-11)
	v.SetSelection(&TextSelection{Start: 6, End: 11})
	got := v.SelectedText()
	if got != "world" {
		t.Errorf("expected 'world', got %q", got)
	}

	// Reversed selection (start > end) should still work
	v.SetSelection(&TextSelection{Start: 11, End: 6})
	got = v.SelectedText()
	if got != "world" {
		t.Errorf("expected 'world' with reversed selection, got %q", got)
	}
}

func TestTextBufferView_GetSelectedText_MultiLine(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello\nworld")

	v := NewTextSurface(tb)

	// Select "llo\nwor" — offsets: h=0, e=1, l=2, l=3, o=4, \n=5, w=6, o=7, r=8
	v.SetSelection(&TextSelection{Start: 2, End: 9})
	got := v.SelectedText()
	if got != "llo\nwor" {
		t.Errorf("expected 'llo\\nwor', got %q", got)
	}
}

func TestTextBufferView_GetPlainText(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello\nworld")

	v := NewTextSurface(tb)
	got := v.PlainText()
	if got != "hello\nworld" {
		t.Errorf("expected 'hello\\nworld', got %q", got)
	}
}

func TestTextBufferView_MeasureForDimensions_NoWrap(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello\nab\nworld!")

	v := NewTextSurface(tb)
	// No wrap mode set

	m := v.MeasureForDimensions(80, 24)
	if m.LineCount != 3 {
		t.Errorf("expected 3 lines, got %d", m.LineCount)
	}
	if m.MaxWidth != 6 {
		t.Errorf("expected max width 6, got %d", m.MaxWidth)
	}
}

func TestTextBufferView_MeasureForDimensions_WithWrap(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("abcdefgh\nxy") // line 0: 8 chars, line 1: 2 chars

	v := NewTextSurface(tb)
	v.SetWrapMode(WrapChar)

	m := v.MeasureForDimensions(3, 20)
	// "abcdefgh" at width 3: ceil(8/3) = 3 virtual lines
	// "xy" at width 3: 1 virtual line
	// Total: 4
	if m.LineCount != 4 {
		t.Errorf("expected 4 lines with wrap, got %d", m.LineCount)
	}
	// MaxWidth is the widest *wrapped* virtual line (3), not the source (8).
	if m.MaxWidth != 3 {
		t.Errorf("expected max width 3 (wrapped), got %d", m.MaxWidth)
	}
}

func TestTextBufferView_MeasureForDimensions_ExplicitWrapWidth(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("abcdefghij") // 10 chars

	v := NewTextSurface(tb)
	v.SetWrapMode(WrapChar)
	v.SetWrapWidth(5) // explicit wrap width

	// MeasureForDimensions uses the explicit wrapWidth if set
	m := v.MeasureForDimensions(80, 20)
	// 10 chars at width 5 (explicit wrapWidth overrides the provided width): 2 lines
	if m.LineCount != 2 {
		t.Errorf("expected 2 lines with explicit wrapWidth=5, got %d", m.LineCount)
	}
}

func TestTextBufferView_GetVirtualLineCount(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("abcdef\nxy")

	v := NewTextSurface(tb)
	v.SetWrapMode(WrapChar)
	v.SetViewport(&Viewport{Width: 3, Height: 20})

	// "abcdef" → 2 virtual lines, "xy" → 1 virtual line
	if v.VirtualLineCount() != 3 {
		t.Errorf("expected 3 virtual lines, got %d", v.VirtualLineCount())
	}
}

func TestTextBufferView_ViewportClipping(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("line1\nline2\nline3\nline4\nline5")

	v := NewTextSurface(tb)
	v.SetViewport(&Viewport{X: 0, Y: 3, Width: 80, Height: 10})

	// Only lines 3 and 4 visible (Y=3, only 2 lines left)
	vlines := v.VirtualLines()
	if len(vlines) != 2 {
		t.Fatalf("expected 2 visible lines, got %d", len(vlines))
	}
	if vlines[0].SourceLine != 3 {
		t.Errorf("expected source line 3, got %d", vlines[0].SourceLine)
	}
	if vlines[1].SourceLine != 4 {
		t.Errorf("expected source line 4, got %d", vlines[1].SourceLine)
	}
}

func TestTextBufferView_SetViewportSize(t *testing.T) {
	tb := NewTextBuffer()
	tb.SetText("hello")

	v := NewTextSurface(tb)

	// No viewport initially
	if v.Viewport() != nil {
		t.Error("expected nil viewport initially")
	}

	// SetViewportSize should create a viewport
	v.SetViewportSize(40, 10)
	vp := v.Viewport()
	if vp == nil {
		t.Fatal("expected viewport to be created")
	}
	if vp.Width != 40 || vp.Height != 10 {
		t.Errorf("expected 40x10, got %dx%d", vp.Width, vp.Height)
	}

	// Update existing viewport dimensions, preserving offset
	v.SetViewport(&Viewport{X: 2, Y: 3, Width: 40, Height: 10})
	v.SetViewportSize(60, 20)
	vp = v.Viewport()
	if vp.X != 2 || vp.Y != 3 {
		t.Errorf("expected offset (2,3) preserved, got (%d,%d)", vp.X, vp.Y)
	}
	if vp.Width != 60 || vp.Height != 20 {
		t.Errorf("expected 60x20, got %dx%d", vp.Width, vp.Height)
	}
}
