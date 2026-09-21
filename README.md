# fluid

A retained-mode terminal UI framework for Go, with a flexbox layout engine
and double-buffered cell rendering.

[![Go](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white&style=flat-square)](https://go.dev/)
[![License](https://img.shields.io/github/license/unaquatic/fluid?style=flat-square)](https://github.com/unaquatic/fluid/blob/main/LICENSE)

## Install

```sh
go get github.com/unaquatic/fluid
```

## Quick start

```go
package main

import (
	"context"
	"log"

	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/terminal"
	"github.com/unaquatic/fluid/widget"
)

func main() {
	r, err := terminal.NewRenderer(terminal.Config{
		InlineMode:  true,
		ExitOnCtrlC: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	r.Root().Add(widget.NewBox(
		widget.BoxWithFlexDirection(layout.DirectionColumn),
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderSingle),
		widget.Children(
			widget.NewText("hello, fluid"),
		),
	))

	if err := r.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
```

## Rendering modes

- **Inline** renders into the terminal's normal scrollback, emitting only the
  rows the content needs. The terminal's own scrollbar keeps working.
- **Fullscreen** uses the alternate screen buffer with a fixed viewport.

## Structure

```
fluid/
  buffer/           cell buffer (2D grid of styled characters), border drawing
  input/            keyboard, mouse, paste events + keybindings + parser
  layout/           flexbox layout engine
  render/           scene graph, base renderable, selection, render context
  style/            color, style (fg/bg/attrs), StyledText, border types & chars
  symbol/           unicode character constants (borders, bars, blocks, scrollbars)
  terminal/         terminal renderer, diff, input loop, hit-testing, backend
  testutil/         test helpers (golden output, mock renderables)
  textbuf/          text buffer, edit buffer, text/edit surface, word wrap, unicode width
  widget/           box, text, textarea, scrollbox, markdown, diff, code, table, tree, etc.
```

## Testing

```sh
go test ./...                  # all tests
go test ./widget/ -v           # widget tests
go test ./render/ -run Inline  # inline render tests
go test ./layout/              # layout engine tests
```
