package html

import (
	"strconv"
	"strings"

	"github.com/unaquatic/fluid/internal/markdown"
)

// commonMark renders the kinds of node of CommonMark, in the form of the
// CommonMark reference renderer.
type commonMark struct{}

// RegisterFuncs implements [NodeRenderer].
func (commonMark) RegisterFuncs(reg NodeRendererFuncRegisterer) {
	reg.Register(markdown.KindDocument, renderNothing)
	reg.Register(markdown.KindParagraph, renderParagraph)
	reg.Register(markdown.KindHeading, renderHeading)
	reg.Register(markdown.KindThematicBreak, renderThematicBreak)
	reg.Register(markdown.KindCodeBlock, renderCodeBlock)
	reg.Register(markdown.KindFencedCodeBlock, renderFencedCodeBlock)
	reg.Register(markdown.KindBlockquote, renderBlockquote)
	reg.Register(markdown.KindList, renderList)
	reg.Register(markdown.KindListItem, renderListItem)
	reg.Register(markdown.KindHTMLBlock, renderHTMLBlock)
	reg.Register(markdown.KindText, renderText)
	reg.Register(markdown.KindCodeSpan, renderCodeSpan)
	reg.Register(markdown.KindEmphasis, renderEmphasis)
	reg.Register(markdown.KindLink, renderLink)
	reg.Register(markdown.KindImage, renderImage)
	reg.Register(markdown.KindAutoLink, renderAutoLink)
	reg.Register(markdown.KindRawHTML, renderRawHTML)
}

// omitted is what a renderer that is not unsafe writes in place of raw
// HTML.
const omitted = "<!-- raw HTML omitted -->"

func renderNothing(*Writer, markdown.Node, bool) (markdown.WalkStatus, error) {
	return markdown.WalkContinue, nil
}

// isTight reports whether n is a block of an item of a tight list, or of a
// tight description of a definition list, which write no paragraph tags.
func isTight(n markdown.Node) bool {
	switch p := n.Parent().(type) {
	case *markdown.ListItem:
		l, ok := p.Parent().(*markdown.List)
		return ok && l.IsTight
	case *markdown.DefinitionDescription:
		return p.IsTight
	}
	return false
}

func renderParagraph(w *Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error) {
	if isTight(n) {
		// A line break parts the paragraph's text from the block after it,
		// which another paragraph does not start with.
		if !entering && n.NextSibling() != nil && n.HasChildren() {
			w.WriteString("\n")
		}
		return markdown.WalkContinue, nil
	}
	if entering {
		w.CR()
		w.WriteString("<p")
		w.WriteAttributes(n, ParagraphAttributeFilter)
		w.WriteString(">")
	} else {
		w.WriteString("</p>\n")
	}
	return markdown.WalkContinue, nil
}

func renderHeading(w *Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error) {
	level := strconv.Itoa(n.(*markdown.Heading).Level)
	if entering {
		w.CR()
		w.WriteString("<h" + level)
		w.WriteAttributes(n, HeadingAttributeFilter)
		w.WriteString(">")
	} else {
		w.WriteString("</h" + level + ">\n")
	}
	return markdown.WalkContinue, nil
}

func renderThematicBreak(w *Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error) {
	if entering {
		w.CR()
		w.WriteString("<hr")
		w.WriteAttributes(n, ThematicBreakAttributeFilter)
		w.WriteString(w.emptyEnd() + "\n")
	}
	return markdown.WalkContinue, nil
}

func renderCodeBlock(w *Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error) {
	if entering {
		w.CR()
		w.WriteString("<pre><code>")
		w.WriteText(n.(*markdown.CodeBlock).Literal)
		w.WriteString("</code></pre>\n")
	}
	return markdown.WalkSkipChildren, nil
}

func renderFencedCodeBlock(w *Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error) {
	if entering {
		b := n.(*markdown.FencedCodeBlock)
		w.CR()
		w.WriteString("<pre><code")
		if lang, _, _ := strings.Cut(strings.ReplaceAll(b.Info, "\t", " "), " "); lang != "" {
			w.WriteString(` class="language-`)
			w.WriteText(lang)
			w.WriteString(`"`)
		}
		w.WriteString(">")
		w.WriteText(b.Literal)
		w.WriteString("</code></pre>\n")
	}
	return markdown.WalkSkipChildren, nil
}

func renderBlockquote(w *Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error) {
	w.CR()
	if entering {
		w.WriteString("<blockquote")
		w.WriteAttributes(n, BlockquoteAttributeFilter)
		w.WriteString(">\n")
	} else {
		w.WriteString("</blockquote>\n")
	}
	return markdown.WalkContinue, nil
}

func renderList(w *Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error) {
	l := n.(*markdown.List)
	tag := "ul"
	if l.IsOrdered() {
		tag = "ol"
	}
	if !entering {
		w.WriteString("</" + tag + ">\n")
		return markdown.WalkContinue, nil
	}
	w.CR()
	w.WriteString("<" + tag)
	if l.IsOrdered() && l.Start != 1 {
		w.WriteString(` start="` + strconv.Itoa(l.Start) + `"`)
	}
	w.WriteAttributes(n, ListAttributeFilter)
	w.WriteString(">\n")
	return markdown.WalkContinue, nil
}

func renderListItem(w *Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error) {
	if !entering {
		w.WriteString("</li>\n")
		return markdown.WalkContinue, nil
	}
	w.CR()
	w.WriteString("<li")
	w.WriteAttributes(n, ListItemAttributeFilter)
	w.WriteString(">")
	switch n.(*markdown.ListItem).Task {
	case markdown.Unchecked:
		w.WriteString(`<input disabled="" type="checkbox"` + w.emptyEnd() + " ")
	case markdown.Checked:
		w.WriteString(`<input checked="" disabled="" type="checkbox"` + w.emptyEnd() + " ")
	}
	return markdown.WalkContinue, nil
}

func renderHTMLBlock(w *Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error) {
	if entering {
		w.CR()
		if w.config.Unsafe {
			w.WriteString(n.(*markdown.HTMLBlock).Literal)
		} else {
			w.WriteString(omitted)
		}
		w.CR()
	}
	return markdown.WalkSkipChildren, nil
}

func renderText(w *Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error) {
	if !entering {
		return markdown.WalkContinue, nil
	}
	t := n.(*markdown.Text)
	w.WriteText(t.Value)
	switch {
	case t.HardLineBreak() || t.SoftLineBreak() && w.config.HardWraps:
		w.WriteString("<br" + w.emptyEnd() + "\n")
	case t.SoftLineBreak() && w.keepsSoftLineBreak(t):
		w.WriteString("\n")
	}
	return markdown.WalkContinue, nil
}

func renderCodeSpan(w *Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error) {
	if entering {
		w.WriteString("<code")
		w.WriteAttributes(n, CodeSpanAttributeFilter)
		w.WriteString(">")
		w.WriteText(w.plainText(n))
		w.WriteString("</code>")
	}
	return markdown.WalkSkipChildren, nil
}

func renderEmphasis(w *Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error) {
	tag := "em"
	if n.(*markdown.Emphasis).Level == 2 {
		tag = "strong"
	}
	if entering {
		w.WriteString("<" + tag)
		w.WriteAttributes(n, EmphasisAttributeFilter)
		w.WriteString(">")
	} else {
		w.WriteString("</" + tag + ">")
	}
	return markdown.WalkContinue, nil
}

// writeDestination writes url as the value of an href or src attribute,
// or nothing in place of a dangerous one unless the renderer is unsafe.
func writeDestination(w *Writer, url string) {
	if w.config.Unsafe || !isDangerousURL(url) {
		w.WriteURL(url)
	}
}

func renderLink(w *Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error) {
	if !entering {
		w.WriteString("</a>")
		return markdown.WalkContinue, nil
	}
	l := n.(*markdown.Link)
	w.WriteString(`<a href="`)
	writeDestination(w, l.Destination)
	w.WriteString(`"`)
	if l.Title != "" {
		w.WriteString(` title="`)
		w.WriteText(l.Title)
		w.WriteString(`"`)
	}
	w.WriteAttributes(n, LinkAttributeFilter)
	w.WriteString(">")
	return markdown.WalkContinue, nil
}

func renderImage(w *Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error) {
	if !entering {
		return markdown.WalkContinue, nil
	}
	img := n.(*markdown.Image)
	w.WriteString(`<img src="`)
	writeDestination(w, img.Destination)
	w.WriteString(`" alt="`)
	w.WriteText(w.plainText(img))
	w.WriteString(`"`)
	if img.Title != "" {
		w.WriteString(` title="`)
		w.WriteText(img.Title)
		w.WriteString(`"`)
	}
	w.WriteAttributes(n, ImageAttributeFilter)
	w.WriteString(w.emptyEnd())
	return markdown.WalkSkipChildren, nil
}

func renderAutoLink(w *Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error) {
	if entering {
		l := n.(*markdown.AutoLink)
		w.WriteString(`<a href="`)
		writeDestination(w, l.URL)
		w.WriteString(`"`)
		w.WriteAttributes(n, LinkAttributeFilter)
		w.WriteString(`>`)
		w.WriteText(l.Label)
		w.WriteString("</a>")
	}
	return markdown.WalkSkipChildren, nil
}

func renderRawHTML(w *Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error) {
	if entering {
		if w.config.Unsafe {
			w.WriteString(n.(*markdown.RawHTML).Value)
		} else {
			w.WriteString(omitted)
		}
	}
	return markdown.WalkSkipChildren, nil
}

// plainText returns the text of the inline content of n without markup, as
// code spans and image descriptions render it, with the line breaks that w
// renders.
func (w *Writer) plainText(n markdown.Node) string {
	var b strings.Builder
	for c := range markdown.Children(n) {
		switch c := c.(type) {
		case *markdown.Text:
			b.WriteString(c.Value)
			if c.HardLineBreak() || c.SoftLineBreak() && w.keepsSoftLineBreak(c) {
				b.WriteString("\n")
			}
		case *markdown.AutoLink:
			b.WriteString(c.Label)
		case *markdown.RawHTML:
		default:
			b.WriteString(w.plainText(c))
		}
	}
	return b.String()
}
