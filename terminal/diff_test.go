package terminal

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/textbuf"
)

func newTestRenderer(w, h int) (*Renderer, *bytes.Buffer) {
	var outBuf bytes.Buffer
	r := &Renderer{
		backend:       NewBackend(strings.NewReader(""), &outBuf),
		currentBuffer: buffer.New(w, h),
		nextBuffer:    buffer.New(w, h),
		done:          make(chan struct{}),
		linkRegistry:  newLinkRegistry(),
	}
	return r, &outBuf
}

func TestBuildSGR_Reset(t *testing.T) {
	sgr := buildSGR(style.Color{}, style.Color{}, 0, style.ColorTrueColor, style.Indexers{})
	// Every generated style begins by resetting prior SGR state.
	if !strings.HasPrefix(sgr, "\x1b[0") {
		t.Errorf("expected SGR to start with reset, got: %q", sgr)
	}
	if !strings.HasSuffix(sgr, "m") {
		t.Errorf("expected SGR to end with 'm', got: %q", sgr)
	}
}

func TestBuildSGR_Bold(t *testing.T) {
	sgr := buildSGR(style.ColorWhite, style.ColorBlack, style.AttrBold, style.ColorTrueColor, style.Indexers{})
	if !strings.Contains(sgr, ";1;") && !strings.Contains(sgr, ";1m") {
		t.Errorf("expected bold attribute in SGR, got: %q", sgr)
	}
}

func TestBuildSGR_Italic(t *testing.T) {
	sgr := buildSGR(style.ColorWhite, style.ColorBlack, style.AttrItalic, style.ColorTrueColor, style.Indexers{})
	if !strings.Contains(sgr, ";3;") && !strings.Contains(sgr, ";3m") {
		t.Errorf("expected italic attribute in SGR, got: %q", sgr)
	}
}

func TestBuildSGR_Underline(t *testing.T) {
	sgr := buildSGR(style.ColorWhite, style.ColorBlack, style.AttrUnderline, style.ColorTrueColor, style.Indexers{})
	if !strings.Contains(sgr, ";4;") {
		t.Errorf("expected underline attribute in SGR, got: %q", sgr)
	}
}

func TestBuildSGR_Strikethrough(t *testing.T) {
	sgr := buildSGR(style.ColorWhite, style.ColorBlack, style.AttrStrikethrough, style.ColorTrueColor, style.Indexers{})
	if !strings.Contains(sgr, ";9;") {
		t.Errorf("expected strikethrough attribute in SGR, got: %q", sgr)
	}
}

func TestBuildSGR_ForegroundColor(t *testing.T) {
	sgr := buildSGR(style.ColorRed, style.ColorTransparent, 0, style.ColorTrueColor, style.Indexers{})
	// Normalized red converts to RGB 255,0,0.
	if !strings.Contains(sgr, "38;2;255;0;0") {
		t.Errorf("expected fg truecolor in SGR, got: %q", sgr)
	}
}

func TestBuildSGR_BackgroundColor(t *testing.T) {
	sgr := buildSGR(style.ColorWhite, style.ColorBlue, 0, style.ColorTrueColor, style.Indexers{})
	// Normalized blue converts to RGB 0,0,255.
	if !strings.Contains(sgr, "48;2;0;0;255") {
		t.Errorf("expected bg truecolor in SGR, got: %q", sgr)
	}
}

func TestBuildSGR_TransparentBgSkipped(t *testing.T) {
	sgr := buildSGR(style.ColorWhite, style.ColorTransparent, 0, style.ColorTrueColor, style.Indexers{})
	if strings.Contains(sgr, "48;2") {
		t.Errorf("expected transparent bg to be skipped in SGR, got: %q", sgr)
	}
}

func TestBuildSGR_MultipleAttributes(t *testing.T) {
	sgr := buildSGR(style.ColorWhite, style.ColorBlack, style.AttrBold|style.AttrItalic|style.AttrUnderline, style.ColorTrueColor, style.Indexers{})
	if !strings.Contains(sgr, ";1;") {
		t.Error("expected bold")
	}
	if !strings.Contains(sgr, ";3;") {
		t.Error("expected italic")
	}
	if !strings.Contains(sgr, ";4;") {
		t.Error("expected underline")
	}
}

func TestDiffAndFlush_UnchangedCellsSkipped(t *testing.T) {
	r, outBuf := newTestRenderer(10, 3)

	// Identical buffers require no cell output.
	r.currentBuffer.SetCell(0, 0, 'A', style.ColorWhite, style.ColorBlack, 0)
	r.nextBuffer.SetCell(0, 0, 'A', style.ColorWhite, style.ColorBlack, 0)

	r.diffAndFlush(r.currentBuffer, r.nextBuffer)

	output := outBuf.String()
	// The unchanged glyph does not appear in the output.
	if strings.Contains(output, "A") {
		t.Error("unchanged cell 'A' should not appear in output")
	}
}

func TestDiffAndFlush_ChangedCellsWritten(t *testing.T) {
	r, outBuf := newTestRenderer(10, 3)

	// A changed cell produces terminal output.
	r.nextBuffer.SetCell(0, 0, 'X', style.ColorWhite, style.ColorBlack, 0)

	r.diffAndFlush(r.currentBuffer, r.nextBuffer)

	output := outBuf.String()
	if !strings.Contains(output, "X") {
		t.Error("changed cell 'X' should appear in output")
	}
}

func TestDiffAndFlush_CursorPositioning(t *testing.T) {
	r, outBuf := newTestRenderer(10, 5)

	// Place one changed cell at zero-based (5,3).
	r.nextBuffer.SetCell(5, 3, 'Z', style.ColorWhite, style.ColorBlack, 0)

	r.diffAndFlush(r.currentBuffer, r.nextBuffer)

	output := outBuf.String()
	// CUP converts it to one-based row 4, column 6.
	if !strings.Contains(output, "\x1b[4;6H") {
		t.Errorf("expected cursor position [4;6H, got: %q", output)
	}
}

func TestDiffAndFlush_WideCharContinuationSkipped(t *testing.T) {
	r, outBuf := newTestRenderer(10, 1)

	// Set one wide CJK character.
	r.nextBuffer.SetCell(0, 0, '\u4e16', style.ColorWhite, style.ColorBlack, 0) // 世 occupies two cells.

	r.diffAndFlush(r.currentBuffer, r.nextBuffer)

	output := outBuf.String()
	// The head emits the wide character once.
	if !strings.Contains(output, "世") {
		t.Error("expected wide character in output")
	}
}

func TestDiffAndFlush_ResetAtEnd(t *testing.T) {
	r, outBuf := newTestRenderer(5, 1)

	r.nextBuffer.SetCell(0, 0, 'A', style.ColorWhite, style.ColorBlack, 0)
	r.diffAndFlush(r.currentBuffer, r.nextBuffer)

	output := outBuf.String()
	if !strings.HasSuffix(output, "\x1b[0m") {
		t.Error("expected reset at end of output")
	}
}

func TestDiffAndFlush_MultipleCellsDifferentPositions(t *testing.T) {
	r, outBuf := newTestRenderer(10, 5)

	// Noncontiguous changes require separate cursor positions.
	r.nextBuffer.SetCell(0, 0, 'A', style.ColorRed, style.ColorBlack, 0)
	r.nextBuffer.SetCell(5, 3, 'B', style.ColorBlue, style.ColorBlack, 0)

	r.diffAndFlush(r.currentBuffer, r.nextBuffer)

	output := outBuf.String()
	if !strings.Contains(output, "A") {
		t.Error("expected 'A' in output")
	}
	if !strings.Contains(output, "B") {
		t.Error("expected 'B' in output")
	}
}

func TestFullFlush_WritesAllCells(t *testing.T) {
	r, outBuf := newTestRenderer(3, 2)

	r.nextBuffer.SetCell(0, 0, 'A', style.ColorWhite, style.ColorBlack, 0)
	r.nextBuffer.SetCell(1, 0, 'B', style.ColorWhite, style.ColorBlack, 0)
	r.nextBuffer.SetCell(2, 0, 'C', style.ColorWhite, style.ColorBlack, 0)
	r.nextBuffer.SetCell(0, 1, 'D', style.ColorWhite, style.ColorBlack, 0)

	r.FullFlush(r.nextBuffer)

	output := outBuf.String()
	for _, ch := range []string{"A", "B", "C", "D"} {
		if !strings.Contains(output, ch) {
			t.Errorf("expected '%s' in full flush output", ch)
		}
	}
	if !strings.HasSuffix(output, "\x1b[0m") {
		t.Error("expected reset at end")
	}
}

func TestRendererStats_Record(t *testing.T) {
	s := newStats(10)
	s.record(16 * time.Millisecond) // Approximately one 60-FPS frame.
	s.record(16 * time.Millisecond)
	s.record(16 * time.Millisecond)

	if s.FrameCount != 3 {
		t.Errorf("expected FrameCount=3, got %d", s.FrameCount)
	}
	if s.LastRenderTime != 16*time.Millisecond {
		t.Errorf("expected LastRenderTime=16ms, got %v", s.LastRenderTime)
	}
	if s.FPS < 50 || s.FPS > 70 {
		t.Errorf("expected FPS around 60, got %f", s.FPS)
	}
}

func TestRendererStats_MaxRenderTime(t *testing.T) {
	s := newStats(10)
	s.record(10 * time.Millisecond)
	s.record(50 * time.Millisecond)
	s.record(20 * time.Millisecond)

	if s.MaxRenderTime != 50*time.Millisecond {
		t.Errorf("expected MaxRenderTime=50ms, got %v", s.MaxRenderTime)
	}
}

func TestRendererStats_Reset(t *testing.T) {
	s := newStats(10)
	s.record(16 * time.Millisecond)
	s.record(16 * time.Millisecond)
	s.reset()

	if s.FrameCount != 0 {
		t.Errorf("expected FrameCount=0 after reset, got %d", s.FrameCount)
	}
	if s.FPS != 0 {
		t.Errorf("expected FPS=0 after reset, got %f", s.FPS)
	}
	if s.MaxRenderTime != 0 {
		t.Errorf("expected MaxRenderTime=0 after reset, got %v", s.MaxRenderTime)
	}
}

func TestRendererStats_SlidingWindow(t *testing.T) {
	s := newStats(3)

	s.record(10 * time.Millisecond)
	s.record(20 * time.Millisecond)
	s.record(30 * time.Millisecond)
	// The first three samples average 20ms.
	if s.AvgRenderTime != 20*time.Millisecond {
		t.Errorf("expected AvgRenderTime=20ms, got %v", s.AvgRenderTime)
	}

	s.record(40 * time.Millisecond)
	// Adding a fourth sample evicts the oldest and produces a 30ms average.
	if s.AvgRenderTime != 30*time.Millisecond {
		t.Errorf("expected AvgRenderTime=30ms after slide, got %v", s.AvgRenderTime)
	}
}

// TestFlushRune_BrokenPair verifies that malformed wide-character storage emits
// spaces that preserve the row's recorded width. The test bypasses normal
// buffer repair to construct each malformed case.
func TestFlushRune_BrokenPair(t *testing.T) {
	const width = 4

	tests := []struct {
		name  string
		setup func(b *buffer.Buffer)
		want  string
	}{
		{
			name: "intact pair emits head only",
			setup: func(b *buffer.Buffer) {
				b.SetCellDirect(1, 0, buffer.Cell{Char: '\u4e16', Width: 2})
				b.SetCellDirect(2, 0, buffer.Cell{Char: 0, Width: -1})
			},
			want: " \u4e16 ",
		},
		{
			name: "stranded continuation claims its column",
			setup: func(b *buffer.Buffer) {
				b.SetCellDirect(2, 0, buffer.Cell{Char: 0, Width: -1})
			},
			want: "    ",
		},
		{
			name: "stranded head gives up its second column",
			setup: func(b *buffer.Buffer) {
				b.SetCellDirect(1, 0, buffer.Cell{Char: '\u4e16', Width: 2})
				b.SetCellDirect(2, 0, buffer.Cell{Char: 'X', Width: 1})
			},
			want: "  X ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := buffer.New(width, 1)
			for x := range width {
				b.SetCellDirect(x, 0, buffer.Cell{Char: ' ', Width: 1})
			}
			tt.setup(b)

			var got strings.Builder
			cols := 0
			for x := range width {
				c := b.Cell(x, 0)
				ch, ok := flushRune(b, x, 0, c)
				if !ok {
					continue
				}
				got.WriteRune(ch)
				if ch == c.Char {
					got.WriteString(c.Combining())
				}
				cols += textbuf.RuneWidth(ch)
			}
			if got.String() != tt.want {
				t.Errorf("emitted %q, want %q", got.String(), tt.want)
			}
			if cols != width {
				t.Errorf("row advances %d columns, want %d", cols, width)
			}
		})
	}
}
