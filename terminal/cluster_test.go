package terminal

import (
	"fmt"
	"strings"
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/textbuf"
)

func TestFlushWideClusters(t *testing.T) {
	for _, width := range []int{3, 4, 130} {
		for _, mode := range []string{"full", "diff", "inline full", "inline diff"} {
			t.Run(fmt.Sprintf("%s/%d", mode, width), func(t *testing.T) {
				checkWideClusterFlush(t, mode, width)
			})
		}
	}
}

func checkWideClusterFlush(t *testing.T, mode string, width int) {
	t.Helper()
	want := "A\u0915" + strings.Repeat("\u093e", width-1) + "Z"
	b := buffer.New(width+3, 1)
	b.DrawClusters(want, 0, 0, b.Width(), style.ColorWhite, style.ColorBlack, 0, nil)
	r, out := newInlineTestRenderer()
	switch mode {
	case "full":
		r.FullFlush(b)
	case "diff":
		r.DiffAndFlush(buffer.New(b.Width(), 1), b)
	case "inline full":
		r.InlineFlush(b)
	case "inline diff":
		r.InlineFlush(buffer.New(b.Width(), 1))
		r.InlineFlush(b)
	}
	term := newDisagreeingTerm(b.Width()+1, 3, textbuf.RuneWidth)
	term.feed(out.String())
	if !strings.HasPrefix(term.row(0), want) {
		t.Errorf("rendered %q, want prefix %q", term.row(0), want)
	}
}

func TestFlushBrokenWideCluster(t *testing.T) {
	for _, col := range []int{0, 1, 2} {
		t.Run(fmt.Sprint(col), func(t *testing.T) {
			b := buffer.New(4, 1)
			b.SetGraphemeCluster(0, 0, "\u0915\u093e\u0903", style.ColorWhite, style.ColorBlack, 0)
			b.PushScissorRect(col, 0, 1, 1)
			b.SetCell(col, 0, 'X', style.ColorWhite, style.ColorBlack, 0)
			b.PopScissorRect()
			var out strings.Builder
			for x := range b.Width() {
				ch, emit := flushRune(b, x, 0, b.Cell(x, 0))
				if !emit {
					t.Fatalf("broken cluster skipped column %d", x)
				}
				out.WriteRune(ch)
			}
			want := []rune("    ")
			want[col] = 'X'
			if out.String() != string(want) {
				t.Errorf("emitted %q, want %q", out.String(), string(want))
			}
		})
	}
}
