// Package terminal runs fluid in a terminal.
//
// A [Renderer] owns the scene's root and its event loop, which applies tree
// changes, lays out and draws each frame, and flushes only the cells that
// changed: inline, into the terminal's normal screen and scrollback, or
// fullscreen, on the alternate screen. [Backend] writes the terminal's escape
// sequences, and [DetectColorMode] picks 24-bit, 256, or 16 colors from the
// environment, which FLUID_COLOR overrides.
//
// # Running
//
// [NewRenderer] takes a [Config], whose fields left zero take their defaults,
// so a Config sets only what it changes. [Renderer.Run] sets up the terminal
// and returns when its context is canceled, the process receives SIGINT or
// SIGTERM, or the renderer is destroyed. Control-C destroys the renderer
// unless a key handler prevents it or [Config.HandleCtrlC] passes the key to
// the application, and [Renderer.Destroy] restores the terminal's modes.
//
// # Inline and fullscreen
//
// By default the renderer draws on the alternate screen and the root fills the
// terminal. With [Config.InlineMode] set, it draws in the normal screen below
// the prompt instead: the root is as wide as the terminal, less one column,
// and as tall as its content, and rows that scroll off the top stay in the
// terminal's scrollback. [Renderer.SetInlineMode] switches between the two
// while the renderer runs.
//
// An inline root has no screen height for its children to fill. It is laid
// out tall enough for any transcript, so a child with a flex grow factor takes
// all of that height. Let inline content be as tall as what it holds, using a
// Box rather than a ScrollBox from package widget, and give anything that
// grows a fixed height.
//
// # Input and focus
//
// The renderer parses terminal input into [input.KeyEvent],
// [input.MouseEvent], and [input.PasteEvent] values. It offers each key event,
// in order, to
//
//   - the raw-input handlers added by [Renderer.AddInputHandler], any of which
//     can consume it;
//   - the handlers subscribed to [Renderer.EventBus], any of which can call
//     PreventDefault to withhold it from the rest;
//   - the focused renderable's HandleKeyPress, and then each of its
//     ancestors', until one returns true or calls StopPropagation.
//
// Mouse and paste events also reach the event bus before any renderable.
// With [Config.UseMouse] set, the renderer sends each mouse event to the
// renderable under the pointer and then to its ancestors, until one stops its
// propagation. Pasted text goes to the focused renderable.
//
// [Renderer.FocusRenderable] gives a renderable focus. A focusable renderable
// can also take it through its own Focus method or, with
// [Config.EnableAutoFocus] set, from a mouse press on it.
// [render.WithFocusable] makes a renderable focusable.
//
// # The event loop
//
// The event loop runs on one goroutine. It dispatches input, runs the
// functions passed to [Renderer.Enqueue], lays out the tree, and draws frames.
// Widgets are not safe for concurrent use, so once the renderer runs, change
// the tree only from that goroutine. Input handlers and widget callbacks
// already run there; any other goroutine passes a function to Enqueue.
//
// [Renderer.Run] draws a frame whenever the tree requests one. Adding or
// removing a renderable requests one, as do widget setters, and requests made
// between two frames coalesce, so a burst of changes is drawn once.
package terminal
