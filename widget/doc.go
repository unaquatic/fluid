// Package widget provides fluid's widgets, which embed
// [render.BaseRenderable].
//
// [Box] and [ScrollBox] contain other widgets. [Text], [TextNode],
// [Textarea], and [Input] show and edit text, [Markdown] and [StreamMarkdown]
// render Markdown as written or as it streams in, and [Code], [Diff], and
// [LineNumber] show highlighted code and diffs. [Select], [TabSelect], [List],
// [Table], [Tree], [Gauge], [Sparkline], [ScrollBar], and [CharFill] cover
// choices, data, and decoration.
//
// Each widget is built with a constructor and functional options named after
// it, as [NewBox] takes [BoxWithBorder], and has setters, such as
// [Text.SetContent], that change it later. Every widget has an option that
// applies [layout.Option] values to its layout node, such as [BoxWithLayout].
// Widgets are not safe for concurrent use; package terminal describes the
// goroutine that may change them.
//
// # Input
//
// [Textarea], [Input], [Select], [TabSelect], [List], [Table], and [Tree] are
// focusable, and take keys once they have focus. Widgets that take input
// report it through callbacks, such as [TextareaWithOnSubmit] and
// [InputWithOnChange]. Textarea and Input map keys to actions through
// [input.KeyBinding] tables, and [TextareaWithKeyBindings] overrides entries
// of [DefaultTextareaBindings].
//
// # Themes
//
// [Markdown], [Code], and [Diff] take a whole [style.Theme], through options
// such as [MarkdownWithTheme]. Other widgets take the colors of its roles, as
// in BoxWithBg(theme.Surface).
package widget
