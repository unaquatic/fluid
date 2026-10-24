package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/testutil"
)

// MaxSide is the most cells a design may be wide or tall.
const MaxSide = 400

// Design is a grid of terminal cells as the editor page paints it and
// exchanges it as JSON.
type Design struct {
	// Width and Height are the size of the grid in cells.
	Width  int `json:"width"`
	Height int `json:"height"`
	// Cells holds the rows of the grid, top to bottom, each Width cells long.
	Cells [][]Cell `json:"cells"`
	// Names maps colors, as #rrggbb, to the names the legend gives them.
	Names map[string]string `json:"names,omitempty"`
}

// Cell is one cell of a Design.
type Cell struct {
	// Ch is the character the cell shows: one code point, a space for none,
	// or "" when the cell continues the wide character to its left.
	Ch string `json:"ch"`
	// Fg is the color of Ch as #rrggbb, or "" for transparent. A cell that
	// shows no character has none.
	Fg string `json:"fg,omitempty"`
	// Bg is the background as #rrggbb, or "" for transparent.
	Bg string `json:"bg,omitempty"`
	// Attrs holds the letters of the cell's attributes, in the order of
	// Attrs.
	Attrs string `json:"attrs,omitempty"`
}

// Attr is a cell attribute that golden frames record.
type Attr struct {
	// Bit is the attribute's flag in buffer.Cell.Attributes.
	Bit style.Attrs
	// Letter marks the attribute in an attrs frame.
	Letter byte
	// SGR is the parameter of the escape sequence that turns it on.
	SGR string
}

// Attrs are the attributes golden frames record, in the order a combination
// lists them.
var Attrs = []Attr{
	{style.AttrBold, 'B', "1"},
	{style.AttrDim, 'D', "2"},
	{style.AttrItalic, 'I', "3"},
	{style.AttrUnderline, 'U', "4"},
	{style.AttrStrikethrough, 'S', "9"},
}

// NewDesign returns the cells of buf as a design whose legend gives the
// colors in names their names. names may be nil.
func NewDesign(buf *buffer.Buffer, names map[style.Color]string) Design {
	d := Design{Width: buf.Width(), Height: buf.Height(), Cells: make([][]Cell, buf.Height())}
	for y := range buf.Height() {
		row := make([]Cell, buf.Width())
		for x := range buf.Width() {
			row[x] = NewCell(buf.Cell(x, y))
		}
		d.Cells[y] = row
	}
	for c, name := range names {
		if d.Names == nil {
			d.Names = make(map[string]string, len(names))
		}
		d.Names[Hex(c)] = name
	}
	return d
}

// NewCell returns c as a design cell.
func NewCell(c buffer.Cell) Cell {
	cell := Cell{Ch: " ", Bg: Hex(c.Bg), Attrs: AttrLetters(c.Attributes)}
	switch {
	case c.IsContinuation():
		cell.Ch = ""
	case !c.IsEmpty():
		cell.Ch = string(c.Char)
		cell.Fg = Hex(c.Fg)
	}
	return cell
}

// ParseDoc reads a golden frames document into its one snapshot, or a golden
// sequence document into a snapshot for each step.
func ParseDoc(doc string) ([]testutil.Snapshot, error) {
	if strings.HasPrefix(doc, "=== step: ") {
		return testutil.ParseSequence(doc)
	}
	s, err := testutil.ParseFrames(doc)
	if err != nil {
		return nil, err
	}
	return []testutil.Snapshot{s}, nil
}

// Buffer returns the cells of d in a buffer, with the legend names by
// color. It reports an error when d is malformed: rows of the wrong length,
// a color that is not #rrggbb, or a cell that holds anything but one
// printable character.
func (d Design) Buffer() (*buffer.Buffer, map[style.Color]string, error) {
	if d.Width < 1 || d.Height < 1 || d.Width > MaxSide || d.Height > MaxSide {
		return nil, nil, fmt.Errorf("size %d×%d is not between 1×1 and %d×%d", d.Width, d.Height, MaxSide, MaxSide)
	}
	if len(d.Cells) != d.Height {
		return nil, nil, fmt.Errorf("%d rows, not %d", len(d.Cells), d.Height)
	}
	buf := buffer.New(d.Width, d.Height)
	for y, row := range d.Cells {
		if len(row) != d.Width {
			return nil, nil, fmt.Errorf("row %d is %d cells wide, not %d", y, len(row), d.Width)
		}
		// covered is the column after the last one a wide character spans.
		covered := 0
		for x, c := range row {
			if c.Ch == "" && x < covered {
				continue
			}
			r, fg, bg, attrs, err := c.Parse()
			if err != nil {
				return nil, nil, fmt.Errorf("cell (%d, %d): %w", x, y, err)
			}
			buf.SetCell(x, y, r, fg, bg, attrs)
			covered = x + buffer.RuneWidth(r)
		}
	}
	names := make(map[style.Color]string, len(d.Names))
	for hex, name := range d.Names {
		c, err := ParseHex(hex)
		if err != nil || hex == "" {
			return nil, nil, fmt.Errorf("legend name %q: color %q is not #rrggbb", name, hex)
		}
		if strings.ContainsFunc(name, unicode.IsControl) {
			return nil, nil, fmt.Errorf("legend name %q holds a control character", name)
		}
		if name = strings.TrimSpace(name); name != "" {
			names[c] = name
		}
	}
	return buf, names, nil
}

// Parse returns the character, colors, and attributes of c. A continuation
// cell that no wide character covers reads as a space.
func (c Cell) Parse() (r rune, fg, bg style.Color, attrs style.Attrs, err error) {
	r = ' '
	if c.Ch != "" {
		var size int
		r, size = utf8.DecodeRuneInString(c.Ch)
		if size != len(c.Ch) || r == utf8.RuneError {
			return 0, fg, bg, 0, fmt.Errorf("character %q is not one code point", c.Ch)
		}
		if !unicode.IsPrint(r) || buffer.RuneWidth(r) < 1 {
			return 0, fg, bg, 0, fmt.Errorf("character %q takes up no column", c.Ch)
		}
	}
	if fg, err = ParseHex(c.Fg); err != nil {
		return 0, fg, bg, 0, fmt.Errorf("foreground: %w", err)
	}
	if r == ' ' {
		fg = buffer.CellClear.Fg
	}
	if bg, err = ParseHex(c.Bg); err != nil {
		return 0, fg, bg, 0, fmt.Errorf("background: %w", err)
	}
	if attrs, err = ParseAttrs(c.Attrs); err != nil {
		return 0, fg, bg, 0, err
	}
	return r, fg, bg, attrs, nil
}

// Frames returns d as the golden frames document that testutil.GoldenFrames
// writes for its cells and names.
func (d Design) Frames() (string, error) {
	buf, names, err := d.Buffer()
	if err != nil {
		return "", err
	}
	return testutil.GoldenFrames(buf, names)
}

// ANSI returns d as WriteANSI writes it.
func (d Design) ANSI() (string, error) {
	buf, _, err := d.Buffer()
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	if err := WriteANSI(&sb, buf); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// WriteANSI writes buf to w as lines of text that color and style their
// characters with SGR escape sequences in 24-bit color, for a terminal to
// show. Transparent colors are the terminal's own. Each line that sets a
// style ends by resetting it, and trailing cells that show nothing are left
// out.
func WriteANSI(w io.Writer, buf *buffer.Buffer) error {
	bw := bufio.NewWriter(w)
	for y := range buf.Height() {
		end := buf.Width()
		for end > 0 && inkless(buf.Cell(end-1, y)) && buf.Cell(end-1, y).Bg.IsTransparent() {
			end--
		}
		var cur SGR
		for x := range end {
			c := buf.Cell(x, y)
			if c.IsContinuation() {
				continue
			}
			want := NewSGR(c)
			if inkless(c) {
				// No ink shows the foreground, so any will do.
				want.Fg = cur.Fg
			}
			if want != cur {
				bw.WriteString(want.String())
				cur = want
			}
			ch := c.Char
			if ch == 0 {
				ch = ' '
			}
			bw.WriteRune(ch)
		}
		if cur != (SGR{}) {
			bw.WriteString("\x1b[0m")
		}
		bw.WriteByte('\n')
	}
	return bw.Flush()
}

// SGR is the style of a cell as the parameters of an SGR escape sequence,
// each part empty for the terminal's default or else a list of parameters
// that each follow a semicolon.
type SGR struct {
	Attrs, Fg, Bg string
}

// NewSGR returns the style of c. A cell that shows no character has no
// foreground.
func NewSGR(c buffer.Cell) SGR {
	var s SGR
	for _, a := range Attrs {
		if c.Attributes&a.Bit != 0 {
			s.Attrs += ";" + a.SGR
		}
	}
	if !c.IsEmpty() && !c.Fg.IsTransparent() {
		s.Fg = fmt.Sprintf(";38;2;%d;%d;%d", c.Fg.R, c.Fg.G, c.Fg.B)
	}
	if !c.Bg.IsTransparent() {
		s.Bg = fmt.Sprintf(";48;2;%d;%d;%d", c.Bg.R, c.Bg.G, c.Bg.B)
	}
	return s
}

// String returns the escape sequence that resets the style and selects s.
func (s SGR) String() string {
	return "\x1b[0" + s.Attrs + s.Fg + s.Bg + "m"
}

// inkless reports whether c draws nothing in its foreground color: no
// character, and no line through or under it.
func inkless(c buffer.Cell) bool {
	return c.IsEmpty() && c.Attributes&(style.AttrUnderline|style.AttrStrikethrough) == 0
}

// AttrLetters returns the letters of the attributes in Attrs that attrs sets.
func AttrLetters(attrs style.Attrs) string {
	var sb strings.Builder
	for _, a := range Attrs {
		if attrs&a.Bit != 0 {
			sb.WriteByte(a.Letter)
		}
	}
	return sb.String()
}

// ParseAttrs returns the attributes whose letters letters holds.
func ParseAttrs(letters string) (style.Attrs, error) {
	var attrs style.Attrs
	for _, l := range letters {
		i := strings.IndexRune("BDIUS", l)
		if i < 0 {
			return 0, fmt.Errorf("unknown attribute %q", l)
		}
		attrs |= Attrs[i].Bit
	}
	return attrs, nil
}

// Hex returns c as #rrggbb, or "" when it is transparent.
func Hex(c style.Color) string {
	if c.IsTransparent() {
		return ""
	}
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// ParseHex reads a color as Hex writes it.
func ParseHex(s string) (style.Color, error) {
	if s == "" {
		return style.ColorTransparent, nil
	}
	hex, ok := strings.CutPrefix(s, "#")
	v, err := strconv.ParseUint(hex, 16, 32)
	if !ok || len(hex) != 6 || err != nil {
		return style.Color{}, fmt.Errorf("color %q is not #rrggbb", s)
	}
	return style.ColorFromInts(uint8(v>>16), uint8(v>>8), uint8(v), 255), nil
}
