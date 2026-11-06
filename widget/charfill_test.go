package widget

import (
	"testing"

	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/testutil"
)

// TestCharFill_SetBg checks that SetBg changes the background the fill draws
// after it was made, and that Bg reports it.
func TestCharFill_SetBg(t *testing.T) {
	bg := style.ColorFromHex("#202020")
	c := NewCharFill(' ', style.ColorWhite, style.ColorFromHex("#ffffff"))
	c.SetBg(bg)
	if c.Bg() != bg {
		t.Fatalf("Bg() = %v, want %v", c.Bg(), bg)
	}
	buf := testutil.RenderWidget(c, 3, 1)
	for x := range 3 {
		if got := buf.Cell(x, 0).Bg; got != bg {
			t.Errorf("cell %d bg = %v, want %v", x, got, bg)
		}
	}
}
