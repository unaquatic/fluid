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

// colorHex returns an RGB hex string for c, or "-" when c is transparent.
func colorHex(c style.Color) string {
	if c.A == 0 {
		return "-"
	}
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// Golden-file assertions.

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

// GoldenSequence collects named GoldenFrames snapshots for one
// state-transition golden file. String labels each snapshot with a header:
//
//	=== step: initial ===
//	=== text ===
//	|Hello     |
//	=== fg ===
//	|AAAAA     |
//	=== legend ===
//	A #ffffff
//	=== step: after scroll ===
//	=== text ===
//	|World     |
//	...
//
// A sequence can capture output before and after any number of state changes.
type GoldenSequence struct {
	steps []goldenStep
}

type goldenStep struct {
	name    string
	content string
	err     error
}

// NewGoldenSequence returns an empty snapshot sequence.
func NewGoldenSequence() *GoldenSequence {
	return &GoldenSequence{}
}

// AddFrames appends a GoldenFrames snapshot of buf named name and returns gs.
// names labels colors in the snapshot's legend and may be nil. An error
// building the snapshot fails AssertGoldenSequence.
func (gs *GoldenSequence) AddFrames(name string, buf *buffer.Buffer, names map[style.Color]string) *GoldenSequence {
	content, err := GoldenFrames(buf, names)
	gs.steps = append(gs.steps, goldenStep{name: name, content: content, err: err})
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
	for _, step := range seq.steps {
		if step.err != nil {
			t.Fatalf("golden sequence %s, step %s: %v", name, step.name, step.err)
		}
	}
	assertGoldenContent(t, name+".seq.golden", seq.String())
}

// lineDiff returns each differing line from expected and actual.
func lineDiff(expected, actual string) string {
	expLines := strings.Split(expected, "\n")
	actLines := strings.Split(actual, "\n")

	var sb strings.Builder
	maxLines := max(len(actLines), len(expLines))

	for i := range maxLines {
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
