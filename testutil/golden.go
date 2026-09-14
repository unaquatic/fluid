package testutil

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/style"
)

// updateGolden controls whether assertions replace their golden files.
var updateGolden = flag.Bool("update", false, "update golden test files")

// Buffer serialization.

// GoldenText returns a deterministic character-grid representation of buf.
// Vertical delimiters make trailing spaces visible:
//
//	|Hello     |
//	|World     |
//
// Zero-value cells become spaces. The output does not include cell styles.
func GoldenText(buf *buffer.Buffer) string {
	var sb strings.Builder
	w := buf.Width()
	h := buf.Height()
	for y := range h {
		sb.WriteByte('|')
		for x := range w {
			cell := buf.Cell(x, y)
			if cell.Char == 0 {
				sb.WriteRune(' ')
			} else {
				sb.WriteRune(cell.Char)
			}
		}
		sb.WriteString("|\n")
	}
	return sb.String()
}

// GoldenStyled returns a deterministic representation of characters,
// foreground colors, and selected text attributes. Its cell format is:
//
//	char[attrs]
//
// Attribute flags are B for bold, D for dim, I for italic, U for underline,
// and S for strikethrough. A {#RRGGBB} tag marks each foreground-color change;
// {-} marks transparency. For example:
//
//	|{#ff0000}H[B]e[B]l[B]l[B]o[B]|
//
// Only the five listed attributes have flags. Background colors and link IDs
// are not encoded.
func GoldenStyled(buf *buffer.Buffer) string {
	var sb strings.Builder
	w := buf.Width()
	h := buf.Height()
	for y := range h {
		sb.WriteByte('|')
		var lastFg string
		for x := range w {
			cell := buf.Cell(x, y)

			// Emit a foreground tag only when the color changes within this row.
			fg := colorHex(cell.Fg)
			if fg != lastFg {
				sb.WriteString("{")
				sb.WriteString(fg)
				sb.WriteString("}")
				lastFg = fg
			}

			// Represent an unset character as one visible space.
			if cell.Char == 0 {
				sb.WriteRune(' ')
			} else {
				sb.WriteRune(cell.Char)
			}

			// Append stable single-letter flags for base text attributes.
			attrs := cell.Attributes & style.AttributeBaseMask
			if attrs != 0 {
				sb.WriteByte('[')
				if attrs&style.AttrBold != 0 {
					sb.WriteByte('B')
				}
				if attrs&style.AttrDim != 0 {
					sb.WriteByte('D')
				}
				if attrs&style.AttrItalic != 0 {
					sb.WriteByte('I')
				}
				if attrs&style.AttrUnderline != 0 {
					sb.WriteByte('U')
				}
				if attrs&style.AttrStrikethrough != 0 {
					sb.WriteByte('S')
				}
				sb.WriteByte(']')
			}
		}
		sb.WriteString("|\n")
	}
	return sb.String()
}

// colorHex returns an RGB hex string for c, or "-" when c is transparent.
func colorHex(c style.Color) string {
	if c.A == 0 {
		return "-"
	}
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// Golden-file assertions.

// AssertGolden compares GoldenText for buf with testdata/<name>.golden.
//
// The -update test flag creates or replaces the file. Without that flag,
// a missing or different file fails the test and prints the actual output.
func AssertGolden(t *testing.T, name string, buf *buffer.Buffer) {
	t.Helper()
	assertGoldenContent(t, name+".golden", GoldenText(buf))
}

// AssertGoldenStyled compares GoldenStyled for buf with
// testdata/<name>.styled.golden.
func AssertGoldenStyled(t *testing.T, name string, buf *buffer.Buffer) {
	t.Helper()
	assertGoldenContent(t, name+".styled.golden", GoldenStyled(buf))
}

// assertGoldenContent updates or compares one file in the package's testdata directory.
func assertGoldenContent(t *testing.T, filename, actual string) {
	t.Helper()

	goldenPath := filepath.Join("testdata", filename)

	if *updateGolden {
		// Update mode creates the package's testdata directory on demand.
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("failed to create testdata dir: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(actual), 0o644); err != nil {
			t.Fatalf("failed to write golden file %s: %v", goldenPath, err)
		}
		t.Logf("updated golden file: %s", goldenPath)
		return
	}

	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("golden file %s does not exist.\n"+
				"Run with -update to create it.\n"+
				"Actual output:\n%s", goldenPath, actual)
		}
		t.Fatalf("failed to read golden file %s: %v", goldenPath, err)
	}

	if string(expected) != actual {
		t.Errorf("golden file mismatch: %s\n"+
			"Run with -update to regenerate.\n\n"+
			"--- Expected ---\n%s\n"+
			"--- Actual ---\n%s\n"+
			"--- Diff ---\n%s",
			goldenPath, string(expected), actual, lineDiff(string(expected), actual))
	}
}

// GoldenSequence collects named snapshots for one state-transition golden file.
// String labels each snapshot with a header:
//
//	=== step: initial ===
//	|Hello     |
//	|World     |
//	=== step: after scroll ===
//	|World     |
//	|          |
//
// A sequence can capture output before and after any number of state changes.
type GoldenSequence struct {
	steps []goldenStep
}

type goldenStep struct {
	name    string
	content string
}

// NewGoldenSequence returns an empty snapshot sequence.
func NewGoldenSequence() *GoldenSequence {
	return &GoldenSequence{}
}

// Add appends a GoldenText snapshot named name and returns gs.
func (gs *GoldenSequence) Add(name string, buf *buffer.Buffer) *GoldenSequence {
	gs.steps = append(gs.steps, goldenStep{name: name, content: GoldenText(buf)})
	return gs
}

// AddStyled appends a GoldenStyled snapshot named name and returns gs.
func (gs *GoldenSequence) AddStyled(name string, buf *buffer.Buffer) *GoldenSequence {
	gs.steps = append(gs.steps, goldenStep{name: name, content: GoldenStyled(buf)})
	return gs
}

// String serializes the named snapshots in insertion order.
func (gs *GoldenSequence) String() string {
	var sb strings.Builder
	for _, step := range gs.steps {
		sb.WriteString(fmt.Sprintf("=== step: %s ===\n", step.name))
		sb.WriteString(step.content)
	}
	return sb.String()
}

// AssertGoldenSequence compares seq with testdata/<name>.seq.golden.
// The -update test flag creates or replaces the file.
func AssertGoldenSequence(t *testing.T, name string, seq *GoldenSequence) {
	t.Helper()
	assertGoldenContent(t, name+".seq.golden", seq.String())
}

// lineDiff returns each differing line from expected and actual.
func lineDiff(expected, actual string) string {
	expLines := strings.Split(expected, "\n")
	actLines := strings.Split(actual, "\n")

	var sb strings.Builder
	maxLines := max(len(actLines), len(expLines))

	for i := 0; i < maxLines; i++ {
		var exp, act string
		if i < len(expLines) {
			exp = expLines[i]
		}
		if i < len(actLines) {
			act = actLines[i]
		}
		if exp != act {
			sb.WriteString(fmt.Sprintf("  line %d:\n", i+1))
			sb.WriteString(fmt.Sprintf("    - %q\n", exp))
			sb.WriteString(fmt.Sprintf("    + %q\n", act))
		}
	}
	return sb.String()
}
