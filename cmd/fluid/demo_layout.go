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

var demoLayoutCmd = demo{
	name:  "layout",
	short: "Flexbox layout engine demo",
	run:   runLayoutDemo,
}

func runLayoutDemo(ctx context.Context, o demoOptions) error {
	r, err := terminal.NewRenderer(terminal.Config{
		UseMouse: !o.noMouse,
	})
	if err != nil {
		return fmt.Errorf("renderer: %w", err)
	}

	root := r.Root()

	outer := widget.NewBox(
		widget.BoxWithID("outer"),
		widget.BoxWithLayout(
			layout.WithFlexGrow(1),
			layout.WithFlexDirection(layout.DirectionColumn),
		),
		widget.Children(
			widget.NewBox(
				widget.BoxWithID("header"),
				widget.BoxWithLayout(
					layout.WithHeight(layout.Points(3)),
					layout.WithPadding(layout.Points(1)),
					layout.WithJustifyContent(layout.JustifyCenter),
					layout.WithAlignItems(layout.AlignCenter),
				),
				widget.BoxWithBorder(style.BorderSides(style.SideBottom)),
				widget.BoxWithBorderStyle(style.BorderSingle),
				widget.Children(
					widget.NewText("Fluid Layout Demo"),
				),
			),

			widget.NewBox(
				widget.BoxWithID("body"),
				widget.BoxWithLayout(
					layout.WithFlexGrow(1),
					layout.WithFlexDirection(layout.DirectionRow),
					layout.WithPadding(layout.Points(1)),
					layout.WithGap(1),
				),
				widget.Children(
					widget.NewBox(
						widget.BoxWithID("col1"),
						widget.BoxWithLayout(
							layout.WithFlexGrow(1),
							layout.WithFlexDirection(layout.DirectionColumn),
							layout.WithPadding(layout.Points(1)),
							layout.WithGap(1),
						),
						widget.BoxWithBorder(style.BorderAll()),
						widget.BoxWithBorderStyle(style.BorderSingle),
						widget.BoxWithTitle("Grow: 1"),
						widget.Children(
							widget.NewText("flexGrow: 1"),
							widget.NewText("This column takes"),
							widget.NewText("1/4 of the space"),
						),
					),

					widget.NewBox(
						widget.BoxWithID("col2"),
						widget.BoxWithLayout(
							layout.WithFlexGrow(2),
							layout.WithFlexDirection(layout.DirectionColumn),
							layout.WithPadding(layout.Points(1)),
							layout.WithGap(1),
						),
						widget.BoxWithBorder(style.BorderAll()),
						widget.BoxWithBorderStyle(style.BorderSingle),
						widget.BoxWithTitle("Grow: 2 (nested)"),
						widget.Children(
							widget.NewText("flexGrow: 2 — 2/4 of space"),

							widget.NewBox(
								widget.BoxWithID("nested-row"),
								widget.BoxWithLayout(
									layout.WithFlexDirection(layout.DirectionRow),
									layout.WithFlexGrow(1),
									layout.WithGap(1),
								),
								widget.Children(
									widget.NewBox(
										widget.BoxWithID("nested-a"),
										widget.BoxWithLayout(
											layout.WithFlexGrow(1),
											layout.WithPadding(layout.Points(1)),
											layout.WithFlexDirection(layout.DirectionColumn),
										),
										widget.BoxWithBorder(style.BorderAll()),
										widget.BoxWithBorderStyle(style.BorderSingle),
										widget.BoxWithTitle("Nested A"),
										widget.Children(
											widget.NewText("Sub-box A"),
											widget.NewText("grow: 1"),
										),
									),
									widget.NewBox(
										widget.BoxWithID("nested-b"),
										widget.BoxWithLayout(
											layout.WithFlexGrow(1),
											layout.WithPadding(layout.Points(1)),
											layout.WithFlexDirection(layout.DirectionColumn),
										),
										widget.BoxWithBorder(style.BorderAll()),
										widget.BoxWithBorderStyle(style.BorderSingle),
										widget.BoxWithTitle("Nested B"),
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
						widget.BoxWithLayout(
							layout.WithFlexGrow(1),
							layout.WithFlexDirection(layout.DirectionColumn),
							layout.WithPadding(layout.Points(1)),
							layout.WithGap(1),
						),
						widget.BoxWithBorder(style.BorderAll()),
						widget.BoxWithBorderStyle(style.BorderSingle),
						widget.BoxWithTitle("Grow: 1"),
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
				widget.BoxWithLayout(
					layout.WithHeight(layout.Points(1)),
					layout.WithFlexDirection(layout.DirectionRow),
					layout.WithJustifyContent(layout.JustifySpaceBetween),
					layout.WithPadding(layout.Points(0)),
				),
				widget.Children(
					widget.NewText(" Esc/Q: Exit"),
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
		if e.Key.Name == "escape" || (e.Key.Name == "q" && !e.Key.Mods.Has(input.ModCtrl) && !e.Key.Mods.Has(input.ModAlt)) {
			r.Destroy()
		}
	})

	return r.RunContinuous(ctx)
}
