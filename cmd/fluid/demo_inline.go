package main

import (
	"context"
	"fmt"
	"time"

	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/terminal"
	"github.com/unaquatic/fluid/widget"
)

var demoInlineCmd = Demo{
	Name:  "inline",
	Short: "Inline (scrollable) rendering mode demo",
	Run:   runInlineDemo,
}

func runInlineDemo(ctx context.Context, o DemoOptions) error {
	r, err := terminal.NewRenderer(terminal.Config{
		InlineMode: true,
		UseMouse:   false,
	})
	if err != nil {
		return fmt.Errorf("renderer: %w", err)
	}

	root := r.Root()

	outer := widget.NewBox(
		widget.BoxWithID("outer"),
		widget.BoxWithFlexDirection(layout.DirectionColumn),
		widget.Children(
			widget.NewBox(
				widget.BoxWithBorder(style.BorderAll()),
				widget.BoxWithBorderStyle(style.BorderSingle),
				widget.Children(
					widget.NewText("Inline Mode Demo"),
				),
			),

			widget.NewText(
				" a: add line | d: delete | q: quit",
				widget.TextWithHeight(layout.Points(1)),
			),
		),
	)
	root.Add(outer)

	contentBox := widget.NewBox(
		widget.BoxWithID("content"),
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderSingle),
		widget.BoxWithTitle("Content"),
		widget.BoxWithFlexDirection(layout.DirectionColumn),
	)
	root.Add(contentBox)

	lineCount := 0
	addLine := func() {
		lineCount++
		t := widget.NewText(
			fmt.Sprintf("  Line %d — added at %s", lineCount, time.Now().Format("15:04:05")),
			widget.TextWithHeight(layout.Points(1)),
		)
		contentBox.Add(t)
		r.RequestRender()
	}

	for range 3 {
		addLine()
	}

	r.EventBus().On(input.EventKeyboard, func(e input.TerminalEvent) {
		if e.Key == nil {
			return
		}
		switch {
		case e.Key.Name == "escape" || (e.Key.Name == "q" && !e.Key.Mods.Has(input.ModCtrl) && !e.Key.Mods.Has(input.ModAlt)):
			r.Destroy()
		case e.Key.Name == "a":
			addLine()
		case e.Key.Name == "d":
			children := contentBox.Children()
			if len(children) > 0 {
				last := children[len(children)-1]
				contentBox.Remove(last.ID())
				r.RequestRender()
			}
		}
	})

	return r.Run(ctx)
}
