package main

import (
	"context"
	"fmt"

	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/terminal"
	"github.com/unaquatic/fluid/widget"
)

var demoNestedScrollCmd = Demo{
	Name:  "nested-scroll",
	Short: "Nested scrollable containers demo",
	Run:   runNestedScrollDemo,
}

func runNestedScrollDemo(ctx context.Context, o DemoOptions) error {
	r, err := terminal.NewRenderer(terminal.Config{
		ExitOnCtrlC: true,
		UseMouse:    !o.NoMouse,
		TargetFPS:   o.FPS,
	})
	if err != nil {
		return fmt.Errorf("renderer: %w", err)
	}

	root := r.Root()

	leftContent := widget.NewBox(widget.BoxWithFlexDirection(layout.DirectionColumn))
	for i := 1; i <= 40; i++ {
		leftContent.Add(widget.NewText(
			fmt.Sprintf("Left panel line %d", i),
			widget.TextWithHeight(layout.Points(1)),
		))
	}

	rightContent := widget.NewBox(widget.BoxWithFlexDirection(layout.DirectionColumn))
	for i := 1; i <= 40; i++ {
		rightContent.Add(widget.NewText(
			fmt.Sprintf("Right panel line %d", i),
			widget.TextWithHeight(layout.Points(1)),
		))
	}

	leftScroll := widget.NewScrollBox(
		widget.ScrollBoxWithFlexGrow(1),
		widget.ScrollBoxWithBorderConfig(style.BorderAll(), style.BorderSingle),
		widget.ScrollBoxWithChildren(leftContent),
	)
	leftSB := widget.NewScrollBar()
	widget.BindScrollBar(leftScroll, leftSB)

	rightScroll := widget.NewScrollBox(
		widget.ScrollBoxWithFlexGrow(1),
		widget.ScrollBoxWithBorderConfig(style.BorderAll(), style.BorderSingle),
		widget.ScrollBoxWithChildren(rightContent),
	)
	rightSB := widget.NewScrollBar()
	widget.BindScrollBar(rightScroll, rightSB)

	mdContent := `# Nested Markdown

This **markdown** content lives inside a ScrollBox, which itself is inside the outer ScrollBox.

## How It Works

- The inner ScrollBox captures scroll events first
- When at a scroll boundary, events **bubble** to the outer ScrollBox
- Each ScrollBox has its own scrollbar

## Code Example

` + "```go" + `
scrollBox := widget.NewScrollBox(
    widget.ScrollBoxWithScrollBarY(true),
)
scrollBox.Add(widget.NewMarkdown(content))
` + "```" + `

## More Content

Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.

> Scroll events propagate from the innermost ScrollBox outward,
> stopping at the first container that can actually scroll.

### List of Items

1. First item in the list
2. Second item with more text
3. Third item
4. Fourth item
5. Fifth item to push content longer

---

*End of nested markdown content*
`
	md := widget.NewMarkdown(mdContent,
		widget.MarkdownWithWrapMode(widget.WrapWord),
	)
	mdScroll := widget.NewScrollBox(
		widget.ScrollBoxWithFlexGrow(1),
		widget.ScrollBoxWithBorderConfig(style.BorderAll(), style.BorderSingle),
		widget.ScrollBoxWithChildren(md),
	)
	mdSB := widget.NewScrollBar()
	widget.BindScrollBar(mdScroll, mdSB)

	section4Lines := make([]render.Renderable, 0, 20)
	for i := 1; i <= 20; i++ {
		section4Lines = append(section4Lines, widget.NewText(
			fmt.Sprintf("  Bottom line %d — scroll down in the outer ScrollBox to see more", i),
			widget.TextWithHeight(layout.Points(1)),
		))
	}

	section1Lines := make([]render.Renderable, 0, 5)
	for i := 1; i <= 5; i++ {
		section1Lines = append(section1Lines, widget.NewText(
			fmt.Sprintf("  Outer line %d — this content scrolls in the outer ScrollBox", i),
			widget.TextWithHeight(layout.Points(1)),
		))
	}

	outerContentChildren := []render.Renderable{
		widget.NewText("Section 1 — Top of outer scroll",
			widget.TextWithHeight(layout.Points(1)),
		),
	}
	outerContentChildren = append(outerContentChildren, section1Lines...)
	outerContentChildren = append(outerContentChildren,
		widget.NewText("Section 2 — Inner scroll panels (scroll independently)",
			widget.TextWithHeight(layout.Points(1)),
		),
		widget.NewBox(
			widget.BoxWithFlexDirection(layout.DirectionRow),
			widget.BoxWithHeight(layout.Points(12)),
			widget.BoxWithGap(layout.Points(2)),
			widget.Children(
				newFlexPanel(leftScroll, leftSB),
				newFlexPanel(rightScroll, rightSB),
			),
		),
		widget.NewText("Section 3 — Markdown in a ScrollBox",
			widget.TextWithHeight(layout.Points(1)),
		),
		widget.NewBox(
			widget.BoxWithHeight(layout.Points(15)),
			widget.BoxWithFlexDirection(layout.DirectionRow),
			widget.Children(mdScroll, mdSB),
		),
		widget.NewText("Section 4 — Below the nested scrolls",
			widget.TextWithHeight(layout.Points(1)),
		),
	)
	outerContentChildren = append(outerContentChildren, section4Lines...)

	outerContent := widget.NewBox(
		widget.BoxWithFlexDirection(layout.DirectionColumn),
		widget.BoxWithGap(layout.Points(1)),
		widget.Children(outerContentChildren...),
	)

	outerScroll := widget.NewScrollBox(
		widget.ScrollBoxWithFlexGrow(1),
		widget.ScrollBoxWithBorderConfig(style.BorderAll(), style.BorderSingle),
		widget.ScrollBoxWithChildren(outerContent),
	)
	outerSB := widget.NewScrollBar()
	widget.BindScrollBar(outerScroll, outerSB)

	container := widget.NewBox(
		widget.BoxWithFlexGrow(1),
		widget.BoxWithFlexDirection(layout.DirectionColumn),
		widget.BoxWithPadding(layout.Points(1)),
		widget.BoxWithGap(layout.Points(1)),
		widget.Children(
			widget.NewText("Nested Scroll Demo",
				widget.TextWithHeight(layout.Points(1)),
			),
			widget.NewText("Scroll inner panels independently. Events bubble to outer when at boundary.",
				widget.TextWithHeight(layout.Points(1)),
			),
			widget.NewBox(
				widget.BoxWithFlexGrow(1),
				widget.BoxWithFlexDirection(layout.DirectionRow),
				widget.Children(outerScroll, outerSB),
			),
			widget.NewText("Mouse wheel: scroll (inner first, then outer) | Esc/q: quit",
				widget.TextWithHeight(layout.Points(1)),
			),
		),
	)
	root.Add(container)

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

// newFlexPanel returns an equal-width row containing a scroll box and scroll bar.
func newFlexPanel(scroll *widget.ScrollBox, sb *widget.ScrollBar) *widget.Box {
	panel := widget.NewBox(
		widget.BoxWithFlexGrow(1),
		widget.BoxWithFlexBasis(layout.Points(0)),
		widget.BoxWithFlexDirection(layout.DirectionRow),
		widget.Children(scroll, sb),
	)
	return panel
}
