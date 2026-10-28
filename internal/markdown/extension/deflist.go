package extension

import (
	"cmp"
	"slices"
	"strings"

	"github.com/unaquatic/fluid/internal/markdown/ast"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// definitionList is the extension DefinitionList.
type definitionList struct{}

// ExtendParser implements [parser.Extension].
func (definitionList) ExtendParser(c *parser.Config) {
	c.Apply(
		parser.WithBlockParser(NewDefinitionListParser(), 101),
		parser.WithBlockParser(NewDefinitionDescriptionParser(), 102),
	)
}

// definitionListParser parses definition lists.
type definitionListParser struct{}

// NewDefinitionListParser returns the parser of definition lists, which
// start at a line that starts with ":" and a space or tab, a description,
// after a paragraph, each of whose lines becomes a term. Such a line also
// adds a description to a list that is the block before it, and adds the
// lines of a paragraph after a list to it as terms. A description's
// content starts after the spaces after its ":", or after 5 columns of
// them if there are 8 or more, and lines after its first are indented as
// far, or are blank. The parser that [NewDefinitionDescriptionParser]
// returns opens the descriptions.
//
// A link reference definition between the paragraph or the list and the
// ":" parts them, so that the ":" starts no description, as goldmark parts
// a paragraph from a new list; goldmark makes such a definition a term of a
// list before it.
func NewDefinitionListParser() parser.BlockParser { return definitionListParser{} }

// Trigger implements [parser.BlockParser].
func (definitionListParser) Trigger() []byte { return []byte{':'} }

// Open implements [parser.BlockParser]. It opens the list, again if it is
// the block before, with the terms, and leaves the ":" to the parser of
// descriptions.
func (p definitionListParser) Open(parent ast.Node, s *parser.BlockState) (ast.Node, parser.State) {
	if _, ok := parent.(*ast.DefinitionList); ok {
		return nil, parser.None
	}
	offset, ok := descriptionOffset(s)
	if !ok {
		return nil, parser.None
	}
	var list *ast.DefinitionList
	if para, ok := parent.(*ast.Paragraph); ok {
		// The line would continue the paragraph, which closes, and goes
		// if it held only link reference definitions.
		s.Close(para, s.LineNumber()-1)
		if para.Parent() == nil {
			return nil, parser.None
		}
		list = addTerms(para, s.Source())
	} else {
		last := parent.LastChild()
		var end int // where the block before ends, and others can start
		switch last := last.(type) {
		case *ast.Paragraph:
			end = last.SourceSpan().End
		case *ast.DefinitionList:
			// The line does not continue the list, which closes, if it
			// is open. Blocks after it start at the line that closed it.
			s.CloseUnmatched()
			end = definitionListsOf(s).closedAt[last]
		default:
			return nil, parser.None
		}
		if linkDefBetween(parent, end, s.LineOffset()+s.NextNonspace()) {
			return nil, parser.None
		}
		switch last := last.(type) {
		case *ast.Paragraph:
			list = addTerms(last, s.Source())
		case *ast.DefinitionList:
			list = last
		}
	}
	list.Offset = offset
	if !s.Reopen(list, p) {
		return nil, parser.None
	}
	return list, parser.HasChildren
}

// descriptionOffset returns the column, from the current one, at which the
// content of a description that starts at the current position of s
// starts, and reports whether one starts there: a ":" that is not
// indented, and at least one column of spaces and tabs.
func descriptionOffset(s *parser.BlockState) (int, bool) {
	rest := s.Rest()
	if s.Indent() != 0 || !strings.HasPrefix(rest, ":") {
		return 0, false
	}
	col := s.Column() + 1
	w := 0
	for i := 1; i < len(rest) && (rest[i] == ' ' || rest[i] == '\t'); i++ {
		n := 1
		if rest[i] == '\t' {
			n = 4 - col%4
		}
		col += n
		w += n
	}
	switch {
	case w == 0:
		return 0, false
	case w >= 8:
		w = 5 // the content starts with indented code
	}
	return 1 + w, true
}

// linkDefBetween reports whether a link reference definition of the
// document of n starts between offsets start and end, where it parts the
// block before from a description.
func linkDefBetween(n ast.Node, start, end int) bool {
	for n.Parent() != nil {
		n = n.Parent()
	}
	doc, ok := n.(*ast.Document)
	if !ok {
		return false
	}
	// The definitions are in the order in which they appear.
	i, _ := slices.BinarySearchFunc(doc.Links, start, func(d *ast.LinkDef, start int) int { return cmp.Compare(d.Start, start) })
	return i < len(doc.Links) && doc.Links[i].Start < end
}

// addTerms makes the lines of paragraph p, which has closed, terms of the
// definition list before it, or of a new one in its place, and returns the
// list. src is the document's source.
func addTerms(p *ast.Paragraph, src string) *ast.DefinitionList {
	parent := p.Parent()
	list, ok := p.PreviousSibling().(*ast.DefinitionList)
	if ok {
		parent.RemoveChild(parent, p)
	} else {
		list = &ast.DefinitionList{}
		start := p.SourceSpan().Start
		list.SetSourceSpan(ast.Span{Start: start, End: start})
		parent.ReplaceChild(parent, p, list)
	}
	for _, l := range p.Lines() {
		start, end := trimmedLine(src, l.Offset)
		t := &ast.DefinitionTerm{}
		t.SetSourceSpan(ast.Span{Start: start, End: end})
		t.SetLines([]ast.Line{{Text: strings.Trim(l.Text, " \t"), Offset: start}})
		list.AppendChild(list, t)
	}
	return list
}

// trimmedLine returns the range of the line of src that starts at offset,
// without its line ending and the spaces and tabs at each end.
func trimmedLine(src string, offset int) (start, end int) {
	end = len(src)
	if i := strings.IndexAny(src[offset:], "\r\n"); i >= 0 {
		end = offset + i
	}
	start = offset
	for start < end && (src[start] == ' ' || src[start] == '\t') {
		start++
	}
	for end > start && (src[end-1] == ' ' || src[end-1] == '\t') {
		end--
	}
	return start, end
}

// definitionListsKey keys the state of the definition lists of a document
// in the values of a [parser.BlockState].
type definitionListsKey struct{}

// definitionLists is the state of the definition lists of a document as it
// is parsed.
type definitionLists struct {
	// blank holds the number of the last blank line that each list took.
	blank map[*ast.DefinitionList]int

	// closedAt holds the offset in the source of the line at which each
	// list that has closed closed.
	closedAt map[*ast.DefinitionList]int
}

// definitionListsOf returns the state of the definition lists of the
// document that s parses.
func definitionListsOf(s *parser.BlockState) *definitionLists {
	d, _ := s.Value(definitionListsKey{}).(*definitionLists)
	if d == nil {
		d = &definitionLists{blank: make(map[*ast.DefinitionList]int), closedAt: make(map[*ast.DefinitionList]int)}
		s.SetValue(definitionListsKey{}, d)
	}
	return d
}

// Continue implements [parser.BlockParser]. A line continues a definition
// list if it is blank or indented to the content of its last description.
func (definitionListParser) Continue(n ast.Node, s *parser.BlockState) parser.State {
	l := n.(*ast.DefinitionList)
	switch {
	case s.Blank():
		definitionListsOf(s).blank[l] = s.LineNumber()
		s.AdvanceNextNonspace()
	case s.Indent() >= l.Offset:
		s.AdvanceColumns(l.Offset)
	default:
		return parser.Close
	}
	return parser.Continue | parser.HasChildren
}

// Close implements [parser.BlockParser]. A list ends with its last
// description.
func (definitionListParser) Close(n ast.Node, s *parser.BlockState) {
	definitionListsOf(s).closedAt[n.(*ast.DefinitionList)] = s.LineOffset()
	if last := n.LastChild(); last != nil {
		sp := n.SourceSpan()
		sp.End = last.SourceSpan().End
		n.SetSourceSpan(sp)
	}
}

// CanInterruptParagraph implements [parser.BlockParser].
func (definitionListParser) CanInterruptParagraph() bool { return true }

// CanAcceptIndentedLine implements [parser.BlockParser].
func (definitionListParser) CanAcceptIndentedLine() bool { return false }

// definitionDescriptionParser parses the descriptions of definition lists.
type definitionDescriptionParser struct{}

// NewDefinitionDescriptionParser returns the parser of the descriptions of
// definition lists, which the parser that [NewDefinitionListParser]
// returns opens the lists of. A description is tight if the line before it
// is not blank, and no blank line parts its blocks, as a list is tight in
// CommonMark. goldmark leaves out the blank lines between blocks, and only
// the first paragraph of a tight description renders without paragraph
// tags.
func NewDefinitionDescriptionParser() parser.BlockParser { return definitionDescriptionParser{} }

// Trigger implements [parser.BlockParser].
func (definitionDescriptionParser) Trigger() []byte { return []byte{':'} }

// Open implements [parser.BlockParser].
func (definitionDescriptionParser) Open(parent ast.Node, s *parser.BlockState) (ast.Node, parser.State) {
	list, ok := parent.(*ast.DefinitionList)
	if !ok || s.Indent() != 0 || !strings.HasPrefix(s.Rest(), ":") {
		return nil, parser.None
	}
	d := &ast.DefinitionDescription{IsTight: true}
	switch prev := list.LastChild().(type) {
	case *ast.DefinitionTerm:
		// The paragraph of the terms ended at the line before, unless a
		// blank line closed it.
		d.IsTight = !blankBetween(s.Source(), prev.SourceSpan().End, s.LineOffset()+s.NextNonspace())
	case *ast.DefinitionDescription:
		// The description before may have taken the blank line, as code
		// does.
		d.IsTight = definitionListsOf(s).blank[list] != s.LineNumber()-1
	}
	s.AdvanceNextNonspace()
	s.Advance(1)
	s.AdvanceColumns(list.Offset - 1)
	return d, parser.HasChildren
}

// Continue implements [parser.BlockParser]; the list decides which lines
// continue its descriptions.
func (definitionDescriptionParser) Continue(ast.Node, *parser.BlockState) parser.State {
	return parser.Continue | parser.HasChildren
}

// Close implements [parser.BlockParser]. A description ends with its last
// block, or its ":" if it has none, and is loose if a blank line parts its
// blocks.
func (definitionDescriptionParser) Close(n ast.Node, s *parser.BlockState) {
	d := n.(*ast.DefinitionDescription)
	sp := d.SourceSpan()
	sp.End = sp.Start + 1
	if last := d.LastChild(); last != nil {
		sp.End = last.SourceSpan().End
	}
	d.SetSourceSpan(sp)
	for c := d.FirstChild(); c != nil && d.IsTight; c = c.NextSibling() {
		if next := c.NextSibling(); next != nil && blankBetween(s.Source(), c.SourceSpan().End, next.SourceSpan().Start) {
			d.IsTight = false
		}
	}
}

// CanInterruptParagraph implements [parser.BlockParser].
func (definitionDescriptionParser) CanInterruptParagraph() bool { return true }

// CanAcceptIndentedLine implements [parser.BlockParser].
func (definitionDescriptionParser) CanAcceptIndentedLine() bool { return false }

// blankBetween reports whether a whole line lies between offsets start and
// end of src, the end of a block and the start of the next: whether two
// line endings or more do.
func blankBetween(src string, start, end int) bool {
	n := 0
	for i := start; i < end; i++ {
		switch src[i] {
		case '\r':
			if i+1 < end && src[i+1] == '\n' {
				i++
			}
			fallthrough
		case '\n':
			if n++; n == 2 {
				return true
			}
		}
	}
	return false
}
