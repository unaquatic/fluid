// Package textbuf stores, edits, wraps, and measures text for terminal widgets.
//
// [TextBuffer] holds lines with their highlights, and [EditBuffer] adds a
// cursor and editing. [TextSurface] and [EditSurface] wrap a buffer into the
// virtual lines of a [Viewport], extmarks track ranges of text through edits,
// and the width functions, such as [StringWidth], measure grapheme clusters in
// terminal cells.
package textbuf
