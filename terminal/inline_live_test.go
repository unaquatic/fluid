package terminal

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/unaquatic/fluid/style"
)

// historyRows is the number of rows above the live row in liveRenderer's
// frame.
const historyRows = 16

// liveRenderer returns a renderer whose frame holds historyRows rows of
// history, a live row two rows tall, and a footer of footer rows. With live
// set, the live area starts at the live row, as SetInlineLiveTop sets it.
// It draws the first frame.
func liveRenderer(t *testing.T, footer int, live bool) (r *Renderer, out *bytes.Buffer, row, foot *glyphRow) {
	t.Helper()
	r, out = newInlineTestRenderer()
	for i := range historyRows / 2 {
		r.root.Add(newGlyphRow("h"+strconv.Itoa(i), rune('a'+i), 39, 2))
	}
	row = newGlyphRow("live", 'L', 39, 2)
	foot = newGlyphRow("footer", 'F', 39, footer)
	r.root.Add(row)
	r.root.Add(foot)
	if live {
		r.SetInlineLiveTop(func() int { return row.Y() })
	}
	r.renderInlineFrame()
	return r, out, row, foot
}

// TestInlineLive_TallLiveAreaKeepsTheScrollback grows the footer under a
// live row until the live area is taller than the terminal, changes the
// live row, as a running call's status does, and shrinks the footer again.
// With a live area, the live row is left out while it does not fit, rather
// than scrolled into the scrollback, so no change rewrites the screen, and
// it shows again, as changed, once it fits. Without one, the change to the
// row in the scrollback rewrites everything.
func TestInlineLive_TallLiveAreaKeepsTheScrollback(t *testing.T) {
	for _, live := range []bool{true, false} {
		t.Run("live="+strconv.FormatBool(live), func(t *testing.T) {
			r, out, row, foot := liveRenderer(t, 4, live)

			foot.resize(testTermHeight)
			r.renderInlineFrame()
			if live {
				if got := frameText(r.inlineCurrentBuf); strings.ContainsRune(got, 'L') {
					t.Errorf("output %q shows the live row, which no longer fits", got)
				}
				if r.inlineViewportTop != historyRows {
					t.Errorf("viewportTop = %d, want the live area's top, %d", r.inlineViewportTop, historyRows)
				}
			}

			row.setGlyph('M')
			r.renderInlineFrame()
			foot.resize(4)
			r.renderInlineFrame()

			if erased := strings.Contains(out.String(), inlineEraseAll); erased == live {
				t.Errorf("a change rewrote the screen and the scrollback = %v, want %v", erased, !live)
			}
			var want string
			for i := range historyRows / 2 {
				want += strings.Repeat(string(rune('a'+i)), 2)
			}
			want += "MMFFFF"
			if got := frameText(r.inlineCurrentBuf); got != want {
				t.Errorf("output %q, want %q", got, want)
			}
		})
	}
}

// TestInlineLive_RowsLeftOutStayUnwritten checks that a change to a row of
// the live area that is left out writes nothing.
func TestInlineLive_RowsLeftOutStayUnwritten(t *testing.T) {
	r, out, row, _ := liveRenderer(t, testTermHeight, true)
	out.Reset()

	row.setGlyph('M')
	r.renderInlineFrame()

	if strings.Contains(out.String(), "M") {
		t.Errorf("a change to a row left out wrote %q", out.String())
	}
}

// TestInlineLive_CursorMovesWithTheOutput checks that the cursor, in the
// footer under rows left out, is inverted at its row in the output.
func TestInlineLive_CursorMovesWithTheOutput(t *testing.T) {
	r, _, _, foot := liveRenderer(t, testTermHeight, true)
	r.cursorVisible = true
	r.cursorX, r.cursorY = 3, foot.Y()+1
	r.renderInlineFrame()

	cut := r.nextBuffer.Height() - testTermHeight - r.inlineLiveFrom
	if !r.inlineLiveFull || cut == 0 {
		t.Fatal("expected rows of the live area to be left out")
	}
	if got := r.inlineOutCell.Y; got != foot.Y()+1-cut {
		t.Errorf("cursor at output row %d, want %d", got, foot.Y()+1-cut)
	}
	if got := r.inlineCurrentBuf.Cell(3, foot.Y()+1-cut); got.Fg.Equals(style.ColorWhite) {
		t.Error("expected the cursor cell to be inverted in the output")
	}
}

// insertRenderer draws history rows and under them a live area of footer
// rows, all one-row glyph rows, below a shell prompt, and feeds the output
// to a terminal. It returns the renderer, its output, which it leaves empty,
// the terminal, and the footer, which starts the live area.
func insertRenderer(t *testing.T, history, footer int) (*Renderer, *bytes.Buffer, *disagreeingTerm, *glyphRow) {
	t.Helper()
	r, out := startedTestRenderer()
	term := newDisagreeingTerm(testTermWidth, testTermHeight, func(rune) int { return 1 })
	term.feed(shellPrompt + "\r\n")
	for i := range history {
		r.root.Add(newGlyphRow("h"+strconv.Itoa(i), rune('a'+i%26), glyphRowWidth, 1))
	}
	foot := newGlyphRow("footer", 'F', glyphRowWidth, footer)
	r.root.Add(foot)
	r.SetInlineLiveTop(func() int { return foot.Y() })
	r.renderInlineFrame()
	term.feed(out.String())
	out.Reset()
	return r, out, term, foot
}

// TestInlineLive_RowsAddedAboveScrollThroughARegion adds rows above a live
// area on a full screen, as a reply gains rows above the input box, and
// checks that each frame writes the rows added and the live area's first
// row alone, through a scroll region, rather than every row of the live
// area, and that the terminal then holds every row of the frame, in order,
// in its scrollback and on screen.
func TestInlineLive_RowsAddedAboveScrollThroughARegion(t *testing.T) {
	for _, tt := range []struct {
		name   string
		footer int  // the rows of the live area
		added  int  // the rows added each frame
		change bool // whether the live area changes too
	}{
		{name: "one free row, a row a frame", footer: testTermHeight - 1, added: 1},
		{name: "one free row, rows a frame", footer: testTermHeight - 1, added: 3},
		{name: "free rows, rows a frame", footer: testTermHeight - 3, added: 4},
		{name: "the live area changes too", footer: testTermHeight - 2, added: 2, change: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r, out, term, foot := insertRenderer(t, 30, tt.footer)
			n := 0
			for frame := range 3 {
				for range tt.added {
					r.root.InsertBefore(newGlyphRow("n"+strconv.Itoa(n), rune('A'+n), glyphRowWidth, 1), foot)
					n++
				}
				want := tt.added + 1
				if tt.change {
					foot.setGlyph(rune('0' + frame))
					want = tt.added + tt.footer
				}
				r.renderInlineFrame()
				written := out.String()
				term.feed(written)
				out.Reset()

				if !strings.Contains(written, "\x1b[1;") {
					t.Errorf("frame %d set no scroll region", frame)
				}
				if got := strings.Count(written, inlineEraseRow); got != want {
					t.Errorf("frame %d wrote %d rows, want %d", frame, got, want)
				}
				if got, want := firstColumn(term.lines()), "$"+frameText(r.nextBuffer); got != want {
					t.Fatalf("frame %d: terminal holds %q, want the prompt and the frame, %q", frame, got, want)
				}
			}
		})
	}
}

// TestInlineLive_FullScreenHoldsStill grows the content above a live area
// exactly as tall as the screen, as a reply does above an input box that
// fills it. With a live area, the rows the content gains are left out, so
// nothing is written, rather than every row of the screen rewritten a row
// lower for each of them, and they show, scrolling into the scrollback,
// once a row of the screen is free again. Without one, every row they add
// rewrites the screen.
func TestInlineLive_FullScreenHoldsStill(t *testing.T) {
	for _, live := range []bool{true, false} {
		t.Run("live="+strconv.FormatBool(live), func(t *testing.T) {
			r, out, row, foot := liveRenderer(t, testTermHeight-2, live)
			out.Reset()

			for i := range 3 {
				r.root.InsertBefore(newGlyphRow("new"+strconv.Itoa(i), rune('x'+i), 39, 1), row)
				r.renderInlineFrame()
			}
			if wrote := out.Len() > 0; wrote == live {
				t.Errorf("rows gained above the live area wrote output = %v, want %v", wrote, !live)
			}

			foot.resize(4)
			r.renderInlineFrame()
			if strings.Contains(out.String(), inlineEraseAll) {
				t.Error("showing the rows left out rewrote the screen and the scrollback")
			}
			if got, want := frameText(r.inlineCurrentBuf), frameText(r.nextBuffer); got != want {
				t.Errorf("output %q, want the frame, %q", got, want)
			}
		})
	}
}
