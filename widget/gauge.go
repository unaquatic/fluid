package widget

import (
	"fmt"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/symbol"
)

// Gauge displays a value from zero to one as a progress bar.
type Gauge struct {
	*render.BaseRenderable

	value     float64 // Clamped to the range [0, 1] by SetValue.
	label     string
	showLabel bool

	fillColor       style.Color
	emptyColor      style.Color
	labelColor      style.Color
	backgroundColor style.Color

	// A gradient blends from fillColor to gradientEnd across the filled cells.
	useGradient bool
	gradientEnd style.Color
}

// GaugeOption configures a Gauge.
type GaugeOption func(*Gauge)

// NewGauge returns a Gauge with a one-row layout and a percentage label.
func NewGauge(opts ...GaugeOption) *Gauge {
	g := &Gauge{
		BaseRenderable:  render.NewBaseRenderable(),
		showLabel:       true,
		fillColor:       style.ColorBrightGreen,
		emptyColor:      style.ColorFromInts(50, 50, 50, 255),
		labelColor:      style.ColorWhite,
		backgroundColor: style.ColorTransparent,
	}

	g.LayoutNode().Height = layout.Points(1)

	for _, opt := range opts {
		opt(g)
	}

	return g
}

// SetValue sets the gauge value, clamped to the range [0, 1].
func (g *Gauge) SetValue(v float64) {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	g.value = v
	g.RequestRender()
}

// Value returns the current value.
func (g *Gauge) Value() float64 { return g.value }

// SetLabel sets the label text.
func (g *Gauge) SetLabel(label string) {
	g.label = label
	g.RequestRender()
}

// RenderSelf draws the gauge.
func (g *Gauge) RenderSelf(buf *buffer.Buffer, deltaTime float64) {
	w := g.Width()
	h := g.Height()
	if w <= 0 || h <= 0 {
		return
	}

	x := g.X()
	y := g.Y()

	fillWidth := min(int(g.value*float64(w)), w)

	for row := range h {
		for col := range w {
			if col < fillWidth {
				fillColor := g.fillColor
				if g.useGradient && fillWidth > 0 {
					t := float32(col) / float32(fillWidth)
					fillColor = g.fillColor.Blend(g.gradientEnd, t)
				}
				buf.SetCell(x+col, y+row, symbol.GaugeFill, fillColor, fillColor, 0)
			} else {
				buf.SetCell(x+col, y+row, symbol.GaugeEmpty, g.emptyColor, g.emptyColor, 0)
			}
		}
	}

	// Center the explicit label, or the percentage when no label is set.
	if g.showLabel {
		label := g.label
		if label == "" {
			label = fmt.Sprintf("%d%%", int(g.value*100))
		}
		labelX := max(x+(w-len(label))/2, x)
		labelY := y + h/2
		buf.DrawText(label, labelX, labelY, g.labelColor)
	}
}

// Gauge options.

// GaugeWithValue sets the initial value.
func GaugeWithValue(v float64) GaugeOption {
	return func(g *Gauge) { g.value = v }
}

// GaugeWithLabel sets the label.
func GaugeWithLabel(label string) GaugeOption {
	return func(g *Gauge) { g.label = label }
}

// GaugeWithShowLabel controls whether the gauge displays its label.
func GaugeWithShowLabel(v bool) GaugeOption {
	return func(g *Gauge) { g.showLabel = v }
}

// GaugeWithFillColor sets the fill color.
func GaugeWithFillColor(c style.Color) GaugeOption {
	return func(g *Gauge) { g.fillColor = c }
}

// GaugeWithEmptyColor sets the empty color.
func GaugeWithEmptyColor(c style.Color) GaugeOption {
	return func(g *Gauge) { g.emptyColor = c }
}

// GaugeWithGradient colors the fill with a gradient from start to end.
func GaugeWithGradient(start, end style.Color) GaugeOption {
	return func(g *Gauge) {
		g.fillColor = start
		g.gradientEnd = end
		g.useGradient = true
	}
}

// GaugeWithID sets the widget ID.
func GaugeWithID(id string) GaugeOption {
	return func(g *Gauge) { g.SetID(id) }
}

// GaugeWithWidth sets the widget width.
func GaugeWithWidth(s layout.Scalar) GaugeOption {
	return func(g *Gauge) { g.LayoutNode().Width = s }
}

// GaugeWithHeight sets the widget height.
func GaugeWithHeight(s layout.Scalar) GaugeOption {
	return func(g *Gauge) { g.LayoutNode().Height = s }
}

// GaugeWithFlexGrow sets the flex-grow factor.
func GaugeWithFlexGrow(v int) GaugeOption {
	return func(g *Gauge) { g.LayoutNode().FlexGrow = v }
}
