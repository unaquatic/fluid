package terminal

import (
	"bytes"
	"strings"
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/style"
)

// ---------------------------------------------------------------------------
// Helper: create a renderer with a captured output buffer
// ---------------------------------------------------------------------------

func newTestRenderer(w, h int) (*Renderer, *bytes.Buffer) {
	var outBuf bytes.Buffer
	r := &Renderer{
		backend:       NewBackend(strings.NewReader(""), &outBuf),
		currentBuffer: buffer.New(w, h),
		nextBuffer:    buffer.New(w, h),
		done:          make(chan struct{}),
		linkRegistry:  NewLinkRegistry(),
	}
	return r, &outBuf
}

// ---------------------------------------------------------------------------
// buildSGR tests
// ---------------------------------------------------------------------------

func TestBuildSGR_Reset(t *testing.T) {
	sgr := buildSGR(style.Color{}, style.Color{}, 0, style.ColorTrueColor)
	// Should start with reset
	if !strings.HasPrefix(sgr, "\x1b[0") {
		t.Errorf("expected SGR to start with reset, got: %q", sgr)
	}
	if !strings.HasSuffix(sgr, "m") {
		t.Errorf("expected SGR to end with 'm', got: %q", sgr)
	}
}

func TestBuildSGR_Bold(t *testing.T) {
	sgr := buildSGR(style.ColorWhite, style.ColorBlack, style.AttrBold, style.ColorTrueColor)
	if !strings.Contains(sgr, ";1;") && !strings.Contains(sgr, ";1m") {
		t.Errorf("expected bold attribute in SGR, got: %q", sgr)
	}
}

func TestBuildSGR_Italic(t *testing.T) {
	sgr := buildSGR(style.ColorWhite, style.ColorBlack, style.AttrItalic, style.ColorTrueColor)
	if !strings.Contains(sgr, ";3;") && !strings.Contains(sgr, ";3m") {
		t.Errorf("expected italic attribute in SGR, got: %q", sgr)
	}
}

func TestBuildSGR_Underline(t *testing.T) {
	sgr := buildSGR(style.ColorWhite, style.ColorBlack, style.AttrUnderline, style.ColorTrueColor)
	if !strings.Contains(sgr, ";4;") {
		t.Errorf("expected underline attribute in SGR, got: %q", sgr)
	}
}

func TestBuildSGR_Strikethrough(t *testing.T) {
	sgr := buildSGR(style.ColorWhite, style.ColorBlack, style.AttrStrikethrough, style.ColorTrueColor)
	if !strings.Contains(sgr, ";9;") {
		t.Errorf("expected strikethrough attribute in SGR, got: %q", sgr)
	}
}

func TestBuildSGR_ForegroundColor(t *testing.T) {
	sgr := buildSGR(style.ColorRed, style.ColorTransparent, 0, style.ColorTrueColor)
	// Red = (1,0,0,1) -> (255,0,0)
	if !strings.Contains(sgr, "38;2;255;0;0") {
		t.Errorf("expected fg truecolor in SGR, got: %q", sgr)
	}
}

func TestBuildSGR_BackgroundColor(t *testing.T) {
	sgr := buildSGR(style.ColorWhite, style.ColorBlue, 0, style.ColorTrueColor)
	// Blue = (0,0,1,1) -> (0,0,255)
	if !strings.Contains(sgr, "48;2;0;0;255") {
		t.Errorf("expected bg truecolor in SGR, got: %q", sgr)
	}
}

func TestBuildSGR_TransparentBgSkipped(t *testing.T) {
	sgr := buildSGR(style.ColorWhite, style.ColorTransparent, 0, style.ColorTrueColor)
	if strings.Contains(sgr, "48;2") {
		t.Errorf("expected transparent bg to be skipped in SGR, got: %q", sgr)
	}
}

func TestBuildSGR_MultipleAttributes(t *testing.T) {
	sgr := buildSGR(style.ColorWhite, style.ColorBlack, style.AttrBold|style.AttrItalic|style.AttrUnderline, style.ColorTrueColor)
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

// ---------------------------------------------------------------------------
// DiffAndFlush tests
// ---------------------------------------------------------------------------

func TestDiffAndFlush_UnchangedCellsSkipped(t *testing.T) {
	r, outBuf := newTestRenderer(10, 3)

	// Set identical content in both buffers
	r.currentBuffer.SetCell(0, 0, 'A', style.ColorWhite, style.ColorBlack, 0)
	r.nextBuffer.SetCell(0, 0, 'A', style.ColorWhite, style.ColorBlack, 0)

	r.DiffAndFlush(r.currentBuffer, r.nextBuffer)

	output := outBuf.String()
	// Should only have the reset at end, no content to write
	if strings.Contains(output, "A") {
		t.Error("unchanged cell 'A' should not appear in output")
	}
}

func TestDiffAndFlush_ChangedCellsWritten(t *testing.T) {
	r, outBuf := newTestRenderer(10, 3)

	// Current is empty, next has content
	r.nextBuffer.SetCell(0, 0, 'X', style.ColorWhite, style.ColorBlack, 0)

	r.DiffAndFlush(r.currentBuffer, r.nextBuffer)

	output := outBuf.String()
	if !strings.Contains(output, "X") {
		t.Error("changed cell 'X' should appear in output")
	}
}

func TestDiffAndFlush_CursorPositioning(t *testing.T) {
	r, outBuf := newTestRenderer(10, 5)

	// Set a cell at (5, 3)
	r.nextBuffer.SetCell(5, 3, 'Z', style.ColorWhite, style.ColorBlack, 0)

	r.DiffAndFlush(r.currentBuffer, r.nextBuffer)

	output := outBuf.String()
	// Should contain cursor move to row 4, col 6 (1-indexed)
	if !strings.Contains(output, "\x1b[4;6H") {
		t.Errorf("expected cursor position [4;6H, got: %q", output)
	}
}

func TestDiffAndFlush_WideCharContinuationSkipped(t *testing.T) {
	r, outBuf := newTestRenderer(10, 1)

	// Set a wide character (e.g. CJK)
	r.nextBuffer.SetCell(0, 0, '\u4e16', style.ColorWhite, style.ColorBlack, 0) // '世' is wide

	r.DiffAndFlush(r.currentBuffer, r.nextBuffer)

	output := outBuf.String()
	// Should contain the wide char
	if !strings.Contains(output, "世") {
		t.Error("expected wide character in output")
	}
}

func TestDiffAndFlush_ResetAtEnd(t *testing.T) {
	r, outBuf := newTestRenderer(5, 1)

	r.nextBuffer.SetCell(0, 0, 'A', style.ColorWhite, style.ColorBlack, 0)
	r.DiffAndFlush(r.currentBuffer, r.nextBuffer)

	output := outBuf.String()
	if !strings.HasSuffix(output, "\x1b[0m") {
		t.Error("expected reset at end of output")
	}
}

func TestDiffAndFlush_MultipleCellsDifferentPositions(t *testing.T) {
	r, outBuf := newTestRenderer(10, 5)

	// Two cells in different locations
	r.nextBuffer.SetCell(0, 0, 'A', style.ColorRed, style.ColorBlack, 0)
	r.nextBuffer.SetCell(5, 3, 'B', style.ColorBlue, style.ColorBlack, 0)

	r.DiffAndFlush(r.currentBuffer, r.nextBuffer)

	output := outBuf.String()
	if !strings.Contains(output, "A") {
		t.Error("expected 'A' in output")
	}
	if !strings.Contains(output, "B") {
		t.Error("expected 'B' in output")
	}
}

// ---------------------------------------------------------------------------
// FullFlush tests
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Stats tests
// ---------------------------------------------------------------------------

func TestRendererStats_Record(t *testing.T) {
	s := NewStats(10)
	s.record(0.016) // ~60fps frame
	s.record(0.016)
	s.record(0.016)

	if s.FrameCount != 3 {
		t.Errorf("expected FrameCount=3, got %d", s.FrameCount)
	}
	if s.LastRenderTime != 0.016 {
		t.Errorf("expected LastRenderTime=0.016, got %f", s.LastRenderTime)
	}
	if s.FPS < 50 || s.FPS > 70 {
		t.Errorf("expected FPS around 60, got %f", s.FPS)
	}
}

func TestRendererStats_MaxRenderTime(t *testing.T) {
	s := NewStats(10)
	s.record(0.010)
	s.record(0.050)
	s.record(0.020)

	if s.MaxRenderTime != 0.050 {
		t.Errorf("expected MaxRenderTime=0.050, got %f", s.MaxRenderTime)
	}
}

func TestRendererStats_Reset(t *testing.T) {
	s := NewStats(10)
	s.record(0.016)
	s.record(0.016)
	s.reset()

	if s.FrameCount != 0 {
		t.Errorf("expected FrameCount=0 after reset, got %d", s.FrameCount)
	}
	if s.FPS != 0 {
		t.Errorf("expected FPS=0 after reset, got %f", s.FPS)
	}
	if s.MaxRenderTime != 0 {
		t.Errorf("expected MaxRenderTime=0 after reset, got %f", s.MaxRenderTime)
	}
}

func TestRendererStats_SlidingWindow(t *testing.T) {
	s := NewStats(3)

	s.record(0.010)
	s.record(0.020)
	s.record(0.030)
	// Window: [0.010, 0.020, 0.030], avg = 0.020
	if s.AvgRenderTime < 0.019 || s.AvgRenderTime > 0.021 {
		t.Errorf("expected AvgRenderTime ~0.020, got %f", s.AvgRenderTime)
	}

	s.record(0.040)
	// Window: [0.020, 0.030, 0.040], avg = 0.030
	if s.AvgRenderTime < 0.029 || s.AvgRenderTime > 0.031 {
		t.Errorf("expected AvgRenderTime ~0.030 after slide, got %f", s.AvgRenderTime)
	}
}
