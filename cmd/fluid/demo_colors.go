package main

import (
	"context"
	"fmt"

	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/terminal"
	"github.com/unaquatic/fluid/widget"
)

var demoColorsCmd = Demo{
	Name:  "colors",
	Short: "16-color, 256-color and named color palettes",
	Run:   runColorsDemo,
}

// NamedColors are style's predefined colors, in the order style declares
// them. ColorTransparent is left out, as it is no color.
var NamedColors = []struct {
	Name  string
	Color style.Color
}{
	{"ColorBlack", style.ColorBlack},
	{"ColorWhite", style.ColorWhite},
	{"ColorRed", style.ColorRed},
	{"ColorGreen", style.ColorGreen},
	{"ColorBlue", style.ColorBlue},
	{"ColorYellow", style.ColorYellow},
	{"ColorCyan", style.ColorCyan},
	{"ColorMagenta", style.ColorMagenta},
	{"ColorSilver", style.ColorSilver},
	{"ColorGray", style.ColorGray},
	{"ColorMaroon", style.ColorMaroon},
	{"ColorOlive", style.ColorOlive},
	{"ColorTeal", style.ColorTeal},
	{"ColorNavy", style.ColorNavy},
	{"ColorPurple", style.ColorPurple},
	{"ColorOrange", style.ColorOrange},
	{"ColorBrightRed", style.ColorBrightRed},
	{"ColorBrightGreen", style.ColorBrightGreen},
	{"ColorBrightYellow", style.ColorBrightYellow},
	{"ColorBrightBlue", style.ColorBrightBlue},
	{"ColorBrightMagenta", style.ColorBrightMagenta},
	{"ColorBrightCyan", style.ColorBrightCyan},
}

// Palette builds styled text a line at a time.
type Palette struct {
	Chunks []style.TextChunk
}

// Text appends s in the surrounding colors.
func (p *Palette) Text(s string) {
	p.Chunks = append(p.Chunks, style.TextChunk{Text: s})
}

// Swatch appends label on c, in black or white, whichever reads better.
func (p *Palette) Swatch(label string, c style.Color) {
	fg := style.ColorWhite
	if c.Luminance() > 0.5 {
		fg = style.ColorBlack
	}
	p.Chunks = append(p.Chunks, style.TextChunk{Text: label, Fg: fg, Bg: c})
}

// Entry appends a swatch of palette entry a, labeled with its number.
func (p *Palette) Entry(a style.ANSI) {
	p.Swatch(fmt.Sprintf(" %3d", a), a.Color())
}

// Line ends the current line.
func (p *Palette) Line() {
	p.Text("\n")
}

// ColorPalettes returns the demo's text: the 16 ANSI colors, the 256-color
// xterm palette, and the named colors with the entries they draw in under
// 256 and 16 colors.
func ColorPalettes() style.StyledText {
	var p Palette

	p.Text("16 colors: fluid's guess at the terminal theme")
	p.Line()
	for row, label := range []string{"standard", "bright"} {
		p.Text("  ")
		for col := range 8 {
			p.Entry(style.ANSI(row*8 + col))
		}
		p.Text("  " + label)
		p.Line()
	}
	p.Line()

	p.Text("256 colors: the xterm palette")
	p.Line()
	p.Text("  ANSI 0-15")
	p.Line()
	p.Text("  ")
	for a := range 16 {
		p.Entry(style.ANSI(a))
	}
	p.Line()
	p.Line()
	// The color cube, entry 16 + 36r + 6g + b, as six blocks of red with
	// green down and blue across, two blocks to a row.
	p.Text("  Cube 16-231")
	p.Line()
	for blocks := range 3 {
		for g := range 6 {
			p.Text("  ")
			for r := blocks * 2; r < blocks*2+2; r++ {
				if r > blocks*2 {
					p.Text("  ")
				}
				for b := range 6 {
					p.Entry(style.ANSI(16 + 36*r + 6*g + b))
				}
			}
			p.Line()
		}
	}
	p.Line()
	p.Text("  Grays 232-255")
	p.Line()
	for row := range 2 {
		p.Text("  ")
		for col := range 12 {
			p.Entry(style.ANSI(232 + row*12 + col))
		}
		p.Line()
	}
	p.Line()

	p.Text("Named colors")
	p.Line()
	p.Text(fmt.Sprintf("  %-20s%-10s%-11s%-7s%s", "name", "hex", "truecolor", "256", "16"))
	p.Line()
	for _, nc := range NamedColors {
		e256, e16 := style.Nearest256.Index(nc.Color), style.Nearest16.Index(nc.Color)
		p.Text(fmt.Sprintf("  %-20s%-10s", nc.Name, nc.Color.ToHex()))
		p.Swatch("         ", nc.Color)
		p.Text("  ")
		p.Swatch(fmt.Sprintf(" %3d ", e256), e256.Color())
		p.Text("  ")
		p.Swatch(fmt.Sprintf(" %3d ", e16), e16.Color())
		p.Line()
	}

	return style.StyledText{Chunks: p.Chunks}
}

// ColorModeName returns the FLUID_COLOR value that selects m.
func ColorModeName(m style.ColorMode) string {
	switch m {
	case style.Color256:
		return "256"
	case style.Color16:
		return "16"
	}
	return "truecolor"
}

func runColorsDemo(ctx context.Context, o DemoOptions) error {
	r, err := terminal.NewRenderer(terminal.Config{
		UseMouse: !o.NoMouse,
	})
	if err != nil {
		return fmt.Errorf("renderer: %w", err)
	}

	palettes := widget.NewStyledText(ColorPalettes(),
		widget.TextWithWrapMode(widget.WrapNone),
	)
	scrollBox := widget.NewScrollBox(
		widget.ScrollBoxWithFlexGrow(1),
		widget.ScrollBoxWithBorderConfig(style.BorderAll(), style.BorderSingle),
		widget.ScrollBoxWithChildren(palettes),
	)
	scrollBar := widget.NewScrollBar()
	widget.BindScrollBar(scrollBox, scrollBar)

	header := fmt.Sprintf("Colors demo | mode: %s (set with FLUID_COLOR=truecolor, 256 or 16)",
		ColorModeName(r.ColorMode()))
	container := widget.NewBox(
		widget.BoxWithFlexGrow(1),
		widget.BoxWithFlexDirection(layout.DirectionColumn),
		widget.BoxWithPadding(layout.Points(1)),
		widget.BoxWithGap(layout.Points(1)),
		widget.Children(
			widget.NewText(header,
				widget.TextWithHeight(layout.Points(1)),
			),
			widget.NewBox(
				widget.BoxWithFlexGrow(1),
				widget.BoxWithFlexDirection(layout.DirectionRow),
				widget.Children(scrollBox, scrollBar),
			),
			widget.NewText("Esc/Q: Exit | Wheel/Arrows/PgUp/PgDn: Scroll",
				widget.TextWithHeight(layout.Points(1)),
			),
		),
	)
	r.Root().Add(container)
	scrollBox.Focus()

	r.EventBus().On(input.EventKeyboard, func(e input.TerminalEvent) {
		if e.Key == nil {
			return
		}
		if e.Key.Name == "escape" || (e.Key.Name == "q" && !e.Key.Mods.Has(input.ModCtrl) && !e.Key.Mods.Has(input.ModAlt)) {
			r.Destroy()
		}
	})

	return r.RunContinuous(ctx)
}
