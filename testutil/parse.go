package testutil

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/style"
)

// Snapshot is a buffer read back from a golden frames document, with the
// color names from the document's legend.
type Snapshot struct {
	// Name is the step name of a sequence snapshot and empty otherwise.
	Name string
	// Buf holds the recorded cells. A cell that shows no character has the
	// foreground of buffer.CellClear, because frames do not record it.
	Buf *buffer.Buffer
	// Names maps each legend color that has a name to that name.
	Names map[style.Color]string
}

// ParseFrames reads a GoldenFrames document back into a Snapshot. Passing the
// snapshot's buffer and names to GoldenFrames reproduces doc.
func ParseFrames(doc string) (Snapshot, error) {
	sections, err := frameSections(doc)
	if err != nil {
		return Snapshot{}, err
	}
	text, err := frameRows(sections["text"], -1)
	if err != nil {
		return Snapshot{}, fmt.Errorf("text frame: %w", err)
	}
	width := 0
	if len(text) > 0 {
		width = len(text[0])
	}
	var marks [3][][]rune
	for i, title := range []string{"fg", "bg", "attrs"} {
		if marks[i], err = markRows(sections, title, width, len(text)); err != nil {
			return Snapshot{}, err
		}
	}
	values, names, err := parseLegend(sections["legend"])
	if err != nil {
		return Snapshot{}, fmt.Errorf("legend: %w", err)
	}
	buf := buffer.New(width, len(text))
	for y, row := range text {
		for x := 0; x < len(row); x++ {
			c, err := decodeCell(row[x], marks[0][y][x], marks[1][y][x], marks[2][y][x], values)
			if err != nil {
				return Snapshot{}, fmt.Errorf("cell (%d, %d): %w", x, y, err)
			}
			buf.SetCell(x, y, c.Char, c.Fg, c.Bg, c.Attributes)
			// A wide character already wrote its continuation cell.
			x += max(buffer.RuneWidth(c.Char), 1) - 1
		}
	}
	return Snapshot{Buf: buf, Names: names}, nil
}

// ParseSequence reads a GoldenSequence document back into its snapshots, in
// order, each named after its step.
func ParseSequence(doc string) ([]Snapshot, error) {
	var names []string
	var bodies []*strings.Builder
	for line := range strings.Lines(doc) {
		if name, ok := stepName(line); ok {
			names = append(names, name)
			bodies = append(bodies, &strings.Builder{})
			continue
		}
		if len(bodies) == 0 {
			return nil, fmt.Errorf("line %q comes before the first step", strings.TrimSuffix(line, "\n"))
		}
		bodies[len(bodies)-1].WriteString(line)
	}
	steps := make([]Snapshot, len(names))
	for i, body := range bodies {
		s, err := ParseFrames(body.String())
		if err != nil {
			return nil, fmt.Errorf("step %s: %w", names[i], err)
		}
		s.Name = names[i]
		steps[i] = s
	}
	return steps, nil
}

// stepName returns the step name that line opens, if it is a step header.
func stepName(line string) (string, bool) {
	name, ok := strings.CutPrefix(strings.TrimSuffix(line, "\n"), "=== step: ")
	if !ok {
		return "", false
	}
	return strings.CutSuffix(name, " ===")
}

// frameSections maps each section title of a frames document to its lines.
func frameSections(doc string) (map[string][]string, error) {
	sections := make(map[string][]string)
	title := ""
	for line := range strings.Lines(doc) {
		line = strings.TrimSuffix(line, "\n")
		if t, ok := sectionTitle(line); ok {
			if _, seen := sections[t]; seen {
				return nil, fmt.Errorf("repeated %s section", t)
			}
			title = t
			sections[t] = []string{}
			continue
		}
		if title == "" {
			return nil, fmt.Errorf("line %q comes before the first section", line)
		}
		sections[title] = append(sections[title], line)
	}
	if _, ok := sections["text"]; !ok {
		return nil, errors.New("no text frame")
	}
	return sections, nil
}

// sectionTitle returns the title of the frames section that line opens.
func sectionTitle(line string) (string, bool) {
	t, ok := strings.CutPrefix(line, "=== ")
	if !ok {
		return "", false
	}
	if t, ok = strings.CutSuffix(t, " ==="); !ok {
		return "", false
	}
	switch t {
	case "text", "fg", "bg", "attrs", "legend":
		return t, true
	}
	return "", false
}

// frameRows returns the runes between the delimiters of each line. Every row
// must hold width runes, or as many as the first row when width is negative.
func frameRows(lines []string, width int) ([][]rune, error) {
	rows := make([][]rune, len(lines))
	for y, line := range lines {
		inner, ok := strings.CutPrefix(line, "|")
		if ok {
			inner, ok = strings.CutSuffix(inner, "|")
		}
		if !ok {
			return nil, fmt.Errorf("row %d lacks its delimiters", y)
		}
		rows[y] = []rune(inner)
		if width < 0 {
			width = len(rows[y])
		}
		if len(rows[y]) != width {
			return nil, fmt.Errorf("row %d is %d cells wide, not %d", y, len(rows[y]), width)
		}
	}
	return rows, nil
}

// markRows returns the rows of the frame titled title, or blank rows when the
// document omits it, checking that they match the text frame's size.
func markRows(sections map[string][]string, title string, width, height int) ([][]rune, error) {
	lines, ok := sections[title]
	if !ok {
		lines = slices.Repeat([]string{"|" + strings.Repeat(" ", width) + "|"}, height)
	}
	if len(lines) != height {
		return nil, fmt.Errorf("%s frame has %d rows, not %d", title, len(lines), height)
	}
	rows, err := frameRows(lines, width)
	if err != nil {
		return nil, fmt.Errorf("%s frame: %w", title, err)
	}
	return rows, nil
}

// parseLegend returns the value of each legend symbol, a color or attribute
// letters, and the name of each named color.
func parseLegend(lines []string) (map[rune]string, map[style.Color]string, error) {
	values := make(map[rune]string)
	names := make(map[style.Color]string)
	for _, line := range lines {
		sym, size := utf8.DecodeRuneInString(line)
		rest, ok := strings.CutPrefix(line[size:], " ")
		if !ok || rest == "" {
			return nil, nil, fmt.Errorf("line %q lacks a value", line)
		}
		value, name, _ := strings.Cut(rest, " ")
		values[sym] = value
		if name == "" {
			continue
		}
		c, err := parseColor(value)
		if err != nil {
			return nil, nil, fmt.Errorf("line %q: %w", line, err)
		}
		names[c] = name
	}
	return values, names, nil
}

// decodeCell returns the cell that shows ch with the fg, bg, and attrs marks.
func decodeCell(ch, fg, bg, attrs rune, values map[rune]string) (buffer.Cell, error) {
	c := buffer.Cell{Char: ch, Fg: buffer.CellClear.Fg}
	var err error
	if ch != ' ' || fg != ' ' {
		if c.Fg, err = symbolColor(fg, FgSymbols, values); err != nil {
			return c, fmt.Errorf("foreground: %w", err)
		}
	}
	if bg != ' ' {
		if c.Bg, err = symbolColor(bg, BgSymbols, values); err != nil {
			return c, fmt.Errorf("background: %w", err)
		}
	}
	if c.Attributes, err = decodeAttrs(attrs, values); err != nil {
		return c, fmt.Errorf("attributes: %w", err)
	}
	return c, nil
}

// symbolColor returns the legend color of sym, which must belong to set.
func symbolColor(sym rune, set string, values map[rune]string) (style.Color, error) {
	value, ok := values[sym]
	if !ok || !strings.ContainsRune(set, sym) {
		return style.Color{}, fmt.Errorf("symbol %q has no color in the legend", sym)
	}
	return parseColor(value)
}

// decodeAttrs returns the attributes that mark stands for: none for a space,
// one for a letter, and the legend's letters for a combination symbol.
func decodeAttrs(mark rune, values map[rune]string) (style.Attrs, error) {
	letters := string(mark)
	switch {
	case mark == ' ':
		return 0, nil
	case strings.ContainsRune(AttrComboSymbols, mark):
		var ok bool
		if letters, ok = values[mark]; !ok {
			return 0, fmt.Errorf("symbol %q has no attributes in the legend", mark)
		}
	}
	var attrs style.Attrs
	for _, l := range letters {
		a := attrOf(l)
		if a == 0 {
			return 0, fmt.Errorf("unknown attribute %q", l)
		}
		attrs |= a
	}
	return attrs, nil
}

// attrOf returns the attribute that letter stands for, or zero when letter
// names none.
func attrOf(letter rune) style.Attrs {
	for _, a := range frameAttrs {
		if rune(a.letter) == letter {
			return a.attr
		}
	}
	return 0
}

// parseColor reads a color as colorHex writes it.
func parseColor(s string) (style.Color, error) {
	if s == "-" {
		return style.ColorTransparent, nil
	}
	hex, ok := strings.CutPrefix(s, "#")
	if !ok || len(hex) != 6 {
		return style.Color{}, fmt.Errorf("color %q is not #rrggbb or -", s)
	}
	v, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return style.Color{}, fmt.Errorf("color %q: %w", s, err)
	}
	return style.ColorFromInts(uint8(v>>16), uint8(v>>8), uint8(v), 255), nil
}
