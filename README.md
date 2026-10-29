<p align="center">
  <a href="https://github.com/unaquatic/fluid">
    <picture>
      <source media="(prefers-color-scheme: dark)" srcset="docs/logo-dark.svg">
      <img src="docs/logo-light.svg" alt="fluid">
    </picture>
  </a>
</p>

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.27-555?labelColor=888&logo=go&logoColor=white&style=flat-square" alt="Go"></a>
  <a href="https://github.com/unaquatic/fluid/blob/main/LICENSE"><img src="https://img.shields.io/github/license/unaquatic/fluid?labelColor=888&color=555&style=flat-square" alt="License"></a>
</p>

<p align="center"><b>A terminal user interface framework written in golang.</b></p>

---

## Install

```sh
go get github.com/unaquatic/fluid
```

## Usage

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
		InlineMode: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	r.Root().Add(widget.NewBox(
		widget.BoxWithLayout(layout.WithFlexDirection(layout.DirectionColumn)),
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

## Examples

```sh
go run github.com/unaquatic/fluid/examples/chat   # an inline chat that streams a reply; enter sends, esc stops or exits
```

[`examples/chat`](examples/chat) changes its widgets only on the renderer's event loop, in three layers:

- `state.go` holds plain data, such as the transcript and whether a reply is streaming.
- `service.go` turns keys and the words of a reply into events that say what the screen must show. It touches no widget, so `service_test.go` needs no terminal.
- `ui.go` owns the widgets and applies those events. Enter reaches it through the input's `OnSubmit`, Esc through the event bus before the focused input, and the reply's goroutine through `Renderer.Enqueue`.

## Colors

The renderer picks 24-bit, 256, or 16 colors from the environment, and maps every color to the nearest one the terminal can show:

- 24-bit when `COLORTERM` is `truecolor` or `24bit`, or `TERM` ends in `-direct`.
- 256 inside tmux or screen, unless `COLORTERM` advertises 24-bit.
- 24-bit for known terminals, such as Ghostty, iTerm2, WezTerm, Kitty, and VS Code.
- 256 when `TERM` contains `256color`, and 16 when `TERM` is empty or `dumb`.
- 24-bit for any other terminal.

Set `FLUID_COLOR` to `truecolor`, `24bit`, `256`, or `16` to choose one yourself.

## Commands

```sh
go run github.com/unaquatic/fluid/cmd/fluid demo markdown                  # run a demo; fluid -h lists all 13
go run github.com/unaquatic/fluid/cmd/fluid design -root .                 # paint golden frames files in the browser
go run github.com/unaquatic/fluid/cmd/fluid design html -o goldens.html .  # draw every golden frames file in color
```

- `fluid demo <name>` runs an interactive demo of the widgets, the layout engine, or the inline and fullscreen modes, with `-no-mouse`.
- `fluid design` serves an editor on port 4098 that paints the golden frames and sequence files under `-root` cell by cell and saves them as `testutil` writes them; `fluid design ansi file...` prints them with ANSI escape sequences.
- `fluid design html [dir...]` writes one HTML page that draws the golden frames and sequence files under the given directories in their colors, with each snapshot's legend and raw file.

## Testing

```sh
go test ./...                  # all tests
go test ./widget/ -v           # widget tests
go test ./render/ -run Inline  # inline render tests
go test ./layout/              # layout engine tests
```

## License

Fluid is released under the [MIT License](LICENSE).
