package extension

import (
	"cmp"
	"slices"
	"strings"

	"github.com/unaquatic/fluid/internal/markdown/ast"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// footnote is the extension Footnote.
type footnote struct{}

// ExtendParser implements [parser.Extension].
func (footnote) ExtendParser(c *parser.Config) {
	c.Apply(
		parser.WithBlockParser(NewFootnoteBlockParser(), 999),
		parser.WithInlineParser(NewFootnoteParser(), 101),
		parser.WithASTTransformer(NewFootnoteASTTransformer(), 999),
	)
}

// footnotesKey keys the footnotes of a document in the values of the states
// of its parser.
type footnotesKey struct{}

// footnotes holds the footnotes of a document as it is parsed.
type footnotes struct {
	list  *ast.FootnoteList        // nil until a definition closes
	byRef map[string]*ast.Footnote // the first definition of each label

	// labelEnd holds the offset in the source of the end of the label of
	// each definition that is open.
	labelEnd map[*ast.Footnote]int
}

// footnotesOf returns the footnotes of the document that s parses.
func footnotesOf(s *parser.BlockState) *footnotes {
	f, _ := s.Value(footnotesKey{}).(*footnotes)
	if f == nil {
		f = &footnotes{byRef: make(map[string]*ast.Footnote), labelEnd: make(map[*ast.Footnote]int)}
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
// that a backslash does not escape. A definition in a footnote is none,
// unlike in goldmark, which opens one there and loses its text. As a
// definition closes, it moves to the document's [ast.FootnoteList],
// which the first to close leaves in its place.
func NewFootnoteBlockParser() parser.BlockParser { return footnoteBlockParser{} }

// Trigger implements [parser.BlockParser].
func (footnoteBlockParser) Trigger() []byte { return []byte{'['} }

// Open implements [parser.BlockParser].
func (footnoteBlockParser) Open(parent ast.Node, s *parser.BlockState) (ast.Node, parser.State) {
	rest := s.Rest()
	n := footnoteLabel(rest)
	if n == 0 || n == len(rest) || rest[n] != ':' || isBlank(rest[2:n-1]) || inFootnote(parent) {
		return nil, parser.None
	}
	s.AdvanceNextNonspace()
	s.Advance(n + 1)
	fn := &ast.Footnote{Ref: replaceNUL(rest[2 : n-1])}
	footnotesOf(s).labelEnd[fn] = s.LineOffset() + s.Offset()
	return fn, parser.HasChildren
}

// Continue implements [parser.BlockParser].
func (footnoteBlockParser) Continue(n ast.Node, s *parser.BlockState) parser.State {
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
func (footnoteBlockParser) Close(n ast.Node, s *parser.BlockState) {
	fn := n.(*ast.Footnote)
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
		f.list = &ast.FootnoteList{}
		f.list.SetSourceSpan(sp)
		parent.InsertBefore(parent, fn, f.list)
	case fn.PreviousSibling() == ast.Node(f.list):
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
func inFootnote(n ast.Node) bool {
	for ; n != nil; n = n.Parent() {
		if _, ok := n.(*ast.Footnote); ok {
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
func (footnoteParser) Parse(parent ast.Node, s *parser.InlineState) (ast.Node, int) {
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
		parent.AppendChild(parent, ast.NewText("!"))
	}
	return &ast.FootnoteLink{Index: fn.Index}, bang + n
}

// footnoteASTTransformer finishes the footnotes of documents.
type footnoteASTTransformer struct{}

// NewFootnoteASTTransformer returns the transformer of documents that
// finishes their footnotes. It counts the references to each footnote,
// removes the footnotes that none refers to, adds a
// [ast.FootnoteBacklink] to each reference to the end of the last
// paragraph of each footnote, or of the footnote if its last block is not a
// paragraph, orders them by their numbers, and moves their list to the end
// of the document, or removes it if it holds none.
func NewFootnoteASTTransformer() parser.ASTTransformer { return footnoteASTTransformer{} }

// Transform implements [parser.ASTTransformer].
func (footnoteASTTransformer) Transform(doc *ast.Document) {
	var list *ast.FootnoteList
	var links []*ast.FootnoteLink
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch n := n.(type) {
		case *ast.FootnoteList:
			if entering && list == nil {
				list = n
			}
		case *ast.FootnoteLink:
			if entering {
				links = append(links, n)
			}
		}
		return ast.WalkContinue, nil
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

	var notes []*ast.Footnote
	for _, c := range slices.Collect(ast.Children(list)) {
		list.RemoveChild(list, c)
		fn, ok := c.(*ast.Footnote)
		if !ok || fn.Index == 0 {
			continue
		}
		var container ast.Node = fn
		if p, ok := fn.LastChild().(*ast.Paragraph); ok {
			container = p
		}
		for i := range refs[fn.Index] {
			container.AppendChild(container, &ast.FootnoteBacklink{Index: fn.Index, RefCount: refs[fn.Index], RefIndex: i})
		}
		notes = append(notes, fn)
	}
	slices.SortFunc(notes, func(a, b *ast.Footnote) int { return cmp.Compare(a.Index, b.Index) })
	for _, fn := range notes {
		list.AppendChild(list, fn)
	}

	if parent := list.Parent(); list.Count == 0 {
		parent.RemoveChild(parent, list)
		return
	}
	end := doc.SourceSpan().End
	list.SetSourceSpan(ast.Span{Start: end, End: end})
	doc.AppendChild(doc, list)
}
