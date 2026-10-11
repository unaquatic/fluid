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

## Testing

```sh
go test ./...                  # all tests
go test ./widget/ -v           # widget tests
go test ./render/ -run Inline  # inline render tests
go test ./layout/              # layout engine tests
```

## License

Fluid is released under the [MIT License](LICENSE).
