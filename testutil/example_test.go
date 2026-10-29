package testutil_test

import (
	"fmt"
	"log"

	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/testutil"
	"github.com/unaquatic/fluid/widget"
)

// This example prints the golden frames of a Text widget, the form that
// AssertGoldenFrames compares with testdata/<name>.frames.golden in a test:
//
//	func TestGreeting(t *testing.T) {
//		buf := testutil.RenderWidget(widget.NewText("say hi"), 7, 1)
//		testutil.AssertGoldenFrames(t, "greeting", buf, nil)
//	}
func ExampleGoldenFrames() {
	textBase := style.ColorFromHex("#cfd2d5")
	text := widget.NewText("say hi", widget.TextWithFg(textBase))

	buf := testutil.RenderWidget(text, 7, 1)
	frames, err := testutil.GoldenFrames(buf, map[style.Color]string{textBase: "TextBase"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(frames)
	// Output:
	// === text ===
	// |say hi |
	// === fg ===
	// |AAA AA |
	// === legend ===
	// A #cfd2d5 TextBase
}
