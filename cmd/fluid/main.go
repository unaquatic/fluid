// Command fluid runs interactive demos of fluid's widgets, layout engine and
// rendering modes in the terminal, and paints and draws golden files.
//
// "fluid demo" runs a demo by name.
//
// "fluid design" serves a page for drawing terminal screens cell by cell:
// typing text, painting foreground and background colors and attributes,
// drawing boxes, and moving regions around. The page opens the golden
// frames and sequence files under a directory, and saves designs there as
// golden frames files, the format that testutil.AssertGoldenFrames checks
// and agents read, written by testutil.GoldenFrames itself. It also exports
// a design as a PNG image or as text with ANSI escape sequences. The palette
// offers the roles of a theme, style.DarculaSolid unless -theme names a
// style.Theme as JSON, and the colors the golden files already use.
//
// "fluid design ansi" prints golden files with ANSI escape sequences.
// "fluid design html" writes one static HTML page that draws every golden
// file under the given directories in color, with each snapshot's legend and
// raw file; transparent cells show the -bg color.
//
// Usage:
//
//	fluid demo name [-no-mouse]
//	fluid design [-port n] [-root dir] [-theme file]
//	fluid design ansi file...
//	fluid design html [-o file] [-bg color] [dir...]
//
// For example, to run a demo, paint a golden file of this repository, print
// it with ANSI escape sequences, and draw every golden file on one page:
//
//	go run ./cmd/fluid demo dashboard
//	go run ./cmd/fluid design
//	open 'http://localhost:4098/?open=widget/testdata/box_border_with_title.frames.golden'
//	go run ./cmd/fluid design ansi widget/testdata/box_border_with_title.frames.golden
//	go run ./cmd/fluid design html -o /tmp/goldens.html .
package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	if err := Run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "fluid:", err)
		os.Exit(1)
	}
}
