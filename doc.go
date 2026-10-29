// Package fluid is a retained-mode terminal user interface framework.
//
// A fluid program builds a tree of widgets once and changes it as events
// arrive. For each frame, fluid lays the tree out with a flexbox engine, draws
// it into a grid of styled cells, and writes to the terminal only the cells
// that changed. It draws either on the alternate screen or inline, in the
// terminal's normal screen and scrollback, and maps every color to the 24-bit,
// 256, or 16 colors the terminal supports.
//
// This package contains no code. It introduces the packages that make up the
// API, each of which documents its own part in full.
//
// # Getting started
//
// A terminal.Renderer owns the terminal, the event loop, and the root of the
// tree. Add widgets to the root and call Run, which returns when its context
// is canceled, the program is interrupted, or the renderer is destroyed:
//
//	package main
//
//	import (
//		"context"
//		"log"
//
//		"github.com/unaquatic/fluid/style"
//		"github.com/unaquatic/fluid/terminal"
//		"github.com/unaquatic/fluid/widget"
//	)
//
//	func main() {
//		r, err := terminal.NewRenderer(terminal.Config{InlineMode: true})
//		if err != nil {
//			log.Fatal(err)
//		}
//		r.Root().Add(widget.NewBox(
//			widget.BoxWithBorder(style.BorderAll()),
//			widget.BoxWithBorderStyle(style.BorderRounded),
//			widget.BoxWithTitle("fluid"),
//			widget.Children(widget.NewText("hello, fluid")),
//		))
//		if err := r.Run(context.Background()); err != nil {
//			log.Fatal(err)
//		}
//	}
//
// The fluid command runs demos of the widgets, the layout engine, and both
// rendering modes; with no demo name it lists them:
//
//	go run github.com/unaquatic/fluid/cmd/fluid demo
//
// # Structuring a program
//
// Widgets are not safe for concurrent use, so a program changes them only on
// the renderer's event loop. A program of more than a few widgets can keep to
// that, and keep its logic testable, in three layers:
//
//   - The state is plain data, such as a transcript and whether a reply is
//     streaming. It holds no widgets.
//   - A service handles what happens, such as a submitted prompt or the next
//     word of a reply. It changes the state and returns events that say what
//     the screen must show, but never touches a widget, so its tests need no
//     terminal.
//   - A ui owns the widgets and applies the service's events to them.
//
// Everything reaches the service on the event loop. Keys arrive through a
// handler on the renderer's event bus, which calls PreventDefault on the keys
// it uses so that the focused widget never sees them, and widget callbacks
// already run there. A goroutine, such as one that streams a reply, hands its
// results to the event loop with Renderer.Enqueue, which runs the functions a
// goroutine enqueues in the order it enqueued them. The chat example is built
// this way:
//
//	go run github.com/unaquatic/fluid/examples/chat
//
// # Packages
//
// Most programs import four packages:
//
//   - [github.com/unaquatic/fluid/terminal] runs the renderer: the rendering
//     modes, the event loop, and how input reaches widgets.
//   - [github.com/unaquatic/fluid/widget] provides the widgets.
//   - [github.com/unaquatic/fluid/layout] sizes and places them.
//   - [github.com/unaquatic/fluid/style] describes colors, text attributes,
//     styled text, borders, and themes.
//
// The others serve programs that handle input themselves, write their own
// widgets, or test them:
//
//   - [github.com/unaquatic/fluid/render] defines the tree of widgets and shows
//     how to write one.
//   - [github.com/unaquatic/fluid/input] defines key, mouse, and paste events,
//     the event bus, and key bindings.
//   - [github.com/unaquatic/fluid/buffer] is the grid of cells that widgets
//     draw into.
//   - [github.com/unaquatic/fluid/textbuf] stores, edits, wraps, and measures
//     text.
//   - [github.com/unaquatic/fluid/syntax] splits source code into tokens for
//     highlighting.
//   - [github.com/unaquatic/fluid/symbol] names the runes of borders, bars,
//     blocks, and scroll bars.
//   - [github.com/unaquatic/fluid/testutil] lays out and draws widgets without
//     a terminal and compares them with golden files.
package fluid
