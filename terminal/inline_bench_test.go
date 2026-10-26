package terminal

import (
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
)

// benchInlineRenderer returns an inline renderer writing to io.Discard, sized
// like a real terminal.
func benchInlineRenderer(width, height int) *Renderer {
	root := NewRootRenderable(width, height)
	cfg := DefaultConfig()
	cfg.InlineMode = true
	r := &Renderer{
		root:          root,
		hitGrid:       NewHitGrid(width, height),
		backend:       NewBackend(strings.NewReader(""), io.Discard),
		config:        cfg,
		currentBuffer: buffer.New(width, height),
		nextBuffer:    buffer.New(width, height),
		width:         width,
		height:        height,
		controlState:  StateAutoStarted,
		parser:        input.NewParser(),
		eventBus:      input.NewEventBus(),
		pendingWork:   make(chan func()),
		wake:          make(chan struct{}, 1),
		stopChan:      make(chan struct{}, 1),
		done:          make(chan struct{}),
		sel:           render.NewSelection(),
		linkRegistry:  newLinkRegistry(),
		stats:         newStats(60),
	}
	root.SetHost(r)
	return r
}

// textRow paints every cell it occupies, so benchmarks measure widgets that
// actually draw rather than empty placeholders.
type textRow struct {
	*render.BaseRenderable
	text string
}

func newTextRow(id string, width, height int) *textRow {
	t := &textRow{
		BaseRenderable: render.NewBaseRenderable(render.WithID(id),
			render.WithWidth(layout.Points(width)), render.WithHeight(layout.Points(height))),
		text: strings.Repeat("transcript ", width/11+1)[:width],
	}
	t.SetSelf(t)
	return t
}

// RenderSelf is the widget content override point, so the base keeps its damage
// bookkeeping, exactly as the real widgets do.
func (t *textRow) RenderSelf(buf *buffer.Buffer, _ time.Duration) {
	for y := range t.Height() {
		buf.DrawText(t.text, t.X(), t.Y()+y, style.ColorWhite)
	}
}

// addTranscript fills the root with bubbles totalling roughly rows rows. Empty
// bubbles isolate the dimension-dependent costs from widget drawing.
func addTranscript(r *Renderer, rows, width int, painted bool) {
	const bubbleRows = 5
	for i := 0; rows > 0; i++ {
		id := fmt.Sprintf("msg-%d", i)
		if painted {
			r.root.Add(newTextRow(id, width-1, bubbleRows))
		} else {
			r.root.Add(render.NewBaseRenderable(render.WithID(id),
				render.WithWidth(layout.Points(width-1)), render.WithHeight(layout.Points(bubbleRows))))
		}
		rows -= bubbleRows
	}
}

// BenchmarkInlineFrame_Keystroke measures one frame in a session of the given
// height, which is the cost paid per keypress while typing.
func BenchmarkInlineFrame_Keystroke(b *testing.B) {
	for _, painted := range []bool{false, true} {
		for _, rows := range []int{500, 5000} {
			b.Run(fmt.Sprintf("painted=%v/rows=%d", painted, rows), func(b *testing.B) {
				const width, height = 120, 40
				r := benchInlineRenderer(width, height)
				addTranscript(r, rows, width, painted)

				// The prompt is the only widget that changes while typing.
				prompt := newTextRow("prompt", width-1, 3)
				r.root.Add(prompt)
				r.renderInlineFrame()

				b.ReportAllocs()
				for b.Loop() {
					// Typing changes the prompt and nothing else.
					prompt.RequestRender()
					r.renderInlineFrame()
				}

				stats := r.root.LastFrameStats
				b.ReportMetric(float64(stats.Layout.Microseconds()), "layout-us")
				b.ReportMetric(float64(stats.BufClear.Microseconds()), "clear-us")
				b.ReportMetric(float64(stats.Render.Microseconds()), "render-us")
				b.ReportMetric(float64(stats.Flush.Microseconds()), "flush-us")
			})
		}
	}
}

// BenchmarkInlineFramePhases breaks a keystroke into its per-frame passes over
// a nested tree, which is the shape a real transcript has.
func BenchmarkInlineFramePhases(b *testing.B) {
	const width, height = 120, 40
	const rowsPerBubble, childrenPerBubble = 5, 4

	for _, bubbles := range []int{100, 1000} {
		b.Run(fmt.Sprintf("bubbles=%d", bubbles), func(b *testing.B) {
			r := benchInlineRenderer(width, height)
			for i := range bubbles {
				outer := render.NewBaseRenderable(render.WithID(fmt.Sprintf("bubble-%d", i)),
					render.WithWidth(layout.Points(width-1)))
				for j := range childrenPerBubble {
					outer.Add(newTextRow(fmt.Sprintf("bubble-%d-%d", i, j),
						width-3, rowsPerBubble/childrenPerBubble+1))
				}
				r.root.Add(outer)
			}
			prompt := newTextRow("prompt", width-1, 3)
			r.root.Add(prompt)
			r.renderInlineFrame()

			nodes := bubbles*(1+childrenPerBubble) + 2
			b.ReportMetric(float64(nodes), "nodes")
			b.ReportAllocs()
			for b.Loop() {
				prompt.RequestRender()
				r.renderInlineFrame()
			}

			s := r.root.LastFrameStats
			b.ReportMetric(float64(s.Layout.Nanoseconds())/1000, "layout-us")
			b.ReportMetric(float64(s.Sync.Nanoseconds())/1000, "sync-us")
			b.ReportMetric(float64(s.Commands.Nanoseconds())/1000, "cmds-us")
			b.ReportMetric(float64(s.Render.Nanoseconds())/1000, "render-us")
		})
	}
}

// BenchmarkInlineDamageScan measures the tree walk that collects damage, which
// runs once per frame regardless of how little changed.
func BenchmarkInlineDamageScan(b *testing.B) {
	const width, height = 120, 40
	for _, bubbles := range []int{100, 1000} {
		b.Run(fmt.Sprintf("bubbles=%d", bubbles), func(b *testing.B) {
			r := benchInlineRenderer(width, height)
			for i := range bubbles {
				outer := render.NewBaseRenderable(render.WithID(fmt.Sprintf("bubble-%d", i)),
					render.WithWidth(layout.Points(width-1)))
				for j := range 4 {
					outer.Add(newTextRow(fmt.Sprintf("bubble-%d-%d", i, j), width-3, 2))
				}
				r.root.Add(outer)
			}
			r.renderInlineFrame()

			b.ReportAllocs()
			for b.Loop() {
				r.root.TakeDamage()
			}
		})
	}
}

// BenchmarkInlineChangedRows measures the per-frame comparison of the previous
// frame against the next one.
func BenchmarkInlineChangedRows(b *testing.B) {
	for _, rows := range []int{50, 500, 5000} {
		b.Run(fmt.Sprintf("rows=%d", rows), func(b *testing.B) {
			const width = 120
			prev := buffer.New(width, rows)
			next := buffer.New(width, rows)
			for y := range rows {
				for x := range width {
					prev.SetCell(x, y, 'a', style.ColorWhite, style.ColorBlack, 0)
					next.SetCell(x, y, 'a', style.ColorWhite, style.ColorBlack, 0)
				}
			}
			next.SetCell(0, rows-1, 'Z', style.ColorWhite, style.ColorBlack, 0)

			b.ReportAllocs()
			for b.Loop() {
				inlineChangedRows(prev, next, 0)
			}
		})
	}
}
