package terminal

import (
	"fmt"
	"slices"
	"testing"
)

func TestParseCPR(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		wantCols []int
		wantRest string
	}{
		{
			name:     "single report",
			in:       "\x1b[1;3R",
			wantCols: []int{3},
		},
		{
			name:     "batched reports",
			in:       "\x1b[1;2R\x1b[1;3R\x1b[1;9R",
			wantCols: []int{2, 3, 9},
		},
		{
			name:     "multi digit column",
			in:       "\x1b[24;132R",
			wantCols: []int{132},
		},
		{
			name: "no reports",
			in:   "",
		},
		{
			// Discard ordinary input received during probing.
			name:     "keystrokes discarded",
			in:       "q\x1b[1;3Rabc\x1b[1;5R",
			wantCols: []int{3, 5},
		},
		{
			// Do not interpret unrelated CSI input as a cursor position report.
			name:     "other csi ignored",
			in:       "\x1b[?62;c\x1b[1;4R",
			wantCols: []int{4},
		},
		{
			name:     "partial report held back",
			in:       "\x1b[1;4R\x1b[1;1",
			wantCols: []int{4},
			wantRest: "\x1b[1;1",
		},
		{
			name: "report with no column",
			in:   "\x1b[5R",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cols, rest := parseCPR([]byte(tt.in))
			if !slices.Equal(cols, tt.wantCols) {
				t.Errorf("columns = %v, want %v", cols, tt.wantCols)
			}
			if string(rest) != tt.wantRest {
				t.Errorf("rest = %q, want %q", rest, tt.wantRest)
			}
		})
	}
}

// TestParseCPR_SplitAcrossReads verifies that a partial trailing report is
// retained for the next read.
func TestParseCPR_SplitAcrossReads(t *testing.T) {
	const reports = "q\x1b[1;12R\x1b[24;132R"
	for split := range len(reports) + 1 {
		t.Run(fmt.Sprintf("split %d", split), func(t *testing.T) {
			cols, rest := parseCPR([]byte(reports[:split]))
			more, rest := parseCPR(append(rest, reports[split:]...))
			if got := append(cols, more...); !slices.Equal(got, []int{12, 132}) {
				t.Errorf("columns = %v, want [12 132]", got)
			}
			if len(rest) != 0 {
				t.Errorf("rest = %q, want empty", rest)
			}
		})
	}
}

func TestParseCPR_ByteAtATime(t *testing.T) {
	var cols []int
	var rest []byte
	for _, b := range []byte("\x1b[1;12R") {
		var more []int
		more, rest = parseCPR(append(rest, b))
		cols = append(cols, more...)
	}
	if !slices.Equal(cols, []int{12}) || len(rest) != 0 {
		t.Errorf("parseCPR = %v, %q; want [12], empty remainder", cols, rest)
	}
}

func TestWidthProbeString(t *testing.T) {
	tests := []struct {
		probe WidthProbe
		want  string
	}{
		{WidthProbeOff, "off"},
		{WidthProbeAuto, "auto"},
		{WidthProbe(9), "WidthProbe(9)"},
	}
	for _, tt := range tests {
		if got := tt.probe.String(); got != tt.want {
			t.Errorf("WidthProbe(%d).String() = %q, want %q", uint8(tt.probe), got, tt.want)
		}
	}
}

func TestWidthProbeValid(t *testing.T) {
	if !WidthProbeOff.Valid() || !WidthProbeAuto.Valid() {
		t.Error("defined values must be valid")
	}
	if WidthProbe(9).Valid() {
		t.Error("undefined value must not be valid")
	}
}

// TestProbeWidths_NonTerminalStdin verifies that probing rejects a nonterminal
// reader without blocking.
func TestProbeWidths_NonTerminalStdin(t *testing.T) {
	r, _ := newTestRenderer(10, 1)
	if _, err := r.backend.ProbeWidths([]string{"A"}); err == nil {
		t.Error("expected an error for non-terminal stdin")
	}
}

func TestProbeWidths_NoSamples(t *testing.T) {
	r, _ := newTestRenderer(10, 1)
	widths, err := r.backend.ProbeWidths(nil)
	if err != nil || widths != nil {
		t.Errorf("ProbeWidths(nil) = %v, %v; want nil, nil", widths, err)
	}
}
