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
// it, as [NewBox] takes [BoxWithBorder].
package widget
