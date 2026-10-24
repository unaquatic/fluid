package terminal

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
)

// startedTestRenderer returns an inline test renderer that reports completed
// setup, so mode changes reach the terminal.
func startedTestRenderer() (*Renderer, *bytes.Buffer) {
	r, outBuf := newInlineTestRenderer()
	r.controlState = StateAutoStarted
	return r, outBuf
}

func TestSetInlineMode_LeavesAlternateScreen(t *testing.T) {
	r, outBuf := startedTestRenderer()
	r.config.InlineMode = false
	r.backend.EnterAlternateScreen()
	outBuf.Reset()

	r.SetInlineMode(true)

	if !strings.Contains(outBuf.String(), "\x1b[?1049l") {
		t.Error("expected the alternate screen to be left")
	}
	if r.backend.AlternateScreen() {
		t.Error("expected AlternateScreen=false")
	}
	if !r.config.InlineMode {
		t.Error("expected InlineMode=true")
	}
}

func TestSetInlineMode_EntersAlternateScreen(t *testing.T) {
	r, outBuf := startedTestRenderer()

	r.SetInlineMode(false)

	if !strings.Contains(outBuf.String(), "\x1b[?1049h") {
		t.Error("expected the alternate screen to be entered")
	}
	if !r.backend.AlternateScreen() {
		t.Error("expected AlternateScreen=true")
	}
	if r.config.InlineMode {
		t.Error("expected InlineMode=false")
	}
}

func TestSetInlineMode_UnchangedModeWritesNothing(t *testing.T) {
	r, outBuf := startedTestRenderer()

	r.SetInlineMode(true)

	if outBuf.Len() != 0 {
		t.Errorf("expected no output, got %q", outBuf.String())
	}
}

func TestSetInlineMode_BeforeSetupWritesNothing(t *testing.T) {
	r, outBuf := newInlineTestRenderer()

	r.SetInlineMode(false)

	if outBuf.Len() != 0 {
		t.Errorf("expected no output, got %q", outBuf.String())
	}
	if r.config.InlineMode {
		t.Error("expected the mode to be recorded")
	}
}

// TestSetInlineMode_KeepsTrackedTerminalState checks that a running renderer
// keeps the last inline frame and its anchor. The alternate screen leaves that
// output on the normal screen, so a return can compare against it.
func TestSetInlineMode_KeepsTrackedTerminalState(t *testing.T) {
	r, _ := startedTestRenderer()
	r.InlineFlush(rowsBuffer(10, strings.Repeat("A", testTermHeight+5), 5))
	if r.inlineViewportTop == 0 {
		t.Fatal("expected the tall frame to advance the viewport")
	}
	top, cursor := r.inlineViewportTop, r.inlineCursorRow

	r.SetInlineMode(false)

	if r.inlineCurrentBuf == nil {
		t.Error("expected the inline frame to be kept")
	}
	if r.inlineViewportTop != top || r.inlineCursorRow != cursor || r.inlineTermHeight != testTermHeight {
		t.Errorf("anchor is top=%d cursor=%d height=%d, want top=%d cursor=%d height=%d",
			r.inlineViewportTop, r.inlineCursorRow, r.inlineTermHeight, top, cursor, testTermHeight)
	}
	if !r.invalidated {
		t.Error("expected the next frame to be drawn in full")
	}
}

// TestSetInlineMode_BeforeSetupDiscardsTrackedTerminalState checks that a
// renderer that is not running drops the inline frame, since nothing ties it
// to the screen.
func TestSetInlineMode_BeforeSetupDiscardsTrackedTerminalState(t *testing.T) {
	r, _ := newInlineTestRenderer()
	r.InlineFlush(rowsBuffer(10, strings.Repeat("A", testTermHeight+5), 5))

	r.SetInlineMode(false)

	if r.inlineCurrentBuf != nil {
		t.Error("expected inlineCurrentBuf=nil")
	}
	if r.inlineViewportTop != 0 || r.inlineCursorRow != 0 || r.inlineTermHeight != 0 {
		t.Errorf("expected the inline anchor to be cleared, got top=%d cursor=%d height=%d",
			r.inlineViewportTop, r.inlineCursorRow, r.inlineTermHeight)
	}
}

// shellPrompt stands for output the shell wrote before the renderer started,
// which inline output must leave alone.
const shellPrompt = "$ fluid"

// Glyphs for the rows drawn inline before the alternate screen is entered.
const (
	shortGlyphs = "abc"
	tallGlyphs  = "abcdefghijklmnopqrstuvwxy" // Taller than the terminal.
)

// glyphRowWidth is the width of the rows drawn by drawInline. Inline output
// leaves the final terminal column free.
const glyphRowWidth = testTermWidth - 1

// drawInline draws one row per glyph inline, below a shell prompt. It returns
// the renderer, its output buffer, which it leaves empty, the terminal that
// received the output, and the rows.
func drawInline(t *testing.T, glyphs string) (*Renderer, *bytes.Buffer, *disagreeingTerm, []*glyphRow) {
	t.Helper()
	r, out := startedTestRenderer()
	term := newDisagreeingTerm(testTermWidth, testTermHeight, func(rune) int { return 1 })
	term.feed(shellPrompt + "\r\n")

	var rows []*glyphRow
	for i, ch := range glyphs {
		row := newGlyphRow(strconv.Itoa(i), ch, glyphRowWidth, 1)
		rows = append(rows, row)
		r.root.Add(row)
	}
	r.renderInlineFrame()
	term.feed(out.String())
	out.Reset()
	return r, out, term, rows
}

// enterFullscreen draws rows inline as drawInline does, then switches to the
// alternate screen and draws a fullscreen frame. It returns what drawInline
// returns, with the output buffer again left empty.
func enterFullscreen(t *testing.T, glyphs string) (*Renderer, *bytes.Buffer, *disagreeingTerm, []*glyphRow) {
	t.Helper()
	r, out, term, rows := drawInline(t, glyphs)

	r.SetInlineMode(false)
	r.renderFrame()
	term.feed(out.String())
	out.Reset()
	return r, out, term, rows
}

// resizeDuring returns a change that resizes the terminal to w by h.
func resizeDuring(w, h int) func(*Renderer, []*glyphRow) {
	return func(r *Renderer, _ []*glyphRow) { r.processResize(w, h) }
}

// firstColumn returns the first character of each line, which identifies the
// row that drew it.
func firstColumn(lines []string) string {
	var sb strings.Builder
	for _, line := range lines {
		if line == "" {
			sb.WriteByte(' ')
			continue
		}
		sb.WriteByte(line[0])
	}
	return sb.String()
}

// TestSetInlineMode_RoundTrip covers the return from the alternate screen. The
// terminal brings back the normal screen with the earlier inline output on it,
// so the first inline frame must replace that output rather than draw a second
// copy below it.
func TestSetInlineMode_RoundTrip(t *testing.T) {
	appendRow := func(r *Renderer, _ []*glyphRow) {
		r.root.Add(newGlyphRow("appended", 'z', glyphRowWidth, 1))
	}
	changeFirstRow := func(_ *Renderer, rows []*glyphRow) {
		rows[0].setGlyph('A')
	}

	for _, tt := range []struct {
		name   string
		glyphs string
		during func(*Renderer, []*glyphRow)
		// want is the first column of the scrollback and the screen.
		want string
		// erase reports whether the return erases the screen and the
		// scrollback. A return that does not must leave unchanged rows alone.
		erase bool
	}{
		{name: "unchanged", glyphs: shortGlyphs, want: "$" + shortGlyphs},
		{name: "appended row", glyphs: shortGlyphs, during: appendRow, want: "$" + shortGlyphs + "z"},
		{name: "appended row after scrolling", glyphs: tallGlyphs, during: appendRow, want: "$" + tallGlyphs + "z"},
		{name: "changed row in the scrollback", glyphs: tallGlyphs, during: changeFirstRow, want: "A" + tallGlyphs[1:], erase: true},
		{name: "height resized", glyphs: shortGlyphs, during: resizeDuring(testTermWidth, testTermHeight-5), want: shortGlyphs, erase: true},
		{name: "width resized", glyphs: shortGlyphs, during: resizeDuring(testTermWidth-5, testTermHeight), want: shortGlyphs, erase: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r, out, term, rows := enterFullscreen(t, tt.glyphs)
			if tt.during != nil {
				tt.during(r, rows)
			}

			r.SetInlineMode(true)
			r.renderInlineFrame()
			term.feed(out.String())

			if got := firstColumn(term.lines()); got != tt.want {
				t.Errorf("terminal rows begin %q, want %q", got, tt.want)
			}
			if got := strings.Contains(out.String(), inlineEraseAll); got != tt.erase {
				t.Errorf("return erased the screen and scrollback: %v, want %v", got, tt.erase)
			}
			last := tt.glyphs[len(tt.glyphs)-1:]
			if !tt.erase && strings.Contains(out.String(), strings.Repeat(last, glyphRowWidth)) {
				t.Errorf("return rewrote the unchanged row %q", last)
			}
		})
	}
}

// TestDestroy_FullscreenErasesInlineOutput covers quitting from the alternate
// screen. Leaving it shows the inline output drawn before it was entered, which
// must not stay behind for the shell to print over.
func TestDestroy_FullscreenErasesInlineOutput(t *testing.T) {
	for _, tt := range []struct {
		name   string
		glyphs string
		during func(*Renderer, []*glyphRow)
		// want is the first column of the scrollback and the screen.
		want string
		// eraseAll reports whether the screen and the scrollback are erased.
		eraseAll bool
		// cursorRow is where the shell resumes, the row the output began on
		// or the top of an erased screen.
		cursorRow int
	}{
		{name: "on screen", glyphs: shortGlyphs, want: "$", cursorRow: 1},
		{name: "in the scrollback", glyphs: tallGlyphs, eraseAll: true},
		{name: "height resized", glyphs: shortGlyphs, during: resizeDuring(testTermWidth, testTermHeight-5), eraseAll: true},
		{name: "width resized", glyphs: shortGlyphs, during: resizeDuring(testTermWidth-5, testTermHeight), eraseAll: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r, out, term, rows := enterFullscreen(t, tt.glyphs)
			if tt.during != nil {
				tt.during(r, rows)
			}

			r.Destroy()
			term.feed(out.String())

			if got := firstColumn(term.lines()); got != tt.want {
				t.Errorf("terminal rows begin %q, want %q", got, tt.want)
			}
			if got := strings.Contains(out.String(), inlineEraseAll); got != tt.eraseAll {
				t.Errorf("erased the screen and scrollback: %v, want %v", got, tt.eraseAll)
			}
			if term.normal != nil {
				t.Error("expected the alternate screen to be left")
			}
			if term.cx != 0 || term.cy != tt.cursorRow {
				t.Errorf("cursor at (%d, %d), want (0, %d)", term.cx, term.cy, tt.cursorRow)
			}
		})
	}
}

// TestDestroy_InlineClearOnExit covers quitting from inline mode. ClearOnExit
// erases the output the way quitting from the alternate screen does, and
// without it the output stays with the cursor on a clean line below it.
func TestDestroy_InlineClearOnExit(t *testing.T) {
	for _, tt := range []struct {
		name        string
		clearOnExit bool
		glyphs      string
		during      func(*Renderer, []*glyphRow)
		// want is the first column of the scrollback and the screen.
		want string
		// eraseAll reports whether the screen and the scrollback are erased.
		eraseAll bool
		// cursorRow is where the shell resumes: below the kept output, on the
		// row the erased output began on, or at the top of an erased screen.
		cursorRow int
	}{
		{name: "kept by default", glyphs: shortGlyphs, want: "$" + shortGlyphs, cursorRow: len(shortGlyphs) + 1},
		{name: "on screen", clearOnExit: true, glyphs: shortGlyphs, want: "$", cursorRow: 1},
		{name: "in the scrollback", clearOnExit: true, glyphs: tallGlyphs, eraseAll: true},
		{name: "height resized", clearOnExit: true, glyphs: shortGlyphs, during: resizeDuring(testTermWidth, testTermHeight-5), eraseAll: true},
		{name: "width resized", clearOnExit: true, glyphs: shortGlyphs, during: resizeDuring(testTermWidth-5, testTermHeight), eraseAll: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r, out, term, rows := drawInline(t, tt.glyphs)
			r.config.ClearOnExit = tt.clearOnExit
			if tt.during != nil {
				tt.during(r, rows)
			}

			r.Destroy()
			term.feed(out.String())

			if got := firstColumn(term.lines()); got != tt.want {
				t.Errorf("terminal rows begin %q, want %q", got, tt.want)
			}
			if got := strings.Contains(out.String(), inlineEraseAll); got != tt.eraseAll {
				t.Errorf("erased the screen and scrollback: %v, want %v", got, tt.eraseAll)
			}
			if term.cx != 0 || term.cy != tt.cursorRow {
				t.Errorf("cursor at (%d, %d), want (0, %d)", term.cx, term.cy, tt.cursorRow)
			}
		})
	}
}

// TestRender covers drawing a frame outside the event loop, which shows a
// change before Destroy ends rendering: hiding the last row erases it at once,
// the frame takes the place of the one requested, and the cursor resumes below
// the rows that remain.
func TestRender(t *testing.T) {
	r, out, term, rows := drawInline(t, shortGlyphs)
	rows[len(rows)-1].SetGone(true)
	r.RequestRender()

	r.Render()
	if r.dirty.Load() {
		t.Error("expected the requested frame to be drawn")
	}
	r.Destroy()
	term.feed(out.String())

	kept := shortGlyphs[:len(shortGlyphs)-1]
	if got, want := firstColumn(term.lines()), "$"+kept; got != want {
		t.Errorf("terminal rows begin %q, want %q", got, want)
	}
	if term.cx != 0 || term.cy != len(kept)+1 {
		t.Errorf("cursor at (%d, %d), want (0, %d)", term.cx, term.cy, len(kept)+1)
	}
}

// TestRender_NotRunning checks that Render writes nothing before setup or
// after Destroy, when no frame belongs on the terminal.
func TestRender_NotRunning(t *testing.T) {
	for _, tt := range []struct {
		name string
		make func() (*Renderer, *bytes.Buffer)
	}{
		{name: "before setup", make: newInlineTestRenderer},
		{name: "destroyed", make: func() (*Renderer, *bytes.Buffer) {
			r, out := startedTestRenderer()
			r.Destroy()
			out.Reset()
			return r, out
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r, out := tt.make()
			r.root.Add(newGlyphRow("0", 'a', glyphRowWidth, 1))

			r.Render()

			if out.Len() != 0 {
				t.Errorf("expected no output, got %q", out.String())
			}
		})
	}
}

// TestDestroy_FullscreenWithoutInlineOutput keeps the exit of a renderer that
// never drew inline: leaving the alternate screen restores the shell as it was.
func TestDestroy_FullscreenWithoutInlineOutput(t *testing.T) {
	r, out := startedTestRenderer()
	r.config.InlineMode = false
	term := newDisagreeingTerm(testTermWidth, testTermHeight, func(rune) int { return 1 })
	term.feed(shellPrompt + "\r\n")
	r.backend.EnterAlternateScreen()
	r.root.Add(newGlyphRow("0", 'a', glyphRowWidth, 1))
	r.renderFrame()
	term.feed(out.String())
	out.Reset()

	r.Destroy()
	term.feed(out.String())

	if strings.Contains(out.String(), inlineEraseBelow) || strings.Contains(out.String(), inlineEraseAll) {
		t.Errorf("expected nothing to be erased, got %q", out.String())
	}
	if got := firstColumn(term.lines()); got != "$" {
		t.Errorf("terminal rows begin %q, want %q", got, "$")
	}
}

func TestSetInlineMode_FullscreenResizesRoot(t *testing.T) {
	r, _ := startedTestRenderer()
	// An inline frame grows the root to the tall layout height.
	r.renderInlineFrame()

	r.SetInlineMode(false)

	if got := r.root.Height(); got != r.height {
		t.Errorf("expected root height %d, got %d", r.height, got)
	}
}

func TestInlineMode(t *testing.T) {
	r, _ := startedTestRenderer()

	if !r.InlineMode() {
		t.Error("expected inline mode to be reported")
	}

	r.SetInlineMode(false)

	if r.InlineMode() {
		t.Error("expected fullscreen mode to be reported")
	}
}
