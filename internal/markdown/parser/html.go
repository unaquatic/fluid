package parser

import (
	"strings"

	"github.com/unaquatic/fluid/internal/markdown"
)

// htmlBlockParser parses HTML blocks.
type htmlBlockParser struct{}

// NewHTMLBlockParser returns the parser of HTML blocks, at a line that
// starts with one of the seven kinds of HTML that CommonMark lists. The
// last of them, a complete open or closing tag of any name, cannot
// interrupt a paragraph. A block starts at its indentation.
func NewHTMLBlockParser() BlockParser { return htmlBlockParser{} }

// Trigger implements [BlockParser].
func (htmlBlockParser) Trigger() []byte { return []byte{'<'} }

// Open implements [BlockParser].
func (htmlBlockParser) Open(parent markdown.Node, s *BlockState) (markdown.Node, State) {
	rest := s.Rest()
	_, inPara := parent.(*markdown.Paragraph)
	lazy := s.Lazy()
	for t := 1; t <= 7; t++ {
		if htmlBlockStarts(t, rest) && (t < 7 || !inPara && !lazy) {
			s.StartAt(s.lineOffset + s.offset)
			return &markdown.HTMLBlock{HTMLBlockType: t}, NoChildren
		}
	}
	return nil, None
}

// Continue implements [BlockParser]. A blank line ends blocks of types 6
// and 7.
func (htmlBlockParser) Continue(n markdown.Node, s *BlockState) State {
	if t := n.(*markdown.HTMLBlock).HTMLBlockType; s.blank && (t == 6 || t == 7) {
		return Close
	}
	return Continue | NoChildren
}

// AddLine implements [LineAdder]. A line that meets the end condition of a
// block of types 1 through 5 is the block's last.
func (htmlBlockParser) AddLine(n markdown.Node, s *BlockState) {
	line := s.Remaining()
	n.SetLines(append(n.Lines(), line))
	if t := n.(*markdown.HTMLBlock).HTMLBlockType; t <= 5 && htmlBlockEnds(t, line.Text) {
		s.Close(n, s.lineNumber)
	}
}

// Close implements [BlockParser]. The block ends before its trailing blank
// lines.
func (htmlBlockParser) Close(n markdown.Node, s *BlockState) {
	lines := n.Lines()
	end := len(lines)
	for end > 1 && strings.Trim(lines[end-1].Text, " ") == "" {
		end--
	}
	n.(*markdown.HTMLBlock).Literal = replaceNUL(joinText(lines[:end], ""))
	n.SetLines(lines[:end])
}

// CanInterruptParagraph implements [BlockParser].
func (htmlBlockParser) CanInterruptParagraph() bool { return true }

// CanAcceptIndentedLine implements [BlockParser].
func (htmlBlockParser) CanAcceptIndentedLine() bool { return false }

// htmlTag returns the length of the HTML open tag, closing tag, comment,
// processing instruction, declaration, or CDATA section at the start of s,
// or 0 if there is none.
func htmlTag(s string) int {
	if len(s) < 3 || s[0] != '<' {
		return 0
	}
	switch c := s[1]; {
	case isLetter(c):
		return openTag(s)
	case c == '/':
		return closingTag(s)
	case c == '?':
		return through(s, 2, "?>")
	case c != '!':
		return 0
	case strings.HasPrefix(s, "<!-->"):
		return 5
	case strings.HasPrefix(s, "<!--->"):
		return 6
	case strings.HasPrefix(s, "<!--"):
		return through(s, 4, "-->")
	case strings.HasPrefix(s, "<![CDATA["):
		return through(s, 9, "]]>")
	case isLetter(s[2]):
		return through(s, 3, ">")
	}
	return 0
}

// htmlTagEnd returns the string that ends the HTML comment, processing
// instruction, declaration, or CDATA section that s starts with, or "" if
// s starts with none of them.
func htmlTagEnd(s string) string {
	switch {
	case strings.HasPrefix(s, "<?"):
		return "?>"
	case strings.HasPrefix(s, "<!--"):
		return "-->"
	case strings.HasPrefix(s, "<![CDATA["):
		return "]]>"
	case strings.HasPrefix(s, "<!"):
		return ">"
	}
	return ""
}

// through returns the offset after the first end in s at or after offset
// i, or 0 if there is none.
func through(s string, i int, end string) int {
	j := strings.Index(s[i:], end)
	if j < 0 {
		return 0
	}
	return i + j + len(end)
}

// openTag returns the length of the HTML open tag at the start of s, or 0.
func openTag(s string) int {
	i := tagNameEnd(s, 1)
	if i == 1 {
		return 0
	}
	for {
		j := skipTagSpace(s, i)
		if j == i || j == len(s) || !isAttrNameStart(s[j]) {
			i = j
			break
		}
		// An attribute, with an optional value.
		i = j + 1
		for i < len(s) && isAttrNameChar(s[i]) {
			i++
		}
		k := skipTagSpace(s, i)
		if k < len(s) && s[k] == '=' {
			i = attrValueEnd(s, skipTagSpace(s, k+1))
			if i == 0 {
				return 0
			}
		}
	}
	if i < len(s) && s[i] == '/' {
		i++
	}
	if i < len(s) && s[i] == '>' {
		return i + 1
	}
	return 0
}

// closingTag returns the length of the HTML closing tag at the start of s,
// or 0.
func closingTag(s string) int {
	if !strings.HasPrefix(s, "</") {
		return 0
	}
	i := tagNameEnd(s, 2)
	if i == 2 {
		return 0
	}
	i = skipTagSpace(s, i)
	if i < len(s) && s[i] == '>' {
		return i + 1
	}
	return 0
}

// tagNameEnd returns the offset after the tag name at offset i of s, or i
// if there is none.
func tagNameEnd(s string, i int) int {
	if i >= len(s) || !isLetter(s[i]) {
		return i
	}
	i++
	for i < len(s) && (isAlnum(s[i]) || s[i] == '-') {
		i++
	}
	return i
}

// skipTagSpace returns the offset after the spaces, tabs, and line endings
// at offset i of s.
func skipTagSpace(s string, i int) int {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n') {
		i++
	}
	return i
}

func isAttrNameStart(c byte) bool { return isLetter(c) || c == '_' || c == ':' }
func isAttrNameChar(c byte) bool  { return isAlnum(c) || c == '_' || c == '.' || c == ':' || c == '-' }

// attrValueEnd returns the offset after the attribute value at offset i of
// s, or 0 if there is none.
func attrValueEnd(s string, i int) int {
	if i >= len(s) {
		return 0
	}
	if q := s[i]; q == '"' || q == '\'' {
		j := strings.IndexByte(s[i+1:], q)
		if j < 0 {
			return 0
		}
		return i + 1 + j + 1
	}
	j := i
	// A NUL character, which CommonMark replaces, is not a control
	// character.
	for j < len(s) && (s[j] > ' ' || s[j] == 0) && !strings.ContainsRune("\"'=<>`", rune(s[j])) {
		j++
	}
	if j == i {
		return 0
	}
	return j
}

// htmlBlockTags holds the tag names that start an HTML block of type 6.
var htmlBlockTags = map[string]bool{
	"address": true, "article": true, "aside": true, "base": true, "basefont": true,
	"blockquote": true, "body": true, "caption": true, "center": true, "col": true,
	"colgroup": true, "dd": true, "details": true, "dialog": true, "dir": true,
	"div": true, "dl": true, "dt": true, "fieldset": true, "figcaption": true,
	"figure": true, "footer": true, "form": true, "frame": true, "frameset": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"head": true, "header": true, "hr": true, "html": true, "iframe": true,
	"legend": true, "li": true, "link": true, "main": true, "menu": true,
	"menuitem": true, "nav": true, "noframes": true, "ol": true, "optgroup": true,
	"option": true, "p": true, "param": true, "search": true, "section": true,
	"summary": true, "table": true, "tbody": true, "td": true, "tfoot": true,
	"th": true, "thead": true, "title": true, "tr": true, "track": true, "ul": true,
}

// htmlRawTags holds the tag names that start an HTML block of type 1.
var htmlRawTags = []string{"pre", "script", "style", "textarea"}

// htmlBlockStarts reports whether line s, from its "<", starts an HTML block
// of type t, as CommonMark numbers its start conditions.
func htmlBlockStarts(t int, s string) bool {
	switch t {
	case 1:
		for _, name := range htmlRawTags {
			if len(s) > len(name) && strings.EqualFold(s[1:1+len(name)], name) {
				rest := s[1+len(name):]
				if rest == "" || rest[0] == ' ' || rest[0] == '\t' || rest[0] == '>' {
					return true
				}
			}
		}
	case 2:
		return strings.HasPrefix(s, "<!--")
	case 3:
		return strings.HasPrefix(s, "<?")
	case 4:
		return len(s) > 2 && s[1] == '!' && isLetter(s[2])
	case 5:
		return strings.HasPrefix(s, "<![CDATA[")
	case 6:
		i := 1
		if strings.HasPrefix(s, "</") {
			i = 2
		}
		j := i
		for j < len(s) && isAlnum(s[j]) {
			j++
		}
		if !htmlBlockTags[strings.ToLower(s[i:j])] {
			return false
		}
		rest := s[j:]
		return rest == "" || rest[0] == ' ' || rest[0] == '\t' || rest[0] == '>' || strings.HasPrefix(rest, "/>")
	case 7:
		n := openTag(s)
		if n == 0 {
			n = closingTag(s)
		}
		return n > 0 && isBlank(s[n:])
	}
	return false
}

// htmlBlockEnds reports whether line s meets the end condition of an HTML
// block of type t, for t from 1 through 5.
func htmlBlockEnds(t int, s string) bool {
	switch t {
	case 1:
		lower := strings.ToLower(s)
		for _, name := range htmlRawTags {
			if strings.Contains(lower, "</"+name+">") {
				return true
			}
		}
	case 2:
		return strings.Contains(s, "-->")
	case 3:
		return strings.Contains(s, "?>")
	case 4:
		return strings.Contains(s, ">")
	case 5:
		return strings.Contains(s, "]]>")
	}
	return false
}
