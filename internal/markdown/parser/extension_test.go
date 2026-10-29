package parser_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/unaquatic/fluid/internal/markdown/ast"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// Kinds of the nodes of the extension that TestExtension parses with.
var (
	kindNote    = ast.NewNodeKind("Note")
	kindMention = ast.NewNodeKind("Mention")
)

// note is a container between a "::: note" line and a ":::" line.
type note struct {
	ast.BaseBlock
}

func (*note) Kind() ast.NodeKind { return kindNote }

// mention is "@" and a name.
type mention struct {
	ast.BaseInline
	Name string
}

func (*mention) Kind() ast.NodeKind { return kindMention }

// notesAndMentions adds notes and mentions to a parser.
type notesAndMentions struct{}

func (notesAndMentions) ExtendParser(c *parser.Config) {
	c.Apply(
		parser.WithBlockParser(notes{}, 50),
		parser.WithInlineParser(mentions{}, 600),
	)
}

type notes struct{}

func (notes) Trigger() []byte { return []byte{':'} }

func (notes) Open(parent ast.Node, s *parser.BlockState) (ast.Node, parser.State) {
	if s.Rest() != "::: note" {
		return nil, parser.None
	}
	s.Advance(len(s.Line()) - s.Offset())
	return &note{}, parser.HasChildren
}

func (notes) Continue(n ast.Node, s *parser.BlockState) parser.State {
	if s.Rest() == ":::" {
		s.Close(n, s.LineNumber())
		return parser.Close
	}
	return parser.Continue | parser.HasChildren
}

func (notes) Close(ast.Node, *parser.BlockState) {}
func (notes) CanInterruptParagraph() bool        { return true }
func (notes) CanAcceptIndentedLine() bool        { return false }

type mentions struct{}

func (mentions) Trigger() []byte { return []byte{'@'} }

func (mentions) Parse(parent ast.Node, s *parser.InlineState) (ast.Node, int) {
	r := s.Rest()
	n := 1
	for n < len(r) && 'a' <= r[n] && r[n] <= 'z' {
		n++
	}
	if n == 1 {
		return nil, 0
	}
	return &mention{Name: r[1:n]}, n
}

// dump returns the tree of n, whose document's source is src, as
// ast.Dump writes it.
func dump(t testing.TB, n ast.Node, src string) string {
	t.Helper()
	var b strings.Builder
	if err := ast.Dump(&b, n, src); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// checkTree checks that src parses with p to the tree want, as dump writes
// it.
func checkTree(t *testing.T, p parser.Parser, src, want string) {
	t.Helper()
	if got := dump(t, p.Parse(src), src); got != want {
		t.Errorf("tree of %q:\n%s\nwant:\n%s", src, got, want)
	}
}

// TestExtension checks that an extension can add a container block that
// closes at a line of its own, and inline content, of its own kinds, and
// that the blocks open in the container end at the line before.
func TestExtension(t *testing.T) {
	const src = "::: note\n> hi @bob\n:::\nafter @ 1"
	checkTree(t, parser.NewParser(parser.WithExtensions(notesAndMentions{})), src, `Document [0:32]
  Note [0:22] "::: note\n> hi @bob\n:::"
    Blockquote [9:18] "> hi @bob"
      Paragraph [11:18] "hi @bob"
        Text "hi "
        Mention Name="bob"
  Paragraph [23:32] "after @ 1"
    Text "after @ 1"
`)
}

// reopener opens the first note in the tip again at a line "+++", if the
// note is the tip's last child.
type reopener struct{}

func (reopener) Trigger() []byte { return []byte{'+'} }

func (reopener) Open(parent ast.Node, s *parser.BlockState) (ast.Node, parser.State) {
	if s.Rest() != "+++" {
		return nil, parser.None
	}
	for c := range ast.Children(parent) {
		if n, ok := c.(*note); ok && s.Reopen(n, notes{}) {
			s.Advance(len(s.Line()) - s.Offset())
			return n, parser.HasChildren
		}
	}
	return nil, parser.None
}

func (reopener) Continue(ast.Node, *parser.BlockState) parser.State { return parser.Close }
func (reopener) Close(ast.Node, *parser.BlockState)                 {}
func (reopener) CanInterruptParagraph() bool                        { return true }
func (reopener) CanAcceptIndentedLine() bool                        { return false }

// TestReopen checks that a block parser can open a block that closed again,
// if it is the last block, and that its span then ends at its new end, and
// cannot if a block follows it.
func TestReopen(t *testing.T) {
	const src = "::: note\nx\n:::\n+++\ny\n:::\n\npara\n\n+++"
	checkTree(t, parser.NewParser(parser.WithExtensions(notesAndMentions{}), parser.WithBlockParser(reopener{}, 60)), src, `Document [0:35]
  Note [0:24] "::: note\nx\n:::\n+++\ny\n:::"
    Paragraph [9:10] "x"
      Text "x"
    Paragraph [19:20] "y"
      Text "y"
  Paragraph [26:30] "para"
    Text "para"
  Paragraph [32:35] "+++"
    Text "+++"
`)
}

// closer closes the paragraph that a line "%" and text would continue, and
// opens nothing.
type closer struct{}

func (closer) Trigger() []byte { return []byte{'%'} }

func (closer) Open(parent ast.Node, s *parser.BlockState) (ast.Node, parser.State) {
	if p, ok := parent.(*ast.Paragraph); ok {
		s.Close(p, s.LineNumber()-1)
	}
	return nil, parser.None
}

func (closer) Continue(ast.Node, *parser.BlockState) parser.State { return parser.Close }
func (closer) Close(ast.Node, *parser.BlockState)                 {}
func (closer) CanInterruptParagraph() bool                        { return true }
func (closer) CanAcceptIndentedLine() bool                        { return false }

// TestCloseContainer checks that a line whose block parser closes the
// paragraph that it would continue, and opens nothing, starts a paragraph
// of its own, as the block parsers try it again in the tip.
func TestCloseContainer(t *testing.T) {
	const src = "> a\n> %b\n> - c"
	checkTree(t, parser.NewParser(parser.WithBlockParser(closer{}, 50)), src, `Document [0:14]
  Blockquote [0:14] "> a\n> %b\n> - c"
    Paragraph [2:3] "a"
      Text "a"
    Paragraph [6:8] "%b"
      Text "%b"
    List Marker='-' IsTight=true [11:14] "- c"
      ListItem Offset=2 [11:14] "- c"
        Paragraph [13:14] "c"
          Text "c"
`)
}

// bangs opens a one-line block at a line that starts with "- !".
type bangs struct{}

func (bangs) Trigger() []byte { return []byte{'-'} }

func (bangs) Open(parent ast.Node, s *parser.BlockState) (ast.Node, parser.State) {
	if !strings.HasPrefix(s.Rest(), "- !") {
		return nil, parser.None
	}
	s.Advance(len(s.Line()) - s.Offset())
	return &ast.ThematicBreak{}, parser.NoChildren
}

func (bangs) Continue(ast.Node, *parser.BlockState) parser.State { return parser.Close }
func (bangs) Close(ast.Node, *parser.BlockState)                 {}
func (bangs) CanInterruptParagraph() bool                        { return true }
func (bangs) CanAcceptIndentedLine() bool                        { return false }

// TestPriority checks that a block parser's priority places it among the
// built-in ones: before list items at 650, which come at 700, and after
// them at 750.
func TestPriority(t *testing.T) {
	for _, tt := range []struct {
		priority int
		want     ast.NodeKind
	}{
		{650, ast.KindThematicBreak},
		{750, ast.KindList},
	} {
		p := parser.NewParser(parser.WithBlockParser(bangs{}, tt.priority))
		if got := p.Parse("- !x").FirstChild().Kind(); got != tt.want {
			t.Errorf("priority %d: block %s, want %s", tt.priority, got, tt.want)
		}
	}
}

// namesKey keys the names that a "name:" paragraph defines, and countKey
// the number of mentions, in the values of a document.
type (
	namesKey struct{}
	countKey struct{}
)

// names takes each paragraph "name: " and a name, which it defines.
type names struct{}

func (names) Transform(p *ast.Paragraph, s *parser.BlockState) {
	name, ok := strings.CutPrefix(p.Lines()[0].Text, "name: ")
	if !ok {
		return
	}
	defined, _ := s.Value(namesKey{}).(map[string]bool)
	if defined == nil {
		defined = make(map[string]bool)
		s.SetValue(namesKey{}, defined)
	}
	defined[name] = true
	p.Parent().RemoveChild(p.Parent(), p)
}

// knownMentions parses mentions of the names that are defined, numbered in
// the order in which the document mentions them.
type knownMentions struct{}

func (knownMentions) Trigger() []byte { return []byte{'@'} }

func (knownMentions) Parse(parent ast.Node, s *parser.InlineState) (ast.Node, int) {
	m, n := mentions{}.Parse(parent, s)
	defined, _ := s.DocumentValue(namesKey{}).(map[string]bool)
	if m == nil || !defined[m.(*mention).Name] {
		return nil, 0
	}
	count, _ := s.DocumentValue(countKey{}).(int)
	s.SetDocumentValue(countKey{}, count+1)
	m.(*mention).Name += strconv.Itoa(count + 1)
	return m, n
}

// TestDocumentValues checks that inline parsers see the values that block
// parsers store for the document, after its blocks, and that values they
// store last through the document's inline content.
func TestDocumentValues(t *testing.T) {
	const src = "@ann @bob\n\n@bob\n\nname: bob"
	p := parser.NewParser(parser.WithParagraphTransformer(names{}, 100), parser.WithInlineParser(knownMentions{}, 600))
	checkTree(t, p, src, `Document [0:26]
  Paragraph [0:9] "@ann @bob"
    Text "@ann "
    Mention Name="bob1"
  Paragraph [11:15] "@bob"
    Mention Name="bob2"
`)
}

// titleMeta sets the metadata field "title" to the text of a document's
// first heading.
type titleMeta struct{}

func (titleMeta) Transform(doc *ast.Document) {
	h, ok := doc.FirstChild().(*ast.Heading)
	if !ok {
		return
	}
	if doc.Meta == nil {
		doc.Meta = make(map[string]any)
	}
	doc.Meta["title"] = h.FirstChild().(*ast.Text).Value
}

// TestMeta checks that the parser sets no metadata, and that a transformer
// of documents can.
func TestMeta(t *testing.T) {
	const src = "# Hi\n\ntext"
	if got := parser.NewParser().Parse(src).Meta; got != nil {
		t.Errorf("Meta = %v without extensions, want nil", got)
	}
	p := parser.NewParser(parser.WithASTTransformer(titleMeta{}, 100))
	if got := p.Parse(src).Meta["title"]; got != "Hi" {
		t.Errorf(`Meta["title"] = %v, want "Hi"`, got)
	}
}

// TestAddOptions checks that options added to a parser after it is made
// take effect.
func TestAddOptions(t *testing.T) {
	p := parser.NewParser()
	p.AddOptions(parser.WithInlineParser(mentions{}, 600))
	checkTree(t, p, "@ann", `Document [0:4]
  Paragraph [0:4] "@ann"
    Mention Name="ann"
`)
}
