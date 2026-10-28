// Package spectest checks the parser against the examples of the CommonMark
// and GitHub Flavored Markdown specifications, which give each example's
// HTML in the form that their reference implementations render it. [HTML]
// renders trees so, for tests only.
package spectest

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// An Example is an example of a specification, as its JSON form holds it.
type Example struct {
	Markdown string `json:"markdown"`
	HTML     string `json:"html"`
	Example  int    `json:"example"`
	Section  string `json:"section"`
}

// Examples reads the examples of a specification from file.
func Examples(t testing.TB, file string) []Example {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var examples []Example
	if err := json.Unmarshal(data, &examples); err != nil {
		t.Fatal(err)
	}
	return examples
}

// Run checks that convert converts each example of the specification in
// file to the example's HTML, in a subtest named for the example's number.
func Run(t *testing.T, file string, convert func(src string) string) {
	for _, ex := range Examples(t, file) {
		t.Run(fmt.Sprint(ex.Example), func(t *testing.T) {
			if got := convert(ex.Markdown); got != ex.HTML {
				t.Errorf("%s, example %d:\nmarkdown:\n%s\ngot:\n%s\nwant:\n%s", ex.Section, ex.Example, ex.Markdown, got, ex.HTML)
			}
		})
	}
}
