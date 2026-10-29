package spectest

import (
	"strconv"
	"strings"

	"github.com/unaquatic/fluid/internal/markdown/ast"
)

// Options configure how HTML renders.
type Options struct {
	// XHTML ends empty elements as XHTML does, as in "<br />", and renders
	// the alignments of table cells as align attributes.
	XHTML bool

	// Unsafe renders raw HTML, and the destinations of links and images
	// that could run script, as they are written, in place of leaving them
	// out.
	Unsafe bool

	// TableAlignAttribute renders the alignments of table cells as align
	// attributes without XHTML too, in place of style attributes.
	TableAlignAttribute bool

	// KeepsSoftLineBreak reports whether the soft line break after a text
	// renders; nil renders all.
	KeepsSoftLineBreak func(t *ast.Text) bool
}

// HTML returns the HTML of n, usually a document, as the reference
// implementations of CommonMark and GFM render it, and goldmark renders
// footnotes and definition lists, for tests to compare. It is no renderer
// of fluid's: tests use it to check the trees that the parser builds
// against the HTML that the specifications give.
func HTML(n ast.Node, o Options) string {
	w := &writer{o: o}
	_ = ast.Walk(n, w.render)
	return w.b.String()
}

// writer writes the HTML of a document.
type writer struct {
	b strings.Builder
	o Options
}

// write writes s as it is.
func (w *writer) write(s string) { w.b.WriteString(s) }

// text writes s, escaping the characters that HTML gives meaning to.
func (w *writer) text(s string) { w.write(escaper.Replace(s)) }

// cr starts a line, unless nothing is written or a line just ended.
func (w *writer) cr() {
	if s := w.b.String(); s != "" && s[len(s)-1] != '\n' {
		w.write("\n")
	}
}

// end returns the end of an empty element.
func (w *writer) end() string {
	if w.o.XHTML {
		return " />"
	}
	return ">"
}

// destination writes url as an attribute value, or nothing in place of
// one that could run script, unless the writer is unsafe.
func (w *writer) destination(url string) {
	if w.o.Unsafe || !isDangerousURL(url) {
		w.text(escapeURL(url))
	}
}

// keeps reports whether the soft line break after t renders.
func (w *writer) keeps(t *ast.Text) bool {
	return w.o.KeepsSoftLineBreak == nil || w.o.KeepsSoftLineBreak(t)
}

var escaper = strings.NewReplacer(`&`, "&amp;", `<`, "&lt;", `>`, "&gt;", `"`, "&quot;")

// omitted is what a writer that is not unsafe writes in place of raw HTML.
const omitted = "<!-- raw HTML omitted -->"

// render writes the HTML of n as Walk enters it and leaves it.
func (w *writer) render(n ast.Node, entering bool) (ast.WalkStatus, error) {
	// open writes the tag that opens n's element as the walk enters n, and
	// the one that closes it as it leaves.
	open := func(start, end string) {
		if entering {
			w.write(start)
		} else {
			w.write(end)
		}
	}
	// Elements that hold no blocks or inline content of their own write all
	// of themselves as the walk enters them.
	switch n := n.(type) {
	case *ast.Paragraph:
		if isTight(n) {
			// A line break parts the paragraph's text from the block after
			// it, which another paragraph does not start with.
			if !entering && n.NextSibling() != nil && n.HasChildren() {
				w.write("\n")
			}
			break
		}
		if entering {
			w.cr()
		}
		open("<p>", "</p>\n")
	case *ast.Heading:
		level := strconv.Itoa(n.Level)
		if entering {
			w.cr()
		}
		open("<h"+level+">", "</h"+level+">\n")
	case *ast.ThematicBreak:
		if entering {
			w.cr()
			w.write("<hr" + w.end() + "\n")
		}
	case *ast.CodeBlock:
		if !entering {
			break
		}
		w.cr()
		w.write("<pre><code>")
		w.text(n.Literal)
		w.write("</code></pre>\n")
		return ast.WalkSkipChildren, nil
	case *ast.FencedCodeBlock:
		if !entering {
			break
		}
		w.cr()
		w.write("<pre><code")
		if lang, _, _ := strings.Cut(strings.ReplaceAll(n.Info, "\t", " "), " "); lang != "" {
			w.write(` class="language-`)
			w.text(lang)
			w.write(`"`)
		}
		w.write(">")
		w.text(n.Literal)
		w.write("</code></pre>\n")
		return ast.WalkSkipChildren, nil
	case *ast.Blockquote:
		w.cr()
		open("<blockquote>\n", "</blockquote>\n")
	case *ast.List:
		tag := "ul"
		if n.IsOrdered() {
			tag = "ol"
		}
		if !entering {
			w.write("</" + tag + ">\n")
			break
		}
		w.cr()
		if n.IsOrdered() && n.Start != 1 {
			w.write(`<ol start="` + strconv.Itoa(n.Start) + `">` + "\n")
		} else {
			w.write("<" + tag + ">\n")
		}
	case *ast.ListItem:
		if !entering {
			w.write("</li>\n")
			break
		}
		w.cr()
		w.write("<li>")
		switch n.Task {
		case ast.Unchecked:
			w.write(`<input disabled="" type="checkbox"` + w.end() + " ")
		case ast.Checked:
			w.write(`<input checked="" disabled="" type="checkbox"` + w.end() + " ")
		}
	case *ast.HTMLBlock:
		if !entering {
			break
		}
		w.cr()
		if w.o.Unsafe {
			w.write(n.Literal)
		} else {
			w.write(omitted)
		}
		w.cr()
		return ast.WalkSkipChildren, nil
	case *ast.Table:
		if entering {
			w.cr()
		}
		open("<table>\n", "</table>\n")
	case *ast.TableHeader:
		open("<thead>\n<tr>\n", "</tr>\n</thead>\n")
		if !entering && n.NextSibling() != nil {
			w.write("<tbody>\n")
		}
	case *ast.TableRow:
		open("<tr>\n", "</tr>\n")
		if !entering && n.NextSibling() == nil {
			w.write("</tbody>\n")
		}
	case *ast.TableCell:
		tag := "td"
		if _, ok := n.Parent().(*ast.TableHeader); ok {
			tag = "th"
		}
		if !entering {
			w.write("</" + tag + ">\n")
			break
		}
		w.write("<" + tag)
		if align := alignName(n.Alignment); align != "" {
			if w.o.XHTML || w.o.TableAlignAttribute {
				w.write(` align="` + align + `"`)
			} else {
				w.write(` style="text-align:` + align + `"`)
			}
		}
		w.write(">")
	case *ast.FootnoteList:
		if entering {
			w.cr()
		}
		open(`<div class="footnotes" role="doc-endnotes">`+"\n<hr"+w.end()+"\n<ol>\n", "</ol>\n</div>\n")
	case *ast.Footnote:
		open(`<li id="fn:`+strconv.Itoa(n.Index)+`">`+"\n", "</li>\n")
	case *ast.DefinitionList:
		if entering {
			w.cr()
		}
		open("<dl>\n", "</dl>\n")
	case *ast.DefinitionTerm:
		if entering {
			w.cr()
		}
		open("<dt>", "</dt>\n")
	case *ast.DefinitionDescription:
		if !entering {
			w.write("</dd>\n")
			break
		}
		w.cr()
		w.write("<dd>")
		if !n.IsTight {
			w.write("\n")
		}

	case *ast.Text:
		if !entering {
			break
		}
		w.text(n.Value)
		switch {
		case n.HardLineBreak():
			w.write("<br" + w.end() + "\n")
		case n.SoftLineBreak() && w.keeps(n):
			w.write("\n")
		}
	case *ast.CodeSpan:
		if !entering {
			break
		}
		w.write("<code>")
		w.text(w.plainText(n))
		w.write("</code>")
		return ast.WalkSkipChildren, nil
	case *ast.Emphasis:
		tag := "em"
		if n.Level == 2 {
			tag = "strong"
		}
		open("<"+tag+">", "</"+tag+">")
	case *ast.Strikethrough:
		open("<del>", "</del>")
	case *ast.Link:
		if !entering {
			w.write("</a>")
			break
		}
		w.write(`<a href="`)
		w.destination(n.Destination)
		w.write(`"`)
		if n.Title != "" {
			w.write(` title="`)
			w.text(n.Title)
			w.write(`"`)
		}
		w.write(">")
	case *ast.Image:
		if !entering {
			break
		}
		w.write(`<img src="`)
		w.destination(n.Destination)
		w.write(`" alt="`)
		w.text(w.plainText(n))
		w.write(`"`)
		if n.Title != "" {
			w.write(` title="`)
			w.text(n.Title)
			w.write(`"`)
		}
		w.write(w.end())
		return ast.WalkSkipChildren, nil
	case *ast.AutoLink:
		if !entering {
			break
		}
		w.write(`<a href="`)
		w.destination(n.URL)
		w.write(`">`)
		w.text(n.Label)
		w.write("</a>")
		return ast.WalkSkipChildren, nil
	case *ast.RawHTML:
		if !entering {
			break
		}
		if w.o.Unsafe {
			w.write(n.Value)
		} else {
			w.write(omitted)
		}
		return ast.WalkSkipChildren, nil
	case *ast.FootnoteLink:
		if entering {
			index := strconv.Itoa(n.Index)
			w.write(`<sup id="fnref` + refNumber(n.RefIndex) + ":" + index + `"><a href="#fn:` + index +
				`" class="footnote-ref" role="doc-noteref">` + index + "</a></sup>")
		}
	case *ast.FootnoteBacklink:
		if entering {
			w.write(`&#160;<a href="#fnref` + refNumber(n.RefIndex) + ":" + strconv.Itoa(n.Index) +
				`" class="footnote-backref" role="doc-backlink">&#x21a9;&#xfe0e;</a>`)
		}
	}
	return ast.WalkContinue, nil
}

// isTight reports whether n is a block of an item of a tight list, or of a
// tight description, which write no paragraph tags.
func isTight(n ast.Node) bool {
	switch p := n.Parent().(type) {
	case *ast.ListItem:
		l, ok := p.Parent().(*ast.List)
		return ok && l.IsTight
	case *ast.DefinitionDescription:
		return p.IsTight
	}
	return false
}

// refNumber returns the number of reference refIndex in the ID of a
// footnote reference: none for the first.
func refNumber(refIndex int) string {
	if refIndex == 0 {
		return ""
	}
	return strconv.Itoa(refIndex)
}

// plainText returns the text of the inline content of n without markup, as
// code spans and image descriptions render it.
func (w *writer) plainText(n ast.Node) string {
	var b strings.Builder
	for c := range ast.Children(n) {
		switch c := c.(type) {
		case *ast.Text:
			b.WriteString(c.Value)
			if c.HardLineBreak() || c.SoftLineBreak() && w.keeps(c) {
				b.WriteString("\n")
			}
		case *ast.AutoLink:
			b.WriteString(c.Label)
		case *ast.RawHTML:
		default:
			b.WriteString(w.plainText(c))
		}
	}
	return b.String()
}

// alignName returns the name of alignment a in HTML, or "" for none.
func alignName(a ast.Alignment) string {
	if a == ast.AlignNone {
		return ""
	}
	return a.String()
}

// escapeURL percent-encodes the bytes of s that a URL cannot hold, keeping
// percent-encoded bytes as they are.
func escapeURL(s string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case isAlnum(c) || strings.IndexByte(";/?:@&=+$,-_.!~*'()#", c) >= 0:
			b.WriteByte(c)
		case c == '%' && i+2 < len(s) && isHex(s[i+1]) && isHex(s[i+2]):
			b.WriteByte(c)
		default:
			b.WriteByte('%')
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&15])
		}
	}
	return b.String()
}

// isDangerousURL reports whether url could run script: one with the scheme
// javascript, vbscript, or file, or data unless it is an image of type gif,
// png, jpeg, or webp.
func isDangerousURL(url string) bool {
	lower := strings.ToLower(strings.TrimLeft(url, " \t\n\r\f"))
	for _, scheme := range []string{"javascript:", "vbscript:", "file:"} {
		if strings.HasPrefix(lower, scheme) {
			return true
		}
	}
	if !strings.HasPrefix(lower, "data:") {
		return false
	}
	for _, safe := range []string{"data:image/png", "data:image/gif", "data:image/jpeg", "data:image/webp"} {
		if strings.HasPrefix(lower, safe) {
			return false
		}
	}
	return true
}

func isAlnum(c byte) bool { return '0' <= c && c <= '9' || 'a' <= c|0x20 && c|0x20 <= 'z' }
func isHex(c byte) bool   { return '0' <= c && c <= '9' || 'a' <= c|0x20 && c|0x20 <= 'f' }
