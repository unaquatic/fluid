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

var demoInlineInputCmd = Demo{
	Name:  "inline-input",
	Short: "Inline mode with cursor/input demo",
	Run:   runInlineInputDemo,
}

func runInlineInputDemo(ctx context.Context, o DemoOptions) error {
	r, err := terminal.NewRenderer(terminal.Config{
		InlineMode: true,
		UseMouse:   false,
	})
	if err != nil {
		return fmt.Errorf("renderer: %w", err)
	}

	root := r.Root()

	titleBox := widget.NewBox(
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderSingle),
		widget.Children(
			widget.NewText("Inline Input Demo"),
		),
	)
	root.Add(titleBox)

	helpText := widget.NewText(
		" Esc: Exit | Alt+Enter: Send",
		widget.TextWithHeight(layout.Points(1)),
	)
	root.Add(helpText)

	msgBox := widget.NewBox(
		widget.BoxWithID("messages"),
		widget.BoxWithFlexDirection(layout.DirectionColumn),
	)
	root.Add(msgBox)

	sep := widget.NewText(
		"─────────────────────────────────────────────────",
		widget.TextWithHeight(layout.Points(1)),
	)
	root.Add(sep)

	inputRow := widget.NewBox(
		widget.BoxWithFlexDirection(layout.DirectionRow),
	)

	inputRow.Add(widget.NewText(
		" > ",
		widget.TextWithWidth(layout.Points(4)),
	))

	ta := widget.NewTextarea(
		widget.TextareaWithPlaceholder("Type a message... (Enter=newline, Alt+Enter=send)"),
		widget.TextareaWithBackgroundColor(style.ColorTransparent),
		widget.TextareaWithWidth(layout.Percent(100)),
		widget.TextareaWithWrapMode(textbuf.WrapWord),
	)
	inputRow.Add(ta)

	inputBox := widget.NewBox(
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderSingle),
		widget.Children(inputRow),
	)
	root.Add(inputBox)

	msgCount := 0
	ta.OnSubmit = func(text string) {
		if text == "" {
			return
		}
		msgCount++
		msg := widget.NewText(
			fmt.Sprintf("  [%d] %s", msgCount, text),
		)
		msgBox.Add(msg)
		ta.SetText("")
		r.RequestRender()
	}

	// Renderer focus initializes the textarea before its first render.
	r.FocusRenderable(ta)

	r.EventBus().On(input.EventKeyboard, func(e input.TerminalEvent) {
		if e.Key == nil {
			return
		}
		if e.Key.Name == "escape" {
			r.Destroy()
		}
	})

	return r.Run(ctx)
}
