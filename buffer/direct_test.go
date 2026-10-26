package buffer

import (
	"testing"

	"github.com/unaquatic/fluid/style"
)

func TestSetCellDirect_PreservesWidePair(t *testing.T) {
	tests := []struct {
		name  string
		write func(*Buffer)
	}{
		{"unchanged", func(b *Buffer) { b.SetCellDirect(4, 0, b.Cell(4, 0)) }},
		{"cursor inversion", func(b *Buffer) {
			c := b.Cell(4, 0)
			c.Fg, c.Bg = c.Bg, c.Fg
			b.SetCellDirect(4, 0, c)
		}},
		{"replacement", func(b *Buffer) {
			b.SetCellDirect(4, 0, Cell{Char: '\u754c', Width: 2, Fg: style.ColorRed, Bg: style.ColorBlue})
		}},
		{"new pair", func(b *Buffer) {
			c := b.Cell(4, 0)
			b.Clear(style.ColorTransparent)
			b.SetCellDirect(4, 0, c)
		}},
		{"repair missing continuation", func(b *Buffer) {
			b.cells[b.cellIndex(5, 0)] = CellClear
			b.SetCellDirect(4, 0, b.Cell(4, 0))
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New(10, 1)
			b.SetCell(4, 0, '\u4e16', style.ColorWhite, style.ColorBlack, style.AttrBold)
			tt.write(b)
			head := b.Cell(4, 0)
			if head.Width != 2 {
				t.Fatalf("head width = %d, want 2", head.Width)
			}
			want := Cell{Fg: head.Fg, Bg: head.Bg, Attributes: head.Attributes, Width: -1}
			if got := b.Cell(5, 0); !got.Equals(want) {
				t.Errorf("continuation = %+v, want %+v", got, want)
			}
		})
	}
}

func TestSetCellDirect_ContinuationKeepsItsHead(t *testing.T) {
	b := New(10, 1)
	b.SetCell(4, 0, '\u4e16', style.ColorWhite, style.ColorBlack, 0)
	head := b.Cell(4, 0)
	cont := b.Cell(5, 0)
	b.SetCellDirect(5, 0, cont)
	if !b.Cell(4, 0).Equals(head) || !b.Cell(5, 0).Equals(cont) {
		t.Fatal("writing the continuation changed the wide pair")
	}
}

func TestSetCellDirect_NarrowOverWideStillDetaches(t *testing.T) {
	for _, x := range []int{4, 5} {
		t.Run(string(rune('0'+x)), func(t *testing.T) {
			b := New(10, 1)
			b.SetCell(4, 0, '\u4e16', style.ColorWhite, style.ColorBlack, 0)
			cell := Cell{Char: 'X', Width: 1, Fg: style.ColorWhite, Bg: style.ColorBlack}
			b.SetCellDirect(x, 0, cell)
			if !b.Cell(x, 0).Equals(cell) {
				t.Fatal("narrow cell was not written")
			}
			if other := b.Cell(9-x, 0); other.Char != ' ' || other.Width != 1 {
				t.Errorf("other half = %+v, want a single-column space", other)
			}
		})
	}
}
