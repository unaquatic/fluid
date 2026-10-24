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

var demoLayoutCmd = Demo{
	Name:  "layout",
	Short: "Flexbox layout engine demo",
	Run:   runLayoutDemo,
}

func runLayoutDemo(ctx context.Context, o DemoOptions) error {
	r, err := terminal.NewRenderer(terminal.Config{
		ExitOnCtrlC: true,
		UseMouse:    !o.NoMouse,
		TargetFPS:   o.FPS,
	})
	if err != nil {
		return fmt.Errorf("renderer: %w", err)
	}

	root := r.Root()

	outer := widget.NewBox(
		widget.BoxWithID("outer"),
		widget.BoxWithFlexGrow(1),
		widget.BoxWithFlexDirection(layout.DirectionColumn),
		widget.Children(
			widget.NewBox(
				widget.BoxWithID("header"),
				widget.BoxWithHeight(layout.Points(3)),
				widget.BoxWithBorder(style.BorderSides(style.SideBottom)),
				widget.BoxWithBorderStyle(style.BorderSingle),
				widget.BoxWithPadding(layout.Points(1)),
				widget.BoxWithJustifyContent(layout.JustifyCenter),
				widget.BoxWithAlignItems(layout.AlignCenter),
				widget.Children(
					widget.NewText("Fluid Layout Demo"),
				),
			),

			widget.NewBox(
				widget.BoxWithID("body"),
				widget.BoxWithFlexGrow(1),
				widget.BoxWithFlexDirection(layout.DirectionRow),
				widget.BoxWithGap(layout.Points(1)),
				widget.BoxWithPadding(layout.Points(1)),
				widget.Children(
					widget.NewBox(
						widget.BoxWithID("col1"),
						widget.BoxWithFlexGrow(1),
						widget.BoxWithBorder(style.BorderAll()),
						widget.BoxWithBorderStyle(style.BorderSingle),
						widget.BoxWithTitle("Grow: 1"),
						widget.BoxWithFlexDirection(layout.DirectionColumn),
						widget.BoxWithPadding(layout.Points(1)),
						widget.BoxWithGap(layout.Points(1)),
						widget.Children(
							widget.NewText("flexGrow: 1"),
							widget.NewText("This column takes"),
							widget.NewText("1/4 of the space"),
						),
					),

					widget.NewBox(
						widget.BoxWithID("col2"),
						widget.BoxWithFlexGrow(2),
						widget.BoxWithBorder(style.BorderAll()),
						widget.BoxWithBorderStyle(style.BorderSingle),
						widget.BoxWithTitle("Grow: 2 (nested)"),
						widget.BoxWithFlexDirection(layout.DirectionColumn),
						widget.BoxWithPadding(layout.Points(1)),
						widget.BoxWithGap(layout.Points(1)),
						widget.Children(
							widget.NewText("flexGrow: 2 — 2/4 of space"),

							widget.NewBox(
								widget.BoxWithID("nested-row"),
								widget.BoxWithFlexDirection(layout.DirectionRow),
								widget.BoxWithFlexGrow(1),
								widget.BoxWithGap(layout.Points(1)),
								widget.Children(
									widget.NewBox(
										widget.BoxWithID("nested-a"),
										widget.BoxWithFlexGrow(1),
										widget.BoxWithBorder(style.BorderAll()),
										widget.BoxWithBorderStyle(style.BorderSingle),
										widget.BoxWithTitle("Nested A"),
										widget.BoxWithPadding(layout.Points(1)),
										widget.BoxWithFlexDirection(layout.DirectionColumn),
										widget.Children(
											widget.NewText("Sub-box A"),
											widget.NewText("grow: 1"),
										),
									),
									widget.NewBox(
										widget.BoxWithID("nested-b"),
										widget.BoxWithFlexGrow(1),
										widget.BoxWithBorder(style.BorderAll()),
										widget.BoxWithBorderStyle(style.BorderSingle),
										widget.BoxWithTitle("Nested B"),
										widget.BoxWithPadding(layout.Points(1)),
										widget.BoxWithFlexDirection(layout.DirectionColumn),
										widget.Children(
											widget.NewText("Sub-box B"),
											widget.NewText("grow: 1"),
										),
									),
								),
							),
						),
					),

					widget.NewBox(
						widget.BoxWithID("col3"),
						widget.BoxWithFlexGrow(1),
						widget.BoxWithBorder(style.BorderAll()),
						widget.BoxWithBorderStyle(style.BorderSingle),
						widget.BoxWithTitle("Grow: 1"),
						widget.BoxWithFlexDirection(layout.DirectionColumn),
						widget.BoxWithPadding(layout.Points(1)),
						widget.BoxWithGap(layout.Points(1)),
						widget.Children(
							widget.NewText("flexGrow: 1"),
							widget.NewText("This column takes"),
							widget.NewText("1/4 of the space"),
							widget.NewText(""),
							widget.NewText("Proportional:"),
							widget.NewText("1 : 2 : 1"),
						),
					),
				),
			),

			widget.NewBox(
				widget.BoxWithID("footer"),
				widget.BoxWithHeight(layout.Points(1)),
				widget.BoxWithFlexDirection(layout.DirectionRow),
				widget.BoxWithJustifyContent(layout.JustifySpaceBetween),
				widget.BoxWithPadding(layout.Points(0)),
				widget.Children(
					widget.NewText(" Esc/q: quit"),
					widget.NewText("Fluid Layout Engine "),
				),
			),
		),
	)
	root.Add(outer)

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
