package widget

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/textbuf"
)

// SyntaxHighlighter converts source code into styled logical lines.
// Implementations may cache parser and style state. Configure themes through
// the concrete implementation's constructor or methods.
type SyntaxHighlighter interface {
	// Highlight returns one styled Line per logical line in content.
	// filetype identifies the source language, such as "go" or "python".
	// defaultFg and defaultBg apply when a token has no explicit color.
	Highlight(content, filetype string, defaultFg, defaultBg style.Color) ([]textbuf.Line, error)
}

// ChromaHighlighter implements SyntaxHighlighter with Chroma lexers and styles.
type ChromaHighlighter struct {
	theme string

	// The lexer is cached for cachedFiletype; chromaStyle is cached for theme.
	cachedFiletype string
	lexer          chroma.Lexer
	chromaStyle    *chroma.Style
}

// NewChromaHighlighter returns a highlighter for the named Chroma theme.
// Common theme names include "monokai", "dracula", and "github".
func NewChromaHighlighter(theme string) *ChromaHighlighter {
	h := &ChromaHighlighter{theme: theme}
	h.resolveStyle()
	return h
}

// SetTheme changes the color theme and invalidates the cached style.
func (h *ChromaHighlighter) SetTheme(theme string) {
	if h.theme == theme {
		return
	}
	h.theme = theme
	h.resolveStyle()
}

// Theme returns the current theme name.
func (h *ChromaHighlighter) Theme() string { return h.theme }

// Highlight tokenizes content and returns one styled Line per logical line.
func (h *ChromaHighlighter) Highlight(content, filetype string, defaultFg, defaultBg style.Color) ([]textbuf.Line, error) {
	h.ensureLexer(filetype)

	lines := strings.Split(content, "\n")

	iterator, err := h.lexer.Tokenise(nil, content)
	if err != nil {
		return h.buildFallbackLines(lines, defaultFg, defaultBg), err
	}

	return h.buildLines(iterator, lines, defaultFg, defaultBg), nil
}

// resolveStyle selects the configured Chroma style or falls back to Monokai.
func (h *ChromaHighlighter) resolveStyle() {
	h.chromaStyle = styles.Get(h.theme)
	if h.chromaStyle == nil {
		h.chromaStyle = styles.Get("monokai")
	}
}

// ensureLexer resolves and caches a coalescing lexer for filetype.
func (h *ChromaHighlighter) ensureLexer(filetype string) {
	if h.cachedFiletype == filetype && h.lexer != nil {
		return
	}
	h.cachedFiletype = filetype
	if filetype != "" {
		h.lexer = lexers.Get(filetype)
	}
	if h.lexer == nil {
		h.lexer = lexers.Fallback
	}
	h.lexer = chroma.Coalesce(h.lexer)
}

// buildFallbackLines applies the default colors without syntax styling.
func (h *ChromaHighlighter) buildFallbackLines(lines []string, defaultFg, defaultBg style.Color) []textbuf.Line {
	result := make([]textbuf.Line, len(lines))
	for i, line := range lines {
		result[i] = textbuf.Line{
			Spans: []textbuf.Span{{
				Text: line,
				Fg:   defaultFg,
				Bg:   defaultBg,
			}},
		}
	}
	return result
}

// buildLines splits Chroma tokens into logical lines of styled spans.
func (h *ChromaHighlighter) buildLines(iterator chroma.Iterator, lines []string, defaultFg, defaultBg style.Color) []textbuf.Line {
	// Keep one span slice per source line while splitting multiline tokens.
	spansByLine := make([][]textbuf.Span, len(lines))

	lineIdx := 0
	tokens := iterator.Tokens()

	for _, token := range tokens {
		tokenFg, tokenBg, tokenAttrs := h.resolveTokenStyle(token.Type, defaultFg, defaultBg)

		// Chroma can return one token for a multiline string or comment.
		text := token.Value
		for {
			nlPos := strings.IndexByte(text, '\n')
			if nlPos == -1 {
				// The remaining token text belongs to the current line.
				if len(text) > 0 && lineIdx < len(spansByLine) {
					spansByLine[lineIdx] = append(spansByLine[lineIdx], textbuf.Span{
						Text:       text,
						Fg:         tokenFg,
						Bg:         tokenBg,
						Attributes: tokenAttrs,
					})
				}
				break
			}

			// Append the portion before the newline to the current line.
			before := text[:nlPos]
			if len(before) > 0 && lineIdx < len(spansByLine) {
				spansByLine[lineIdx] = append(spansByLine[lineIdx], textbuf.Span{
					Text:       before,
					Fg:         tokenFg,
					Bg:         tokenBg,
					Attributes: tokenAttrs,
				})
			}

			// Continue the token on the next source line.
			lineIdx++
			text = text[nlPos+1:]
		}
	}

	// Represent empty source lines with non-nil span slices.
	result := make([]textbuf.Line, len(spansByLine))
	for i, spans := range spansByLine {
		if spans == nil {
			spans = []textbuf.Span{}
		}
		result[i] = textbuf.Line{Spans: spans}
	}
	return result
}

// resolveTokenStyle returns the terminal style for a Chroma token type.
func (h *ChromaHighlighter) resolveTokenStyle(tokenType chroma.TokenType, defaultFg, defaultBg style.Color) (fg, bg style.Color, attrs uint32) {
	// Chroma uses Error for text a lexer cannot classify, including valid syntax
	// that a lexer does not understand. Render such text with the defaults rather
	// than presenting a lexer limitation as a source-code error.
	if tokenType == chroma.Error {
		return defaultFg, defaultBg, 0
	}

	entry := h.chromaStyle.Get(tokenType)

	if entry.Colour.IsSet() {
		fg = style.ColorFromInts(entry.Colour.Red(), entry.Colour.Green(), entry.Colour.Blue(), 255)
	} else {
		fg = defaultFg
	}

	bg = defaultBg

	if entry.Bold == chroma.Yes {
		attrs |= style.AttrBold
	}
	if entry.Italic == chroma.Yes {
		attrs |= style.AttrItalic
	}
	if entry.Underline == chroma.Yes {
		attrs |= style.AttrUnderline
	}

	return fg, bg, attrs
}
