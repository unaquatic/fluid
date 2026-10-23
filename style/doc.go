// Package style describes how cells look.
//
// It provides [Color] and the escape sequences that select it in 24-bit,
// 256-color, and 16-color terminals, text attributes, [Style], styled text,
// and border styles. A [Theme] names the roles widgets draw with, such as
// surfaces, Markdown, syntax tokens, and selections; [ThemeFromSeeds] derives
// one from a few OKLCH seed colors, and [DarculaSolid] defines one directly.
package style
