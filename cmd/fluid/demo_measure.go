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

var demoMeasureCmd = Demo{
	Name:  "measure",
	Short: "Textarea intrinsic sizing with MeasureFunc",
	Run:   runMeasureDemo,
}

func runMeasureDemo(ctx context.Context, o DemoOptions) error {
	r, err := terminal.NewRenderer(terminal.Config{
		UseMouse: !o.NoMouse,
	})
	if err != nil {
		return fmt.Errorf("renderer: %w", err)
	}

	root := r.Root()

	autoHeightTA := widget.NewTextarea(
		widget.TextareaWithID("auto-height"),
		widget.TextareaWithText("Type here and watch the box grow.\nAdd more lines with Enter!"),
		widget.TextareaWithWrapMode(textbuf.WrapWord),
		widget.TextareaWithShowCursor(true),
		widget.TextareaWithLayout(
			layout.WithHeight(layout.Auto()),
			layout.WithMaxHeight(layout.Points(10)),
		),
	)

	fixedTA := widget.NewTextarea(
		widget.TextareaWithID("fixed"),
		widget.TextareaWithText("This textarea has a fixed height of 5 lines.\nIt scrolls when content exceeds the viewport.\nLine 3\nLine 4\nLine 5\nLine 6 (scroll to see)"),
		widget.TextareaWithWrapMode(textbuf.WrapWord),
		widget.TextareaWithShowCursor(true),
		widget.TextareaWithLayout(layout.WithHeight(layout.Points(5))),
	)

	autoTA := widget.NewTextarea(
		widget.TextareaWithID("auto"),
		widget.TextareaWithText("short"),
		widget.TextareaWithWrapMode(textbuf.WrapNone),
		widget.TextareaWithShowCursor(true),
		widget.TextareaWithLayout(
			layout.WithWidth(layout.Auto()),
			layout.WithHeight(layout.Auto()),
			layout.WithMaxHeight(layout.Points(8)),
		),
	)

	statusText := widget.NewText("",
		widget.TextWithLayout(layout.WithHeight(layout.Points(1))),
	)

	autoHeightBox := widget.NewBox(
		widget.BoxWithID("auto-height-box"),
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderSingle),
		widget.BoxWithTitle("Auto Height"),
		widget.BoxWithLayout(layout.WithPadding(layout.Points(1))),
		widget.Children(autoHeightTA),
	)
	fixedBox := widget.NewBox(
		widget.BoxWithID("fixed-box"),
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderSingle),
		widget.BoxWithTitle("Fixed Height"),
		widget.BoxWithLayout(layout.WithPadding(layout.Points(1))),
		widget.Children(fixedTA),
	)
	autoBox := widget.NewBox(
		widget.BoxWithID("auto-box"),
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderSingle),
		widget.BoxWithTitle("Auto Width + Height"),
		widget.BoxWithLayout(layout.WithPadding(layout.Points(1))),
		widget.Children(autoTA),
	)

	container := widget.NewBox(
		widget.BoxWithID("container"),
		widget.BoxWithLayout(
			layout.WithFlexGrow(1),
			layout.WithPadding(layout.Points(1)),
			layout.WithFlexDirection(layout.DirectionColumn),
			layout.WithGap(1),
		),
		widget.Children(
			widget.NewText("MeasureFunc Demo - Textarea Intrinsic Sizing",
				widget.TextWithLayout(layout.WithHeight(layout.Points(1))),
			),
			widget.NewText("Auto Height (grows with content, max 10 lines):",
				widget.TextWithLayout(layout.WithHeight(layout.Points(1))),
			),
			autoHeightBox,
			widget.NewText("Fixed Height (5 lines, scrolls when content overflows):",
				widget.TextWithLayout(layout.WithHeight(layout.Points(1))),
			),
			fixedBox,
			widget.NewText("Auto Width + Height (no wrap, sizes to content):",
				widget.TextWithLayout(layout.WithHeight(layout.Points(1))),
			),
			autoBox,
			widget.NewText("Esc/Q: Exit | Tab: Cycle focus",
				widget.TextWithLayout(layout.WithHeight(layout.Points(1))),
			),
			statusText,
		),
	)
	root.Add(container)

	textareas := []*widget.Textarea{autoHeightTA, fixedTA, autoTA}
	boxes := []*widget.Box{autoHeightBox, fixedBox, autoBox}
	focusIdx := 0

	updateBorders := func(active int) {
		for i, box := range boxes {
			_ = i == active // Focus colors are intentionally disabled.
			box.RequestRender()
		}
	}

	updateStatus := func(ta *widget.Textarea) {
		row, col := ta.CursorPosition()
		lines := ta.LineCount()
		vlines := ta.TotalVirtualLineCount()
		w, h := ta.Width(), ta.Height()
		statusText.SetContent(fmt.Sprintf(
			"Ln %d, Col %d | Lines: %d | VLines: %d | Size: %dx%d",
			row+1, col+1, lines, vlines, w, h,
		))
	}

	for _, ta := range textareas {
		ta.OnContentChange = func(_ string) { updateStatus(ta) }
		ta.OnCursorChange = func(_, _ int) { updateStatus(ta) }
	}

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
			focusIdx = (focusIdx + 1) % len(textareas)
			r.FocusRenderable(textareas[focusIdx])
			updateBorders(focusIdx)
			updateStatus(textareas[focusIdx])
			return
		}
	})

	if err := r.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	r.FocusRenderable(textareas[focusIdx])
	updateBorders(focusIdx)
	updateStatus(textareas[focusIdx])

	return r.Wait(ctx)
}
