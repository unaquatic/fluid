package terminal

import (
	"bytes"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
)

// renderables returns rows as the renderables CommitInline takes.
func renderables(rows []*glyphRow) []render.Renderable {
	out := make([]render.Renderable, len(rows))
	for i, row := range rows {
		out[i] = row
	}
	return out
}

// drawAndFeed draws a frame and feeds its output to term, and returns what
// the frame wrote, leaving out empty.
func drawAndFeed(r *Renderer, out *bytes.Buffer, term *disagreeingTerm) string {
	r.renderInlineFrame()
	written := out.String()
	term.feed(written)
	out.Reset()
	return written
}

// TestCommitInline_TerminalKeepsTheRows commits the leading rows of a frame
// on screen and of one taller than the screen, and checks that the frame
// then holds only the rest, that committing writes nothing, as the terminal
// already shows every row, and that the frames after it change and add rows
// below the committed ones, which stay as they were.
func TestCommitInline_TerminalKeepsTheRows(t *testing.T) {
	for _, tt := range []struct {
		name   string
		glyphs string
		commit int
	}{
		{name: "on screen", glyphs: "abcdef", commit: 4},
		{name: "in the scrollback", glyphs: tallGlyphs, commit: 20},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r, out, term, rows := drawInline(t, tt.glyphs)

			if got := r.CommitInline(renderables(rows[:tt.commit])...); got != tt.commit {
				t.Fatalf("committed %d rows, want %d", got, tt.commit)
			}
			if written := drawAndFeed(r, out, term); written != "" {
				t.Errorf("the frame after the commit wrote %q, want nothing", written)
			}
			if got, want := frameText(r.nextBuffer), tt.glyphs[tt.commit:]; got != want {
				t.Errorf("the frame holds %q, want the rows that stay, %q", got, want)
			}
			if got, want := r.root.ChildrenCount(), len(rows)-tt.commit; got != want {
				t.Errorf("the root holds %d rows, want %d", got, want)
			}

			// A committed row has left the scene.
			rows[0].setGlyph('X')
			if written := drawAndFeed(r, out, term); written != "" {
				t.Errorf("a change to a committed row wrote %q, want nothing", written)
			}

			rows[tt.commit].setGlyph('A')
			for i := range 3 {
				r.root.Add(newGlyphRow("z"+strconv.Itoa(i), 'z', glyphRowWidth, 1))
			}
			drawAndFeed(r, out, term)

			want := "$" + tt.glyphs[:tt.commit] + "A" + tt.glyphs[tt.commit+1:] + "zzz"
			if got := firstColumn(term.lines()); got != want {
				t.Errorf("terminal rows begin %q, want %q", got, want)
			}
		})
	}
}

// TestCommitInline_StopsAtARowItCannotCommit checks which leading rows
// CommitInline commits: unchanged ones, in the root's order, past gone ones,
// up to the last row, which holds the cursor.
func TestCommitInline_StopsAtARowItCannotCommit(t *testing.T) {
	for _, tt := range []struct {
		name   string
		before func(*Renderer, []*glyphRow)
		commit func([]*glyphRow) []*glyphRow
		want   int
	}{
		{
			name:   "unchanged rows",
			commit: func(rows []*glyphRow) []*glyphRow { return rows[:3] },
			want:   3,
		},
		{
			name:   "the last row",
			commit: func(rows []*glyphRow) []*glyphRow { return rows },
			want:   5,
		},
		{
			name:   "a changed row",
			before: func(_ *Renderer, rows []*glyphRow) { rows[2].setGlyph('C') },
			commit: func(rows []*glyphRow) []*glyphRow { return rows[:5] },
			want:   2,
		},
		{
			name: "a row laid out anew",
			before: func(_ *Renderer, rows []*glyphRow) {
				rows[1].LayoutNode().MarginBottom = layout.Points(1)
				rows[1].LayoutNode().MarkDirty()
			},
			commit: func(rows []*glyphRow) []*glyphRow { return rows[:3] },
			want:   1,
		},
		{
			name: "a row laid out since the frame",
			before: func(r *Renderer, rows []*glyphRow) {
				rows[1].LayoutNode().MarginTop = layout.Points(1)
				rows[1].LayoutNode().MarkDirty()
				r.root.CalculateLayout()
			},
			commit: func(rows []*glyphRow) []*glyphRow { return rows[:3] },
			want:   1,
		},
		{
			name: "a row the next overlaps",
			before: func(r *Renderer, rows []*glyphRow) {
				rows[1].LayoutNode().MarginBottom = layout.Points(-1)
				rows[1].LayoutNode().MarkDirty()
				r.renderInlineFrame()
			},
			commit: func(rows []*glyphRow) []*glyphRow { return rows[:3] },
			want:   1,
		},
		{
			name: "the cursor's row",
			before: func(r *Renderer, _ []*glyphRow) {
				r.SetCursorPosition(0, 1, true)
				r.renderInlineFrame()
			},
			commit: func(rows []*glyphRow) []*glyphRow { return rows[:3] },
			want:   1,
		},
		{
			name: "a row whose ID a gone one has",
			before: func(r *Renderer, rows []*glyphRow) {
				rows[0].SetGone(true)
				rows[0].SetID(rows[1].ID())
				r.renderInlineFrame()
			},
			commit: func(rows []*glyphRow) []*glyphRow { return rows[1:3] },
			want:   0,
		},
		{
			name:   "a row after one that stays",
			commit: func(rows []*glyphRow) []*glyphRow { return rows[1:3] },
			want:   0,
		},
		{
			name:   "rows out of order",
			commit: func(rows []*glyphRow) []*glyphRow { return []*glyphRow{rows[1], rows[0]} },
			want:   0,
		},
		{
			name: "a row after a gone one",
			before: func(r *Renderer, rows []*glyphRow) {
				rows[0].SetGone(true)
				r.renderInlineFrame()
			},
			commit: func(rows []*glyphRow) []*glyphRow { return rows[1:3] },
			want:   2,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r, _, _, rows := drawInline(t, "abcdef")
			if tt.before != nil {
				tt.before(r, rows)
			}
			commit := tt.commit(rows)

			if got := r.CommitInline(renderables(commit)...); got != tt.want {
				t.Fatalf("committed %d rows, want %d", got, tt.want)
			}
			for i, row := range commit {
				if inRoot := row.Parent() != nil; inRoot == (i < tt.want) {
					t.Errorf("row %q in the root: %v, want %v", row.ID(), inRoot, i >= tt.want)
				}
			}
		})
	}
}

// TestCommitInline_CommitsNothing covers the states in which no row can be
// committed, as nothing ties the frame to the terminal or rows of the root
// do not end where the next begin.
func TestCommitInline_CommitsNothing(t *testing.T) {
	for _, tt := range []struct {
		name string
		// root sets the root up before the first frame.
		root func(*layout.Node)
		// before runs after it.
		before func(*Renderer, []*glyphRow)
		// noFrame leaves the first frame undrawn.
		noFrame bool
		// commit returns the rows to commit, the first two when nil.
		commit func([]*glyphRow) []*glyphRow
	}{
		{name: "before the first frame", noFrame: true},
		{name: "outside inline mode", before: func(r *Renderer, _ []*glyphRow) { r.SetInlineMode(false) }},
		{name: "after a width resize", before: func(r *Renderer, _ []*glyphRow) { r.processResize(testTermWidth-5, testTermHeight) }},
		{name: "after a height resize", before: func(r *Renderer, _ []*glyphRow) { r.processResize(testTermWidth, testTermHeight-5) }},
		{
			name: "while the output is due to be written again",
			before: func(r *Renderer, rows []*glyphRow) {
				r.SetInlineReprint(func() {})
				r.CommitInline(rows[0])
				r.renderInlineFrame()
				r.ReprintInline()
			},
			commit: func(rows []*glyphRow) []*glyphRow { return rows[1:2] },
		},
		{name: "on a root with top padding", root: func(n *layout.Node) { n.PaddingTop = layout.Points(1) }},
		{name: "on a root with gaps", root: func(n *layout.Node) { n.Gap = 1 }},
		{name: "on a root with row gaps", root: func(n *layout.Node) { n.RowGap = 1 }},
		{name: "on a centred root", root: func(n *layout.Node) { n.JustifyContent = layout.JustifyCenter }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r, _ := startedTestRenderer()
			if tt.root != nil {
				tt.root(r.root.LayoutNode())
			}
			var rows []*glyphRow
			for i, ch := range "abc" {
				row := newGlyphRow(strconv.Itoa(i), ch, glyphRowWidth, 1)
				rows = append(rows, row)
				r.root.Add(row)
			}
			if !tt.noFrame {
				r.renderInlineFrame()
			}
			if tt.before != nil {
				tt.before(r, rows)
			}
			commit := rows[:2]
			if tt.commit != nil {
				commit = tt.commit(rows)
			}

			if got := r.CommitInline(renderables(commit)...); got != 0 {
				t.Errorf("committed %d rows, want none", got)
			}
		})
	}
}

// TestCommitInline_RewritePutsTheRowsBack covers what writes the whole output
// again, which erases the screen and the scrollback: the rows committed are
// put back first, and the terminal ends up holding every row once, after
// which they can be committed again. Without a function to put them back,
// they are lost.
func TestCommitInline_RewritePutsTheRowsBack(t *testing.T) {
	const commit = 2
	for _, tt := range []struct {
		name   string
		during func(*Renderer, []*glyphRow)
		// noReprint leaves SetInlineReprint unset.
		noReprint bool
		want      string
	}{
		{name: "width resized", during: resizeDuring(testTermWidth-5, testTermHeight), want: tallGlyphs},
		{
			name: "height resized",
			// The frame is written again once something in it changes, as
			// an input box fitted to the screen does.
			during: func(r *Renderer, rows []*glyphRow) {
				r.processResize(testTermWidth, testTermHeight-5)
				rows[len(rows)-1].setGlyph('Y')
			},
			want: tallGlyphs[:len(tallGlyphs)-1] + "Y",
		},
		{name: "ReprintInline", during: func(r *Renderer, _ []*glyphRow) { r.ReprintInline() }, want: tallGlyphs},
		{
			name: "changed row in the scrollback",
			// The frame holds every row but the committed ones, and its first
			// is in the scrollback.
			during: func(_ *Renderer, rows []*glyphRow) { rows[commit].setGlyph('C') },
			want:   tallGlyphs[:commit] + "C" + tallGlyphs[commit+1:],
		},
		{
			name: "return from the alternate screen",
			during: func(r *Renderer, _ []*glyphRow) {
				r.SetInlineMode(false)
				r.renderFrame()
				r.SetInlineMode(true)
			},
			want: tallGlyphs,
		},
		{
			name:      "no function to put them back",
			during:    resizeDuring(testTermWidth-5, testTermHeight),
			noReprint: true,
			want:      tallGlyphs[commit:],
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r, out, term, rows := drawInline(t, tallGlyphs)
			reprints := 0
			if !tt.noReprint {
				r.SetInlineReprint(func() {
					reprints++
					for i, row := range rows[:commit] {
						r.root.Add(row, i)
					}
				})
			}
			if got := r.CommitInline(renderables(rows[:commit])...); got != commit {
				t.Fatalf("committed %d rows, want %d", got, commit)
			}
			drawAndFeed(r, out, term)

			tt.during(r, rows)
			var written strings.Builder
			for range 2 {
				written.WriteString(drawAndFeed(r, out, term))
			}

			want := 1
			if tt.noReprint {
				want = 0
			}
			if reprints != want {
				t.Errorf("the rows were put back %d times, want %d", reprints, want)
			}
			if got := strings.Count(written.String(), inlineEraseAll); got != 1 {
				t.Errorf("erased the screen and the scrollback %d times, want once", got)
			}
			if got := firstColumn(term.lines()); got != tt.want {
				t.Errorf("terminal rows begin %q, want %q", got, tt.want)
			}

			// The scene holds every row again.
			r.ReprintInline()
			if written := drawAndFeed(r, out, term); written != "" {
				t.Errorf("ReprintInline after the rewrite wrote %q, want nothing", written)
			}
			if !tt.noReprint {
				if got := r.CommitInline(renderables(rows[:commit])...); got != commit {
					t.Errorf("committed %d rows again, want %d", got, commit)
				}
			}
		})
	}
}

// TestReprintInline_NothingCommitted checks that ReprintInline leaves the
// output alone while the scene holds every row of it.
func TestReprintInline_NothingCommitted(t *testing.T) {
	r, out, term, _ := drawInline(t, shortGlyphs)
	r.SetInlineReprint(func() { t.Error("the rows were put back, with none committed") })

	r.ReprintInline()
	if written := drawAndFeed(r, out, term); written != "" {
		t.Errorf("the frame wrote %q, want nothing", written)
	}
}

// TestCommitInline_RowsAddedAboveTheLiveArea commits every row above a live
// area, which leaves the committed rows on screen above the frame, and then
// adds rows above the live area, from the frame of the commit on, which
// scroll into the scrollback through the rows of the screen the committed
// rows hold.
func TestCommitInline_RowsAddedAboveTheLiveArea(t *testing.T) {
	const history = 30
	r, out, term, foot := insertRenderer(t, history, testTermHeight-3)
	// Committing removes rows from the slice the root holds.
	committed := slices.Clone(r.root.ChildrenRaw()[:history])
	if got := r.CommitInline(committed...); got != history {
		t.Fatalf("committed %d rows, want %d", got, history)
	}
	if r.inlineViewportTop >= 0 {
		t.Fatalf("viewport top is %d, want the committed rows on screen above the frame", r.inlineViewportTop)
	}

	var glyphs strings.Builder
	for i := range history {
		glyphs.WriteRune(rune('a' + i%26))
	}
	for frame := range 3 {
		for i := range 2 {
			r.root.InsertBefore(newGlyphRow("n"+strconv.Itoa(frame*2+i), rune('A'+frame*2+i), glyphRowWidth, 1), foot)
		}
		written := drawAndFeed(r, out, term)

		if !strings.Contains(written, "\x1b[1;") {
			t.Errorf("frame %d set no scroll region", frame)
		}
		if got, want := firstColumn(term.lines()), "$"+glyphs.String()+frameText(r.nextBuffer); got != want {
			t.Fatalf("frame %d: terminal holds %q, want %q", frame, got, want)
		}
	}
}

// TestCommitInline_OncePerFrame commits a row, and then the next, which
// CommitInline commits only once the frame put the rows where the commit
// moved them, under a gone row the root keeps first.
func TestCommitInline_OncePerFrame(t *testing.T) {
	r, out := startedTestRenderer()
	term := newDisagreeingTerm(testTermWidth, testTermHeight, func(rune) int { return 1 })
	term.feed(shellPrompt + "\r\n")
	gone := newGlyphRow("gone", 'g', glyphRowWidth, 1)
	gone.SetGone(true)
	r.root.Add(gone)
	var rows []*glyphRow
	for i, ch := range "bcdef" {
		row := newGlyphRow(strconv.Itoa(i), ch, glyphRowWidth, 1)
		rows = append(rows, row)
		r.root.Add(row)
	}
	drawAndFeed(r, out, term)

	if got := r.CommitInline(rows[0]); got != 1 {
		t.Fatalf("committed %d rows, want 1", got)
	}
	if got := r.CommitInline(rows[1]); got != 0 {
		t.Errorf("committed %d rows before the frame, want none", got)
	}
	drawAndFeed(r, out, term)
	if got := r.CommitInline(rows[1]); got != 1 {
		t.Errorf("committed %d rows after the frame, want 1", got)
	}
	rows[len(rows)-1].setGlyph('F')
	drawAndFeed(r, out, term)

	if got, want := firstColumn(term.lines()), "$bcdeF"; got != want {
		t.Errorf("terminal rows begin %q, want %q", got, want)
	}
}

// TestCommitInline_StopsAtARowWhoseChildChanged checks that a row is not
// committed while a change under it is to be painted.
func TestCommitInline_StopsAtARowWhoseChildChanged(t *testing.T) {
	child := newGlyphRow("child", 'c', glyphRowWidth, 1)
	row := render.NewBaseRenderable(render.WithID("row"))
	row.Add(child)
	r, _ := startedTestRenderer()
	r.root.Add(row)
	r.root.Add(newGlyphRow("last", 'l', glyphRowWidth, 1))
	r.renderInlineFrame()

	child.setGlyph('C')
	if got := r.CommitInline(row); got != 0 {
		t.Errorf("committed %d rows, want none", got)
	}
}

// TestDestroy_ClearOnExitKeepsCommittedRows checks that erasing inline output
// on exit leaves the rows committed to the terminal, which are no longer part
// of it.
func TestDestroy_ClearOnExitKeepsCommittedRows(t *testing.T) {
	r, out, term, rows := drawInline(t, shortGlyphs)
	r.config.ClearOnExit = true
	if got := r.CommitInline(rows[0]); got != 1 {
		t.Fatalf("committed %d rows, want 1", got)
	}
	drawAndFeed(r, out, term)

	r.Destroy()
	term.feed(out.String())

	if got, want := firstColumn(term.lines()), "$"+shortGlyphs[:1]; got != want {
		t.Errorf("terminal rows begin %q, want %q", got, want)
	}
	if strings.Contains(out.String(), inlineEraseAll) {
		t.Error("erased the screen and the scrollback, want the output erased in place")
	}
}
