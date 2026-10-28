package extension

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/unaquatic/fluid/internal/markdown/ast"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// linkify is an extension of extended autolinks.
type linkify struct {
	options []LinkifyOption
}

// NewLinkify returns an extension that adds extended autolinks, configured
// by opts: the parser that [NewLinkifyParser] returns, with priority 100,
// at the start of each word.
func NewLinkify(opts ...LinkifyOption) parser.Extension { return &linkify{options: opts} }

// ExtendParser implements [parser.Extension].
func (e *linkify) ExtendParser(c *parser.Config) {
	c.Apply(parser.WithWordParser(NewLinkifyParser(e.options...), 100))
}

// A LinkifyOption configures extended autolinks.
type LinkifyOption func(c *LinkifyConfig)

// A LinkifyConfig holds which extended autolinks are recognized.
type LinkifyConfig struct {
	// AllowedProtocols holds the schemes of URL autolinks, each with its
	// colon, such as "https:". A URL autolink is a scheme, "//", and a
	// domain.
	AllowedProtocols []string

	// WWW reports whether "www." and a domain is an autolink.
	WWW bool

	// Email reports whether an email address is an autolink.
	Email bool
}

// WithLinkifyAllowedProtocols sets the schemes of URL autolinks, such as
// "https:", in place of "http:", "https:", and "ftp:". A colon is added to
// a scheme without one.
func WithLinkifyAllowedProtocols(protocols ...string) LinkifyOption {
	return func(c *LinkifyConfig) {
		c.AllowedProtocols = c.AllowedProtocols[:0]
		for _, p := range protocols {
			if !strings.HasSuffix(p, ":") {
				p += ":"
			}
			c.AllowedProtocols = append(c.AllowedProtocols, p)
		}
	}
}

// WithLinkifyWWW sets whether "www." and a domain is an autolink.
func WithLinkifyWWW(v bool) LinkifyOption { return func(c *LinkifyConfig) { c.WWW = v } }

// WithLinkifyEmail sets whether an email address is an autolink.
func WithLinkifyEmail(v bool) LinkifyOption { return func(c *LinkifyConfig) { c.Email = v } }

// linkifyParser parses extended autolinks.
type linkifyParser struct {
	config LinkifyConfig
}

// NewLinkifyParser returns the parser of extended autolinks, configured by
// opts: "www." and a domain, "http://", "https://", or "ftp://" and a
// domain, or an email address, each at the start of a line or after
// whitespace, "*", "_", "~", or "(", and not in the text of a link.
func NewLinkifyParser(opts ...LinkifyOption) parser.InlineParser {
	p := &linkifyParser{config: LinkifyConfig{
		AllowedProtocols: []string{"http:", "https:", "ftp:"},
		WWW:              true,
		Email:            true,
	}}
	for _, opt := range opts {
		opt(&p.config)
	}
	return p
}

// Trigger implements [parser.InlineParser]; an extended autolink starts
// with a letter or digit.
func (*linkifyParser) Trigger() []byte { return alnums }

// alnums holds the ASCII letters and digits.
var alnums = []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")

// Parse implements [parser.InlineParser].
func (p *linkifyParser) Parse(parent ast.Node, s *parser.InlineState) (ast.Node, int) {
	src, i := s.Source(), s.Pos()
	if s.InLinkText() || !autolinkBoundary(src, i) {
		return nil, 0
	}
	if text, url, n := p.extendedAutolink(src[i:]); n > 0 {
		return &ast.AutoLink{AutoLinkType: ast.AutoLinkURL, URL: url, Label: text}, n
	}
	if !p.config.Email {
		return nil, 0
	}
	if n := emailAutolink(s, i); n > 0 {
		return &ast.AutoLink{AutoLinkType: ast.AutoLinkEmail, URL: "mailto:" + src[i:i+n], Label: src[i : i+n]}, n
	}
	return nil, 0
}

// emailScan caches, for the inline content an [parser.InlineState] parses,
// how far the address characters before an "@" run, and which "@" a valid
// domain does not follow, as checks at each underscore of a long run would
// otherwise scan it again.
type emailScan struct {
	localEnd, noDomain int
}

// emailAutolink returns the length of the extended email autolink at
// offset i of the content that s parses, or 0.
func emailAutolink(s *parser.InlineState, i int) int {
	src := s.Source()
	e, _ := s.Value(emailScan{}).(*emailScan)
	if e == nil {
		e = &emailScan{noDomain: -1}
		s.SetValue(emailScan{}, e)
	}
	if i >= e.localEnd {
		e.localEnd = i
		for e.localEnd < len(src) && isEmailLocal(src[e.localEnd]) {
			e.localEnd++
		}
	}
	at := e.localEnd
	if at == len(src) || src[at] != '@' || at == e.noDomain {
		return 0
	}
	n := emailDomain(src[at+1:])
	if n == 0 {
		e.noDomain = at
		return 0
	}
	return at + 1 + n - i
}

// autolinkBoundary reports whether an extended autolink can start at
// offset i of s: at its start, or after whitespace, "*", "_", "~", or "(".
func autolinkBoundary(s string, i int) bool {
	if i == 0 {
		return true
	}
	r, _ := utf8.DecodeLastRuneInString(s[:i])
	return isUnicodeSpace(r) || r == '*' || r == '_' || r == '~' || r == '('
}

// extendedAutolink returns the text and URL of the extended www or URL
// autolink at the start of s, and its length, or 0 if there is none.
func (p *linkifyParser) extendedAutolink(s string) (text, url string, n int) {
	if strings.HasPrefix(s, "www.") {
		if !p.config.WWW {
			return "", "", 0
		}
		if d := domainEnd(s); d > len("www.") {
			if n = autolinkEnd(s, d); n > len("www.") {
				return s[:n], "http://" + s[:n], n
			}
		}
		return "", "", 0
	}
	for _, proto := range p.config.AllowedProtocols {
		if scheme := proto + "//"; strings.HasPrefix(s, scheme) {
			if d := domainEnd(s[len(scheme):]); d > 0 {
				if n = autolinkEnd(s, len(scheme)+d); n > len(scheme) {
					return s[:n], s[:n], n
				}
			}
			return "", "", 0
		}
	}
	return "", "", 0
}

// domainEnd returns the length of the valid domain at the start of s, or 0
// if there is none: segments of characters other than whitespace and
// punctuation, except underscores and hyphens, parted by periods, with at
// least one period and no underscore in the last two segments.
func domainEnd(s string) int {
	if strings.HasPrefix(s, ".") {
		return 0
	}
	periods, last, lastButOne := 0, 0, 0 // underscores in the last segments
	i := 0
scan:
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == '_':
			last++
		case r == '.':
			periods++
			lastButOne, last = last, 0
		case r == '-':
		case unicode.IsSpace(r) || isPunct(r):
			break scan
		}
		i += size
	}
	if periods == 0 || last > 0 || lastButOne > 0 {
		return 0
	}
	return i
}

// autolinkEnd returns the length of the extended autolink in s whose domain
// ends at offset i: the characters up to whitespace or "<", without the
// trailing punctuation, unmatched closing parentheses, and entity reference
// that GFM leaves out of the link.
func autolinkEnd(s string, i int) int {
	for i < len(s) && s[i] != '<' && !isASCIISpace(s[i]) {
		i++
	}
	opening, closing := strings.Count(s[:i], "("), strings.Count(s[:i], ")")
	for i > 0 {
		switch s[i-1] {
		case ')':
			if closing <= opening {
				return i
			}
			closing--
			i--
		case '?', '!', '.', ',', ':', '*', '_', '~', '\'', '"':
			i--
		case ';':
			j := i - 2
			for j > 0 && isAlnum(s[j]) {
				j--
			}
			if j < i-2 && s[j] == '&' {
				i = j
			} else {
				i--
			}
		default:
			return i
		}
	}
	return i
}

// isEmailLocal reports whether c can be in the address of an extended email
// autolink before its "@": a letter, a digit, ".", "-", "_", or "+".
func isEmailLocal(c byte) bool {
	return isAlnum(c) || c == '.' || c == '-' || c == '_' || c == '+'
}

// emailDomain returns the length of the domain of an extended email
// autolink at the start of s, which follows its "@", or 0 if there is none:
// letters, digits, "-", and "_", with periods between them and a letter
// last.
func emailDomain(s string) int {
	i, periods := 0, 0
	for ; i < len(s); i++ {
		c := s[i]
		if c == '.' && i+1 < len(s) && isAlnum(s[i+1]) {
			periods++
		} else if !isAlnum(c) && c != '-' && c != '_' {
			break
		}
	}
	if i == 0 || periods == 0 || !isLetter(s[i-1]) {
		return 0
	}
	return i
}

// isASCIISpace reports whether c is an ASCII whitespace character.
func isASCIISpace(c byte) bool {
	return c == ' ' || '\t' <= c && c <= '\r'
}
