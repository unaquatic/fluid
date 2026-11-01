package terminal

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
)

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
