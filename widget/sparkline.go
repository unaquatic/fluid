package widget

import (
	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/symbol"
)

// sparkBlocks are vertical-fill block characters for sparkline rendering
// (bottom to top), sourced from symbols.BarNineLevels.
var sparkBlocks = symbol.BarNineLevels[:]

// Sparkline is a compact line chart using block characters.
type Sparkline struct {
	*render.BaseRenderable

	data     []float64
	maxValue float64 // 0 = auto-scale
	minValue float64

	fg              style.Color
	backgroundColor style.Color
}

// SparklineOption configures a Sparkline.
type SparklineOption func(*Sparkline)

// NewSparkline creates a Sparkline widget.
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

// SetData sets the sparkline data points.
func (s *Sparkline) SetData(data []float64) {
	s.data = data
	s.RequestRender()
}

// AddPoint appends a data point.
func (s *Sparkline) AddPoint(v float64) {
	s.data = append(s.data, v)
	// Keep data within width
	if w := s.Width(); w > 0 && len(s.data) > w {
		s.data = s.data[len(s.data)-w:]
	}
	s.RequestRender()
}

// RenderSelf draws the sparkline.
func (s *Sparkline) RenderSelf(buf *buffer.Buffer, deltaTime float64) {
	w := s.Width()
	h := s.Height()
	if w <= 0 || h <= 0 || len(s.data) == 0 {
		return
	}

	x := s.X()
	y := s.Y()

	// Fill background
	if !s.backgroundColor.IsTransparent() {
		buf.FillRect(x, y, w, h, s.backgroundColor)
	}

	// Find min/max
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

	// Draw sparkline (right-aligned if data is shorter than width)
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

		// For multi-row sparklines, fill from bottom up
		for row := range h {
			screenY := y + h - 1 - row
			cellFraction := normalized*float64(h) - float64(row)

			if cellFraction >= 1.0 {
				buf.SetCell(col, screenY, sparkBlocks[levels], s.fg, s.backgroundColor, 0)
			} else if cellFraction > 0 {
				level := max(int(cellFraction*float64(levels)), 0)
				if level > levels {
					level = levels
				}
				buf.SetCell(col, screenY, sparkBlocks[level], s.fg, s.backgroundColor, 0)
			}
		}
	}
}

// --- Sparkline Options ---

// SparklineWithFg sets the foreground color.
func SparklineWithFg(c style.Color) SparklineOption {
	return func(s *Sparkline) { s.fg = c }
}

// SparklineWithBg sets the background color.
func SparklineWithBg(c style.Color) SparklineOption {
	return func(s *Sparkline) { s.backgroundColor = c }
}

// SparklineWithMaxValue sets the maximum value for scaling.
func SparklineWithMaxValue(v float64) SparklineOption {
	return func(s *Sparkline) { s.maxValue = v }
}

// SparklineWithID sets the widget ID.
func SparklineWithID(id string) SparklineOption {
	return func(s *Sparkline) { s.SetID(id) }
}

// SparklineWithWidth sets the widget width.
func SparklineWithWidth(sc layout.Scalar) SparklineOption {
	return func(s *Sparkline) { s.LayoutNode().Width = sc }
}

// SparklineWithHeight sets the widget height.
func SparklineWithHeight(sc layout.Scalar) SparklineOption {
	return func(s *Sparkline) { s.LayoutNode().Height = sc }
}

// SparklineWithFlexGrow sets flex grow.
func SparklineWithFlexGrow(v int) SparklineOption {
	return func(s *Sparkline) { s.LayoutNode().FlexGrow = v }
}
