// Package style describes how cells look.
//
// It provides [Color] and the escape sequences that select it in 24-bit,
// 256-color, and 16-color terminals, text attributes, [Style], styled text,
// and border styles. A [Theme] names the roles widgets draw with, such as
// surfaces, Markdown, syntax tokens, and selections; [ThemeFromSeeds] derives
// one from a few OKLCH seed colors, and [DarculaSolid] defines one directly.
//
// A Color is an RGBA color, and [ColorTransparent] leaves the background
// beneath a cell showing. The terminal renderer maps each color to the
// terminal's [ColorMode], which terminal.DetectColorMode picks from the
// environment.
//
// [StyledText] is a sequence of [TextChunk] runs, each with its own colors and
// attributes, which [NewStyledText] joins from helpers such as [TextBold],
// [Plain], and [Fg]. A chunk with a transparent foreground or background
// inherits the surrounding one.
package style
