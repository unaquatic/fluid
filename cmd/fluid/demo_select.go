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

var demoSelectCmd = demo{
	name:  "select",
	short: "Select and TabSelect widget demo",
	run:   runSelectDemo,
}

func runSelectDemo(ctx context.Context, o demoOptions) error {
	r, err := terminal.NewRenderer(terminal.Config{
		UseMouse: !o.noMouse,
	})
	if err != nil {
		return fmt.Errorf("renderer: %w", err)
	}

	root := r.Root()

	tabs := widget.NewTabSelect([]widget.TabSelectItem{
		{Name: "General"},
		{Name: "Appearance"},
		{Name: "Keybindings"},
		{Name: "Extensions"},
		{Name: "About"},
	})

	selectOptions := []widget.SelectItem{
		{Name: "Language", Description: "Set the display language"},
		{Name: "Font Size", Description: "Adjust editor font size"},
		{Name: "Tab Size", Description: "Number of spaces per tab"},
		{Name: "Word Wrap", Description: "Enable or disable word wrapping"},
		{Name: "Line Numbers", Description: "Show line numbers in editor"},
		{Name: "Minimap", Description: "Show code minimap"},
		{Name: "Breadcrumbs", Description: "Show file breadcrumbs"},
		{Name: "Auto Save", Description: "Automatic file saving"},
		{Name: "Format On Save", Description: "Format code when saving"},
		{Name: "Theme", Description: "Select color theme"},
		{Name: "Icon Theme", Description: "Select file icon theme"},
		{Name: "Cursor Style", Description: "Change cursor appearance"},
		{Name: "Smooth Scrolling", Description: "Enable smooth scroll animation"},
		{Name: "Bracket Pairs", Description: "Colorize matching brackets"},
		{Name: "Indent Guides", Description: "Show indentation guides"},
	}
	sel := widget.NewSelect(selectOptions,
		widget.SelectWithShowDescription(true),
		widget.SelectWithLayout(layout.WithFlexGrow(1)),
	)

	detailName := widget.NewText("Language")
	detailDesc := widget.NewText("Set the display language")
	detailTab := widget.NewText("Tab: General")

	selectBox := widget.NewBox(
		widget.BoxWithLayout(
			layout.WithFlexGrow(1),
			layout.WithFlexDirection(layout.DirectionColumn),
		),
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderSingle),
		widget.BoxWithTitle("Options"),
		widget.Children(sel),
	)
	detailBox := widget.NewBox(
		widget.BoxWithLayout(
			layout.WithFlexGrow(1),
			layout.WithFlexDirection(layout.DirectionColumn),
			layout.WithPadding(layout.Points(1)),
			layout.WithGap(1),
		),
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderSingle),
		widget.BoxWithTitle("Details"),
		widget.Children(
			detailName,
			detailDesc,
			widget.NewText("", widget.TextWithLayout(layout.WithHeight(layout.Points(1)))),
			detailTab,
		),
	)

	container := widget.NewBox(
		widget.BoxWithLayout(
			layout.WithFlexGrow(1),
			layout.WithFlexDirection(layout.DirectionColumn),
			layout.WithPadding(layout.Points(1)),
			layout.WithGap(1),
		),
		widget.Children(
			widget.NewText("Select & TabSelect Demo",
				widget.TextWithLayout(layout.WithHeight(layout.Points(1))),
			),
			widget.NewBox(
				widget.BoxWithBorder(style.BorderSides(style.SideBottom)),
				widget.BoxWithBorderStyle(style.BorderSingle),
				widget.BoxWithLayout(layout.WithHeight(layout.Points(3))),
				widget.Children(tabs),
			),
			widget.NewBox(
				widget.BoxWithLayout(
					layout.WithFlexGrow(1),
					layout.WithFlexDirection(layout.DirectionRow),
					layout.WithGap(2),
				),
				widget.Children(selectBox, detailBox),
			),
			widget.NewText(
				"Esc/Q: Exit | Tab: Switch focus | Up/Down: Navigate | Left/Right: Tabs | Enter: Select",
				widget.TextWithLayout(layout.WithHeight(layout.Points(1))),
			),
		),
	)
	root.Add(container)

	sel.OnSelectionChanged = func(idx int, item widget.SelectItem) {
		detailName.SetContent(item.Name)
		detailDesc.SetContent(item.Description)
	}
	sel.OnItemSelected = func(idx int, item widget.SelectItem) {
		detailName.SetContent(fmt.Sprintf("[Selected] %s", item.Name))
	}
	tabs.OnSelectionChanged = func(idx int, item widget.TabSelectItem) {
		detailTab.SetContent(fmt.Sprintf("Tab: %s", item.Name))
	}

	focusables := []render.Renderable{tabs, sel}
	focusIdx := 1

	r.FocusRenderable(focusables[focusIdx])

	r.EventBus().On(input.EventKeyboard, func(e input.TerminalEvent) {
		if e.Key == nil {
			return
		}
		k := e.Key

		if k.Name == "escape" || (k.Name == "q" && !k.Mods.Has(input.ModCtrl) && !k.Mods.Has(input.ModAlt)) {
			r.Destroy()
			return
		}
		if k.Name == "tab" && !k.Mods.Has(input.ModCtrl) && !k.Mods.Has(input.ModAlt) {
			k.PreventDefault()
			focusIdx = (focusIdx + 1) % len(focusables)
			r.FocusRenderable(focusables[focusIdx])
			return
		}
	})

	return r.RunContinuous(ctx)
}
