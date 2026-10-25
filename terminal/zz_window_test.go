package terminal

import (
	"fmt"
	"testing"

	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/widget"
)

// growStream simulates /demo: a message grows a row at a time, with the prompt
// below it, exactly as a chat app lays the transcript out inline.
func growStream(t *testing.T, startRows, addRows int) {
	const width, height = 120, 40
	r := benchInlineRenderer(width, height)

	filler := widget.NewBox(widget.BoxWithID("filler"),
		widget.BoxWithLayout(
			layout.WithFlexDirection(layout.DirectionColumn),
			layout.WithFlexShrink(0),
		))
	for i := range startRows {
		filler.Add(newGlyphRow(fmt.Sprintf("f%d", i), 'f', width-3, 1))
	}
	r.root.Add(filler)

	bubble := widget.NewBox(widget.BoxWithID("bubble"),
		widget.BoxWithLayout(
			layout.WithFlexDirection(layout.DirectionColumn),
			layout.WithFlexShrink(0),
		))
	r.root.Add(bubble)
	r.root.Add(render.NewBaseRenderable(render.WithID("prompt"),
		render.WithWidth(layout.Points(width-1)), render.WithHeight(layout.Points(3))))
	r.renderInlineFrame()

	var fulls, diffs, skipped, bytes int
	var total int64
	for i := range addRows {
		bubble.Add(newGlyphRow(fmt.Sprintf("b%d", i), 'b', width-3, 1))
		before := r.inlineViewportTop
		r.renderInlineFrame()
		s := r.root.LastFrameStats
		total += s.Total.Nanoseconds()
		bytes += s.FlushBytes
		switch {
		case s.FlushBytes == 0:
			skipped++
		case r.inlineViewportTop != before || s.FlushBytes < 4096:
			diffs++
		default:
			fulls++
		}
	}
	t.Logf("start=%-5d content=%-5d vpTop=%-5d | %d frames: avg %.0f us, %d KB written",
		startRows, r.nextBuffer.Height(), r.inlineViewportTop, addRows,
		float64(total)/float64(addRows)/1000, bytes/1024)
}

func TestWindowBoundary(t *testing.T) {
	t.Log("=== content stays INSIDE the 40-row window ===")
	growStream(t, 5, 30)
	t.Log("=== content goes OUTSIDE the window ===")
	growStream(t, 5, 300)
	growStream(t, 2000, 300)
}
