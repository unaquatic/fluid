package symbol

import (
	"testing"
	"unicode/utf8"
)

// ---------------------------------------------------------------------------
// Border presets
// ---------------------------------------------------------------------------

func TestSingleCharsAreValid(t *testing.T) {
	if SingleChars.TopLeft != '┌' {
		t.Errorf("TopLeft = %q, want '┌'", SingleChars.TopLeft)
	}
	if SingleChars.TopRight != '┐' {
		t.Errorf("TopRight = %q, want '┐'", SingleChars.TopRight)
	}
	if SingleChars.BottomLeft != '└' {
		t.Errorf("BottomLeft = %q, want '└'", SingleChars.BottomLeft)
	}
	if SingleChars.BottomRight != '┘' {
		t.Errorf("BottomRight = %q, want '┘'", SingleChars.BottomRight)
	}
	if SingleChars.Horizontal != '─' {
		t.Errorf("Horizontal = %q, want '─'", SingleChars.Horizontal)
	}
	if SingleChars.Vertical != '│' {
		t.Errorf("Vertical = %q, want '│'", SingleChars.Vertical)
	}
}

func TestDoubleCharsAreValid(t *testing.T) {
	if DoubleChars.TopLeft != '╔' {
		t.Errorf("TopLeft = %q, want '╔'", DoubleChars.TopLeft)
	}
	if DoubleChars.Horizontal != '═' {
		t.Errorf("Horizontal = %q, want '═'", DoubleChars.Horizontal)
	}
	if DoubleChars.Vertical != '║' {
		t.Errorf("Vertical = %q, want '║'", DoubleChars.Vertical)
	}
}

func TestRoundedCharsHaveRoundedCorners(t *testing.T) {
	if RoundedChars.TopLeft != '╭' {
		t.Errorf("TopLeft = %q, want '╭'", RoundedChars.TopLeft)
	}
	if RoundedChars.BottomRight != '╯' {
		t.Errorf("BottomRight = %q, want '╯'", RoundedChars.BottomRight)
	}
}

func TestHeavyCharsAreThick(t *testing.T) {
	if HeavyChars.Horizontal != '━' {
		t.Errorf("Horizontal = %q, want '━'", HeavyChars.Horizontal)
	}
	if HeavyChars.Vertical != '┃' {
		t.Errorf("Vertical = %q, want '┃'", HeavyChars.Vertical)
	}
}

// ---------------------------------------------------------------------------
// Bar levels
// ---------------------------------------------------------------------------

func TestBarNineLevelsLength(t *testing.T) {
	if len(BarNineLevels) != 9 {
		t.Fatalf("BarNineLevels length = %d, want 9", len(BarNineLevels))
	}
	if BarNineLevels[0] != ' ' {
		t.Errorf("BarNineLevels[0] = %q, want ' '", BarNineLevels[0])
	}
	if BarNineLevels[8] != '█' {
		t.Errorf("BarNineLevels[8] = %q, want '█'", BarNineLevels[8])
	}
}

// ---------------------------------------------------------------------------
// Block / Quadrants
// ---------------------------------------------------------------------------

func TestQuadrantCharsLength(t *testing.T) {
	if len(QuadrantChars) != 16 {
		t.Fatalf("QuadrantChars length = %d, want 16", len(QuadrantChars))
	}
	if QuadrantChars[0] != ' ' {
		t.Errorf("QuadrantChars[0] = %q, want ' '", QuadrantChars[0])
	}
	if QuadrantChars[15] != '█' {
		t.Errorf("QuadrantChars[15] = %q, want '█'", QuadrantChars[15])
	}
}

func TestFullBlock(t *testing.T) {
	if FullBlock != '█' {
		t.Errorf("FullBlock = %q, want '█'", FullBlock)
	}
}

// ---------------------------------------------------------------------------
// GrayscaleRamp
// ---------------------------------------------------------------------------

func TestGrayscaleRampNonEmpty(t *testing.T) {
	if len(GrayscaleRamp) == 0 {
		t.Fatal("GrayscaleRamp is empty")
	}
	// First char should be space (lowest intensity)
	runes := []rune(GrayscaleRamp)
	if runes[0] != ' ' {
		t.Errorf("GrayscaleRamp[0] = %q, want ' '", runes[0])
	}
}

// ---------------------------------------------------------------------------
// Scrollbar sets
// ---------------------------------------------------------------------------

func TestScrollBarVerticalThumb(t *testing.T) {
	if ScrollBarVertical.Thumb != '█' {
		t.Errorf("ScrollBarVertical.Thumb = %q, want '█'", ScrollBarVertical.Thumb)
	}
}

func TestScrollBarHorizontalThumb(t *testing.T) {
	if ScrollBarHorizontal.Thumb != '▀' {
		t.Errorf("ScrollBarHorizontal.Thumb = %q, want '▀'", ScrollBarHorizontal.Thumb)
	}
}

// ---------------------------------------------------------------------------
// Shade / Gauge
// ---------------------------------------------------------------------------

func TestShadeConstants(t *testing.T) {
	if ShadeLight != '░' {
		t.Errorf("ShadeLight = %q, want '░'", ShadeLight)
	}
	if ShadeFull != '█' {
		t.Errorf("ShadeFull = %q, want '█'", ShadeFull)
	}
}

func TestGaugeConstants(t *testing.T) {
	if GaugeFill != '█' {
		t.Errorf("GaugeFill = %q, want '█'", GaugeFill)
	}
	if GaugeEmpty != '░' {
		t.Errorf("GaugeEmpty = %q, want '░'", GaugeEmpty)
	}
}

// ---------------------------------------------------------------------------
// Line constants
// ---------------------------------------------------------------------------

func TestLineConstants(t *testing.T) {
	if Horizontal != '─' {
		t.Errorf("Horizontal = %q, want '─'", Horizontal)
	}
	if Vertical != '│' {
		t.Errorf("Vertical = %q, want '│'", Vertical)
	}
}

// ---------------------------------------------------------------------------
// Markers
// ---------------------------------------------------------------------------

func TestMarkerConstants(t *testing.T) {
	if BulletFilled != '●' {
		t.Errorf("BulletFilled = %q, want '●'", BulletFilled)
	}
	if TriangleUp != '▲' {
		t.Errorf("TriangleUp = %q, want '▲'", TriangleUp)
	}
	if TriangleDown != '▼' {
		t.Errorf("TriangleDown = %q, want '▼'", TriangleDown)
	}
	if ArrowUp != '↑' {
		t.Errorf("ArrowUp = %q, want '↑'", ArrowUp)
	}
	if ArrowDown != '↓' {
		t.Errorf("ArrowDown = %q, want '↓'", ArrowDown)
	}
}

func TestTreeIndicators(t *testing.T) {
	if TreeExpanded != "▼" {
		t.Errorf("TreeExpanded = %q, want \"▼\"", TreeExpanded)
	}
	if TreeCollapsed != "▶" {
		t.Errorf("TreeCollapsed = %q, want \"▶\"", TreeCollapsed)
	}
}

// ---------------------------------------------------------------------------
// DOT
// ---------------------------------------------------------------------------

func TestDOT(t *testing.T) {
	if DOT != '·' {
		t.Errorf("DOT = %q, want '·'", DOT)
	}
	if !utf8.ValidRune(DOT) {
		t.Error("DOT is not valid UTF-8")
	}
}
