package extension

import (
	"cmp"
	"slices"
	"strconv"
	"strings"

	"github.com/unaquatic/fluid/internal/markdown"
	"github.com/unaquatic/fluid/internal/markdown/html"
	"github.com/unaquatic/fluid/internal/markdown/md"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// footnote is an extension of footnotes.
type footnote struct {
	options []FootnoteOption
}

// NewFootnote returns an extension that adds footnotes, as PHP Markdown
// Extra defines them, configured by opts: the parser that
// [NewFootnoteBlockParser] returns, with priority 999, the parser that
// [NewFootnoteParser] returns, with priority 101, the transformer of
// documents that [NewFootnoteASTTransformer] returns, with priority 999,
// and the renderer that [NewFootnoteHTMLRenderer] returns, with priority
// 500.
func NewFootnote(opts ...FootnoteOption) Extension { return &footnote{options: opts} }

// Extend implements [md.Extender].
func (e *footnote) Extend(m md.Markdown) {
	m.Parser().AddOptions(parser.WithExtensions(e))
	m.Renderer().AddOptions(html.WithNodeRenderer(NewFootnoteHTMLRenderer(e.options...), 500))
}

// ExtendParser implements [parser.Extension].
func (e *footnote) ExtendParser(c *parser.Config) {
	c.Apply(
		parser.WithBlockParser(NewFootnoteBlockParser(), 999),
		parser.WithInlineParser(NewFootnoteParser(), 101),
		parser.WithASTTransformer(NewFootnoteASTTransformer(), 999),
	)
}

// A FootnoteOption configures how footnotes render.
type FootnoteOption func(c *FootnoteConfig)

// A FootnoteConfig holds how footnotes render. In its titles, classes, and
// backlink HTML, "^^" stands for the number of the footnote, and "%%" for
// the number of references to it.
type FootnoteConfig struct {
	// IDPrefix comes before the IDs of the elements of footnotes, and of
	// the references to them, such as "fn:1" and "fnref:1".
	IDPrefix string

	// IDPrefixFunction returns the prefix of the IDs of the element of
	// node n, if IDPrefix is empty.
	IDPrefixFunction func(n markdown.Node) string

	// LinkTitle is the title of the links of references, if any.
	LinkTitle string

	// BacklinkTitle is the title of the links back to references, if any.
	BacklinkTitle string

	// LinkClass is the class of the links of references.
	LinkClass string

	// BacklinkClass is the class of the links back to references.
	BacklinkClass string

	// BacklinkHTML is the content of the links back to references, as
	// HTML.
	BacklinkHTML string
}

// WithFootnoteIDPrefix sets the prefix of the IDs of footnotes and of the
// references to them.
func WithFootnoteIDPrefix(prefix string) FootnoteOption {
	return func(c *FootnoteConfig) { c.IDPrefix = prefix }
}

// WithFootnoteIDPrefixFunction sets the function that returns the prefix of
// the IDs of the element of a node, if no prefix is set.
func WithFootnoteIDPrefixFunction(f func(n markdown.Node) string) FootnoteOption {
	return func(c *FootnoteConfig) { c.IDPrefixFunction = f }
}

// WithFootnoteLinkTitle sets the title of the links of references.
func WithFootnoteLinkTitle(title string) FootnoteOption {
	return func(c *FootnoteConfig) { c.LinkTitle = title }
}

// WithFootnoteBacklinkTitle sets the title of the links back to
// references.
func WithFootnoteBacklinkTitle(title string) FootnoteOption {
	return func(c *FootnoteConfig) { c.BacklinkTitle = title }
}

// WithFootnoteLinkClass sets the class of the links of references, in place
// of "footnote-ref".
func WithFootnoteLinkClass(class string) FootnoteOption {
	return func(c *FootnoteConfig) { c.LinkClass = class }
}

// WithFootnoteBacklinkClass sets the class of the links back to references,
// in place of "footnote-backref".
func WithFootnoteBacklinkClass(class string) FootnoteOption {
	return func(c *FootnoteConfig) { c.BacklinkClass = class }
}

// WithFootnoteBacklinkHTML sets the content of the links back to
// references, as HTML, in place of "&#x21a9;&#xfe0e;", a hooked arrow.
func WithFootnoteBacklinkHTML(s string) FootnoteOption {
	return func(c *FootnoteConfig) { c.BacklinkHTML = s }
}

// footnotesKey keys the footnotes of a document in the values of the states
// of its parser.
type footnotesKey struct{}

// footnotes holds the footnotes of a document as it is parsed.
type footnotes struct {
	list  *markdown.FootnoteList        // nil until a definition closes
	byRef map[string]*markdown.Footnote // the first definition of each label

	// labelEnd holds the offset in the source of the end of the label of
	// each definition that is open.
	labelEnd map[*markdown.Footnote]int
}

// footnotesOf returns the footnotes of the document that s parses.
func footnotesOf(s *parser.BlockState) *footnotes {
	f, _ := s.Value(footnotesKey{}).(*footnotes)
	if f == nil {
		f = &footnotes{byRef: make(map[string]*markdown.Footnote), labelEnd: make(map[*markdown.Footnote]int)}
		s.SetValue(footnotesKey{}, f)
	}
	return f
}

// footnoteBlockParser parses footnote definitions.
type footnoteBlockParser struct{}

// NewFootnoteBlockParser returns the parser of footnote definitions: "[^", a
// label, and "]:" at the start of a line, and the blocks after them on the
// line, on the lines after it that are indented 4 columns, and on blank
// lines. A label is not blank, and holds no "[" or line ending, nor a "]"
// that a backslash does not escape. A definition in a footnote is none. As
// a definition closes, it moves to the document's
// [markdown.FootnoteList], which the first to close leaves in its place.
func NewFootnoteBlockParser() parser.BlockParser { return footnoteBlockParser{} }

// Trigger implements [parser.BlockParser].
func (footnoteBlockParser) Trigger() []byte { return []byte{'['} }

// Open implements [parser.BlockParser].
func (footnoteBlockParser) Open(parent markdown.Node, s *parser.BlockState) (markdown.Node, parser.State) {
	rest := s.Rest()
	n := footnoteLabel(rest)
	if n == 0 || n == len(rest) || rest[n] != ':' || isBlank(rest[2:n-1]) || inFootnote(parent) {
		return nil, parser.None
	}
	s.AdvanceNextNonspace()
	s.Advance(n + 1)
	fn := &markdown.Footnote{Ref: replaceNUL(rest[2 : n-1])}
	footnotesOf(s).labelEnd[fn] = s.LineOffset() + s.Offset()
	return fn, parser.HasChildren
}

// Continue implements [parser.BlockParser].
func (footnoteBlockParser) Continue(n markdown.Node, s *parser.BlockState) parser.State {
	switch {
	case s.Blank():
		s.AdvanceNextNonspace()
	case s.Indent() >= 4:
		s.AdvanceColumns(4)
	default:
		return parser.Close
	}
	return parser.Continue | parser.HasChildren
}

// Close implements [parser.BlockParser]. It ends the definition's span at
// its last block, or its label, and moves it to the document's list of
// footnotes. The list's span, until the definitions move to the end of the
// document, runs from the first definition through the last that directly
// follows it.
func (footnoteBlockParser) Close(n markdown.Node, s *parser.BlockState) {
	fn := n.(*markdown.Footnote)
	f := footnotesOf(s)
	sp := fn.SourceSpan()
	if last := fn.LastChild(); last != nil {
		sp.End = last.SourceSpan().End
	} else {
		sp.End = f.labelEnd[fn]
	}
	delete(f.labelEnd, fn)
	fn.SetSourceSpan(sp)

	parent := fn.Parent()
	switch {
	case f.list == nil:
		f.list = &markdown.FootnoteList{}
		f.list.SetSourceSpan(sp)
		parent.InsertBefore(parent, fn, f.list)
	case fn.PreviousSibling() == markdown.Node(f.list):
		ls := f.list.SourceSpan()
		ls.End = max(ls.End, sp.End)
		f.list.SetSourceSpan(ls)
	}
	f.list.AppendChild(f.list, fn)
	if _, ok := f.byRef[fn.Ref]; !ok {
		f.byRef[fn.Ref] = fn
	}
}

// CanInterruptParagraph implements [parser.BlockParser].
func (footnoteBlockParser) CanInterruptParagraph() bool { return true }

// CanAcceptIndentedLine implements [parser.BlockParser].
func (footnoteBlockParser) CanAcceptIndentedLine() bool { return false }

// inFootnote reports whether n is in a footnote, or is one.
func inFootnote(n markdown.Node) bool {
	for ; n != nil; n = n.Parent() {
		if _, ok := n.(*markdown.Footnote); ok {
			return true
		}
	}
	return false
}

// footnoteLabel returns the length of the "[^", label, and "]" at the start
// of s, or 0 if there are none: a "]" that a backslash does not escape ends
// the label, and a "[" or a line ending before it, or the end of s, makes
// none.
func footnoteLabel(s string) int {
	if !strings.HasPrefix(s, "[^") {
		return 0
	}
	for i := 2; i < len(s); i++ {
		switch s[i] {
		case '\\':
			if i+1 < len(s) && isASCIIPunct(s[i+1]) {
				i++
			}
		case ']':
			return i + 1
		case '[', '\n':
			return 0
		}
	}
	return 0
}

// footnoteParser parses references to footnotes.
type footnoteParser struct{}

// NewFootnoteParser returns the parser of references to footnotes: "[^", the
// label of a footnote that the document defines, and "]". A "!" before
// them stays text, as an image cannot start there. The parser numbers each
// footnote as the document first refers to it.
func NewFootnoteParser() parser.InlineParser { return footnoteParser{} }

// Trigger implements [parser.InlineParser].
func (footnoteParser) Trigger() []byte { return []byte{'!', '['} }

// Parse implements [parser.InlineParser].
func (footnoteParser) Parse(parent markdown.Node, s *parser.InlineState) (markdown.Node, int) {
	f, _ := s.DocumentValue(footnotesKey{}).(*footnotes)
	if f == nil || f.list == nil {
		return nil, 0
	}
	rest := s.Rest()
	bang := 0
	if rest[0] == '!' {
		bang = 1
	}
	n := footnoteLabel(rest[bang:])
	if n == 0 {
		return nil, 0
	}
	fn := f.byRef[rest[bang+2:bang+n-1]]
	if fn == nil {
		return nil, 0
	}
	if fn.Index == 0 {
		f.list.Count++
		fn.Index = f.list.Count
	}
	if bang > 0 {
		parent.AppendChild(parent, markdown.NewText("!"))
	}
	return &markdown.FootnoteLink{Index: fn.Index}, bang + n
}

// footnoteASTTransformer finishes the footnotes of documents.
type footnoteASTTransformer struct{}

// NewFootnoteASTTransformer returns the transformer of documents that
// finishes their footnotes. It counts the references to each footnote,
// removes the footnotes that none refers to, adds a
// [markdown.FootnoteBacklink] to each reference to the end of the last
// paragraph of each footnote, or of the footnote if its last block is not a
// paragraph, orders them by their numbers, and moves their list to the end
// of the document, or removes it if it holds none.
func NewFootnoteASTTransformer() parser.ASTTransformer { return footnoteASTTransformer{} }

// Transform implements [parser.ASTTransformer].
func (footnoteASTTransformer) Transform(doc *markdown.Document) {
	var list *markdown.FootnoteList
	var links []*markdown.FootnoteLink
	_ = markdown.Walk(doc, func(n markdown.Node, entering bool) (markdown.WalkStatus, error) {
		switch n := n.(type) {
		case *markdown.FootnoteList:
			if entering && list == nil {
				list = n
			}
		case *markdown.FootnoteLink:
			if entering {
				links = append(links, n)
			}
		}
		return markdown.WalkContinue, nil
	})
	if list == nil {
		return
	}

	refs := make(map[int]int) // the references to each footnote, by number
	for _, l := range links {
		refs[l.Index]++
	}
	before := make(map[int]int)
	for _, l := range links {
		l.RefCount = refs[l.Index]
		l.RefIndex = before[l.Index]
		before[l.Index]++
	}

	var notes []*markdown.Footnote
	for _, c := range slices.Collect(markdown.Children(list)) {
		list.RemoveChild(list, c)
		fn, ok := c.(*markdown.Footnote)
		if !ok || fn.Index == 0 {
			continue
		}
		var container markdown.Node = fn
		if p, ok := fn.LastChild().(*markdown.Paragraph); ok {
			container = p
		}
		for i := range refs[fn.Index] {
			container.AppendChild(container, &markdown.FootnoteBacklink{Index: fn.Index, RefCount: refs[fn.Index], RefIndex: i})
		}
		notes = append(notes, fn)
	}
	slices.SortFunc(notes, func(a, b *markdown.Footnote) int { return cmp.Compare(a.Index, b.Index) })
	for _, fn := range notes {
		list.AppendChild(list, fn)
	}

	if parent := list.Parent(); list.Count == 0 {
		parent.RemoveChild(parent, list)
		return
	}
	end := doc.SourceSpan().End
	list.SetSourceSpan(markdown.Span{Start: end, End: end})
	doc.AppendChild(doc, list)
}

// footnoteHTMLRenderer renders footnotes.
type footnoteHTMLRenderer struct {
	config FootnoteConfig
}

// NewFootnoteHTMLRenderer returns the renderer of footnotes and of the
// references to them, configured by opts. A reference renders a sup
// element with a link to its footnote. The list of footnotes renders a div
// element with an hr element and an ol element, in which each footnote
// renders an li element.
func NewFootnoteHTMLRenderer(opts ...FootnoteOption) html.NodeRenderer {
	r := &footnoteHTMLRenderer{config: FootnoteConfig{
		LinkClass:     "footnote-ref",
		BacklinkClass: "footnote-backref",
		BacklinkHTML:  "&#x21a9;&#xfe0e;",
	}}
	for _, opt := range opts {
		opt(&r.config)
	}
	return r
}

// RegisterFuncs implements [html.NodeRenderer].
func (r *footnoteHTMLRenderer) RegisterFuncs(reg html.NodeRendererFuncRegisterer) {
	reg.Register(markdown.KindFootnoteLink, r.renderFootnoteLink)
	reg.Register(markdown.KindFootnoteBacklink, r.renderFootnoteBacklink)
	reg.Register(markdown.KindFootnote, r.renderFootnote)
	reg.Register(markdown.KindFootnoteList, r.renderFootnoteList)
}

func (r *footnoteHTMLRenderer) renderFootnoteLink(w *html.Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error) {
	if entering {
		l := n.(*markdown.FootnoteLink)
		index := strconv.Itoa(l.Index)
		w.WriteString(`<sup id="`)
		r.writeRefID(w, n, l.Index, l.RefIndex)
		w.WriteString(`"><a href="#`)
		r.writeID(w, n, l.Index)
		w.WriteString(`" class="`)
		w.WriteText(footnoteTemplate(r.config.LinkClass, l.Index, l.RefCount))
		if r.config.LinkTitle != "" {
			w.WriteString(`" title="`)
			w.WriteText(footnoteTemplate(r.config.LinkTitle, l.Index, l.RefCount))
		}
		w.WriteString(`" role="doc-noteref">` + index + "</a></sup>")
	}
	return markdown.WalkContinue, nil
}

func (r *footnoteHTMLRenderer) renderFootnoteBacklink(w *html.Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error) {
	if entering {
		l := n.(*markdown.FootnoteBacklink)
		w.WriteString(`&#160;<a href="#`)
		r.writeRefID(w, n, l.Index, l.RefIndex)
		w.WriteString(`" class="`)
		w.WriteText(footnoteTemplate(r.config.BacklinkClass, l.Index, l.RefCount))
		if r.config.BacklinkTitle != "" {
			w.WriteString(`" title="`)
			w.WriteText(footnoteTemplate(r.config.BacklinkTitle, l.Index, l.RefCount))
		}
		w.WriteString(`" role="doc-backlink">`)
		w.WriteString(footnoteTemplate(r.config.BacklinkHTML, l.Index, l.RefCount))
		w.WriteString("</a>")
	}
	return markdown.WalkContinue, nil
}

func (r *footnoteHTMLRenderer) renderFootnote(w *html.Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error) {
	if !entering {
		w.WriteString("</li>\n")
		return markdown.WalkContinue, nil
	}
	w.WriteString(`<li id="`)
	r.writeID(w, n, n.(*markdown.Footnote).Index)
	w.WriteString(`"`)
	w.WriteAttributes(n, html.ListItemAttributeFilter)
	w.WriteString(">\n")
	return markdown.WalkContinue, nil
}

func (r *footnoteHTMLRenderer) renderFootnoteList(w *html.Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error) {
	if !entering {
		w.WriteString("</ol>\n</div>\n")
		return markdown.WalkContinue, nil
	}
	w.CR()
	w.WriteString(`<div class="footnotes" role="doc-endnotes"`)
	w.WriteAttributes(n, html.GlobalAttributeFilter)
	hr := "<hr>"
	if w.Config().XHTML {
		hr = "<hr />"
	}
	w.WriteString(">\n" + hr + "\n<ol>\n")
	return markdown.WalkContinue, nil
}

// writeID writes the ID of footnote index, for the element of node n.
func (r *footnoteHTMLRenderer) writeID(w *html.Writer, n markdown.Node, index int) {
	w.WriteText(r.idPrefix(n))
	w.WriteString("fn:" + strconv.Itoa(index))
}

// writeRefID writes the ID of the reference to footnote index that
// refIndex references come before, for the element of node n.
func (r *footnoteHTMLRenderer) writeRefID(w *html.Writer, n markdown.Node, index, refIndex int) {
	w.WriteText(r.idPrefix(n))
	w.WriteString("fnref")
	if refIndex > 0 {
		w.WriteString(strconv.Itoa(refIndex))
	}
	w.WriteString(":" + strconv.Itoa(index))
}

// idPrefix returns the prefix of the ID of the element of node n.
func (r *footnoteHTMLRenderer) idPrefix(n markdown.Node) string {
	if r.config.IDPrefix == "" && r.config.IDPrefixFunction != nil {
		return r.config.IDPrefixFunction(n)
	}
	return r.config.IDPrefix
}

// footnoteTemplate returns s with each "^^" replaced by index and each "%%"
// by refCount.
func footnoteTemplate(s string, index, refCount int) string {
	if !strings.Contains(s, "^^") && !strings.Contains(s, "%%") {
		return s
	}
	s = strings.ReplaceAll(s, "^^", strconv.Itoa(index))
	return strings.ReplaceAll(s, "%%", strconv.Itoa(refCount))
}
