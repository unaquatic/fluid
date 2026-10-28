package parser

import (
	"strconv"
	"strings"

	"github.com/unaquatic/fluid/internal/markdown/ast"
)

// An IDs makes the IDs of the elements of a document, such as the "id"
// attributes of headings, unique within the document. A [Parser] makes one
// for each document that it parses, which [BlockState.IDs] returns.
type IDs interface {
	// Generate returns a new ID, which it records as used, for an element
	// of kind, made from value, such as a heading's text.
	Generate(value string, kind ast.NodeKind) string

	// Put records id, which an element has, as used.
	Put(id string)
}

// NewIDs returns the IDs that a [Parser] makes for each document unless
// [WithIDs] sets others. Generate makes an ID of the ASCII letters and
// digits of value, in lower case, and a hyphen for each space, hyphen, and
// underscore, leaving out other characters and the spaces at each end. An
// ID with no characters is "heading" for a heading and "id" for any other
// element. A used ID gains a hyphen and the first number from 1 that makes
// it unused.
func NewIDs() IDs { return &idSet{used: make(map[string]bool)} }

// idSet is the [IDs] that NewIDs returns.
type idSet struct {
	used map[string]bool
}

// Generate implements [IDs].
func (s *idSet) Generate(value string, kind ast.NodeKind) string {
	value = strings.Trim(value, " \t\n\v\f\r")
	var b strings.Builder
	for i := range len(value) {
		switch c := value[i]; {
		case 'A' <= c && c <= 'Z':
			b.WriteByte(c + 'a' - 'A')
		case isAlnum(c):
			b.WriteByte(c)
		case isSpaceByte(c) || c == '-' || c == '_':
			b.WriteByte('-')
		}
	}
	id := b.String()
	if id == "" {
		id = "id"
		if kind == ast.KindHeading {
			id = "heading"
		}
	}
	if !s.used[id] {
		s.used[id] = true
		return id
	}
	for n := 1; ; n++ {
		if numbered := id + "-" + strconv.Itoa(n); !s.used[numbered] {
			s.used[numbered] = true
			return numbered
		}
	}
}

// Put implements [IDs].
func (s *idSet) Put(id string) { s.used[id] = true }
