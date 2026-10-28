package parser

import (
	"html"
	"strings"
	"unicode/utf8"

	"github.com/unaquatic/fluid/internal/markdown/ast"
)

// escapeParser parses backslash escapes.
type escapeParser struct{}

// NewEscapeParser returns the parser of backslash escapes, which make an
// ASCII punctuation character literal, and of the hard line breaks that a
// backslash at the end of a line makes. With [WithEscapedSpace], an escaped
// space stands for nothing.
func NewEscapeParser() InlineParser { return escapeParser{} }

// Trigger implements [InlineParser].
func (escapeParser) Trigger() []byte { return []byte{'\\'} }

// Parse implements [InlineParser].
func (escapeParser) Parse(parent ast.Node, s *InlineState) (ast.Node, int) {
	r := s.Rest()
	switch {
	case len(r) > 1 && r[1] == '\n':
		br := ast.NewText("")
		br.SetHardLineBreak(true)
		return br, 2
	case len(r) > 1 && isASCIIPunct(r[1]):
		return ast.NewText(r[1:2]), 2
	case len(r) > 1 && r[1] == ' ' && s.syn.escapedSpace && !endsLine(r[2:]):
		return ast.NewText(""), 2
	}
	return ast.NewText(`\`), 1
}

// endsLine reports whether only spaces and tabs come before the end of the
// line at the start of s, so that they are the spaces at the end of a line,
// which a backslash before them does not escape.
func endsLine(s string) bool {
	s = strings.TrimLeft(s, " \t")
	return s == "" || s[0] == '\n'
}

// entityParser parses entity and numeric character references.
type entityParser struct{}

// NewEntityParser returns the parser of entity and numeric character
// references.
func NewEntityParser() InlineParser { return entityParser{} }

// Trigger implements [InlineParser].
func (entityParser) Trigger() []byte { return []byte{'&'} }

// Parse implements [InlineParser].
func (entityParser) Parse(parent ast.Node, s *InlineState) (ast.Node, int) {
	text, n := entity(s.Rest())
	if n == 0 {
		return nil, 0
	}
	return ast.NewText(text), n
}

// unescape resolves the backslash escapes and entities in s.
func unescape(s string) string {
	if !strings.ContainsAny(s, "\\&") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		switch c := s[i]; {
		case c == '\\' && i+1 < len(s) && isASCIIPunct(s[i+1]):
			b.WriteByte(s[i+1])
			i += 2
			continue
		case c == '&':
			if text, n := entity(s[i:]); n > 0 {
				b.WriteString(text)
				i += n
				continue
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// entity returns the text that the entity or numeric character reference at
// the start of s stands for, and its length, or 0 if there is none.
func entity(s string) (string, int) {
	if len(s) < 3 || s[0] != '&' {
		return "", 0
	}
	if s[1] == '#' {
		i, base, digits := 2, 10, 7
		if s[i]|0x20 == 'x' {
			i, base, digits = 3, 16, 6
		}
		start := i
		var r rune
		for ; i < len(s) && i-start < digits; i++ {
			d := hexValue(s[i])
			if d < 0 || d >= base {
				break
			}
			r = r*rune(base) + rune(d)
		}
		if i == start || i >= len(s) || s[i] != ';' {
			return "", 0
		}
		if r == 0 || !utf8.ValidRune(r) {
			r = utf8.RuneError
		}
		return string(r), i + 1
	}

	// A named entity is an HTML5 entity name between "&" and ";".
	i := 1
	for i < len(s) && i <= 32 && isAlnum(s[i]) {
		i++
	}
	if i < 3 || !isLetter(s[1]) || i >= len(s) || s[i] != ';' {
		return "", 0
	}
	ref := s[:i+1]
	// UnescapeString also decodes the longest entity that prefixes a name,
	// leaving the rest of the name and the ";" after it, which no entity
	// is shorter than.
	text := html.UnescapeString(ref)
	if text == ref || utf8.RuneCountInString(text) > 2 {
		return "", 0
	}
	return text, i + 1
}

// hexValue returns the value of hexadecimal digit c, or -1.
func hexValue(c byte) int {
	switch {
	case isDigit(c):
		return int(c - '0')
	case 'a' <= c|0x20 && c|0x20 <= 'f':
		return int(c|0x20-'a') + 10
	}
	return -1
}
