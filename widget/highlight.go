package widget

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/textbuf"
)

// SyntaxHighlighter tokenizes source code and returns lines of spans.
// Implementations may cache lexers/styles internally for performance.
// Theme configuration is implementation-specific and should be set via
// the concrete type's constructor or setter methods.
type SyntaxHighlighter interface {
	// Highlight tokenizes content and returns one Line per logical line.
	// filetype is the language identifier (e.g. "go", "python").
	// defaultFg/defaultBg are used as fallback when the highlighter does not
	// specify a color for a given token type.
	Highlight(content, filetype string, defaultFg, defaultBg style.Color) ([]textbuf.Line, error)
}

// ChromaHighlighter implements SyntaxHighlighter using the Chroma
// library for tokenization and styling.
type ChromaHighlighter struct {
	theme string

	// Cached state to avoid re-resolving on every call.
	cachedFiletype string
	lexer          chroma.Lexer
	chromaStyle    *chroma.Style
}

// NewChromaHighlighter creates a new ChromaHighlighter with the given theme
// (e.g. "monokai", "dracula", "github").
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

// Highlight tokenizes content and returns one Line per logical line.
func (h *ChromaHighlighter) Highlight(content, filetype string, defaultFg, defaultBg style.Color) ([]textbuf.Line, error) {
	h.ensureLexer(filetype)

	lines := strings.Split(content, "\n")

	iterator, err := h.lexer.Tokenise(nil, content)
	if err != nil {
		return h.buildFallbackLines(lines, defaultFg, defaultBg), err
	}

	return h.buildLines(iterator, lines, defaultFg, defaultBg), nil
}

// resolveStyle resolves the Chroma style from the theme name.
func (h *ChromaHighlighter) resolveStyle() {
	h.chromaStyle = styles.Get(h.theme)
	if h.chromaStyle == nil {
		h.chromaStyle = styles.Get("monokai")
	}
}

// ensureLexer lazily resolves the Chroma lexer, caching it until filetype changes.
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

// buildFallbackLines creates unstyled lines (fallback on tokenization error).
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

// buildLines converts Chroma tokens into Lines.
func (h *ChromaHighlighter) buildLines(iterator chroma.Iterator, lines []string, defaultFg, defaultBg style.Color) []textbuf.Line {
	// Pre-allocate one slice of spans per logical line.
	spansByLine := make([][]textbuf.Span, len(lines))

	lineIdx := 0
	tokens := iterator.Tokens()

	for _, token := range tokens {
		tokenFg, tokenBg, tokenAttrs := h.resolveTokenStyle(token.Type, defaultFg, defaultBg)

		// A single token may span multiple lines (e.g. multiline strings/comments).
		text := token.Value
		for {
			nlPos := strings.IndexByte(text, '\n')
			if nlPos == -1 {
				// No newline -- entire remaining text belongs to current line.
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

			// Text before the newline.
			before := text[:nlPos]
			if len(before) > 0 && lineIdx < len(spansByLine) {
				spansByLine[lineIdx] = append(spansByLine[lineIdx], textbuf.Span{
					Text:       before,
					Fg:         tokenFg,
					Bg:         tokenBg,
					Attributes: tokenAttrs,
				})
			}

			// Move to next line.
			lineIdx++
			text = text[nlPos+1:]
		}
	}

	// Ensure every line has at least an empty span list.
	result := make([]textbuf.Line, len(spansByLine))
	for i, spans := range spansByLine {
		if spans == nil {
			spans = []textbuf.Span{}
		}
		result[i] = textbuf.Line{Spans: spans}
	}
	return result
}

// resolveTokenStyle maps a Chroma token type to fg/bg colors and attributes.
func (h *ChromaHighlighter) resolveTokenStyle(tokenType chroma.TokenType, defaultFg, defaultBg style.Color) (fg, bg style.Color, attrs uint32) {
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
