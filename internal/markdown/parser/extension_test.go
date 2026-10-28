package parser_test

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/unaquatic/fluid/internal/markdown"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// Kinds of the nodes of the extension that TestExtension parses with.
var (
	kindNote    = markdown.NewNodeKind("Note")
	kindMention = markdown.NewNodeKind("Mention")
)

// note is a container between a "::: note" line and a ":::" line.
type note struct {
	markdown.BaseBlock
}

func (*note) Kind() markdown.NodeKind { return kindNote }

// mention is "@" and a name.
type mention struct {
	markdown.BaseInline
	Name string
}

func (*mention) Kind() markdown.NodeKind { return kindMention }

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

func (notes) Open(parent markdown.Node, s *parser.BlockState) (markdown.Node, parser.State) {
	if s.Rest() != "::: note" {
		return nil, parser.None
	}
	s.Advance(len(s.Line()) - s.Offset())
	return &note{}, parser.HasChildren
}

func (notes) Continue(n markdown.Node, s *parser.BlockState) parser.State {
	if s.Rest() == ":::" {
		s.Close(n, s.LineNumber())
		return parser.Close
	}
	return parser.Continue | parser.HasChildren
}

func (notes) Close(markdown.Node, *parser.BlockState) {}
func (notes) CanInterruptParagraph() bool             { return true }
func (notes) CanAcceptIndentedLine() bool             { return false }

type mentions struct{}

func (mentions) Trigger() []byte { return []byte{'@'} }

func (mentions) Parse(parent markdown.Node, s *parser.InlineState) (markdown.Node, int) {
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

// tree returns the kind of each node in n, in the order they start, with
// the source that a block's span covers or the text of inline content.
func tree(src string, n markdown.Node) []string {
	var out []string
	_ = markdown.Walk(n, func(n markdown.Node, entering bool) (markdown.WalkStatus, error) {
		if !entering {
			return markdown.WalkContinue, nil
		}
		switch n := n.(type) {
		case *markdown.Document:
		case *markdown.Text:
			out = append(out, fmt.Sprintf("Text %q", n.Value))
		case *mention:
			out = append(out, "Mention "+n.Name)
		default:
			s := n.SourceSpan()
			out = append(out, fmt.Sprintf("%s %q", n.Kind(), src[s.Start:s.End]))
		}
		return markdown.WalkContinue, nil
	})
	return out
}

// TestExtension checks that an extension can add a container block that
// closes at a line of its own, and inline content, of its own kinds, and
// that the blocks open in the container end at the line before.
func TestExtension(t *testing.T) {
	const src = "::: note\n> hi @bob\n:::\nafter @ 1"
	p := parser.NewParser(parser.WithExtensions(notesAndMentions{}))
	want := []string{
		`Note "::: note\n> hi @bob\n:::"`,
		`Blockquote "> hi @bob"`,
		`Paragraph "hi @bob"`,
		`Text "hi "`,
		`Mention bob`,
		`Paragraph "after @ 1"`,
		`Text "after @ 1"`,
	}
	if got := tree(src, p.Parse(src)); !slices.Equal(got, want) {
		t.Errorf("tree:\ngot  %q\nwant %q", got, want)
	}
}

// reopener opens the first note in the tip again at a line "+++", if the
// note is the tip's last child.
type reopener struct{}

func (reopener) Trigger() []byte { return []byte{'+'} }

func (reopener) Open(parent markdown.Node, s *parser.BlockState) (markdown.Node, parser.State) {
	if s.Rest() != "+++" {
		return nil, parser.None
	}
	for c := range markdown.Children(parent) {
		if n, ok := c.(*note); ok && s.Reopen(n, notes{}) {
			s.Advance(len(s.Line()) - s.Offset())
			return n, parser.HasChildren
		}
	}
	return nil, parser.None
}

func (reopener) Continue(markdown.Node, *parser.BlockState) parser.State { return parser.Close }
func (reopener) Close(markdown.Node, *parser.BlockState)                 {}
func (reopener) CanInterruptParagraph() bool                             { return true }
func (reopener) CanAcceptIndentedLine() bool                             { return false }

// TestReopen checks that a block parser can open a block that closed again,
// if it is the last block, and that its span then ends at its new end, and
// cannot if a block follows it.
func TestReopen(t *testing.T) {
	const src = "::: note\nx\n:::\n+++\ny\n:::\n\npara\n\n+++"
	p := parser.NewParser(parser.WithExtensions(notesAndMentions{}), parser.WithBlockParser(reopener{}, 60))
	want := []string{
		`Note "::: note\nx\n:::\n+++\ny\n:::"`,
		`Paragraph "x"`,
		`Text "x"`,
		`Paragraph "y"`,
		`Text "y"`,
		`Paragraph "para"`,
		`Text "para"`,
		`Paragraph "+++"`,
		`Text "+++"`,
	}
	if got := tree(src, p.Parse(src)); !slices.Equal(got, want) {
		t.Errorf("tree:\ngot  %q\nwant %q", got, want)
	}
}

// closer closes the paragraph that a line "%" and text would continue, and
// opens nothing.
type closer struct{}

func (closer) Trigger() []byte { return []byte{'%'} }

func (closer) Open(parent markdown.Node, s *parser.BlockState) (markdown.Node, parser.State) {
	if p, ok := parent.(*markdown.Paragraph); ok {
		s.Close(p, s.LineNumber()-1)
	}
	return nil, parser.None
}

func (closer) Continue(markdown.Node, *parser.BlockState) parser.State { return parser.Close }
func (closer) Close(markdown.Node, *parser.BlockState)                 {}
func (closer) CanInterruptParagraph() bool                             { return true }
func (closer) CanAcceptIndentedLine() bool                             { return false }

// TestCloseContainer checks that a line whose block parser closes the
// paragraph that it would continue, and opens nothing, starts a paragraph
// of its own, as the block parsers try it again in the tip.
func TestCloseContainer(t *testing.T) {
	const src = "> a\n> %b\n> - c"
	p := parser.NewParser(parser.WithBlockParser(closer{}, 50))
	want := []string{
		`Blockquote "> a\n> %b\n> - c"`,
		`Paragraph "a"`,
		`Text "a"`,
		`Paragraph "%b"`,
		`Text "%b"`,
		`List "- c"`,
		`ListItem "- c"`,
		`Paragraph "c"`,
		`Text "c"`,
	}
	if got := tree(src, p.Parse(src)); !slices.Equal(got, want) {
		t.Errorf("tree:\ngot  %q\nwant %q", got, want)
	}
}

// bangs opens a one-line block at a line that starts with "- !".
type bangs struct{}

func (bangs) Trigger() []byte { return []byte{'-'} }

func (bangs) Open(parent markdown.Node, s *parser.BlockState) (markdown.Node, parser.State) {
	if !strings.HasPrefix(s.Rest(), "- !") {
		return nil, parser.None
	}
	s.Advance(len(s.Line()) - s.Offset())
	return &markdown.ThematicBreak{}, parser.NoChildren
}

func (bangs) Continue(markdown.Node, *parser.BlockState) parser.State { return parser.Close }
func (bangs) Close(markdown.Node, *parser.BlockState)                 {}
func (bangs) CanInterruptParagraph() bool                             { return true }
func (bangs) CanAcceptIndentedLine() bool                             { return false }

// TestPriority checks that a block parser's priority places it among the
// built-in ones: before list items at 650, which come at 700, and after
// them at 750.
func TestPriority(t *testing.T) {
	for _, tt := range []struct {
		priority int
		want     markdown.NodeKind
	}{
		{650, markdown.KindThematicBreak},
		{750, markdown.KindList},
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

func (names) Transform(p *markdown.Paragraph, s *parser.BlockState) {
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

func (knownMentions) Parse(parent markdown.Node, s *parser.InlineState) (markdown.Node, int) {
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
	want := []string{`Paragraph "@ann @bob"`, `Text "@ann "`, "Mention bob1", `Paragraph "@bob"`, "Mention bob2"}
	if got := tree(src, p.Parse(src)); !slices.Equal(got, want) {
		t.Errorf("tree:\ngot  %q\nwant %q", got, want)
	}
}

// titleMeta sets the metadata field "title" to the text of a document's
// first heading.
type titleMeta struct{}

func (titleMeta) Transform(doc *markdown.Document) {
	h, ok := doc.FirstChild().(*markdown.Heading)
	if !ok {
		return
	}
	if doc.Meta == nil {
		doc.Meta = make(map[string]any)
	}
	doc.Meta["title"] = h.FirstChild().(*markdown.Text).Value
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
	if got := tree("@ann", p.Parse("@ann")); !slices.Equal(got, []string{`Paragraph "@ann"`, "Mention ann"}) {
		t.Errorf("tree %q, want a mention", got)
	}
}
