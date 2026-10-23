package terminal

import (
	"fmt"
	"testing"

	"github.com/unaquatic/fluid/layout"
)

func benchFlatSession(rows, width, height int) (*Renderer, *textRow) {
	r := benchInlineRenderer(width, height)
	for i := 0; rows > 0; i++ {
		r.root.Add(newTextRow(fmt.Sprintf("msg-%d", i), width-1, 5))
		rows -= 5
	}
	prompt := newTextRow("prompt", width-1, 3)
	r.root.Add(prompt)
	r.renderInlineFrame()
	return r, prompt
}

func BenchmarkZZ_SameHeight(b *testing.B) {
	for _, rows := range []int{500, 5000} {
		b.Run(fmt.Sprintf("rows=%d", rows), func(b *testing.B) {
			r, prompt := benchFlatSession(rows, 120, 40)
			b.ReportAllocs()
			for range b.N {
				prompt.RequestRender()
				r.renderInlineFrame()
			}
		})
	}
}

func BenchmarkZZ_HeightChange(b *testing.B) {
	for _, rows := range []int{500, 5000} {
		b.Run(fmt.Sprintf("rows=%d", rows), func(b *testing.B) {
			r, prompt := benchFlatSession(rows, 120, 40)
			tall := true
			b.ReportAllocs()
			for range b.N {
				if tall {
					prompt.LayoutNode().Height = layout.Points(9)
				} else {
					prompt.LayoutNode().Height = layout.Points(3)
				}
				tall = !tall
				prompt.LayoutNode().MarkDirty()
				prompt.RequestRender()
				r.renderInlineFrame()
			}
		})
	}
}
