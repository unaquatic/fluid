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

var demoSelectCmd = Demo{
	Name:  "select",
	Short: "Select and TabSelect widget demo",
	Run:   runSelectDemo,
}

func runSelectDemo(ctx context.Context, o DemoOptions) error {
	r, err := terminal.NewRenderer(terminal.Config{
		ExitOnCtrlC: true,
		UseMouse:    !o.NoMouse,
		TargetFPS:   o.FPS,
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
		widget.SelectWithFlexGrow(1),
	)

	detailName := widget.NewText("Language")
	detailDesc := widget.NewText("Set the display language")
	detailTab := widget.NewText("Tab: General")

	selectBox := widget.NewBox(
		widget.BoxWithFlexGrow(1),
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderSingle),
		widget.BoxWithTitle("Options"),
		widget.BoxWithFlexDirection(layout.DirectionColumn),
		widget.Children(sel),
	)
	detailBox := widget.NewBox(
		widget.BoxWithFlexGrow(1),
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderSingle),
		widget.BoxWithTitle("Details"),
		widget.BoxWithFlexDirection(layout.DirectionColumn),
		widget.BoxWithPadding(layout.Points(1)),
		widget.BoxWithGap(layout.Points(1)),
		widget.Children(
			detailName,
			detailDesc,
			widget.NewText("", widget.TextWithHeight(layout.Points(1))),
			detailTab,
		),
	)

	container := widget.NewBox(
		widget.BoxWithFlexGrow(1),
		widget.BoxWithFlexDirection(layout.DirectionColumn),
		widget.BoxWithPadding(layout.Points(1)),
		widget.BoxWithGap(layout.Points(1)),
		widget.Children(
			widget.NewText("Select & TabSelect Demo",
				widget.TextWithHeight(layout.Points(1)),
			),
			widget.NewBox(
				widget.BoxWithBorder(style.BorderSides(style.SideBottom)),
				widget.BoxWithBorderStyle(style.BorderSingle),
				widget.BoxWithHeight(layout.Points(3)),
				widget.Children(tabs),
			),
			widget.NewBox(
				widget.BoxWithFlexGrow(1),
				widget.BoxWithFlexDirection(layout.DirectionRow),
				widget.BoxWithGap(layout.Points(2)),
				widget.Children(selectBox, detailBox),
			),
			widget.NewText(
				"Tab: switch focus | Up/Down: navigate | Left/Right: tabs | Enter: select | Esc/q: quit",
				widget.TextWithHeight(layout.Points(1)),
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
	tabs.OnSelectionChanged = func(idx int) {
		tabNames := []string{"General", "Appearance", "Keybindings", "Extensions", "About"}
		if idx < len(tabNames) {
			detailTab.SetContent(fmt.Sprintf("Tab: %s", tabNames[idx]))
		}
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
