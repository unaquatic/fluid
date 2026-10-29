package extension

import (
	"iter"
	"strings"

	"github.com/unaquatic/fluid/internal/markdown/ast"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// table is the extension Table.
type table struct{}

// ExtendParser implements [parser.Extension].
func (table) ExtendParser(c *parser.Config) {
	c.Apply(parser.WithBlockParser(NewTableParser(), 900))
}

// tableParser parses tables.
type tableParser struct{}

// NewTableParser returns the parser of tables. A table starts at a
// delimiter row under a paragraph whose last line, the header row, has as
// many cells: the header row becomes the table's, and the lines before it
// stay a paragraph.
func NewTableParser() parser.BlockParser { return tableParser{} }

// triedKey keys the set of paragraphs that a delimiter row has failed to
// turn into a table, which no later line tries again, in the values of a
// [parser.BlockState].
type triedKey struct{}

// Trigger implements [parser.BlockParser]; a delimiter row starts with a
// pipe, a colon, a hyphen, or a vertical tab or form feed before one.
func (tableParser) Trigger() []byte { return []byte("|:-\v\f") }

// Open implements [parser.BlockParser].
func (tableParser) Open(parent ast.Node, s *parser.BlockState) (ast.Node, parser.State) {
	para, ok := parent.(*ast.Paragraph)
	tried, _ := s.Value(triedKey{}).(map[ast.Node]bool)
	if !ok || len(para.Lines()) == 0 || tried[para] {
		return nil, parser.None
	}
	align, ok := delimiterRow(s.Rest())
	if !ok {
		return nil, parser.None
	}
	lines := para.Lines()
	header := lines[len(lines)-1]
	if count(splitRow(header.Text)) != len(align) {
		if tried == nil {
			tried = make(map[ast.Node]bool)
			s.SetValue(triedKey{}, tried)
		}
		tried[para] = true
		return nil, parser.None
	}
	para.SetLines(lines[:len(lines)-1])
	s.Close(para, s.LineNumber()-2)
	t := &ast.Table{Alignments: align}
	t.AppendChild(t, newRow(&ast.TableHeader{}, header, align))
	s.StartAt(header.Offset)
	s.Advance(len(s.Line()) - s.Offset())
	return t, parser.NoChildren
}

// Continue implements [parser.BlockParser]; a line with cells continues a
// table, as a row of it, unless another block starts on it.
func (tableParser) Continue(n ast.Node, s *parser.BlockState) parser.State {
	if s.Blank() || count(splitRow(s.Rest())) == 0 {
		return parser.Close
	}
	return parser.Continue
}

// AddLine implements [parser.LineAdder]. The delimiter row that opens a
// table is consumed, which leaves nothing to add.
func (tableParser) AddLine(n ast.Node, s *parser.BlockState) {
	if line := s.Remaining(); line.Text != "" {
		t := n.(*ast.Table)
		t.AppendChild(t, newRow(&ast.TableRow{}, line, t.Alignments))
	}
}

// Close implements [parser.BlockParser].
func (tableParser) Close(ast.Node, *parser.BlockState) {}

// CanInterruptParagraph implements [parser.BlockParser].
func (tableParser) CanInterruptParagraph() bool { return true }

// CanAcceptIndentedLine implements [parser.BlockParser].
func (tableParser) CanAcceptIndentedLine() bool { return false }

// newRow returns row with a cell for each column in align, from the cells
// of line: a row with fewer cells gains empty ones, and one with more loses
// those past the last column.
func newRow(row ast.Node, line ast.Line, align []ast.Alignment) ast.Node {
	end := line.Offset + len(line.Text)
	row.SetSourceSpan(ast.Span{Start: line.Offset, End: end})
	i := 0
	for c := range splitRow(line.Text) {
		if i == len(align) {
			break
		}
		cell := &ast.TableCell{Alignment: align[i]}
		cell.SetSourceSpan(ast.Span{Start: line.Offset + c.start, End: line.Offset + c.end})
		cell.SetLines([]ast.Line{{Text: c.src, Offset: line.Offset + c.start}})
		row.AppendChild(row, cell)
		i++
	}
	for ; i < len(align); i++ {
		cell := &ast.TableCell{Alignment: align[i]}
		cell.SetSourceSpan(ast.Span{Start: end, End: end})
		row.AppendChild(row, cell)
	}
	return row
}

// count returns the number of values that seq yields.
func count[T any](seq iter.Seq[T]) int {
	n := 0
	for range seq {
		n++
	}
	return n
}

// cell is a cell of a table row: its source, and its range in the row.
type cell struct {
	src        string
	start, end int
}

// splitRow returns the cells of table row line, which is read from its
// first character that is not a space or tab: the text between pipes that
// backslashes do not escape, trimmed, with "\|" replaced by "|".
func splitRow(line string) iter.Seq[cell] {
	return func(yield func(cell) bool) {
		i := pipeEnd(line, 0) // an optional leading pipe
		for i < len(line) {
			j := i
			for j < len(line) && line[j] != '|' {
				if line[j] == '\\' && j+1 < len(line) && line[j+1] == '|' {
					j++
				}
				j++
			}
			e := pipeEnd(line, j)
			if j > i || e > 0 {
				a, b := i, j
				for a < b && isCellSpace(line[a]) {
					a++
				}
				for b > a && isCellSpace(line[b-1]) {
					b--
				}
				if !yield(cell{strings.ReplaceAll(line[a:b], `\|`, "|"), a, b}) {
					return
				}
			}
			i = j + e
			if e == 0 {
				return
			}
		}
	}
}

func isCellSpace(c byte) bool { return c == ' ' || '\t' <= c && c <= '\r' }

// pipeEnd returns the length of the pipe, and the spaces after it, at offset
// i of line, or 0 if there is no pipe.
func pipeEnd(line string, i int) int {
	if i >= len(line) || line[i] != '|' {
		return 0
	}
	j := i + 1
	for j < len(line) && isTableSpace(line[j]) {
		j++
	}
	return j - i
}

func isTableSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\v' || c == '\f' }

// delimiterRow returns the column alignments of table delimiter row line,
// which is read from its first character that is not a space or tab, and
// reports whether line is one: cells of hyphens, each with an optional
// colon before and after them, parted by pipes.
func delimiterRow(line string) ([]ast.Alignment, bool) {
	i := 0
	if strings.HasPrefix(line, "|") {
		i = 1
	}
	var align []ast.Alignment
	for {
		n, a := delimiterCell(line[i:])
		if n == 0 {
			if len(align) == 0 || line[i-1] != '|' {
				return nil, false
			}
			break // the trailing pipe
		}
		align = append(align, a)
		i += n
		if i == len(line) || line[i] != '|' {
			break
		}
		i++
	}
	for ; i < len(line); i++ {
		if !isTableSpace(line[i]) {
			return nil, false
		}
	}
	return align, true
}

// delimiterCell returns the length, and the alignment, of the delimiter row
// cell at the start of s, or 0 if there is none.
func delimiterCell(s string) (int, ast.Alignment) {
	i := 0
	for i < len(s) && isTableSpace(s[i]) {
		i++
	}
	left := i < len(s) && s[i] == ':'
	if left {
		i++
	}
	n := runLen(s, i, '-')
	if n == 0 {
		return 0, ast.AlignNone
	}
	i += n
	right := i < len(s) && s[i] == ':'
	if right {
		i++
	}
	for i < len(s) && isTableSpace(s[i]) {
		i++
	}
	switch {
	case left && right:
		return i, ast.AlignCenter
	case left:
		return i, ast.AlignLeft
	case right:
		return i, ast.AlignRight
	}
	return i, ast.AlignNone
}
