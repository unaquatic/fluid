package terminal

import (
	"fmt"
	"strings"
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/style"
)

// disagreeingTerm is a terminal whose advance for some runes differs from the
// width fluid assigned them, the way iTerm advances four columns for
// composed hangul jamo and one for a VS16 emoji.
// It models deferred auto-wrap, scrolling, and the controls used by inline output.
type disagreeingTerm struct {
	w, h     int
	grid     [][]rune
	cx, cy   int
	pending  bool
	autowrap bool
	scrolled int
	adv      func(rune) int
}

func (s *disagreeingTerm) lineFeed() {
	s.pending = false
	s.cy++
	if s.cy < s.h {
		return
	}
	copy(s.grid, s.grid[1:])
	s.grid[s.h-1] = []rune(strings.Repeat(".", s.w))
	s.cy = s.h - 1
	s.scrolled++
}

func (s *disagreeingTerm) put(r rune) {
	w := s.adv(r)
	if w == 0 {
		return
	}
	if s.pending {
		s.cx = 0
		s.lineFeed()
	}
	if s.cy >= 0 && s.cy < s.h && s.cx >= 0 && s.cx < s.w {
		s.grid[s.cy][s.cx] = r
	}
	s.cx += w
	if s.cx >= s.w {
		s.cx = s.w - 1
		s.pending = s.autowrap
	}
}

func (s *disagreeingTerm) csi(params string, command byte) {
	switch command {
	case 'H':
		y, x := cupParams(params)
		s.cy, s.cx, s.pending = y-1, x-1, false
	case 'G':
		s.cx, s.pending = max(atoiParam(params), 1)-1, false
	case 'A':
		s.cy, s.pending = max(s.cy-max(atoiParam(params), 1), 0), false
	case 'B':
		s.cy, s.pending = min(s.cy+max(atoiParam(params), 1), s.h-1), false
	case 'K':
		if params == "2" {
			s.grid[s.cy] = []rune(strings.Repeat(".", s.w))
		}
	case 'h', 'l':
		if params == "?7" {
			s.autowrap = command == 'h'
		}
	}
}

func newDisagreeingTerm(w, h int, adv func(rune) int) *disagreeingTerm {
	g := make([][]rune, h)
	for i := range g {
		g[i] = make([]rune, w)
		for j := range g[i] {
			g[i][j] = '.'
		}
	}
	return &disagreeingTerm{w: w, h: h, grid: g, autowrap: true, adv: adv}
}

func (s *disagreeingTerm) feed(data string) {
	for i := 0; i < len(data); {
		if data[i] == 0x1b && i+1 < len(data) && data[i+1] == '[' {
			j := i + 2
			if j < len(data) && data[j] == '?' {
				j++
			}
			for j < len(data) && (data[j] == ';' || (data[j] >= '0' && data[j] <= '9')) {
				j++
			}
			if j < len(data) {
				s.csi(data[i+2:j], data[j])
				i = j + 1
				continue
			}
		}
		r, sz := firstRune(data[i:])
		switch r {
		case '\r':
			s.cx, s.pending = 0, false
		case '\n':
			s.lineFeed()
		default:
			s.put(r)
		}
		i += sz
	}
}

func (s *disagreeingTerm) row(y int) string { return string(s.grid[y]) }

func cupParams(params string) (y, x int) {
	y, x = 1, 1
	p := strings.Split(params, ";")
	if len(p) > 0 && p[0] != "" {
		y = atoiParam(p[0])
	}
	if len(p) > 1 && p[1] != "" {
		x = atoiParam(p[1])
	}
	return
}

func atoiParam(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return n
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func firstRune(s string) (rune, int) {
	for _, r := range s {
		return r, len(string(r))
	}
	return 0, 1
}

// jamoTerm advances a column for each conjoining jamo, so the cluster costs
// four columns where fluid budgeted two.
func jamoTerm(r rune) int {
	switch r {
	case 0x1161, 0x11A8:
		return 1
	case 0x1100:
		return 2
	}
	return 1
}

// TestReanchor_DriftBoundedToOneGlyph is the regression for the resize
// symptom: FullFlush wrote a row contiguously with no absolute positioning
// after the row start, so a terminal that disagreed about one cluster shifted
// everything after it. DiffAndFlush only avoided this by accident, because
// skipping unchanged cells forced a CUP; during a scroll, where every cell
// changes, it drifted too.
func TestReanchor_DriftBoundedToOneGlyph(t *testing.T) {
	const W = 30
	const tail = "|END"

	build := func() *buffer.Buffer {
		b := buffer.New(W, 1)
		b.Clear(style.ColorTransparent)
		b.DrawClusters("ab \u1100\u1161\u11a8 "+tail, 0, 0, W,
			style.ColorWhite, style.ColorTransparent, 0, nil)
		return b
	}

	// Where the buffer says the tail sits, which is where it must appear.
	want := columnOf(bufferText(build(), W), tail)
	if want < 0 {
		t.Fatal("tail missing from buffer")
	}

	t.Run("FullFlush", func(t *testing.T) {
		r, out := newTestRenderer(W, 1)
		r.FullFlush(build())
		term := newDisagreeingTerm(W, 1, jamoTerm)
		term.feed(out.String())
		if got := columnOf(term.row(0), tail); got != want {
			t.Errorf("tail at column %d, want %d\n  row: %q", got, want, term.row(0))
		}
	})

	t.Run("DiffAndFlush every cell changed", func(t *testing.T) {
		cur := buffer.New(W, 1)
		cur.Clear(style.ColorBlack)
		r, out := newTestRenderer(W, 1)
		r.DiffAndFlush(cur, build())
		term := newDisagreeingTerm(W, 1, jamoTerm)
		term.feed(out.String())
		if got := columnOf(term.row(0), tail); got != want {
			t.Errorf("tail at column %d, want %d\n  row: %q", got, want, term.row(0))
		}
	})
}

// TestReanchor_NarrowEmojiTerminal covers the opposite drift direction: a
// terminal that renders an emoji sequence in one column where fluid
// budgeted two.
func TestReanchor_NarrowEmojiTerminal(t *testing.T) {
	const W = 30
	const tail = "|END"
	narrow := func(r rune) int {
		if r == 0x2764 {
			return 1 // rendered as text presentation despite the VS16
		}
		if r == 0xFE0F {
			return 0
		}
		return 1
	}

	b := buffer.New(W, 1)
	b.Clear(style.ColorTransparent)
	b.DrawClusters("ab \u2764\uFE0F "+tail, 0, 0, W,
		style.ColorWhite, style.ColorTransparent, 0, nil)

	want := columnOf(bufferText(b, W), tail)
	r, out := newTestRenderer(W, 1)
	r.FullFlush(b)
	term := newDisagreeingTerm(W, 1, narrow)
	term.feed(out.String())
	if got := columnOf(term.row(0), tail); got != want {
		t.Errorf("tail at column %d, want %d\n  row: %q", got, want, term.row(0))
	}
}

// TestReanchor_NoCostForPlainText keeps the fix from taxing ordinary output:
// ASCII cells are certain, so no re-anchor sequences may be emitted.
func TestReanchor_NoCostForPlainText(t *testing.T) {
	b := buffer.New(20, 1)
	b.Clear(style.ColorTransparent)
	b.DrawClusters("plain ascii text", 0, 0, 20,
		style.ColorWhite, style.ColorTransparent, 0, nil)

	r, out := newTestRenderer(20, 1)
	r.FullFlush(b)
	if n := strings.Count(out.String(), "\x1b["); n > 5 {
		t.Errorf("plain text emitted %d escape sequences; expected only row + SGR + auto-wrap controls", n)
	}
	if strings.Contains(out.String(), "G") {
		t.Errorf("plain text must not re-anchor: %q", out.String())
	}
}

// columnOf returns the column where sub starts, counting runes rather than
// bytes: one grid cell is one rune, and these rows contain multi-byte runes.
func columnOf(row, sub string) int {
	r := []rune(row)
	t := []rune(sub)
	for i := 0; i+len(t) <= len(r); i++ {
		if string(r[i:i+len(t)]) == sub {
			return i
		}
	}
	return -1
}

// bufferText renders a row the way the buffer intends it, ignoring terminal
// disagreement, to establish where content ought to land.
func bufferText(b *buffer.Buffer, w int) string {
	var sb strings.Builder
	for x := range w {
		c := b.Cell(x, 0)
		if c.Width == -1 {
			continue
		}
		ch := c.Char
		if ch == 0 {
			ch = ' '
		}
		sb.WriteRune(ch)
		for range max(int(c.Width)-1, 0) {
			sb.WriteRune(' ') // wide cell occupies its continuation column
		}
	}
	return sb.String()
}

func rightEdgeBuffer(w, h, row int) *buffer.Buffer {
	b := buffer.New(w, h)
	for y := range h {
		text := "row-" + string(rune('a'+y))
		if y == row {
			text = strings.Repeat("A", w-2) + "\u1100\u1161\u11a8"
		}
		b.DrawClusters(text, 0, y, w, style.ColorWhite, style.ColorTransparent, 0, nil)
	}
	return b
}

func TestReanchor_RightEdgeDoesNotWrap(t *testing.T) {
	for _, mode := range []string{"full", "diff", "inline full", "inline diff"} {
		for _, row := range []int{0, 2} {
			t.Run(fmt.Sprintf("%s/row %d", mode, row), func(t *testing.T) {
				checkRightEdgeFlush(t, mode, row)
			})
		}
	}
}

func TestReanchor_RightEdgeWrapsWithoutClamp(t *testing.T) {
	const w, h = 12, 3
	r, out := newTestRenderer(w, h)
	r.FullFlush(rightEdgeBuffer(w, h, h-1))
	term := newDisagreeingTerm(w, h, jamoTerm)
	term.feed(strings.ReplaceAll(out.String(), "\x1b[?7l", ""))
	if term.scrolled == 0 {
		t.Fatal("simulator failed to detect bottom-row overflow with auto-wrap enabled")
	}
}

func checkRightEdgeFlush(t *testing.T, mode string, row int) {
	t.Helper()
	const w, h = 12, 3
	r, out := newInlineTestRenderer()
	r.width, r.height = w, h+1
	width, height := w, h
	if strings.HasPrefix(mode, "inline") {
		width, height = w-1, h+1 // inline reserves a column and needs a trailing row
	}
	term := newDisagreeingTerm(w, height, jamoTerm)
	prev, next := rightEdgeBuffer(width, h, -1), rightEdgeBuffer(width, h, row)
	switch mode {
	case "full":
		r.FullFlush(next)
	case "diff":
		r.FullFlush(prev)
		term.feed(out.String())
		out.Reset()
		r.DiffAndFlush(prev, next)
	case "inline full":
		r.InlineFlush(next)
	case "inline diff":
		r.InlineFlush(prev)
		term.feed(out.String())
		out.Reset()
		r.InlineFlush(next)
	}
	term.feed(out.String())
	checkRightEdgeRows(t, term, next, row)
	if strings.HasPrefix(mode, "inline") && (term.cx != 0 || term.cy != h) {
		t.Errorf("inline cursor = (%d, %d), want (0, %d)", term.cx, term.cy, h)
	}
}

func checkRightEdgeRows(t *testing.T, term *disagreeingTerm, next *buffer.Buffer, row int) {
	t.Helper()
	if term.scrolled != 0 {
		t.Errorf("screen scrolled %d times", term.scrolled)
	}
	if !term.autowrap {
		t.Error("flush did not restore auto-wrap")
	}
	for y := range next.Height() {
		if y != row && !strings.HasPrefix(term.row(y), "row-"+string(rune('a'+y))) {
			t.Errorf("row %d corrupted: %q", y, term.row(y))
		}
	}
}
