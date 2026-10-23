package buffer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/textbuf"
)

func TestWideClusterStorage(t *testing.T) {
	for _, width := range []int{3, 4, 130} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			cluster := "\u0915" + strings.Repeat("\u093e", width-1)
			if textbuf.GraphemeClusterCount(cluster) != 1 || textbuf.StringWidth(cluster) != width {
				t.Fatal("sample must be one grapheme with the expected width")
			}
			b := New(width+2, 1)
			used := b.DrawClusters(cluster+"X", 0, 0, b.Width(), style.ColorWhite, style.ColorBlack, 0, nil)
			if used != width+1 || b.Cell(0, 0).Width != width || b.Cell(width, 0).Char != 'X' {
				t.Fatalf("incorrect cluster placement: used %d, head %+v", used, b.Cell(0, 0))
			}
			for offset := 1; offset < width; offset++ {
				c := b.Cell(offset, 0)
				if !c.IsContinuation() || c.Width != -offset {
					t.Errorf("column %d = %+v, want offset %d", offset, c, -offset)
				}
			}
			head := b.Cell(0, 0)
			head.Fg, head.Bg = head.Bg, head.Fg
			b.SetCellDirect(0, 0, head)
			for offset := 1; offset < width; offset++ {
				want := Cell{Fg: head.Fg, Bg: head.Bg, Width: -offset}
				if !b.Cell(offset, 0).Equals(want) {
					t.Errorf("direct update did not reconstruct column %d", offset)
				}
			}
		})
	}
}

func TestWideClusterOverdraw(t *testing.T) {
	writes := []struct {
		name string
		draw func(*Buffer, int)
	}{
		{"cell", func(b *Buffer, x int) { b.SetCell(x, 0, 'X', style.ColorWhite, style.ColorBlack, 0) }},
		{"direct", func(b *Buffer, x int) { b.SetCellDirect(x, 0, NewCell('X', style.ColorWhite, style.ColorBlack, 0)) }},
		{"blend", func(b *Buffer, x int) { b.SetCellBlended(x, 0, 'X', style.ColorWhite, style.ColorBlack, 0) }},
		{"fill", func(b *Buffer, x int) { b.FillRect(x, 0, 1, 1, style.ColorBlack) }},
		{"blit", func(b *Buffer, x int) {
			src := New(1, 1)
			src.SetCell(0, 0, 'X', style.ColorWhite, style.ColorBlack, 0)
			b.DrawFrameBuffer(x, 0, src)
		}},
	}
	for _, tt := range writes {
		for x := range 3 {
			t.Run(fmt.Sprintf("%s/%d", tt.name, x), func(t *testing.T) {
				b := New(5, 1)
				b.SetGraphemeCluster(0, 0, "\u0915\u093e\u0903", style.ColorWhite, style.ColorBlack, 0)
				tt.draw(b, x)
				for col := range 3 {
					if b.Cell(col, 0).Width != 1 {
						t.Errorf("column %d still belongs to a cluster: %+v", col, b.Cell(col, 0))
					}
				}
			})
		}
	}
}

func TestWideClusterClipping(t *testing.T) {
	for _, width := range []int{2, 3} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			b := New(width, 1)
			b.PushScissorRect(0, 0, 2, 1)
			b.SetGraphemeCluster(0, 0, "\u0915\u093e\u0903", style.ColorWhite, style.ColorBlack, 0)
			b.PopScissorRect()
			for x := range width {
				if !b.Cell(x, 0).Equals(CellClear) {
					t.Errorf("partially clipped cluster wrote column %d", x)
				}
			}
		})
	}
}

func TestWideClusterBlit(t *testing.T) {
	src := New(3, 1)
	src.SetGraphemeCluster(0, 0, "\u0915\u093e\u0903", style.ColorWhite, style.ColorBlack, 0)
	dst := New(5, 1)
	dst.DrawFrameBuffer(1, 0, src)
	for x := range src.Width() {
		if !dst.Cell(x+1, 0).Equals(src.Cell(x, 0)) {
			t.Errorf("blit changed cluster column %d", x)
		}
	}
}
