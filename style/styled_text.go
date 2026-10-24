package style

import (
	"strings"
	"unicode/utf8"
)

// TextChunk is a run of text with uniform styling.
type TextChunk struct {
	// Text is the text of the run.
	Text string
	// Fg is the foreground color; a transparent one inherits the
	// surrounding foreground.
	Fg Color
	// Bg is the background color; a transparent one inherits the
	// surrounding background.
	Bg Color
	// Attributes is a bitmask of the Attr flags, such as AttrBold.
	Attributes Attrs
	// Link is the link URL, or empty for no link.
	Link string
}

// StyledText is a sequence of styled text chunks.
type StyledText struct {
	// Chunks are the runs of text, in order.
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
		return TextChunk{Text: text, Fg: c}
	}
}

// Bg returns a function that applies ci as a background color.
func Bg(ci ColorInput) func(string) TextChunk {
	c := ParseColor(ci)
	return func(text string) TextChunk {
		return TextChunk{Text: text, Bg: c}
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
	return TextChunk{
		Text:       text,
		Fg:         s.Fg,
		Bg:         s.Bg,
		Attributes: s.Attributes,
	}
}

// FgBlack returns a text chunk with a black foreground.
func FgBlack(text string) TextChunk {
	return TextChunk{Text: text, Fg: ColorBlack}
}

// FgRed returns a text chunk with a red foreground.
func FgRed(text string) TextChunk {
	return TextChunk{Text: text, Fg: ColorRed}
}

// FgGreen returns a text chunk with a green foreground.
func FgGreen(text string) TextChunk {
	return TextChunk{Text: text, Fg: ColorGreen}
}

// FgYellow returns a text chunk with a yellow foreground.
func FgYellow(text string) TextChunk {
	return TextChunk{Text: text, Fg: ColorYellow}
}

// FgBlue returns a text chunk with a blue foreground.
func FgBlue(text string) TextChunk {
	return TextChunk{Text: text, Fg: ColorBlue}
}

// FgMagenta returns a text chunk with a magenta foreground.
func FgMagenta(text string) TextChunk {
	return TextChunk{Text: text, Fg: ColorMagenta}
}

// FgCyan returns a text chunk with a cyan foreground.
func FgCyan(text string) TextChunk {
	return TextChunk{Text: text, Fg: ColorCyan}
}

// FgWhite returns a text chunk with a white foreground.
func FgWhite(text string) TextChunk {
	return TextChunk{Text: text, Fg: ColorWhite}
}

// FgBrightBlack returns a text chunk with a bright-black foreground.
func FgBrightBlack(text string) TextChunk {
	return TextChunk{Text: text, Fg: ColorBrightBlack}
}

// FgBrightRed returns a text chunk with a bright-red foreground.
func FgBrightRed(text string) TextChunk {
	return TextChunk{Text: text, Fg: ColorBrightRed}
}

// FgBrightGreen returns a text chunk with a bright-green foreground.
func FgBrightGreen(text string) TextChunk {
	return TextChunk{Text: text, Fg: ColorBrightGreen}
}

// FgBrightYellow returns a text chunk with a bright-yellow foreground.
func FgBrightYellow(text string) TextChunk {
	return TextChunk{Text: text, Fg: ColorBrightYellow}
}

// FgBrightBlue returns a text chunk with a bright-blue foreground.
func FgBrightBlue(text string) TextChunk {
	return TextChunk{Text: text, Fg: ColorBrightBlue}
}

// FgBrightMagenta returns a text chunk with a bright-magenta foreground.
func FgBrightMagenta(text string) TextChunk {
	return TextChunk{Text: text, Fg: ColorBrightMagenta}
}

// FgBrightCyan returns a text chunk with a bright-cyan foreground.
func FgBrightCyan(text string) TextChunk {
	return TextChunk{Text: text, Fg: ColorBrightCyan}
}

// FgBrightWhite returns a text chunk with a bright-white foreground.
func FgBrightWhite(text string) TextChunk {
	return TextChunk{Text: text, Fg: ColorBrightWhite}
}

// BgBlack returns a text chunk with a black background.
func BgBlack(text string) TextChunk {
	return TextChunk{Text: text, Bg: ColorBlack}
}

// BgRed returns a text chunk with a red background.
func BgRed(text string) TextChunk {
	return TextChunk{Text: text, Bg: ColorRed}
}

// BgGreen returns a text chunk with a green background.
func BgGreen(text string) TextChunk {
	return TextChunk{Text: text, Bg: ColorGreen}
}

// BgYellow returns a text chunk with a yellow background.
func BgYellow(text string) TextChunk {
	return TextChunk{Text: text, Bg: ColorYellow}
}

// BgBlue returns a text chunk with a blue background.
func BgBlue(text string) TextChunk {
	return TextChunk{Text: text, Bg: ColorBlue}
}

// BgMagenta returns a text chunk with a magenta background.
func BgMagenta(text string) TextChunk {
	return TextChunk{Text: text, Bg: ColorMagenta}
}

// BgCyan returns a text chunk with a cyan background.
func BgCyan(text string) TextChunk {
	return TextChunk{Text: text, Bg: ColorCyan}
}

// BgWhite returns a text chunk with a white background.
func BgWhite(text string) TextChunk {
	return TextChunk{Text: text, Bg: ColorWhite}
}
