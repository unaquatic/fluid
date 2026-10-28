package extension

import (
	"iter"
	"strings"

	"github.com/unaquatic/fluid/internal/markdown"
	"github.com/unaquatic/fluid/internal/markdown/html"
	"github.com/unaquatic/fluid/internal/markdown/md"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// table is an extension of tables.
type table struct {
	options []TableOption
}

// NewTable returns an extension that adds tables, configured by opts: the
// parser that [NewTableParser] returns, with priority 900, after the block
// parsers of CommonMark, and the renderer that [NewTableHTMLRenderer]
// returns, with priority 500.
func NewTable(opts ...TableOption) Extension { return &table{options: opts} }

// Extend implements [md.Extender].
func (e *table) Extend(m md.Markdown) {
	m.Parser().AddOptions(parser.WithExtensions(e))
	m.Renderer().AddOptions(html.WithNodeRenderer(NewTableHTMLRenderer(e.options...), 500))
}

// ExtendParser implements [parser.Extension].
func (e *table) ExtendParser(c *parser.Config) {
	c.Apply(parser.WithBlockParser(NewTableParser(), 900))
}

// A TableOption configures the extension of tables.
type TableOption func(c *TableConfig)

// A TableConfig holds how tables render.
type TableConfig struct {
	// CellAlignMethod is how a cell's alignment renders.
	CellAlignMethod TableCellAlignMethod
}

// A TableCellAlignMethod is how the alignment of a table cell renders.
type TableCellAlignMethod int

const (
	// TableCellAlignDefault renders alignments as TableCellAlignAttribute
	// does with XHTML, and as TableCellAlignStyle does without.
	TableCellAlignDefault TableCellAlignMethod = iota

	// TableCellAlignAttribute renders an alignment as an align attribute,
	// such as align="right".
	TableCellAlignAttribute

	// TableCellAlignStyle renders an alignment as a style attribute, such
	// as style="text-align:right".
	TableCellAlignStyle

	// TableCellAlignNone renders no alignments.
	TableCellAlignNone
)

// WithTableCellAlignMethod sets how the alignments of cells render.
func WithTableCellAlignMethod(m TableCellAlignMethod) TableOption {
	return func(c *TableConfig) { c.CellAlignMethod = m }
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
func (tableParser) Open(parent markdown.Node, s *parser.BlockState) (markdown.Node, parser.State) {
	para, ok := parent.(*markdown.Paragraph)
	tried, _ := s.Value(triedKey{}).(map[markdown.Node]bool)
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
			tried = make(map[markdown.Node]bool)
			s.SetValue(triedKey{}, tried)
		}
		tried[para] = true
		return nil, parser.None
	}
	para.SetLines(lines[:len(lines)-1])
	s.Close(para, s.LineNumber()-2)
	t := &markdown.Table{Alignments: align}
	t.AppendChild(t, newRow(&markdown.TableHeader{}, header, align))
	s.StartAt(header.Offset)
	s.Advance(len(s.Line()) - s.Offset())
	return t, parser.NoChildren
}

// Continue implements [parser.BlockParser]; a line with cells continues a
// table, as a row of it, unless another block starts on it.
func (tableParser) Continue(n markdown.Node, s *parser.BlockState) parser.State {
	if s.Blank() || count(splitRow(s.Rest())) == 0 {
		return parser.Close
	}
	return parser.Continue
}

// AddLine implements [parser.LineAdder]. The delimiter row that opens a
// table is consumed, which leaves nothing to add.
func (tableParser) AddLine(n markdown.Node, s *parser.BlockState) {
	if line := s.Remaining(); line.Text != "" {
		t := n.(*markdown.Table)
		t.AppendChild(t, newRow(&markdown.TableRow{}, line, t.Alignments))
	}
}

// Close implements [parser.BlockParser].
func (tableParser) Close(markdown.Node, *parser.BlockState) {}

// CanInterruptParagraph implements [parser.BlockParser].
func (tableParser) CanInterruptParagraph() bool { return true }

// CanAcceptIndentedLine implements [parser.BlockParser].
func (tableParser) CanAcceptIndentedLine() bool { return false }

// newRow returns row with a cell for each column in align, from the cells
// of line: a row with fewer cells gains empty ones, and one with more loses
// those past the last column.
func newRow(row markdown.Node, line markdown.Line, align []markdown.Alignment) markdown.Node {
	end := line.Offset + len(line.Text)
	row.SetSourceSpan(markdown.Span{Start: line.Offset, End: end})
	i := 0
	for c := range splitRow(line.Text) {
		if i == len(align) {
			break
		}
		cell := &markdown.TableCell{Alignment: align[i]}
		cell.SetSourceSpan(markdown.Span{Start: line.Offset + c.start, End: line.Offset + c.end})
		cell.SetLines([]markdown.Line{{Text: c.src, Offset: line.Offset + c.start}})
		row.AppendChild(row, cell)
		i++
	}
	for ; i < len(align); i++ {
		cell := &markdown.TableCell{Alignment: align[i]}
		cell.SetSourceSpan(markdown.Span{Start: end, End: end})
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
func delimiterRow(line string) ([]markdown.Alignment, bool) {
	i := 0
	if strings.HasPrefix(line, "|") {
		i = 1
	}
	var align []markdown.Alignment
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
func delimiterCell(s string) (int, markdown.Alignment) {
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
		return 0, markdown.AlignNone
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
		return i, markdown.AlignCenter
	case left:
		return i, markdown.AlignLeft
	case right:
		return i, markdown.AlignRight
	}
	return i, markdown.AlignNone
}

// tableHTMLRenderer renders tables.
type tableHTMLRenderer struct {
	config TableConfig
}

// NewTableHTMLRenderer returns the renderer of tables, configured by opts.
// A table renders a thead element for its header row and, if it has other
// rows, a tbody element for them.
func NewTableHTMLRenderer(opts ...TableOption) html.NodeRenderer {
	r := &tableHTMLRenderer{}
	for _, opt := range opts {
		opt(&r.config)
	}
	return r
}

// RegisterFuncs implements [html.NodeRenderer].
func (r *tableHTMLRenderer) RegisterFuncs(reg html.NodeRendererFuncRegisterer) {
	reg.Register(markdown.KindTable, renderTable)
	reg.Register(markdown.KindTableHeader, renderTableHeader)
	reg.Register(markdown.KindTableRow, renderTableRow)
	reg.Register(markdown.KindTableCell, r.renderTableCell)
}

// Filters of the attributes of the elements of tables, as HTML defines
// them.
var (
	// TableAttributeFilter allows the attributes of table elements.
	TableAttributeFilter = html.GlobalAttributeFilter.Extend(
		"align", "bgcolor", "border", "cellpadding", "cellspacing", "frame", "rules", "summary", "width",
	)

	// TableHeaderAttributeFilter allows the attributes of thead elements,
	// which the header rows of tables render, in which the tr elements get
	// none.
	TableHeaderAttributeFilter = html.GlobalAttributeFilter.Extend("align", "bgcolor", "char", "charoff", "valign")

	// TableRowAttributeFilter allows the attributes of the tr elements of
	// body rows.
	TableRowAttributeFilter = html.GlobalAttributeFilter.Extend("align", "bgcolor", "char", "charoff", "valign")

	// TableThCellAttributeFilter allows the attributes of th elements,
	// which the cells of header rows render.
	TableThCellAttributeFilter = html.GlobalAttributeFilter.Extend(
		"abbr", "align", "axis", "bgcolor", "char", "charoff", "colspan", "headers", "height", "rowspan",
		"scope", "valign", "width",
	)

	// TableTdCellAttributeFilter allows the attributes of td elements,
	// which the cells of body rows render.
	TableTdCellAttributeFilter = html.GlobalAttributeFilter.Extend(
		"abbr", "align", "axis", "bgcolor", "char", "charoff", "colspan", "headers", "height", "rowspan",
		"scope", "valign", "width",
	)
)

func renderTable(w *html.Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error) {
	if entering {
		w.CR()
		w.WriteString("<table")
		w.WriteAttributes(n, TableAttributeFilter)
		w.WriteString(">\n")
	} else {
		w.WriteString("</table>\n")
	}
	return markdown.WalkContinue, nil
}

func renderTableHeader(w *html.Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error) {
	if entering {
		w.WriteString("<thead")
		w.WriteAttributes(n, TableHeaderAttributeFilter)
		w.WriteString(">\n<tr>\n")
		return markdown.WalkContinue, nil
	}
	w.WriteString("</tr>\n</thead>\n")
	if n.NextSibling() != nil {
		w.WriteString("<tbody>\n")
	}
	return markdown.WalkContinue, nil
}

func renderTableRow(w *html.Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error) {
	if entering {
		w.WriteString("<tr")
		w.WriteAttributes(n, TableRowAttributeFilter)
		w.WriteString(">\n")
		return markdown.WalkContinue, nil
	}
	w.WriteString("</tr>\n")
	if n.NextSibling() == nil {
		w.WriteString("</tbody>\n")
	}
	return markdown.WalkContinue, nil
}

// renderTableCell renders a cell. Its attribute "align" replaces the
// alignment that the align method writes as one, and its attribute "style"
// comes before the alignment that the style method writes, after ";".
func (r *tableHTMLRenderer) renderTableCell(w *html.Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error) {
	tag, filter := "td", TableTdCellAttributeFilter
	if _, ok := n.Parent().(*markdown.TableHeader); ok {
		tag, filter = "th", TableThCellAttributeFilter
	}
	if !entering {
		w.WriteString("</" + tag + ">\n")
		return markdown.WalkContinue, nil
	}
	w.WriteString("<" + tag)
	merged := false // whether the attribute "style" is written with the alignment
	if align := alignName(n.(*markdown.TableCell).Alignment); align != "" {
		method := r.config.CellAlignMethod
		if method == TableCellAlignDefault {
			method = TableCellAlignStyle
			if w.Config().XHTML {
				method = TableCellAlignAttribute
			}
		}
		switch method {
		case TableCellAlignAttribute:
			if _, ok := n.Attribute("align"); !ok {
				w.WriteString(` align="` + align + `"`)
			}
		case TableCellAlignStyle:
			style := "text-align:" + align
			switch v, _ := n.Attribute("style"); v := v.(type) {
			case string:
				style = v + ";" + style
			case []byte:
				style = string(v) + ";" + style
			}
			w.WriteAttribute("style", style)
			merged = true
		}
	}
	for _, a := range n.Attributes() {
		if (a.Name != "style" || !merged) && filter.Allows(a.Name) {
			w.WriteAttribute(a.Name, a.Value)
		}
	}
	w.WriteString(">")
	return markdown.WalkContinue, nil
}

// alignName returns the name of alignment a in HTML, or "" for none.
func alignName(a markdown.Alignment) string {
	switch a {
	case markdown.AlignLeft:
		return "left"
	case markdown.AlignCenter:
		return "center"
	case markdown.AlignRight:
		return "right"
	}
	return ""
}
