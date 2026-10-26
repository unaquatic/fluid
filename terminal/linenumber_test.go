package terminal

import (
	"fmt"
	"strings"
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/textbuf"
	"github.com/unaquatic/fluid/widget"
)

// codeLines returns n lines of code, "line 0" to "line n-1".
func codeLines(n int) string {
	src := make([]string, n)
	for i := range n {
		src[i] = fmt.Sprintf("line %d", i)
	}
	return strings.Join(src, "\n")
}

// newGutterScene lays a gutter for a Code widget out in a row before the code,
// as a standalone LineNumber is used, and paints a first full frame.
//
// It returns two functions. frame paints the next frame as the inline renderer
// does, clearing and repainting only the damage, and returns what the terminal
// shows. full paints the same scene in full onto a blank screen, which is what
// frame must match.
func newGutterScene(t *testing.T, code *widget.Code, ln *widget.LineNumber) (frame, full func() string) {
	t.Helper()
	const width, height = 30, 4
	row := widget.NewBox(
		widget.BoxWithLayout(layout.WithFlexDirection(layout.DirectionRow), layout.WithFlexGrow(1)),
		widget.Children(ln, code),
	)
	root := NewRootRenderable(width, height)
	root.Add(row)
	screen := buffer.New(width, height)

	frame = func() string {
		root.PrepareLayout(0)
		root.SyncPositions(0)
		if damage := root.TakeDamage(); !damage.IsEmpty() {
			// Clear the damage first, as the inline renderer does.
			screen.FillRect(damage.X, damage.Y, damage.Width, damage.Height, style.ColorBlack)
			root.Draw(screen, 0, damage)
		}
		return screen.String()
	}
	full = func() string {
		blank := buffer.New(width, height)
		root.PrepareLayout(0)
		root.SyncPositions(0)
		root.ResetDamage()
		root.Draw(blank, 0, buffer.Rect{})
		return blank.String()
	}

	root.PrepareLayout(0)
	root.SyncPositions(0)
	root.ResetDamage()
	root.Draw(screen, 0, buffer.Rect{})
	return frame, full
}

// newCode returns a Code widget of n lines that grows to fill its row.
func newCode(n int) *widget.Code {
	return widget.NewCode(codeLines(n),
		widget.CodeWithWrapMode(textbuf.WrapNone),
		widget.CodeWithLayout(layout.WithFlexGrow(1)),
	)
}

// TestLineNumberStandalone checks that a LineNumber beside its target, rather
// than inside a Diff that sizes it, takes the width of its numbers and repaints
// whenever its target changes what it shows.
func TestLineNumberStandalone(t *testing.T) {
	t.Run("takes the width of its numbers", func(t *testing.T) {
		code := newCode(30)
		ln := widget.NewLineNumber(code)
		frame, _ := newGutterScene(t, code, ln)

		if got, want := ln.Width(), ln.ComputedWidth(); got != want {
			t.Errorf("gutter width = %d, want its computed width %d", got, want)
		}
		if got := frame(); !strings.HasPrefix(got, "  1 line 0") {
			t.Errorf("gutter numbers do not precede the code:\n%s", got)
		}
	})

	t.Run("repaints when its target scrolls", func(t *testing.T) {
		code := newCode(30)
		ln := widget.NewLineNumber(code)
		frame, full := newGutterScene(t, code, ln)

		code.SetScrollY(10)
		got := frame()
		if want := full(); got != want {
			t.Errorf("after scrolling, frame:\n%s\nwant:\n%s", got, want)
		}
		if !strings.HasPrefix(got, " 11 line 10") {
			t.Errorf("gutter does not number the scrolled lines:\n%s", got)
		}
	})

	t.Run("widens when its target gains a digit", func(t *testing.T) {
		code := newCode(30)
		ln := widget.NewLineNumber(code)
		frame, full := newGutterScene(t, code, ln)
		before := ln.Width()

		code.SetContent(codeLines(1000))
		code.SetScrollY(996)
		frame() // Paints the old width, which then grows.
		got := frame()

		if ln.Width() != ln.ComputedWidth() || ln.Width() <= before {
			t.Errorf("gutter width = %d, want its computed width %d, wider than %d",
				ln.Width(), ln.ComputedWidth(), before)
		}
		if want := full(); got != want {
			t.Errorf("after widening, frame:\n%s\nwant:\n%s", got, want)
		}
		if !strings.HasPrefix(got, " 997 line 996") {
			t.Errorf("gutter does not number the last lines:\n%s", got)
		}
	})

	t.Run("damages nothing once settled", func(t *testing.T) {
		code := newCode(30)
		ln := widget.NewLineNumber(code)
		frame, _ := newGutterScene(t, code, ln)
		code.SetScrollY(10)
		frame()

		// The root repaints a renderable that reports either of these.
		if ln.IsDirty() {
			t.Error("an unchanged gutter reports itself dirty")
		}
		if d := ln.Damage(); !d.IsEmpty() {
			t.Errorf("an unchanged gutter reports damage %+v", d)
		}
	})

	t.Run("hidden numbers damage nothing when the target scrolls", func(t *testing.T) {
		code := newCode(30)
		ln := widget.NewLineNumber(code, widget.LineNumberWithShow(false))
		frame, _ := newGutterScene(t, code, ln)
		code.SetScrollY(10)
		frame()

		if ln.Width() != 0 {
			t.Errorf("gutter width = %d, want 0 with its numbers hidden", ln.Width())
		}
		if ln.IsDirty() {
			t.Error("a gutter with no area reports itself dirty")
		}
	})

	t.Run("shows hidden numbers in the next frame", func(t *testing.T) {
		code := newCode(30)
		ln := widget.NewLineNumber(code, widget.LineNumberWithShow(false))
		frame, full := newGutterScene(t, code, ln)

		ln.SetShowLineNumbers(true)
		got := frame()
		if ln.Width() != ln.ComputedWidth() || ln.Width() == 0 {
			t.Errorf("gutter width = %d, want its computed width %d", ln.Width(), ln.ComputedWidth())
		}
		if want := full(); got != want {
			t.Errorf("after showing the numbers, frame:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("widens for a sign column in the next frame", func(t *testing.T) {
		code := newCode(30)
		ln := widget.NewLineNumber(code)
		frame, full := newGutterScene(t, code, ln)
		before := ln.Width()

		ln.SetLineSigns(map[int]widget.LineSign{0: {Before: "+++"}})
		got := frame()
		if ln.Width() != ln.ComputedWidth() || ln.Width() <= before {
			t.Errorf("gutter width = %d, want its computed width %d, wider than %d",
				ln.Width(), ln.ComputedWidth(), before)
		}
		if want := full(); got != want {
			t.Errorf("after adding a sign column, frame:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("keeps an explicit width", func(t *testing.T) {
		code := newCode(5)
		ln := widget.NewLineNumber(code, widget.LineNumberWithLayout(layout.WithWidth(layout.Points(7))))
		frame, _ := newGutterScene(t, code, ln)
		frame()

		if got := ln.Width(); got != 7 {
			t.Errorf("gutter width = %d, want the explicit 7", got)
		}
	})

	t.Run("clips numbers wider than an explicit width", func(t *testing.T) {
		code := newCode(1000)
		ln := widget.NewLineNumber(code, widget.LineNumberWithLayout(layout.WithWidth(layout.Points(2))))
		code.SetScrollY(996)
		frame, _ := newGutterScene(t, code, ln)

		got := frame()
		if !strings.HasPrefix(got, "99line 996") {
			t.Errorf("gutter does not clip its numbers at its edge:\n%s", got)
		}
	})
}
