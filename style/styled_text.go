package style

import (
	"strings"
	"unicode/utf8"
)

// TextChunk is a run of text with uniform styling.
type TextChunk struct {
	Text       string
	Fg         *Color // Nil inherits the surrounding foreground.
	Bg         *Color // Nil inherits the surrounding background.
	Attributes uint32
	Link       string // The link URL, or empty for no link.
}

// StyledText is a sequence of styled text chunks.
type StyledText struct {
	Chunks []TextChunk
}

// PlainText returns all chunk text without styling metadata.
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

// Append returns the concatenation of st and other with a newly allocated chunk
// slice.
func (st StyledText) Append(other StyledText) StyledText {
	chunks := make([]TextChunk, 0, len(st.Chunks)+len(other.Chunks))
	chunks = append(chunks, st.Chunks...)
	chunks = append(chunks, other.Chunks...)
	return StyledText{Chunks: chunks}
}

// String returns the same unstyled content as PlainText.
func (st StyledText) String() string {
	return st.PlainText()
}

// NewStyledText returns styled text containing chunks.
func NewStyledText(chunks ...TextChunk) StyledText {
	return StyledText{Chunks: chunks}
}

// Plain returns an unstyled text chunk.
func Plain(text string) TextChunk {
	return TextChunk{Text: text}
}

// TextBold returns a bold text chunk.
func TextBold(text string) TextChunk {
	return TextChunk{Text: text, Attributes: AttrBold}
}

// TextItalic returns an italic text chunk.
func TextItalic(text string) TextChunk {
	return TextChunk{Text: text, Attributes: AttrItalic}
}

// TextUnderline returns an underlined text chunk.
func TextUnderline(text string) TextChunk {
	return TextChunk{Text: text, Attributes: AttrUnderline}
}

// TextDim returns a dim text chunk.
func TextDim(text string) TextChunk {
	return TextChunk{Text: text, Attributes: AttrDim}
}

// TextStrikethrough returns a strikethrough text chunk.
func TextStrikethrough(text string) TextChunk {
	return TextChunk{Text: text, Attributes: AttrStrikethrough}
}

// TextInverse returns an inverse-video text chunk.
func TextInverse(text string) TextChunk {
	return TextChunk{Text: text, Attributes: AttrInverse}
}

// Fg returns a function that applies ci as a foreground color.
func Fg(ci ColorInput) func(string) TextChunk {
	c := ParseColor(ci)
	return func(text string) TextChunk {
		return TextChunk{Text: text, Fg: &c}
	}
}

// Bg returns a function that applies ci as a background color.
func Bg(ci ColorInput) func(string) TextChunk {
	c := ParseColor(ci)
	return func(text string) TextChunk {
		return TextChunk{Text: text, Bg: &c}
	}
}

// Link returns a function that applies an underlined hyperlink to text.
func Link(url string) func(string) TextChunk {
	return func(text string) TextChunk {
		return TextChunk{Text: text, Link: url, Attributes: AttrUnderline}
	}
}

// Styled returns a text chunk with s applied.
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

// FgBlack returns a text chunk with a black foreground.
func FgBlack(text string) TextChunk {
	c := ColorBlack
	return TextChunk{Text: text, Fg: &c}
}

// FgRed returns a text chunk with a red foreground.
func FgRed(text string) TextChunk {
	c := ColorRed
	return TextChunk{Text: text, Fg: &c}
}

// FgGreen returns a text chunk with a green foreground.
func FgGreen(text string) TextChunk {
	c := ColorGreen
	return TextChunk{Text: text, Fg: &c}
}

// FgYellow returns a text chunk with a yellow foreground.
func FgYellow(text string) TextChunk {
	c := ColorYellow
	return TextChunk{Text: text, Fg: &c}
}

// FgBlue returns a text chunk with a blue foreground.
func FgBlue(text string) TextChunk {
	c := ColorBlue
	return TextChunk{Text: text, Fg: &c}
}

// FgMagenta returns a text chunk with a magenta foreground.
func FgMagenta(text string) TextChunk {
	c := ColorMagenta
	return TextChunk{Text: text, Fg: &c}
}

// FgCyan returns a text chunk with a cyan foreground.
func FgCyan(text string) TextChunk {
	c := ColorCyan
	return TextChunk{Text: text, Fg: &c}
}

// FgWhite returns a text chunk with a white foreground.
func FgWhite(text string) TextChunk {
	c := ColorWhite
	return TextChunk{Text: text, Fg: &c}
}

// FgBrightBlack returns a text chunk with a bright-black foreground.
func FgBrightBlack(text string) TextChunk {
	c := ColorBrightBlack
	return TextChunk{Text: text, Fg: &c}
}

// FgBrightRed returns a text chunk with a bright-red foreground.
func FgBrightRed(text string) TextChunk {
	c := ColorBrightRed
	return TextChunk{Text: text, Fg: &c}
}

// FgBrightGreen returns a text chunk with a bright-green foreground.
func FgBrightGreen(text string) TextChunk {
	c := ColorBrightGreen
	return TextChunk{Text: text, Fg: &c}
}

// FgBrightYellow returns a text chunk with a bright-yellow foreground.
func FgBrightYellow(text string) TextChunk {
	c := ColorBrightYellow
	return TextChunk{Text: text, Fg: &c}
}

// FgBrightBlue returns a text chunk with a bright-blue foreground.
func FgBrightBlue(text string) TextChunk {
	c := ColorBrightBlue
	return TextChunk{Text: text, Fg: &c}
}

// FgBrightMagenta returns a text chunk with a bright-magenta foreground.
func FgBrightMagenta(text string) TextChunk {
	c := ColorBrightMagenta
	return TextChunk{Text: text, Fg: &c}
}

// FgBrightCyan returns a text chunk with a bright-cyan foreground.
func FgBrightCyan(text string) TextChunk {
	c := ColorBrightCyan
	return TextChunk{Text: text, Fg: &c}
}

// FgBrightWhite returns a text chunk with a bright-white foreground.
func FgBrightWhite(text string) TextChunk {
	c := ColorBrightWhite
	return TextChunk{Text: text, Fg: &c}
}

// BgBlack returns a text chunk with a black background.
func BgBlack(text string) TextChunk {
	c := ColorBlack
	return TextChunk{Text: text, Bg: &c}
}

// BgRed returns a text chunk with a red background.
func BgRed(text string) TextChunk {
	c := ColorRed
	return TextChunk{Text: text, Bg: &c}
}

// BgGreen returns a text chunk with a green background.
func BgGreen(text string) TextChunk {
	c := ColorGreen
	return TextChunk{Text: text, Bg: &c}
}

// BgYellow returns a text chunk with a yellow background.
func BgYellow(text string) TextChunk {
	c := ColorYellow
	return TextChunk{Text: text, Bg: &c}
}

// BgBlue returns a text chunk with a blue background.
func BgBlue(text string) TextChunk {
	c := ColorBlue
	return TextChunk{Text: text, Bg: &c}
}

// BgMagenta returns a text chunk with a magenta background.
func BgMagenta(text string) TextChunk {
	c := ColorMagenta
	return TextChunk{Text: text, Bg: &c}
}

// BgCyan returns a text chunk with a cyan background.
func BgCyan(text string) TextChunk {
	c := ColorCyan
	return TextChunk{Text: text, Bg: &c}
}

// BgWhite returns a text chunk with a white background.
func BgWhite(text string) TextChunk {
	c := ColorWhite
	return TextChunk{Text: text, Bg: &c}
}
