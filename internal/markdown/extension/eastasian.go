package extension

import (
	"unicode"
	"unicode/utf8"

	"github.com/unaquatic/fluid/internal/markdown/ast"
)

// An EastAsianLineBreaks is how a renderer renders the soft line breaks
// between East Asian characters, which their scripts do not part with
// spaces, as browsers and terminals show line breaks.
type EastAsianLineBreaks int

const (
	// EastAsianLineBreaksNone renders them as other soft line breaks.
	EastAsianLineBreaksNone EastAsianLineBreaks = iota

	// EastAsianLineBreaksSimple leaves out a soft line break between two
	// wide characters: Han, Hiragana, Katakana, Hangul, modifier letters,
	// and CJK symbols and punctuation. Pandoc's extension
	// east_asian_line_breaks leaves them out so.
	EastAsianLineBreaksSimple

	// EastAsianLineBreaksCSS3Draft leaves out a soft line break as the
	// segment break transformation rules of CSS Text Module Level 3, in
	// the draft of 29 April 2020, do: next to a zero width space; between
	// two fullwidth, wide, or halfwidth characters, unless one is Hangul;
	// and next to a character of the space-discarding set, or punctuation,
	// or an ideographic space.
	EastAsianLineBreaksCSS3Draft
)

// keepsBreak reports whether style renders a soft line break between the
// characters last and next.
func (style EastAsianLineBreaks) keepsBreak(last, next rune) bool {
	switch style {
	case EastAsianLineBreaksSimple:
		return !isEastAsianWide(last) || !isEastAsianWide(next)
	case EastAsianLineBreaksCSS3Draft:
		if last == '\u200B' || next == '\u200B' {
			return false
		}
		if isFullOrWideOrHalf(last) && isFullOrWideOrHalf(next) {
			return unicode.Is(unicode.Hangul, last) || unicode.Is(unicode.Hangul, next)
		}
		for _, r := range [2]rune{last, next} {
			if unicode.Is(spaceDiscarding, r) || unicode.IsPunct(r) || r == '\u3000' {
				return false
			}
		}
	}
	return true
}

// KeepsSoftLineBreak reports whether the soft line break after text t
// renders in style: whether the last character of t and the first of the
// inline content after it let it, if both have characters. The first
// character of content such as emphasis, a link, or a code span is that of
// its text; raw HTML and images have none. goldmark keeps the line break
// only before text, and so leaves out those before other content, as in
// "a\n*b*", between letters too.
//
// The characters are those of the text, with escapes and entities
// resolved, and without the escaped spaces of [parser.WithEscapedSpace],
// where goldmark sees the source.
func (style EastAsianLineBreaks) KeepsSoftLineBreak(t *ast.Text) bool {
	if style == EastAsianLineBreaksNone || t.Value == "" {
		return true
	}
	next, ok := runeAfter(t)
	if !ok {
		return true
	}
	last, _ := utf8.DecodeLastRuneInString(t.Value)
	return style.keepsBreak(last, next)
}

// runeAfter returns the first character of the inline content after n, in
// the inline content that holds it, if it renders as text.
func runeAfter(n ast.Node) (rune, bool) {
	for ; n != nil && n.Type() == ast.TypeInline; n = n.Parent() {
		if next := n.NextSibling(); next != nil {
			return firstRune(next)
		}
	}
	return 0, false
}

// firstRune returns the first character of inline content n, if it renders
// as text.
func firstRune(n ast.Node) (rune, bool) {
	var s string
	switch n := n.(type) {
	case *ast.Text:
		s = n.Value
	case *ast.AutoLink:
		s = n.Label
	case *ast.RawHTML, *ast.Image:
	default:
		if c := n.FirstChild(); c != nil {
			return firstRune(c)
		}
	}
	if s == "" {
		return 0, false
	}
	r, _ := utf8.DecodeRuneInString(s)
	return r, true
}

// isEastAsianWide reports whether r is a wide East Asian character: Han,
// Hiragana, Katakana, or Hangul, a modifier letter, or CJK symbols and
// punctuation.
func isEastAsianWide(r rune) bool {
	return unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul, unicode.Lm) ||
		0x3000 <= r && r <= 0x303F
}

// spaceDiscarding is the space-discarding set of the draft of CSS Text
// Module Level 3: the blocks of CJK characters, Yi, and kana, and their
// radicals, strokes, and forms.
var spaceDiscarding = &unicode.RangeTable{
	R16: []unicode.Range16{
		{Lo: 0x2E80, Hi: 0x2EFF, Stride: 1}, // CJK Radicals Supplement
		{Lo: 0x2F00, Hi: 0x2FDF, Stride: 1}, // Kangxi Radicals
		{Lo: 0x2FF0, Hi: 0x30FF, Stride: 1}, // Ideographic Description Characters through Katakana
		{Lo: 0x3130, Hi: 0x319F, Stride: 1}, // Hangul Compatibility Jamo and Kanbun
		{Lo: 0x31C0, Hi: 0x31FF, Stride: 1}, // CJK Strokes and Katakana Phonetic Extensions
		{Lo: 0x3300, Hi: 0x4DBF, Stride: 1}, // CJK Compatibility through Extension A
		{Lo: 0x4E00, Hi: 0x9FFF, Stride: 1}, // CJK Unified Ideographs
		{Lo: 0xA000, Hi: 0xA4CF, Stride: 1}, // Yi Syllables and Radicals
		{Lo: 0xF900, Hi: 0xFAFF, Stride: 1}, // CJK Compatibility Ideographs
		{Lo: 0xFE10, Hi: 0xFE1F, Stride: 1}, // Vertical Forms
		{Lo: 0xFE30, Hi: 0xFE6F, Stride: 1}, // CJK Compatibility Forms and Small Form Variants
		{Lo: 0xFF00, Hi: 0xFFEF, Stride: 1}, // Halfwidth and Fullwidth Forms
	},
	R32: []unicode.Range32{
		{Lo: 0x1B000, Hi: 0x1B16F, Stride: 1}, // Kana Supplement through Small Kana Extension
		{Lo: 0x20000, Hi: 0x2A6DF, Stride: 1}, // CJK Unified Ideographs Extension B
		{Lo: 0x2A700, Hi: 0x2EBEF, Stride: 1}, // Extensions C through F
		{Lo: 0x2F800, Hi: 0x2FA1F, Stride: 1}, // CJK Compatibility Ideographs Supplement
		{Lo: 0x30000, Hi: 0x3134F, Stride: 1}, // Extension G
	},
}

// isFullOrWideOrHalf reports whether the East Asian width of r, as Unicode
// Standard Annex #11 gives it, is fullwidth, wide, or halfwidth.
func isFullOrWideOrHalf(r rune) bool {
	return unicode.Is(eastAsianFullWideHalf, r)
}

// eastAsianFullWideHalf holds the fullwidth, halfwidth, and wide characters
// of revision 36 of Unicode Standard Annex #11, East Asian Width, in the
// ranges of goldmark's table of them, from which the rules of
// EastAsianLineBreaksCSS3Draft come.
var eastAsianFullWideHalf = &unicode.RangeTable{
	R16: []unicode.Range16{
		{Lo: 0x1100, Hi: 0x115F, Stride: 1},
		{Lo: 0x11A3, Hi: 0x11A7, Stride: 1},
		{Lo: 0x11FA, Hi: 0x11FF, Stride: 1},
		{Lo: 0x20A9, Hi: 0x20A9, Stride: 1},
		{Lo: 0x2329, Hi: 0x232A, Stride: 1},
		{Lo: 0x2E80, Hi: 0x2E99, Stride: 1},
		{Lo: 0x2E9B, Hi: 0x2EF3, Stride: 1},
		{Lo: 0x2F00, Hi: 0x2FD5, Stride: 1},
		{Lo: 0x2FF0, Hi: 0x2FFB, Stride: 1},
		{Lo: 0x3000, Hi: 0x303E, Stride: 1},
		{Lo: 0x3041, Hi: 0x3096, Stride: 1},
		{Lo: 0x3099, Hi: 0x30FF, Stride: 1},
		{Lo: 0x3105, Hi: 0x312D, Stride: 1},
		{Lo: 0x3131, Hi: 0x318E, Stride: 1},
		{Lo: 0x3190, Hi: 0x31BA, Stride: 1},
		{Lo: 0x31C0, Hi: 0x31E3, Stride: 1},
		{Lo: 0x31F0, Hi: 0x321E, Stride: 1},
		{Lo: 0x3220, Hi: 0x3247, Stride: 1},
		{Lo: 0x3250, Hi: 0x32FE, Stride: 1},
		{Lo: 0x3300, Hi: 0x4DBF, Stride: 1},
		{Lo: 0x4E00, Hi: 0xA48C, Stride: 1},
		{Lo: 0xA490, Hi: 0xA4C6, Stride: 1},
		{Lo: 0xA960, Hi: 0xA97C, Stride: 1},
		{Lo: 0xAC00, Hi: 0xD7A3, Stride: 1},
		{Lo: 0xD7B0, Hi: 0xD7C6, Stride: 1},
		{Lo: 0xD7CB, Hi: 0xD7FB, Stride: 1},
		{Lo: 0xF900, Hi: 0xFAFF, Stride: 1},
		{Lo: 0xFE10, Hi: 0xFE19, Stride: 1},
		{Lo: 0xFE30, Hi: 0xFE52, Stride: 1},
		{Lo: 0xFE54, Hi: 0xFE66, Stride: 1},
		{Lo: 0xFE68, Hi: 0xFE6B, Stride: 1},
		{Lo: 0xFF01, Hi: 0xFFBE, Stride: 1},
		{Lo: 0xFFC2, Hi: 0xFFC7, Stride: 1},
		{Lo: 0xFFCA, Hi: 0xFFCF, Stride: 1},
		{Lo: 0xFFD2, Hi: 0xFFD7, Stride: 1},
		{Lo: 0xFFDA, Hi: 0xFFDC, Stride: 1},
		{Lo: 0xFFE0, Hi: 0xFFE6, Stride: 1},
		{Lo: 0xFFE8, Hi: 0xFFEE, Stride: 1},
	},
	R32: []unicode.Range32{
		{Lo: 0x1B000, Hi: 0x1B001, Stride: 1},
		{Lo: 0x1F200, Hi: 0x1F202, Stride: 1},
		{Lo: 0x1F210, Hi: 0x1F23A, Stride: 1},
		{Lo: 0x1F240, Hi: 0x1F248, Stride: 1},
		{Lo: 0x1F250, Hi: 0x1F251, Stride: 1},
		{Lo: 0x20000, Hi: 0x2FFFD, Stride: 1},
		{Lo: 0x30000, Hi: 0x3FFFD, Stride: 1},
	},
}
