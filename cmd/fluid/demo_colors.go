package main

import (
	"context"
	"fmt"

	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/terminal"
	"github.com/unaquatic/fluid/textbuf"
	"github.com/unaquatic/fluid/widget"
)

var demoColorsCmd = demo{
	name:  "colors",
	short: "16-color, 256-color and named color palettes",
	run:   runColorsDemo,
}

// namedColors are style's predefined colors, in the order style declares
// them. ColorTransparent is left out, as it is no color.
var namedColors = []struct {
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

// palette builds styled text a line at a time.
type palette struct {
	chunks []style.TextChunk
}

// text appends s in the surrounding colors.
func (p *palette) text(s string) {
	p.chunks = append(p.chunks, style.TextChunk{Text: s})
}

// swatch appends label on c, in black or white, whichever reads better.
func (p *palette) swatch(label string, c style.Color) {
	fg := style.ColorWhite
	if c.Luminance() > 0.5 {
		fg = style.ColorBlack
	}
	p.chunks = append(p.chunks, style.TextChunk{Text: label, Fg: fg, Bg: c})
}

// entry appends a swatch of palette entry a, labeled with its number.
func (p *palette) entry(a style.ANSI) {
	p.swatch(fmt.Sprintf(" %3d", a), a.Color())
}

// line ends the current line.
func (p *palette) line() {
	p.text("\n")
}

// colorPalettes returns the demo's text: the 16 ANSI colors, the 256-color
// xterm palette, and the named colors with the entries they draw in under
// 256 and 16 colors.
func colorPalettes() style.StyledText {
	var p palette

	p.text("16 colors: fluid's guess at the terminal theme")
	p.line()
	for row, label := range []string{"standard", "bright"} {
		p.text("  ")
		for col := range 8 {
			p.entry(style.ANSI(row*8 + col))
		}
		p.text("  " + label)
		p.line()
	}
	p.line()

	p.text("256 colors: the xterm palette")
	p.line()
	p.text("  ANSI 0-15")
	p.line()
	p.text("  ")
	for a := range 16 {
		p.entry(style.ANSI(a))
	}
	p.line()
	p.line()
	// The color cube, entry 16 + 36r + 6g + b, as six blocks of red with
	// green down and blue across, two blocks to a row.
	p.text("  Cube 16-231")
	p.line()
	for blocks := range 3 {
		for g := range 6 {
			p.text("  ")
			for r := blocks * 2; r < blocks*2+2; r++ {
				if r > blocks*2 {
					p.text("  ")
				}
				for b := range 6 {
					p.entry(style.ANSI(16 + 36*r + 6*g + b))
				}
			}
			p.line()
		}
	}
	p.line()
	p.text("  Grays 232-255")
	p.line()
	for row := range 2 {
		p.text("  ")
		for col := range 12 {
			p.entry(style.ANSI(232 + row*12 + col))
		}
		p.line()
	}
	p.line()

	p.text("Named colors")
	p.line()
	p.text(fmt.Sprintf("  %-20s%-10s%-11s%-7s%s", "name", "hex", "truecolor", "256", "16"))
	p.line()
	for _, nc := range namedColors {
		e256, e16 := style.Nearest256.Index(nc.Color), style.Nearest16.Index(nc.Color)
		p.text(fmt.Sprintf("  %-20s%-10s", nc.Name, nc.Color.ToHex()))
		p.swatch("         ", nc.Color)
		p.text("  ")
		p.swatch(fmt.Sprintf(" %3d ", e256), e256.Color())
		p.text("  ")
		p.swatch(fmt.Sprintf(" %3d ", e16), e16.Color())
		p.line()
	}

	return style.StyledText{Chunks: p.chunks}
}

// colorModeName returns the FLUID_COLOR value that selects m.
func colorModeName(m style.ColorMode) string {
	switch m {
	case style.Color256:
		return "256"
	case style.Color16:
		return "16"
	}
	return "truecolor"
}

func runColorsDemo(ctx context.Context, o demoOptions) error {
	r, err := terminal.NewRenderer(terminal.Config{
		UseMouse: !o.noMouse,
	})
	if err != nil {
		return fmt.Errorf("renderer: %w", err)
	}

	palettes := widget.NewStyledText(colorPalettes(),
		widget.TextWithWrapMode(textbuf.WrapNone),
	)
	scrollBox := widget.NewScrollBox(
		widget.ScrollBoxWithLayout(layout.WithFlexGrow(1)),
		widget.ScrollBoxWithBorderConfig(style.BorderAll(), style.BorderSingle),
		widget.ScrollBoxWithChildren(palettes),
	)
	scrollBar := widget.NewScrollBar()
	widget.BindScrollBar(scrollBox, scrollBar)

	header := fmt.Sprintf("Colors demo | mode: %s (set with FLUID_COLOR=truecolor, 256 or 16)",
		colorModeName(r.ColorMode()))
	container := widget.NewBox(
		widget.BoxWithLayout(
			layout.WithFlexGrow(1),
			layout.WithFlexDirection(layout.DirectionColumn),
			layout.WithPadding(layout.Points(1)),
			layout.WithGap(1),
		),
		widget.Children(
			widget.NewText(header,
				widget.TextWithLayout(layout.WithHeight(layout.Points(1))),
			),
			widget.NewBox(
				widget.BoxWithLayout(
					layout.WithFlexGrow(1),
					layout.WithFlexDirection(layout.DirectionRow),
				),
				widget.Children(scrollBox, scrollBar),
			),
			widget.NewText("Esc/Q: Exit | Wheel/Arrows/PgUp/PgDn: Scroll",
				widget.TextWithLayout(layout.WithHeight(layout.Points(1))),
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
