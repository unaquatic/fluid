package main

import (
	"context"
	"fmt"

	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/terminal"
	"github.com/unaquatic/fluid/widget"
)

var demoCodeCmd = Demo{
	Name:  "code",
	Short: "Syntax-highlighted code viewer demo",
	Run:   runCodeDemo,
}

const sampleGoCode = `package main

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Fibonacci calculates the nth Fibonacci number using memoization.
func Fibonacci(n int) int {
	memo := make(map[int]int)
	return fibHelper(n, memo)
}

func fibHelper(n int, memo map[int]int) int {
	if n <= 1 {
		return n
	}
	if v, ok := memo[n]; ok {
		return v
	}
	result := fibHelper(n-1, memo) + fibHelper(n-2, memo)
	memo[n] = result
	return result
}

// PrimeFactors returns the prime factorization of n.
func PrimeFactors(n int) []int {
	var factors []int
	for d := 2; d*d <= n; d++ {
		for n%d == 0 {
			factors = append(factors, d)
			n /= d
		}
	}
	if n > 1 {
		factors = append(factors, n)
	}
	return factors
}

// Point represents a 2D point.
type Point struct {
	X, Y float64
}

// Distance calculates the Euclidean distance between two points.
func (p Point) Distance(other Point) float64 {
	dx := p.X - other.X
	dy := p.Y - other.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// String returns a string representation of the point.
func (p Point) String() string {
	return fmt.Sprintf("(%0.2f, %0.2f)", p.X, p.Y)
}

// ReverseString reverses a UTF-8 string.
func ReverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// WordCount counts the number of words in a string.
func WordCount(s string) int {
	return len(strings.Fields(s))
}

func main() {
	// Fibonacci
	for i := 0; i < 20; i++ {
		fmt.Printf("F(%d) = %d\n", i, Fibonacci(i))
	}

	// Prime factors
	fmt.Println("Prime factors of 360:", PrimeFactors(360))

	// Points and distance
	a := Point{3, 4}
	b := Point{7, 1}
	fmt.Printf("Distance from %s to %s = %.2f\n", a, b, a.Distance(b))

	// String operations
	msg := "Hello, Fluid!"
	fmt.Println("Reversed:", ReverseString(msg))
	fmt.Println("Word count:", WordCount(msg))

	// Time
	start := time.Now()
	_ = Fibonacci(30)
	elapsed := time.Since(start)
	fmt.Printf("Fib(30) took %v\n", elapsed)
}
`

func runCodeDemo(ctx context.Context, o DemoOptions) error {
	r, err := terminal.NewRenderer(terminal.Config{
		ExitOnCtrlC: true,
		UseMouse:    !o.NoMouse,
		TargetFPS:   o.FPS,
	})
	if err != nil {
		return fmt.Errorf("renderer: %w", err)
	}

	root := r.Root()

	codeWidget := widget.NewCode(sampleGoCode,
		widget.CodeWithFiletype("go"),
		widget.CodeWithTheme(style.DarculaSolid()),
		widget.CodeWithWrapMode(widget.WrapNone),
		widget.CodeWithFlexGrow(1),
		widget.CodeWithActiveLine(0),
	)
	lineNumbers := widget.NewLineNumber(codeWidget)

	wrapMode := widget.WrapNone
	statusText := widget.NewText(
		"Wrap: OFF | Scroll: mouse wheel or Up/Down | w: toggle wrap | Esc/q: quit",
		widget.TextWithHeight(layout.Points(1)),
	)

	container := widget.NewBox(
		widget.BoxWithFlexGrow(1),
		widget.BoxWithFlexDirection(layout.DirectionColumn),
		widget.BoxWithPadding(layout.Points(1)),
		widget.BoxWithGap(layout.Points(1)),
		widget.Children(
			widget.NewText("Code Viewer Demo — Go Syntax Highlighting",
				widget.TextWithHeight(layout.Points(1)),
			),
			widget.NewBox(
				widget.BoxWithFlexGrow(1),
				widget.BoxWithFlexDirection(layout.DirectionRow),
				widget.BoxWithBorder(style.BorderAll()),
				widget.BoxWithBorderStyle(style.BorderSingle),
				widget.BoxWithTitle("main.go"),
				widget.Children(lineNumbers, codeWidget),
			),
			statusText,
		),
	)
	root.Add(container)

	r.EventBus().On(input.EventKeyboard, func(e input.TerminalEvent) {
		if e.Key == nil {
			return
		}
		k := e.Key

		if k.Name == "escape" || (k.Name == "q" && !k.Ctrl && !k.Alt) {
			r.Destroy()
			return
		}
		if k.Name == "w" && !k.Ctrl && !k.Alt {
			if wrapMode == widget.WrapNone {
				wrapMode = widget.WrapWord
				codeWidget.SetWrapMode(widget.WrapWord)
				statusText.SetContent("Wrap: WORD | Scroll: mouse wheel or Up/Down | w: toggle wrap | Esc/q: quit")
			} else {
				wrapMode = widget.WrapNone
				codeWidget.SetWrapMode(widget.WrapNone)
				statusText.SetContent("Wrap: OFF | Scroll: mouse wheel or Up/Down | w: toggle wrap | Esc/q: quit")
			}
			return
		}
	})

	return r.RunContinuous(ctx)
}
