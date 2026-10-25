package widget

import (
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/symbol"
)

// sparkBlocks contains the bottom-to-top fill levels used for each cell.
var sparkBlocks = symbol.BarNineLevels[:]

// Sparkline is a compact line chart using block characters.
type Sparkline struct {
	*render.BaseRenderable

	data     []float64
	maxValue float64 // Zero enables scaling from the data range.
	minValue float64

	fg              style.Color
	backgroundColor style.Color
}

// SparklineOption configures a Sparkline.
type SparklineOption func(*Sparkline)

// NewSparkline returns a one-row Sparkline for data.
func NewSparkline(data []float64, opts ...SparklineOption) *Sparkline {
	s := &Sparkline{
		BaseRenderable:  render.NewBaseRenderable(),
		data:            data,
		fg:              style.ColorBrightGreen,
		backgroundColor: style.ColorTransparent,
	}

	s.LayoutNode().Height = layout.Points(1)

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// SetData replaces the sparkline data points.
func (s *Sparkline) SetData(data []float64) {
	s.data = data
	s.RequestRender()
}

// AddPoint appends a data point.
func (s *Sparkline) AddPoint(v float64) {
	s.data = append(s.data, v)
	// Retain only points that can be displayed at the current width.
	if w := s.Width(); w > 0 && len(s.data) > w {
		s.data = s.data[len(s.data)-w:]
	}
	s.RequestRender()
}

// RenderSelf draws the sparkline.
func (s *Sparkline) RenderSelf(buf *buffer.Buffer, deltaTime time.Duration) {
	w := s.Width()
	h := s.Height()
	if w <= 0 || h <= 0 || len(s.data) == 0 {
		return
	}

	x := s.X()
	y := s.Y()

	// Fill the viewport when an opaque background is configured.
	if !s.backgroundColor.IsTransparent() {
		buf.FillRect(x, y, w, h, s.backgroundColor)
	}

	// Derive both bounds from the data when automatic scaling is enabled.
	minVal := s.minValue
	maxVal := s.maxValue
	if maxVal == 0 {
		maxVal = s.data[0]
		minVal = s.data[0]
		for _, v := range s.data {
			if v > maxVal {
				maxVal = v
			}
			if v < minVal {
				minVal = v
			}
		}
	}

	valRange := maxVal - minVal
	if valRange == 0 {
		valRange = 1
	}

	// Show the newest visible points, right-aligned when they do not fill the width.
	dataLen := len(s.data)
	startIdx := 0
	if dataLen > w {
		startIdx = dataLen - w
	}
	startX := x
	if dataLen < w {
		startX = x + w - dataLen
	}

	levels := len(sparkBlocks) - 1

	for i := startIdx; i < dataLen; i++ {
		col := startX + (i - startIdx)
		if col >= x+w {
			break
		}

		normalized := (s.data[i] - minVal) / valRange
		if normalized < 0 {
			normalized = 0
		}
		if normalized > 1 {
			normalized = 1
		}

		// Fill each data column from the bottom upward.
		for row := range h {
			screenY := y + h - 1 - row
			cellFraction := normalized*float64(h) - float64(row)

			if cellFraction >= 1.0 {
				buf.SetCell(col, screenY, sparkBlocks[levels], s.fg, s.backgroundColor, 0)
			} else if cellFraction > 0 {
				level := min(max(int(cellFraction*float64(levels)), 0), levels)
				buf.SetCell(col, screenY, sparkBlocks[level], s.fg, s.backgroundColor, 0)
			}
		}
	}
}

// Sparkline options.

// SparklineWithFg sets the foreground color.
func SparklineWithFg(c style.Color) SparklineOption {
	return func(s *Sparkline) { s.fg = c }
}

// SparklineWithBg sets the background color.
func SparklineWithBg(c style.Color) SparklineOption {
	return func(s *Sparkline) { s.backgroundColor = c }
}

// SparklineWithMaxValue sets the upper scaling bound; zero enables automatic scaling.
func SparklineWithMaxValue(v float64) SparklineOption {
	return func(s *Sparkline) { s.maxValue = v }
}

// SparklineWithID sets the widget ID.
func SparklineWithID(id string) SparklineOption {
	return func(s *Sparkline) { s.SetID(id) }
}

// SparklineWithLayout sets layout properties of the widget, such as its size,
// flex factors, margins, and padding, as opts set them in order.
func SparklineWithLayout(opts ...layout.Option) SparklineOption {
	return func(w *Sparkline) {
		for _, opt := range opts {
			opt(w.LayoutNode())
		}
	}
}
