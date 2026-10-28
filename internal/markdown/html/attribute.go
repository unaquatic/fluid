package html

import (
	"maps"
	"strconv"
	"strings"

	"github.com/unaquatic/fluid/internal/markdown"
)

// An AttributeFilter is the set of the names of the attributes that an
// element can have, of those of its node, which [Writer.WriteAttributes]
// writes. The zero AttributeFilter allows every attribute.
type AttributeFilter struct {
	names map[string]bool
}

// NewAttributeFilter returns the filter that allows the attributes of
// names.
func NewAttributeFilter(names ...string) AttributeFilter {
	f := AttributeFilter{names: make(map[string]bool, len(names))}
	for _, name := range names {
		f.names[name] = true
	}
	return f
}

// Extend returns the filter that allows the attributes that f allows and
// those of names. If f allows every attribute, so does the filter.
func (f AttributeFilter) Extend(names ...string) AttributeFilter {
	if f.names == nil {
		return f
	}
	g := AttributeFilter{names: maps.Clone(f.names)}
	for _, name := range names {
		g.names[name] = true
	}
	return g
}

// Allows reports whether f allows the attribute named name: one of its
// names, or any that starts with "data-".
func (f AttributeFilter) Allows(name string) bool {
	return f.names == nil || f.names[name] || strings.HasPrefix(name, "data-")
}

// Filters of the attributes of the elements that the renderer of
// CommonMark writes, as HTML defines them.
var (
	// GlobalAttributeFilter allows the global attributes, which every
	// element can have.
	GlobalAttributeFilter = NewAttributeFilter(
		"accesskey", "autocapitalize", "autofocus", "class", "contenteditable", "dir", "draggable",
		"enterkeyhint", "hidden", "id", "inert", "inputmode", "is", "itemid", "itemprop", "itemref",
		"itemscope", "itemtype", "lang", "part", "role", "slot", "spellcheck", "style", "tabindex",
		"title", "translate",
	)

	// HeadingAttributeFilter allows the attributes of h1 through h6
	// elements.
	HeadingAttributeFilter = GlobalAttributeFilter

	// BlockquoteAttributeFilter allows the attributes of blockquote
	// elements.
	BlockquoteAttributeFilter = GlobalAttributeFilter.Extend("cite")

	// ListAttributeFilter allows the attributes of ol and ul elements.
	ListAttributeFilter = GlobalAttributeFilter.Extend("start", "reversed", "type")

	// ListItemAttributeFilter allows the attributes of li elements.
	ListItemAttributeFilter = GlobalAttributeFilter.Extend("value")

	// ParagraphAttributeFilter allows the attributes of p elements.
	ParagraphAttributeFilter = GlobalAttributeFilter

	// ThematicBreakAttributeFilter allows the attributes of hr elements.
	ThematicBreakAttributeFilter = GlobalAttributeFilter.Extend("align", "color", "noshade", "size", "width")

	// LinkAttributeFilter allows the attributes of the a elements of links
	// and autolinks.
	LinkAttributeFilter = GlobalAttributeFilter.Extend(
		"download", "href", "lang", "media", "ping", "referrerpolicy", "rel", "shape", "target",
	)

	// CodeSpanAttributeFilter allows the attributes of the code elements of
	// code spans.
	CodeSpanAttributeFilter = GlobalAttributeFilter

	// EmphasisAttributeFilter allows the attributes of em and strong
	// elements.
	EmphasisAttributeFilter = GlobalAttributeFilter

	// ImageAttributeFilter allows the attributes of img elements.
	ImageAttributeFilter = GlobalAttributeFilter.Extend(
		"align", "border", "crossorigin", "decoding", "height", "importance", "intrinsicsize", "ismap",
		"loading", "referrerpolicy", "sizes", "srcset", "usemap", "width",
	)
)

// WriteAttributes writes the attributes of n that filter allows, in the
// order in which they were set, as WriteAttribute writes each.
func (w *Writer) WriteAttributes(n markdown.Node, filter AttributeFilter) {
	for _, a := range n.Attributes() {
		if filter.Allows(a.Name) {
			w.WriteAttribute(a.Name, a.Value)
		}
	}
}

// WriteAttribute writes an attribute named name with value: a space, the
// name, "=", and the value in double quotes, escaped as text. The value can
// be a string, a []byte, a bool, a float64, or an int. It writes nothing
// for a value of another type, or a name that HTML does not allow, such as
// one with a space or a quote.
func (w *Writer) WriteAttribute(name string, value any) {
	var v string
	switch value := value.(type) {
	case string:
		v = value
	case []byte:
		v = string(value)
	case bool:
		v = strconv.FormatBool(value)
	case float64:
		v = strconv.FormatFloat(value, 'g', -1, 64)
	case int:
		v = strconv.Itoa(value)
	default:
		return
	}
	if !isAttributeName(name) {
		return
	}
	w.WriteString(" " + name + `="`)
	w.WriteText(v)
	w.WriteString(`"`)
}

// isAttributeName reports whether name is a name that HTML allows for an
// attribute: one or more characters other than controls, spaces, quotes,
// ">", "/", and "=".
func isAttributeName(name string) bool {
	if name == "" {
		return false
	}
	for i := range len(name) {
		if c := name[i]; c <= ' ' || c == 0x7f || strings.IndexByte(`"'>/=<`, c) >= 0 {
			return false
		}
	}
	return true
}
