package widget

import "testing"

// TestMarkdownText checks that MarkdownText keeps the text that Markdown
// shows and drops its syntax, block by block.
func TestMarkdownText(t *testing.T) {
	tests := []struct {
		name, src, want string
	}{
		{name: "plain", src: "add a Divide method", want: "add a Divide method"},
		{name: "empty", src: "", want: ""},
		{name: "code span", src: "add `Divide` to calc.go", want: "add Divide to calc.go"},
		{name: "emphasis", src: "a **zero** check, *then* ~~more~~ tests", want: "a zero check, then more tests"},
		{name: "link", src: "see [the spec](https://spec.example) and <https://docs.example>", want: "see the spec and https://docs.example"},
		{name: "image", src: "![a diagram](d.png) here", want: "a diagram here"},
		{name: "escape", src: `a \*literal\* star`, want: "a *literal* star"},
		{name: "heading", src: "## Plan\n\nWrite the tests.", want: "Plan\nWrite the tests."},
		{name: "setext heading", src: "Plan\n====", want: "Plan"},
		{name: "soft break", src: "one\ntwo", want: "one two"},
		{name: "hard break", src: "one  \ntwo", want: "one\ntwo"},
		{name: "bullets", src: "- read `calc.go`\n- edit it", want: "read calc.go\nedit it"},
		{name: "numbers", src: "1. read\n2. edit", want: "read\nedit"},
		{name: "tasks", src: "- [x] read\n- [ ] edit", want: "read\nedit"},
		{name: "nested list", src: "- read\n  - calc.go\n- edit", want: "read\ncalc.go\nedit"},
		{name: "quote", src: "> keep **this**\n>\n> and that", want: "keep this\nand that"},
		{name: "fenced code", src: "run\n\n```go\nx := 1\n\ty := 2\n```", want: "run\nx := 1\n    y := 2"},
		{name: "indented code", src: "run\n\n    go test", want: "run\ngo test"},
		{name: "table", src: "| a | b |\n|---|---|\n| `1` | **2** |", want: "a b\n1 2"},
		{name: "rule", src: "above\n\n---\n\nbelow", want: "above\nbelow"},
		{name: "html block", src: "<div>\nhi\n</div>", want: "<div>\nhi\n</div>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MarkdownText(tt.src); got != tt.want {
				t.Errorf("MarkdownText(%q) = %q, want %q", tt.src, got, tt.want)
			}
		})
	}
}
