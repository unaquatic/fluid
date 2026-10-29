package parser

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/unaquatic/fluid/internal/markdown/ast"
)

// bracket is an entry of the stack of "[" and "![" link openers.
type bracket struct {
	node      *ast.Text
	prev      *bracket
	prevDelim *Delimiter // top of the delimiter stack at the opener
	index     int        // offset of the "[" in the subject
	image     bool
	active    bool // whether a link can still close the opener

	// bracketAfter reports whether another opener follows this one.
	bracketAfter bool
}

// addBracket pushes a link opener for node, at offset index, to the
// bracket stack.
func (s *InlineState) addBracket(node *ast.Text, index int, image bool) {
	if s.brackets != nil {
		s.brackets.bracketAfter = true
	}
	s.brackets = &bracket{node: node, prev: s.brackets, prevDelim: s.delimStack, index: index, image: image, active: true}
}

// parseBang adds an image opener or "!" to b.
func (s *InlineState) parseBang(b ast.Node) {
	s.pos++
	if s.pos < len(s.src) && s.src[s.pos] == '[' {
		t := ast.NewText("![")
		b.AppendChild(b, t)
		s.addBracket(t, s.pos, true)
		s.pos++
		return
	}
	b.AppendChild(b, ast.NewText("!"))
}

// parseCloseBracket adds the link or image that the "]" at pos closes to
// b, or, if it closes none, the "]" as text.
func (s *InlineState) parseCloseBracket(b ast.Node) {
	s.pos++
	start := s.pos
	opener := s.brackets
	if opener == nil {
		b.AppendChild(b, ast.NewText("]"))
		return
	}
	if !opener.active {
		b.AppendChild(b, ast.NewText("]"))
		s.brackets = opener.prev
		return
	}

	var url, title string
	matched := false

	// An inline link.
	if s.pos < len(s.src) && s.src[s.pos] == '(' {
		i := spnl(s.src, s.pos+1)
		if dest, n, ok := linkDestination(s.src[i:]); ok {
			i += n
			j := spnl(s.src, i)
			if j > i && j < len(s.src) && s.titleCloses(j) {
				if t, tn, ok := linkTitle(s.src[j:]); ok {
					title = t
					j = spnl(s.src, j+tn)
				}
			}
			if j < len(s.src) && s.src[j] == ')' {
				s.pos = j + 1
				url = dest
				matched = true
			}
		}
	}

	// A full, collapsed, or shortcut reference link.
	if !matched {
		var ref string
		n := linkLabel(s.src[s.pos:])
		switch {
		case n > 2:
			ref = s.src[s.pos+1 : s.pos+n-1]
		case !opener.bracketAfter:
			// The reference is the link text, which holds no brackets
			// then.
			ref = s.src[opener.index+1 : start-1]
		}
		s.pos += n
		if def, ok := s.refs[normalizeLabel(ref)]; ok && (n > 2 || !opener.bracketAfter) {
			url, title = def.URL, def.Title
			matched = true
		}
	}

	if !matched {
		s.brackets = opener.prev
		s.pos = start
		b.AppendChild(b, ast.NewText("]"))
		return
	}

	var link ast.Node = &ast.Link{Destination: url, Title: title}
	if opener.image {
		link = &ast.Image{Destination: url, Title: title}
	}
	for t := opener.node.NextSibling(); t != nil; {
		next := t.NextSibling()
		link.AppendChild(link, t)
		t = next
	}
	b.AppendChild(b, link)
	s.processEmphasis(opener.prevDelim)
	s.brackets = opener.prev
	b.RemoveChild(b, opener.node)

	// Links cannot contain links, so a link deactivates the link openers
	// before it.
	if !opener.image {
		for o := s.brackets; o != nil; o = o.prev {
			if !o.image {
				o.active = false
			}
		}
	}
}

// titleCloses reports whether a character that could close a link title
// that starts at offset i follows it.
func (s *InlineState) titleCloses(i int) bool {
	switch c := s.src[i]; c {
	case '"', '\'':
		return s.HasAfter(s.src[i:i+1], i+1)
	case '(':
		return s.HasAfter(")", i+1)
	}
	return false
}

// spnl returns the offset after the spaces and tabs at offset i of s, and
// at most one line ending among them.
func spnl(s string, i int) int {
	for i < len(s) && isSpaceOrTab(s[i]) {
		i++
	}
	if i < len(s) && s[i] == '\n' {
		i++
		for i < len(s) && isSpaceOrTab(s[i]) {
			i++
		}
	}
	return i
}

// linkLabel returns the length of the link label at the start of s, or 0 if
// there is none.
func linkLabel(s string) int {
	if !strings.HasPrefix(s, "[") {
		return 0
	}
	chars := 0
	for i := 1; i < len(s); {
		switch s[i] {
		case '[':
			return 0
		case ']':
			return i + 1
		case '\\':
			if i+1 == len(s) {
				return 0
			}
			i++ // an escaped character, even a bracket, does not end the label
		}
		_, size := utf8.DecodeRuneInString(s[i:])
		i += size
		if chars++; chars > 999 {
			return 0
		}
	}
	return 0
}

// linkDestination returns the link destination at the start of s, with
// escapes and entities resolved, and its length.
func linkDestination(s string) (string, int, bool) {
	if strings.HasPrefix(s, "<") {
		for i := 1; i < len(s); i++ {
			switch s[i] {
			case '>':
				return unescape(s[1:i]), i + 1, true
			case '<', '\n':
				return "", 0, false
			case '\\':
				if i+1 == len(s) || s[i+1] == '\n' {
					return "", 0, false
				}
				i++
			}
		}
		return "", 0, false
	}
	parens := 0
	i := 0
loop:
	for i < len(s) {
		switch c := s[i]; {
		case c == '\\' && i+1 < len(s) && isASCIIPunct(s[i+1]):
			i += 2
			continue
		case c == '(':
			// Nesting is limited, as CommonMark allows, so that a long
			// run of openers is not scanned again for each link.
			if parens++; parens > 32 {
				return "", 0, false
			}
		case c == ')':
			if parens == 0 {
				break loop
			}
			parens--
		case c <= ' ' && c != 0 || c == 0x7f:
			// A NUL character, which CommonMark replaces, is not a control
			// character.
			break loop
		}
		i++
	}
	if i == 0 && !strings.HasPrefix(s, ")") || parens != 0 {
		return "", 0, false
	}
	return unescape(s[:i]), i, true
}

// linkTitle returns the link title at the start of s, with escapes and
// entities resolved, and its length.
func linkTitle(s string) (string, int, bool) {
	if s == "" {
		return "", 0, false
	}
	open, close := s[0], s[0]
	switch open {
	case '"', '\'':
	case '(':
		close = ')'
	default:
		return "", 0, false
	}
	for i := 1; i < len(s); i++ {
		switch c := s[i]; {
		case c == '\\':
			i++
		case c == close:
			return unescape(s[1:i]), i + 1, true
		case c == '(' && open == '(':
			return "", 0, false
		}
	}
	return "", 0, false
}

// normalizeLabel returns the form of link label s that matches labels of the
// same link: case-folded, with its runs of whitespace collapsed to one space,
// and trimmed.
func normalizeLabel(s string) string {
	s = replaceNUL(s)
	var b strings.Builder
	space := false
	for _, r := range s {
		switch r {
		case ' ', '\t', '\r', '\n':
			space = b.Len() > 0
			continue
		}
		if space {
			b.WriteByte(' ')
			space = false
		}
		switch r = unicode.ToLower(unicode.ToUpper(r)); r {
		case 'ß':
			b.WriteString("ss")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
