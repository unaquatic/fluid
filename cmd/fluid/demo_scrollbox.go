package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/terminal"
	"github.com/unaquatic/fluid/widget"
)

var demoScrollboxCmd = demo{
	name:  "scrollbox",
	short: "Scrollable container demo",
	run:   runScrollboxDemo,
}

// pendingLine contains content queued for the render thread.
type pendingLine struct {
	text  string
	color style.Color
}

func runScrollboxDemo(ctx context.Context, o demoOptions) error {
	r, err := terminal.NewRenderer(terminal.Config{
		UseMouse: !o.noMouse,
	})
	if err != nil {
		return fmt.Errorf("renderer: %w", err)
	}

	root := r.Root()

	// Build static content before rendering starts.
	staticContent := widget.NewBox(widget.BoxWithLayout(layout.WithFlexDirection(layout.DirectionColumn)))
	for i := 1; i <= 100_000; i++ {
		staticContent.Add(widget.NewText(
			fmt.Sprintf("Line %3d — Lorem ipsum dolor sit amet, consectetur adipiscing elit", i),
			widget.TextWithLayout(layout.WithHeight(layout.Points(1))),
		))
	}

	staticScroll := widget.NewScrollBox(
		widget.ScrollBoxWithLayout(layout.WithFlexGrow(1)),
		widget.ScrollBoxWithBorderConfig(style.BorderAll(), style.BorderSingle),
		widget.ScrollBoxWithChildren(staticContent),
	)
	staticSB := widget.NewScrollBar()
	widget.BindScrollBar(staticScroll, staticSB)

	dynamicContent := widget.NewBox(widget.BoxWithLayout(layout.WithFlexDirection(layout.DirectionColumn)))
	dynamicScroll := widget.NewScrollBox(
		widget.ScrollBoxWithLayout(layout.WithFlexGrow(1)),
		widget.ScrollBoxWithBorderConfig(style.BorderAll(), style.BorderSingle),
		widget.ScrollBoxWithAutoScroll(widget.AutoScrollBottom),
		widget.ScrollBoxWithChildren(dynamicContent),
	)
	dynamicSB := widget.NewScrollBar()
	widget.BindScrollBar(dynamicScroll, dynamicSB)

	statusText := widget.NewText(
		"Auto-scroll: ON | Lines: 0",
		widget.TextWithLayout(layout.WithHeight(layout.Points(1))),
	)
	frameStatsText := widget.NewText("",
		widget.TextWithLayout(layout.WithHeight(layout.Points(1))),
	)

	container := widget.NewBox(
		widget.BoxWithID("container"),
		widget.BoxWithLayout(
			layout.WithFlexGrow(1),
			layout.WithFlexDirection(layout.DirectionColumn),
			layout.WithPadding(layout.Points(1)),
			layout.WithGap(1),
		),
		widget.Children(
			widget.NewText("ScrollBox Demo",
				widget.TextWithLayout(layout.WithHeight(layout.Points(1))),
			),

			widget.NewBox(
				widget.BoxWithLayout(
					layout.WithFlexDirection(layout.DirectionRow),
					layout.WithFlexGrow(1),
					layout.WithGap(2),
				),
				widget.Children(
					newFlexColumn(
						widget.NewText("Static Content (100k lines)",
							widget.TextWithLayout(layout.WithHeight(layout.Points(1))),
						),
						widget.NewBox(
							widget.BoxWithLayout(
								layout.WithFlexGrow(1),
								layout.WithFlexDirection(layout.DirectionRow),
							),
							widget.Children(staticScroll, staticSB),
						),
					),

					newFlexColumn(
						widget.NewText("Dynamic Content (auto-append)",
							widget.TextWithLayout(layout.WithHeight(layout.Points(1))),
						),
						widget.NewBox(
							widget.BoxWithLayout(
								layout.WithFlexGrow(1),
								layout.WithFlexDirection(layout.DirectionRow),
							),
							widget.Children(dynamicScroll, dynamicSB),
						),
					),
				),
			),

			statusText,
			frameStatsText,
			widget.NewText("Esc/Q: Exit | Wheel: Scroll | A: Add line",
				widget.TextWithLayout(layout.WithHeight(layout.Points(1))),
			),
		),
	)
	root.Add(container)

	// Transfer background additions to the render thread.
	pendingCh := make(chan pendingLine, 256)
	var lineCount atomic.Int32

	queueLine := func() {
		n := int(lineCount.Add(1))
		now := time.Now().Format("15:04:05")
		select {
		case pendingCh <- pendingLine{
			text:  fmt.Sprintf("[%s] Event #%d — new entry appended", now, n),
			color: style.ColorWhite,
		}:
		default:
		}
		r.RequestRender()
	}

	// Seed initial content before rendering starts.
	for range 5 {
		n := int(lineCount.Add(1))
		now := time.Now().Format("15:04:05")
		dynamicContent.Add(widget.NewText(
			fmt.Sprintf("[%s] Event #%d — initial entry", now, n),
			widget.TextWithLayout(layout.WithHeight(layout.Points(1))),
		))
	}
	statusText.SetContent(fmt.Sprintf("Auto-scroll: ON | Lines: %d", lineCount.Load()))

	go func() {
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			if r.IsDestroyed() {
				return
			}
			queueLine()
		}
	}()

	r.EventBus().On(input.EventKeyboard, func(e input.TerminalEvent) {
		if e.Key == nil {
			return
		}
		k := e.Key
		if k.Name == "escape" || (k.Name == "q" && !k.Mods.Has(input.ModCtrl) && !k.Mods.Has(input.ModAlt)) {
			r.Destroy()
			return
		}
		if k.Name == "a" && !k.Mods.Has(input.ModCtrl) && !k.Mods.Has(input.ModAlt) {
			queueLine()
			return
		}
	})

	return r.RunContinuous(ctx)
}

// newFlexColumn returns an equal-width column for the scrollbox demo.
func newFlexColumn(children ...render.Renderable) *widget.Box {
	b := widget.NewBox(
		widget.BoxWithLayout(
			layout.WithFlexGrow(1),
			layout.WithFlexBasis(layout.Points(0)),
			layout.WithFlexDirection(layout.DirectionColumn),
			layout.WithGap(1),
		),
		widget.Children(children...),
	)
	return b
}
