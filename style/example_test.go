package style_test

import (
	"fmt"

	"github.com/unaquatic/fluid/style"
)

// This example builds styled text from chunks with their own styles.
func ExampleNewStyledText() {
	st := style.NewStyledText(
		style.TextBold("fluid"),
		style.Plain(" draws in "),
		style.Fg(style.ColorString("#e5c07b"))("color"),
	)
	fmt.Println(st.PlainText())
	for _, c := range st.Chunks {
		fg := "inherited" // A transparent foreground keeps the widget's own.
		if !c.Fg.IsTransparent() {
			fg = c.Fg.ToHex()
		}
		fmt.Printf("%q bold=%t fg=%s\n", c.Text, c.Attributes.Has(style.AttrBold), fg)
	}
	// Output:
	// fluid draws in color
	// "fluid" bold=true fg=inherited
	// " draws in " bold=false fg=inherited
	// "color" bold=false fg=#e5c07b
}
