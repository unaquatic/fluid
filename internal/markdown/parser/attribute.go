package parser

import (
	"slices"
	"strconv"
	"strings"

	"github.com/unaquatic/fluid/internal/markdown"
)

// ParseAttributes parses the list of attributes in braces at the start of
// s, after spaces, such as {#id .class key=value}. It returns the
// attributes, in order, and the length of the list with the spaces before
// it, or nil and 0 if there is none.
//
// "#" and a name sets the attribute "id", and "." and a name adds the name
// to the attribute "class", whose values join with spaces. A name, "=", and
// a value sets any attribute. A value is a string in double quotes, with
// the escapes of JSON; a number, as a float64; true, false, or null, as a
// bool or nil; a word of ASCII letters, digits, "_", ":", ".", and "-", as
// a string; values in brackets, as a []any; or attributes in braces, as a
// []markdown.Attribute. The value of "class" must be a string. Commas may
// part attributes and values. Brackets and braces nest at most 32 deep.
func ParseAttributes(s string) ([]markdown.Attribute, int) {
	p := &attrScanner{s: s}
	attrs, ok := p.attributes()
	if !ok {
		return nil, 0
	}
	return attrs, p.i
}

// maxAttrDepth is how deep brackets and braces can nest in a list of
// attributes, so that a list that does not close is not scanned again from
// each of the braces in it, nor deeper than the stack goes.
const maxAttrDepth = 32

// attrScanner scans attributes in s from offset i, in depth brackets and
// braces.
type attrScanner struct {
	s     string
	i     int
	depth int
}

// enter enters brackets or braces, and reports whether they nest no
// deeper than maxAttrDepth; leave leaves them.
func (p *attrScanner) enter() bool {
	p.depth++
	return p.depth <= maxAttrDepth
}

func (p *attrScanner) leave() { p.depth-- }

// peek returns the byte at the offset, or -1 at the end.
func (p *attrScanner) peek() int { return peek(p.s, p.i) }

// skipSpaces skips the spaces, tabs, and line endings at the offset.
func (p *attrScanner) skipSpaces() {
	for p.i < len(p.s) && isSpaceByte(p.s[p.i]) {
		p.i++
	}
}

// attributes scans a list of attributes in braces, after spaces.
func (p *attrScanner) attributes() ([]markdown.Attribute, bool) {
	p.skipSpaces()
	if p.peek() != '{' {
		return nil, false
	}
	defer p.leave()
	if !p.enter() {
		return nil, false
	}
	p.i++
	attrs := []markdown.Attribute{}
	for {
		if p.peek() == '}' {
			p.i++
			return attrs, true
		}
		a, ok := p.attribute()
		if !ok {
			return nil, false
		}
		i := slices.IndexFunc(attrs, func(b markdown.Attribute) bool { return b.Name == "class" })
		if a.Name == "class" && i >= 0 {
			attrs[i].Value = attrs[i].Value.(string) + " " + a.Value.(string)
		} else {
			attrs = append(attrs, a)
		}
		p.skipSpaces()
		if p.peek() == ',' {
			p.i++
			p.skipSpaces()
		}
	}
}

// attribute scans an attribute, after spaces.
func (p *attrScanner) attribute() (markdown.Attribute, bool) {
	p.skipSpaces()
	if c := p.peek(); c == '#' || c == '.' {
		p.i++
		start := p.i
		for p.i < len(p.s) && isShorthandByte(p.s[p.i]) {
			p.i++
		}
		name := "class"
		if c == '#' {
			name = "id"
		}
		return markdown.Attribute{Name: name, Value: replaceNUL(p.s[start:p.i])}, true
	}
	name, ok := p.word()
	if !ok {
		return markdown.Attribute{}, false
	}
	p.skipSpaces()
	if p.peek() != '=' {
		return markdown.Attribute{}, false
	}
	p.i++
	v, ok := p.value()
	if !ok {
		return markdown.Attribute{}, false
	}
	if _, isString := v.(string); name == "class" && !isString {
		return markdown.Attribute{}, false
	}
	return markdown.Attribute{Name: name, Value: v}, true
}

// value scans the value of an attribute, after spaces.
func (p *attrScanner) value() (any, bool) {
	p.skipSpaces()
	switch c := p.peek(); {
	case c == -1:
		return nil, false
	case c == '{':
		attrs, ok := p.attributes()
		return attrs, ok
	case c == '[':
		return p.array()
	case c == '"':
		return p.quoted()
	case c == '-' || c == '+' || isDigit(byte(c)):
		return p.number()
	}
	w, ok := p.word()
	switch {
	case !ok:
		return nil, false
	case w == "true":
		return true, true
	case w == "false":
		return false, true
	case w == "null":
		return nil, true
	}
	return w, true
}

// word scans a name, or a value that is a word: an ASCII letter, "_", or
// ":", and then ASCII letters, digits, "_", ":", ".", and "-".
func (p *attrScanner) word() (string, bool) {
	start := p.i
	if c := p.peek(); c == -1 || !isLetter(byte(c)) && c != '_' && c != ':' {
		return "", false
	}
	p.i++
	for p.i < len(p.s) {
		if c := p.s[p.i]; !isAlnum(c) && c != '_' && c != ':' && c != '.' && c != '-' {
			break
		}
		p.i++
	}
	return p.s[start:p.i], true
}

// array scans values in brackets.
func (p *attrScanner) array() ([]any, bool) {
	defer p.leave()
	if !p.enter() {
		return nil, false
	}
	p.i++ // the "["
	values := []any{}
	for first := true; ; first = false {
		switch c := p.peek(); {
		case c == ']':
			p.i++
			return values, true
		case c == ',' && !first:
			p.i++
		}
		v, ok := p.value()
		if !ok {
			return nil, false
		}
		values = append(values, v)
		p.skipSpaces()
	}
}

// quoted scans a string in double quotes, which ends on its line.
func (p *attrScanner) quoted() (string, bool) {
	var b strings.Builder
	for i := p.i + 1; i < len(p.s) && p.s[i] != '\n'; i++ {
		c := p.s[i]
		if c == '"' {
			p.i = i + 1
			return replaceNUL(b.String()), true
		}
		if c == '\\' && i+1 < len(p.s) && p.s[i+1] != '\n' {
			if e := strings.IndexByte(`"/\bfnrt`, p.s[i+1]); e >= 0 {
				b.WriteByte("\"/\\\b\f\n\r\t"[e])
				i++
				continue
			}
		}
		b.WriteByte(c)
	}
	return "", false
}

// number scans a number: a sign, digits, a fraction, and an exponent, of
// which the sign, the fraction, and the exponent are optional.
func (p *attrScanner) number() (float64, bool) {
	start := p.i
	if c := p.peek(); c == '-' || c == '+' {
		p.i++
	}
	if c := p.peek(); c == -1 || !isDigit(byte(c)) {
		return 0, false
	}
	p.digits()
	if p.peek() == '.' {
		p.i++
		p.digits()
	}
	if c := p.peek(); c == 'e' || c == 'E' {
		p.i++
		if c := p.peek(); c == '-' || c == '+' {
			p.i++
		}
		p.digits()
	}
	f, err := strconv.ParseFloat(p.s[start:p.i], 64)
	return f, err == nil
}

// digits skips the decimal digits at the offset.
func (p *attrScanner) digits() {
	for p.i < len(p.s) && isDigit(p.s[p.i]) {
		p.i++
	}
}

// isShorthandByte reports whether c can be in the name after "#" or ".":
// any byte but whitespace and ASCII punctuation other than "_", "-", ":",
// and ".".
func isShorthandByte(c byte) bool {
	return !isSpaceByte(c) && (!isASCIIPunct(c) || c == '_' || c == '-' || c == ':' || c == '.')
}

// isSpaceByte reports whether c is an ASCII space, tab, or line ending.
func isSpaceByte(c byte) bool {
	return c == ' ' || '\t' <= c && c <= '\r'
}

// trailingAttributes returns the attributes in braces at the end of text,
// the content of a heading, after spaces, and text before them, without
// the spaces before them. A brace that a backslash escapes starts none. If
// none ends text, it returns nil and text.
func trailingAttributes(text string) ([]markdown.Attribute, string) {
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '\\':
			if i+1 < len(text) && isASCIIPunct(text[i+1]) {
				i++
			}
		case '{':
			if attrs, n := ParseAttributes(text[i:]); n > 0 && isBlank(text[i+n:]) {
				return attrs, strings.TrimRight(text[:i], " \t\n\v\f\r")
			}
		}
	}
	return nil, text
}
