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

// updateGolden is set by `go test -update` to regenerate golden files.
var updateGolden = flag.Bool("update", false, "update golden test files")

// ---------------------------------------------------------------------------
// Buffer serialisation formats
// ---------------------------------------------------------------------------

// GoldenText converts a buffer to a deterministic plain-text representation
// suitable for golden file comparison. Each row is pipe-delimited to make
// trailing spaces visible. The format is:
//
//	|Hello     |
//	|World     |
//
// This captures the exact character grid without style information.
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

// GoldenStyled converts a buffer to a styled text representation that encodes
// character, foreground color, and attributes. Each cell is encoded as:
//
//	char[attrs] where attrs are single-letter flags: B=bold, I=italic, U=underline,
//	D=dim, S=strikethrough. Foreground color changes are marked inline as {#RRGGBB}.
//
// Example row: {#ff0000}H[B]e[B]l[B]l[B]o[B]
//
// This format captures style information for regression testing while remaining
// human-readable.
func GoldenStyled(buf *buffer.Buffer) string {
	var sb strings.Builder
	w := buf.Width()
	h := buf.Height()
	for y := range h {
		sb.WriteByte('|')
		var lastFg string
		for x := range w {
			cell := buf.Cell(x, y)

			// Emit fg color tag when it changes
			fg := colorHex(cell.Fg)
			if fg != lastFg {
				sb.WriteString("{")
				sb.WriteString(fg)
				sb.WriteString("}")
				lastFg = fg
			}

			// Emit character
			if cell.Char == 0 {
				sb.WriteRune(' ')
			} else {
				sb.WriteRune(cell.Char)
			}

			// Emit attribute flags
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

// colorHex returns a hex color string like "#ff0000" for a Color value,
// or "-" for transparent/zero alpha.
func colorHex(c style.Color) string {
	if c.A == 0 {
		return "-"
	}
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// ---------------------------------------------------------------------------
// Single-snapshot assertions
// ---------------------------------------------------------------------------

// AssertGolden compares the buffer's plain-text rendering against a golden file
// named "<name>.golden" in the testdata/ directory relative to the test file.
//
// If the -update flag is set, the golden file is written/updated instead.
// If the golden file does not exist and -update is not set, the test fails
// and prints the actual output so you can review and create it.
func AssertGolden(t *testing.T, name string, buf *buffer.Buffer) {
	t.Helper()
	assertGoldenContent(t, name+".golden", GoldenText(buf))
}

// AssertGoldenStyled is like AssertGolden but uses the styled format that
// encodes fg color and text attributes.
func AssertGoldenStyled(t *testing.T, name string, buf *buffer.Buffer) {
	t.Helper()
	assertGoldenContent(t, name+".styled.golden", GoldenStyled(buf))
}

// assertGoldenContent is the shared implementation for golden file comparison.
func assertGoldenContent(t *testing.T, filename, actual string) {
	t.Helper()

	goldenPath := filepath.Join("testdata", filename)

	if *updateGolden {
		// Create testdata/ directory if it doesn't exist
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

// ---------------------------------------------------------------------------
// Multi-step sequence assertions
// ---------------------------------------------------------------------------

// GoldenSequence captures multiple named buffer snapshots into a single
// golden file. Each step is labeled with a header line like:
//
//	=== step: initial ===
//	|Hello     |
//	|World     |
//	=== step: after scroll ===
//	|World     |
//	|          |
//
// This enables testing state transitions: render → mutate → render again.
type GoldenSequence struct {
	steps []goldenStep
}

type goldenStep struct {
	name    string
	content string
}

// NewGoldenSequence creates an empty sequence builder.
func NewGoldenSequence() *GoldenSequence {
	return &GoldenSequence{}
}

// Add captures a plain-text buffer snapshot with the given step name.
func (gs *GoldenSequence) Add(name string, buf *buffer.Buffer) *GoldenSequence {
	gs.steps = append(gs.steps, goldenStep{name: name, content: GoldenText(buf)})
	return gs
}

// AddStyled captures a styled buffer snapshot with the given step name.
func (gs *GoldenSequence) AddStyled(name string, buf *buffer.Buffer) *GoldenSequence {
	gs.steps = append(gs.steps, goldenStep{name: name, content: GoldenStyled(buf)})
	return gs
}

// String serializes all steps into a single string for golden file comparison.
func (gs *GoldenSequence) String() string {
	var sb strings.Builder
	for _, step := range gs.steps {
		sb.WriteString(fmt.Sprintf("=== step: %s ===\n", step.name))
		sb.WriteString(step.content)
	}
	return sb.String()
}

// AssertGoldenSequence compares a multi-step sequence against a golden file.
// Use -update to regenerate.
func AssertGoldenSequence(t *testing.T, name string, seq *GoldenSequence) {
	t.Helper()
	assertGoldenContent(t, name+".seq.golden", seq.String())
}

// ---------------------------------------------------------------------------
// Diff helper
// ---------------------------------------------------------------------------

// lineDiff produces a simple line-by-line diff between two strings.
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
