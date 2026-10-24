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

var demoTabCmd = Demo{
	Name:  "tab",
	Short: "Tab handling demo",
	Run:   runTabDemo,
}

const sampleTabCode = "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfor i := 0; i < 10; i++ {\n\t\tif i%2 == 0 {\n\t\t\tfmt.Println(\"even:\", i)\n\t\t} else {\n\t\t\tfmt.Println(\"odd:\", i)\n\t\t}\n\t}\n\n\t// Mixed tabs and spaces\n\tfmt.Println(\n\t\t\"hello\",\n\t\t\"world\",\n\t)\n}\n"

func runTabDemo(ctx context.Context, o DemoOptions) error {
	r, err := terminal.NewRenderer(terminal.Config{
		UseMouse: !o.NoMouse,
	})
	if err != nil {
		return fmt.Errorf("renderer: %w", err)
	}

	root := r.Root()
	tabWidth := 4

	codeWidget := widget.NewCode(sampleTabCode,
		widget.CodeWithFiletype("go"),
		widget.CodeWithTheme(style.DarculaSolid()),
		widget.CodeWithWrapMode(widget.WrapWord),
		widget.CodeWithFlexGrow(1),
	)

	ta := widget.NewTextarea(
		widget.TextareaWithText("Line 1\n\tIndented once\n\t\tIndented twice\n\t\t\tIndented thrice\nNo indent\n\tMixed:\ttwo\ttabs\n"),
		widget.TextareaWithFlexGrow(1),
		widget.TextareaWithWrapMode(widget.WrapWord),
	)

	statusText := widget.NewText(
		fmt.Sprintf("Tab width: %d | Esc/Q: Exit | T: Cycle 2/4/8", tabWidth),
		widget.TextWithHeight(layout.Points(1)),
	)

	container := widget.NewBox(
		widget.BoxWithFlexGrow(1),
		widget.BoxWithFlexDirection(layout.DirectionColumn),
		widget.BoxWithPadding(layout.Points(1)),
		widget.BoxWithGap(layout.Points(1)),
		widget.Children(
			widget.NewText("Tab Handling Demo",
				widget.TextWithHeight(layout.Points(1)),
			),
			widget.NewBox(
				widget.BoxWithFlexGrow(1),
				widget.BoxWithFlexDirection(layout.DirectionRow),
				widget.BoxWithGap(layout.Points(1)),
				widget.Children(
					widget.NewBox(
						widget.BoxWithFlexGrow(1),
						widget.BoxWithFlexDirection(layout.DirectionColumn),
						widget.BoxWithBorder(style.BorderAll()),
						widget.BoxWithBorderStyle(style.BorderSingle),
						widget.BoxWithTitle("Code (read-only)"),
						widget.Children(codeWidget),
					),
					widget.NewBox(
						widget.BoxWithFlexGrow(1),
						widget.BoxWithFlexDirection(layout.DirectionColumn),
						widget.BoxWithBorder(style.BorderAll()),
						widget.BoxWithBorderStyle(style.BorderSingle),
						widget.BoxWithTitle("Textarea (editable)"),
						widget.Children(ta),
					),
				),
			),
			statusText,
		),
	)
	root.Add(container)

	r.EventBus().On(input.EventKeyboard, func(e input.TerminalEvent) {
		if e.Key == nil {
			return
		}
		k := e.Key

		if k.Name == "escape" || (k.Name == "q" && !k.Mods.Has(input.ModCtrl) && !k.Mods.Has(input.ModAlt)) {
			r.Destroy()
			return
		}
		if k.Name == "t" && !k.Mods.Has(input.ModCtrl) && !k.Mods.Has(input.ModAlt) {
			switch tabWidth {
			case 2:
				tabWidth = 4
			case 4:
				tabWidth = 8
			default:
				tabWidth = 2
			}
			codeWidget.TextBuffer().SetTabWidth(tabWidth)
			ta.TextBuffer().SetTabWidth(tabWidth)
			statusText.SetContent(fmt.Sprintf("Tab width: %d | Esc/Q: Exit | T: Cycle 2/4/8", tabWidth))
			return
		}
	})

	return r.RunContinuous(ctx)
}
