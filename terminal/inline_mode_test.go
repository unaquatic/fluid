package terminal

import (
	"bytes"
	"strings"
	"testing"
)

// startedTestRenderer returns an inline test renderer that reports completed
// setup, so mode changes reach the terminal.
func startedTestRenderer() (*Renderer, *bytes.Buffer) {
	r, outBuf := newInlineTestRenderer()
	r.controlState = StateAutoStarted
	return r, outBuf
}

func TestSetInlineMode_LeavesAlternateScreen(t *testing.T) {
	r, outBuf := startedTestRenderer()
	r.config.InlineMode = false
	r.config.Fullscreen = true
	r.backend.EnterAlternateScreen()
	outBuf.Reset()

	r.SetInlineMode(true)

	if !strings.Contains(outBuf.String(), "\x1b[?1049l") {
		t.Error("expected the alternate screen to be left")
	}
	if r.backend.AlternateScreen() {
		t.Error("expected AlternateScreen=false")
	}
	if !r.config.InlineMode || r.config.Fullscreen {
		t.Errorf("expected inline config, got InlineMode=%v Fullscreen=%v",
			r.config.InlineMode, r.config.Fullscreen)
	}
}

func TestSetInlineMode_EntersAlternateScreen(t *testing.T) {
	r, outBuf := startedTestRenderer()

	r.SetInlineMode(false)

	if !strings.Contains(outBuf.String(), "\x1b[?1049h") {
		t.Error("expected the alternate screen to be entered")
	}
	if !r.backend.AlternateScreen() {
		t.Error("expected AlternateScreen=true")
	}
	if r.config.InlineMode || !r.config.Fullscreen {
		t.Errorf("expected fullscreen config, got InlineMode=%v Fullscreen=%v",
			r.config.InlineMode, r.config.Fullscreen)
	}
}

func TestSetInlineMode_UnchangedModeWritesNothing(t *testing.T) {
	r, outBuf := startedTestRenderer()

	r.SetInlineMode(true)

	if outBuf.Len() != 0 {
		t.Errorf("expected no output, got %q", outBuf.String())
	}
}

func TestSetInlineMode_BeforeSetupWritesNothing(t *testing.T) {
	r, outBuf := newInlineTestRenderer()

	r.SetInlineMode(false)

	if outBuf.Len() != 0 {
		t.Errorf("expected no output, got %q", outBuf.String())
	}
	if r.config.InlineMode {
		t.Error("expected the mode to be recorded")
	}
}

func TestSetInlineMode_DiscardsTrackedTerminalState(t *testing.T) {
	r, _ := startedTestRenderer()
	r.InlineFlush(rowsBuffer(10, strings.Repeat("A", testTermHeight+5), 5))
	if r.inlineViewportTop == 0 {
		t.Fatal("expected the tall frame to advance the viewport")
	}

	r.SetInlineMode(false)

	if r.inlineCurrentBuf != nil {
		t.Error("expected inlineCurrentBuf=nil")
	}
	if r.inlineViewportTop != 0 || r.inlineCursorRow != 0 || r.inlineTermHeight != 0 {
		t.Errorf("expected the inline anchor to be cleared, got top=%d cursor=%d height=%d",
			r.inlineViewportTop, r.inlineCursorRow, r.inlineTermHeight)
	}
	if !r.invalidated {
		t.Error("expected the next frame to repaint completely")
	}
}

func TestSetInlineMode_FullscreenResizesRoot(t *testing.T) {
	r, _ := startedTestRenderer()
	// An inline frame grows the root to the tall layout height.
	r.renderInlineFrame()

	r.SetInlineMode(false)

	if got := r.root.Height(); got != r.height {
		t.Errorf("expected root height %d, got %d", r.height, got)
	}
}

func TestInlineMode(t *testing.T) {
	r, _ := startedTestRenderer()

	if !r.InlineMode() {
		t.Error("expected inline mode to be reported")
	}

	r.SetInlineMode(false)

	if r.InlineMode() {
		t.Error("expected fullscreen mode to be reported")
	}
}
