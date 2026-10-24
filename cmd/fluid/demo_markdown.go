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

var demoMarkdownCmd = Demo{
	Name:  "markdown",
	Short: "Markdown renderer demo",
	Run:   runMarkdownDemo,
}

const sampleMarkdown = `# Fluid Markdown Renderer

A pure Go terminal UI framework with **retained-mode rendering**, flexbox layout, and rich widgets.

## Features

### Core Architecture
- **Retained scene graph** with dirty tracking and opacity compositing
- **Pure Go flexbox** layout engine — no CGo, no Yoga
- **Double-buffered rendering** with cell-level diffing
- **Color color model** with alpha blending and opacity stacking

### Widget System
- Box, Text, Input, Textarea, ScrollBox, Select
- Code viewer with **syntax highlighting**
- Markdown renderer (you're looking at it!)
- Table, List, Gauge, Sparkline, Tree

## Quick Start

Install Fluid:

` + "```" + `bash
go get github.com/unaquatic/fluid
` + "```" + `

Create a simple application:

` + "```" + `go
package main

import (
    "github.com/unaquatic/fluid/widget"
)

func main() {
    r, _ := terminal.NewRenderer(terminal.Config{
        ExitOnCtrlC: true,
    })
    defer r.Destroy()

    box := widget.NewBox(
        widget.BoxWithFlexGrow(1),
    )
    box.Add(widget.NewText("Hello, Fluid!"))
    r.Root().Add(box)

    r.Start()
}
` + "```" + `

## Layout System

Fluid uses a **flexbox-based layout engine** written in pure Go:

1. **Direction** — Row or Column
2. **Flex Grow/Shrink** — Proportional space distribution
3. **Justify Content** — Main axis alignment
4. **Align Items** — Cross axis alignment
5. **Wrapping** — Multi-line flex containers

> The layout engine handles all standard flexbox properties including
> percentage-based sizing, min/max constraints, and absolute positioning.

---

## Animation

The animation system provides:

- 20 built-in easing functions
- Keyframe timelines with ` + "`" + `Add` + "`" + `, ` + "`" + `AddAt` + "`" + `, ` + "`" + `Once` + "`" + `
- Looping and alternating animations
- Property-based animation for opacity, translate, and color

### Easing Functions

| Category | Functions |
|----------|-----------|
| Quad     | InQuad, OutQuad, InOutQuad |
| Cubic    | InCubic, OutCubic, InOutCubic |
| Expo     | InExpo, OutExpo, InOutExpo |
| Bounce   | InBounce, OutBounce |
| Elastic  | OutElastic |
| Back     | InBack, OutBack, InOutBack |

---

## Contributing

Pull requests are welcome. For major changes, please open an issue first.

### Development

` + "```" + `bash
# Run tests
go test ./fluid/...

# Build demos
go build -o fluid ./cmd/fluid
./fluid demo markdown
` + "```" + `

---

*Built with Fluid — a terminal UI framework for Go*
`

func runMarkdownDemo(ctx context.Context, o DemoOptions) error {
	r, err := terminal.NewRenderer(terminal.Config{
		ExitOnCtrlC: true,
		UseMouse:    !o.NoMouse,
		TargetFPS:   o.FPS,
	})
	if err != nil {
		return fmt.Errorf("renderer: %w", err)
	}

	root := r.Root()

	md := widget.NewMarkdown(sampleMarkdown,
		widget.MarkdownWithWrapMode(widget.WrapWord),
	)
	scrollBox := widget.NewScrollBox(
		widget.ScrollBoxWithFlexGrow(1),
		widget.ScrollBoxWithBorderConfig(style.BorderAll(), style.BorderSingle),
		widget.ScrollBoxWithChildren(md),
	)
	scrollBar := widget.NewScrollBar()
	widget.BindScrollBar(scrollBox, scrollBar)

	container := widget.NewBox(
		widget.BoxWithFlexGrow(1),
		widget.BoxWithFlexDirection(layout.DirectionColumn),
		widget.BoxWithPadding(layout.Points(1)),
		widget.BoxWithGap(layout.Points(1)),
		widget.Children(
			widget.NewText("Markdown Viewer Demo",
				widget.TextWithHeight(layout.Points(1)),
			),
			widget.NewBox(
				widget.BoxWithFlexGrow(1),
				widget.BoxWithFlexDirection(layout.DirectionRow),
				widget.Children(scrollBox, scrollBar),
			),
			widget.NewText("Scroll: mouse wheel | Esc/q: quit",
				widget.TextWithHeight(layout.Points(1)),
			),
		),
	)
	root.Add(container)

	r.EventBus().On(input.EventKeyboard, func(e input.TerminalEvent) {
		if e.Key == nil {
			return
		}
		if e.Key.Name == "escape" || (e.Key.Name == "q" && !e.Key.Ctrl && !e.Key.Alt) {
			r.Destroy()
		}
	})

	return r.RunContinuous(ctx)
}
