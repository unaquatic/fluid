package style

import (
	"strings"
	"unicode/utf8"
)

// TextChunk is a run of text with uniform styling.
type TextChunk struct {
	Text       string
	Fg         *Color // nil = inherit
	Bg         *Color // nil = inherit
	Attributes uint32
	Link       string // URL, empty = no link
}

// StyledText is a sequence of styled text chunks.
type StyledText struct {
	Chunks []TextChunk
}

// --- StyledText Methods ---

// PlainText concatenates all chunk text into a plain string.
func (st StyledText) PlainText() string {
	var b strings.Builder
	for _, c := range st.Chunks {
		b.WriteString(c.Text)
	}
	return b.String()
}

// Len returns the total rune count across all chunks.
func (st StyledText) Len() int {
	total := 0
	for _, c := range st.Chunks {
		total += utf8.RuneCountInString(c.Text)
	}
	return total
}

// Append concatenates two StyledTexts, returning a new StyledText.
func (st StyledText) Append(other StyledText) StyledText {
	chunks := make([]TextChunk, 0, len(st.Chunks)+len(other.Chunks))
	chunks = append(chunks, st.Chunks...)
	chunks = append(chunks, other.Chunks...)
	return StyledText{Chunks: chunks}
}

// String returns the plain text representation (same as PlainText).
func (st StyledText) String() string {
	return st.PlainText()
}

// --- NewStyledText Constructor ---

// NewStyledText creates a StyledText from variadic chunks.
func NewStyledText(chunks ...TextChunk) StyledText {
	return StyledText{Chunks: chunks}
}

// --- Builder / Helper Functions ---

// Plain creates a plain unstyled chunk.
func Plain(text string) TextChunk {
	return TextChunk{Text: text}
}

// TextBold creates a bold chunk.
func TextBold(text string) TextChunk {
	return TextChunk{Text: text, Attributes: AttrBold}
}

// TextItalic creates an italic chunk.
func TextItalic(text string) TextChunk {
	return TextChunk{Text: text, Attributes: AttrItalic}
}

// TextUnderline creates an underlined chunk.
func TextUnderline(text string) TextChunk {
	return TextChunk{Text: text, Attributes: AttrUnderline}
}

// TextDim creates a dim chunk.
func TextDim(text string) TextChunk {
	return TextChunk{Text: text, Attributes: AttrDim}
}

// TextStrikethrough creates a strikethrough chunk.
func TextStrikethrough(text string) TextChunk {
	return TextChunk{Text: text, Attributes: AttrStrikethrough}
}

// TextInverse creates an inverse chunk.
func TextInverse(text string) TextChunk {
	return TextChunk{Text: text, Attributes: AttrInverse}
}

// Fg returns a curried function that creates a TextChunk with a custom foreground.
func Fg(ci ColorInput) func(string) TextChunk {
	c := ParseColor(ci)
	return func(text string) TextChunk {
		return TextChunk{Text: text, Fg: &c}
	}
}

// Bg returns a curried function that creates a TextChunk with a custom background.
func Bg(ci ColorInput) func(string) TextChunk {
	c := ParseColor(ci)
	return func(text string) TextChunk {
		return TextChunk{Text: text, Bg: &c}
	}
}

// Link returns a curried function that creates a TextChunk with a hyperlink.
func Link(url string) func(string) TextChunk {
	return func(text string) TextChunk {
		return TextChunk{Text: text, Link: url, Attributes: AttrUnderline}
	}
}

// Styled creates a TextChunk with a full Style applied.
func Styled(text string, s Style) TextChunk {
	fg := s.Fg
	bg := s.Bg
	return TextChunk{
		Text:       text,
		Fg:         &fg,
		Bg:         &bg,
		Attributes: s.Attributes,
	}
}

// --- Convenience Color Functions (Foreground) ---

// FgBlack creates a chunk with black foreground.
func FgBlack(text string) TextChunk {
	c := ColorBlack
	return TextChunk{Text: text, Fg: &c}
}

// FgRed creates a chunk with red foreground.
func FgRed(text string) TextChunk {
	c := ColorRed
	return TextChunk{Text: text, Fg: &c}
}

// FgGreen creates a chunk with green foreground.
func FgGreen(text string) TextChunk {
	c := ColorGreen
	return TextChunk{Text: text, Fg: &c}
}

// FgYellow creates a chunk with yellow foreground.
func FgYellow(text string) TextChunk {
	c := ColorYellow
	return TextChunk{Text: text, Fg: &c}
}

// FgBlue creates a chunk with blue foreground.
func FgBlue(text string) TextChunk {
	c := ColorBlue
	return TextChunk{Text: text, Fg: &c}
}

// FgMagenta creates a chunk with magenta foreground.
func FgMagenta(text string) TextChunk {
	c := ColorMagenta
	return TextChunk{Text: text, Fg: &c}
}

// FgCyan creates a chunk with cyan foreground.
func FgCyan(text string) TextChunk {
	c := ColorCyan
	return TextChunk{Text: text, Fg: &c}
}

// FgWhite creates a chunk with white foreground.
func FgWhite(text string) TextChunk {
	c := ColorWhite
	return TextChunk{Text: text, Fg: &c}
}

// --- Bright Foreground ---

// FgBrightBlack creates a chunk with bright black (gray) foreground.
func FgBrightBlack(text string) TextChunk {
	c := ColorBrightBlack
	return TextChunk{Text: text, Fg: &c}
}

// FgBrightRed creates a chunk with bright red foreground.
func FgBrightRed(text string) TextChunk {
	c := ColorBrightRed
	return TextChunk{Text: text, Fg: &c}
}

// FgBrightGreen creates a chunk with bright green foreground.
func FgBrightGreen(text string) TextChunk {
	c := ColorBrightGreen
	return TextChunk{Text: text, Fg: &c}
}

// FgBrightYellow creates a chunk with bright yellow foreground.
func FgBrightYellow(text string) TextChunk {
	c := ColorBrightYellow
	return TextChunk{Text: text, Fg: &c}
}

// FgBrightBlue creates a chunk with bright blue foreground.
func FgBrightBlue(text string) TextChunk {
	c := ColorBrightBlue
	return TextChunk{Text: text, Fg: &c}
}

// FgBrightMagenta creates a chunk with bright magenta foreground.
func FgBrightMagenta(text string) TextChunk {
	c := ColorBrightMagenta
	return TextChunk{Text: text, Fg: &c}
}

// FgBrightCyan creates a chunk with bright cyan foreground.
func FgBrightCyan(text string) TextChunk {
	c := ColorBrightCyan
	return TextChunk{Text: text, Fg: &c}
}

// FgBrightWhite creates a chunk with bright white foreground.
func FgBrightWhite(text string) TextChunk {
	c := ColorBrightWhite
	return TextChunk{Text: text, Fg: &c}
}

// --- Convenience Color Functions (Background) ---

// BgBlack creates a chunk with black background.
func BgBlack(text string) TextChunk {
	c := ColorBlack
	return TextChunk{Text: text, Bg: &c}
}

// BgRed creates a chunk with red background.
func BgRed(text string) TextChunk {
	c := ColorRed
	return TextChunk{Text: text, Bg: &c}
}

// BgGreen creates a chunk with green background.
func BgGreen(text string) TextChunk {
	c := ColorGreen
	return TextChunk{Text: text, Bg: &c}
}

// BgYellow creates a chunk with yellow background.
func BgYellow(text string) TextChunk {
	c := ColorYellow
	return TextChunk{Text: text, Bg: &c}
}

// BgBlue creates a chunk with blue background.
func BgBlue(text string) TextChunk {
	c := ColorBlue
	return TextChunk{Text: text, Bg: &c}
}

// BgMagenta creates a chunk with magenta background.
func BgMagenta(text string) TextChunk {
	c := ColorMagenta
	return TextChunk{Text: text, Bg: &c}
}

// BgCyan creates a chunk with cyan background.
func BgCyan(text string) TextChunk {
	c := ColorCyan
	return TextChunk{Text: text, Bg: &c}
}

// BgWhite creates a chunk with white background.
func BgWhite(text string) TextChunk {
	c := ColorWhite
	return TextChunk{Text: text, Bg: &c}
}
