package parser

import (
	"strings"

	"github.com/unaquatic/fluid/internal/markdown/ast"
)

// autoLinkParser parses autolinks.
type autoLinkParser struct{}

// NewAutoLinkParser returns the parser of autolinks: absolute URIs and
// email addresses in angle brackets.
func NewAutoLinkParser() InlineParser { return autoLinkParser{} }

// Trigger implements [InlineParser].
func (autoLinkParser) Trigger() []byte { return []byte{'<'} }

// Parse implements [InlineParser].
func (autoLinkParser) Parse(parent ast.Node, s *InlineState) (ast.Node, int) {
	r := s.Rest()
	if n := emailAutolink(r); n > 0 {
		addr := r[1 : n-1]
		return &ast.AutoLink{AutoLinkType: ast.AutoLinkEmail, URL: "mailto:" + addr, Label: addr}, n
	}
	if n := uriAutolink(r); n > 0 {
		uri := r[1 : n-1]
		return &ast.AutoLink{AutoLinkType: ast.AutoLinkURL, URL: uri, Label: uri}, n
	}
	return nil, 0
}

// uriAutolink returns the length of the URI autolink at the start of s, or
// 0.
func uriAutolink(s string) int {
	if len(s) < 4 || s[0] != '<' || !isLetter(s[1]) {
		return 0
	}
	i := 2
	for i < len(s) && (isAlnum(s[i]) || s[i] == '+' || s[i] == '.' || s[i] == '-') {
		i++
	}
	if n := i - 1; n < 2 || n > 32 || i == len(s) || s[i] != ':' {
		return 0
	}
	for i++; i < len(s); i++ {
		switch c := s[i]; {
		case c == '>':
			return i + 1
		case c == '<' || c <= ' ' || c == 0x7f:
			return 0
		}
	}
	return 0
}

// emailAutolink returns the length of the email autolink at the start of s,
// or 0.
func emailAutolink(s string) int {
	if len(s) < 4 || s[0] != '<' {
		return 0
	}
	i := 1
	for i < len(s) && (isAlnum(s[i]) || strings.IndexByte(".!#$%&'*+/=?^_`{|}~-", s[i]) >= 0) {
		i++
	}
	if i == 1 || i == len(s) || s[i] != '@' {
		return 0
	}
	// Labels of letters, digits, and hyphens, at most 63 long and with a
	// letter or digit at each end, separated by periods.
	for {
		i++
		start := i
		for i < len(s) && i-start < 63 && (isAlnum(s[i]) || s[i] == '-') {
			i++
		}
		if i == start || s[start] == '-' || s[i-1] == '-' {
			return 0
		}
		if i == len(s) || s[i] != '.' {
			break
		}
	}
	if i < len(s) && s[i] == '>' {
		return i + 1
	}
	return 0
}
