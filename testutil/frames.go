package testutil

import (
	"fmt"
	"iter"
	"slices"
	"strings"
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/style"
)

// Frame symbol sets. Each set is disjoint from the others, so one legend
// covers every frame.
const (
	// FgSymbols mark the foreground colors of cells that show a character.
	FgSymbols = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	// BgSymbols mark opaque background colors.
	BgSymbols = "░▒▓█"
	// AttrComboSymbols mark cells that combine several attributes.
	AttrComboSymbols = "123456789"
)

// frameAttrs are the attributes the attrs frame shows, each with its letter,
// in the order a combination lists them.
var frameAttrs = []struct {
	attr   style.Attrs
	letter byte
}{
	{style.AttrBold, 'B'},
	{style.AttrDim, 'D'},
	{style.AttrItalic, 'I'},
	{style.AttrUnderline, 'U'},
	{style.AttrStrikethrough, 'S'},
}

// frameAttrMask selects the attributes in frameAttrs.
const frameAttrMask = style.AttrBold | style.AttrDim | style.AttrItalic |
	style.AttrUnderline | style.AttrStrikethrough

// GoldenFrames returns a deterministic representation of buf as a series of
// frames, each drawing one property of every cell between vertical
// delimiters, followed by a legend. For example:
//
//	=== text ===
//	|say hi |
//	=== fg ===
//	|AAA BB |
//	=== bg ===
//	|░░░░░░ |
//	=== attrs ===
//	|    BB |
//	=== legend ===
//	A #cfd2d5 TextBase
//	B #5af78d
//	░ #262b27 SuccessSurface
//
// The text frame is GoldenText. The fg frame marks the foreground color of
// each cell that shows a character with FgSymbols, the bg frame marks each
// opaque background with BgSymbols, and the attrs frame marks a single
// attribute with its letter (B bold, D dim, I italic, U underline, S
// strikethrough) and a combination with AttrComboSymbols. Symbols are
// assigned in order of first appearance, reading rows top to bottom and cells
// left to right, and a space marks a cell without the property. The fg, bg,
// and attrs frames are omitted when they mark no cell.
//
// The legend lists each symbol with its color and, when names holds one, the
// color's name, then each combination symbol with its letters. names may be
// nil. GoldenFrames reports an error when buf has more distinct values of a
// property than its symbol set holds.
func GoldenFrames(buf *buffer.Buffer, names map[style.Color]string) (string, error) {
	f := NewFrames()
	var sb strings.Builder
	sb.WriteString("=== text ===\n")
	sb.WriteString(GoldenText(buf))
	frames := []struct {
		title string
		mark  func(buffer.Cell) (rune, error)
	}{
		{"fg", f.Fg},
		{"bg", f.Bg},
		{"attrs", f.Attrs},
	}
	for _, fr := range frames {
		frame, err := DrawFrame(buf, fr.mark)
		if err != nil {
			return "", fmt.Errorf("%s frame: %w", fr.title, err)
		}
		if blankFrame(frame) {
			continue
		}
		sb.WriteString("=== " + fr.title + " ===\n")
		sb.WriteString(frame)
	}
	if legend := f.Legend(names); legend != "" {
		sb.WriteString("=== legend ===\n")
		sb.WriteString(legend)
	}
	return sb.String(), nil
}

// AssertGoldenFrames compares GoldenFrames for buf with
// testdata/<name>.frames.golden. names labels colors in the legend and may be
// nil. The -update test flag creates or replaces the file.
func AssertGoldenFrames(t *testing.T, name string, buf *buffer.Buffer, names map[style.Color]string) {
	t.Helper()
	frames, err := GoldenFrames(buf, names)
	if err != nil {
		t.Fatalf("golden frames %s: %v", name, err)
	}
	assertGoldenContent(t, name+".frames.golden", frames)
}

// DrawFrame returns the rune that mark returns for every cell of buf, one row
// per line between vertical delimiters.
func DrawFrame(buf *buffer.Buffer, mark func(buffer.Cell) (rune, error)) (string, error) {
	var sb strings.Builder
	for y := range buf.Height() {
		sb.WriteByte('|')
		for x := range buf.Width() {
			r, err := mark(buf.Cell(x, y))
			if err != nil {
				return "", fmt.Errorf("cell (%d, %d): %w", x, y, err)
			}
			sb.WriteRune(r)
		}
		sb.WriteString("|\n")
	}
	return sb.String(), nil
}

// blankFrame reports whether frame marks no cell.
func blankFrame(frame string) bool {
	return !strings.ContainsFunc(frame, func(r rune) bool {
		return r != '|' && r != ' ' && r != '\n'
	})
}

// Frames assigns the symbols of the fg, bg, and attrs frames of one
// GoldenFrames representation.
type Frames struct {
	fg     *Symbols[style.Color]
	bg     *Symbols[style.Color]
	combos *Symbols[style.Attrs]
}

// NewFrames returns a Frames with no symbols assigned.
func NewFrames() *Frames {
	return &Frames{
		fg:     NewSymbols[style.Color](FgSymbols),
		bg:     NewSymbols[style.Color](BgSymbols),
		combos: NewSymbols[style.Attrs](AttrComboSymbols),
	}
}

// Fg marks the foreground color of c, or a space when c shows no character.
func (f *Frames) Fg(c buffer.Cell) (rune, error) {
	if c.Char == 0 || c.Char == ' ' {
		return ' ', nil
	}
	r, err := f.fg.Symbol(c.Fg)
	if err != nil {
		return 0, fmt.Errorf("foreground %s: %w", colorHex(c.Fg), err)
	}
	return r, nil
}

// Bg marks the background color of c, or a space when it is transparent.
func (f *Frames) Bg(c buffer.Cell) (rune, error) {
	if c.Bg.IsTransparent() {
		return ' ', nil
	}
	r, err := f.bg.Symbol(c.Bg)
	if err != nil {
		return 0, fmt.Errorf("background %s: %w", colorHex(c.Bg), err)
	}
	return r, nil
}

// Attrs marks the attributes of c: a space for none, a letter for one, and a
// combination symbol for several.
func (f *Frames) Attrs(c buffer.Cell) (rune, error) {
	attrs := c.Attributes & frameAttrMask
	if attrs == 0 {
		return ' ', nil
	}
	if letters := attrLetters(attrs); len(letters) == 1 {
		return rune(letters[0]), nil
	}
	r, err := f.combos.Symbol(attrs)
	if err != nil {
		return 0, fmt.Errorf("attributes %s: %w", attrLetters(attrs), err)
	}
	return r, nil
}

// Legend returns one line per assigned symbol: foreground colors, then
// background colors, then attribute combinations. A color line holds the
// symbol, the color, and its name in names when there is one; a combination
// line holds the symbol and its attribute letters.
func (f *Frames) Legend(names map[style.Color]string) string {
	var sb strings.Builder
	for _, colors := range []*Symbols[style.Color]{f.fg, f.bg} {
		for r, c := range colors.All() {
			sb.WriteString(fmt.Sprintf("%c %s", r, colorHex(c)))
			if name := names[c]; name != "" {
				sb.WriteString(" " + name)
			}
			sb.WriteByte('\n')
		}
	}
	for r, attrs := range f.combos.All() {
		sb.WriteString(fmt.Sprintf("%c %s\n", r, attrLetters(attrs)))
	}
	return sb.String()
}

// attrLetters returns the letters of the frame attributes set in attrs.
func attrLetters(attrs style.Attrs) string {
	var letters []byte
	for _, a := range frameAttrs {
		if attrs&a.attr != 0 {
			letters = append(letters, a.letter)
		}
	}
	return string(letters)
}

// Symbols assigns the runes of a symbol set to keys in order of first
// appearance.
type Symbols[K comparable] struct {
	set  []rune
	keys []K
}

// NewSymbols returns an assignment over the runes of set with no key assigned.
func NewSymbols[K comparable](set string) *Symbols[K] {
	return &Symbols[K]{set: []rune(set)}
}

// Symbol returns the symbol of k, assigning the next unused one when k is new.
// It reports an error when every symbol is in use.
func (s *Symbols[K]) Symbol(k K) (rune, error) {
	if i := slices.Index(s.keys, k); i >= 0 {
		return s.set[i], nil
	}
	if len(s.keys) == len(s.set) {
		return 0, fmt.Errorf("more than %d distinct values", len(s.set))
	}
	s.keys = append(s.keys, k)
	return s.set[len(s.keys)-1], nil
}

// All yields each assigned symbol with its key, in order of assignment.
func (s *Symbols[K]) All() iter.Seq2[rune, K] {
	return func(yield func(rune, K) bool) {
		for i, k := range s.keys {
			if !yield(s.set[i], k) {
				return
			}
		}
	}
}
