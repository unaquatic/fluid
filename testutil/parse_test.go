package testutil

import (
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/style"
)

var (
	green   = style.ColorFromHex("#5af78d")
	surface = style.ColorFromHex("#262b27")
)

// styledBuffer returns a buffer that uses every kind of frame mark.
func styledBuffer() *buffer.Buffer {
	buf := buffer.New(12, 3)
	for x, r := range "say hi" {
		buf.SetCell(x+1, 1, r, green, surface, 0)
	}
	buf.SetCell(8, 1, 'B', style.ColorWhite, style.ColorTransparent, style.AttrBold)
	buf.SetCell(9, 1, 'X', style.ColorWhite, style.ColorTransparent, style.AttrBold|style.AttrItalic)
	buf.SetCell(10, 1, ' ', style.ColorWhite, style.ColorTransparent, style.AttrUnderline)
	buf.SetCell(0, 2, 'T', style.ColorTransparent, style.ColorTransparent, style.AttrDim)
	buf.SetCell(2, 2, '界', green, surface, style.AttrStrikethrough)
	return buf
}

func TestParseFramesRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		buf   *buffer.Buffer
		names map[style.Color]string
	}{
		{name: "styled", buf: styledBuffer(), names: map[style.Color]string{surface: "SuccessSurface", green: "Success"}},
		{name: "unnamed", buf: styledBuffer()},
		{name: "blank", buf: buffer.New(4, 2)},
		{name: "empty", buf: buffer.New(0, 0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := GoldenFrames(tt.buf, tt.names)
			if err != nil {
				t.Fatal(err)
			}
			s, err := ParseFrames(doc)
			if err != nil {
				t.Fatalf("ParseFrames: %v\n%s", err, doc)
			}
			got, err := GoldenFrames(s.Buf, s.Names)
			if err != nil {
				t.Fatal(err)
			}
			if got != doc {
				t.Errorf("round trip changed the document\n--- want ---\n%s--- got ---\n%s", doc, got)
			}
			if !maps.Equal(s.Names, tt.names) && len(s.Names)+len(tt.names) > 0 {
				t.Errorf("names = %v, want %v", s.Names, tt.names)
			}
		})
	}
}

func TestParseSequenceRoundTrip(t *testing.T) {
	names := map[style.Color]string{surface: "SuccessSurface"}
	seq := NewGoldenSequence().
		AddFrames("initial", styledBuffer(), names).
		AddFrames("cleared", buffer.New(12, 3), nil)
	doc := seq.String()

	steps, err := ParseSequence(doc)
	if err != nil {
		t.Fatalf("ParseSequence: %v", err)
	}
	again := NewGoldenSequence()
	for _, s := range steps {
		again.AddFrames(s.Name, s.Buf, s.Names)
	}
	if got := again.String(); got != doc {
		t.Errorf("round trip changed the sequence\n--- want ---\n%s--- got ---\n%s", doc, got)
	}
}

func TestParseFramesErrors(t *testing.T) {
	tests := []struct {
		name, doc, want string
	}{
		{"no text", "=== fg ===\n|A|\n", "no text frame"},
		{"stray line", "hello\n=== text ===\n|a|\n", "before the first section"},
		{"ragged", "=== text ===\n|ab|\n|a|\n", "row 1 is 1 cells wide"},
		{"no delimiters", "=== text ===\nab\n", "lacks its delimiters"},
		{"frame height", "=== text ===\n|a|\n|b|\n=== fg ===\n|A|\n", "fg frame has 1 rows"},
		{"unknown symbol", "=== text ===\n|a|\n=== fg ===\n|A|\n", `symbol 'A' has no color`},
		{"bad color", "=== text ===\n|a|\n=== fg ===\n|A|\n=== legend ===\nA #12\n", `color "#12"`},
		{"missing foreground", "=== text ===\n|a|\n", "foreground"},
		{"unknown attribute", "=== text ===\n| |\n=== attrs ===\n|Q|\n", `unknown attribute 'Q'`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFrames(tt.doc)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("ParseFrames error = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}

// TestParseCommittedGoldens checks that every widget golden file parses and
// that formatting the result reproduces the file exactly.
func TestParseCommittedGoldens(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "widget", "testdata", "*.golden"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no golden files found")
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			doc := string(data)
			got, err := reformat(path, doc)
			if err != nil {
				t.Fatal(err)
			}
			if got != doc {
				t.Errorf("reformatting changed %s\n%s", path, lineDiff(doc, got))
			}
		})
	}
}

// reformat parses the golden file at path and formats it again.
func reformat(path, doc string) (string, error) {
	if !strings.HasSuffix(path, ".seq.golden") {
		s, err := ParseFrames(doc)
		if err != nil {
			return "", err
		}
		return GoldenFrames(s.Buf, s.Names)
	}
	steps, err := ParseSequence(doc)
	if err != nil {
		return "", err
	}
	seq := NewGoldenSequence()
	for _, s := range steps {
		seq.AddFrames(s.Name, s.Buf, s.Names)
	}
	return seq.String(), nil
}
