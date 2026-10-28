package ast

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

// Dump writes the tree of n to w, a line for each node, each indented two
// spaces for each level that it lies below n. A line holds the node's kind,
// the exported fields of its type that are not zero, as Name=value, and its
// attributes, in braces. A block's line ends with its span, as
// [start:end], and, but for a document's, the source that the span covers,
// quoted, if src, the source of the document, holds it. A text's line holds
// its value, quoted, and the line break after it, if any, as softbreak or
// hardbreak.
//
//	Document [0:25]
//	  Heading Level=1 [0:7] "# Title"
//	    Text "Title"
//	  Paragraph [9:25] "Some *emph*\ntext"
//	    Text "Some "
//	    Emphasis Level=1
//	      Text "emph"
//	    Text "" softbreak
//	    Text "text"
func Dump(w io.Writer, n Node, src string) error {
	return dump(w, n, src, 0)
}

// dump writes the lines of n and its descendants to w, n's at depth.
func dump(w io.Writer, n Node, src string, depth int) error {
	var b strings.Builder
	b.WriteString(strings.Repeat("  ", depth))
	b.WriteString(n.Kind().String())
	if t, ok := n.(*Text); ok {
		b.WriteString(" " + strconv.Quote(t.Value))
		switch {
		case t.HardLineBreak():
			b.WriteString(" hardbreak")
		case t.SoftLineBreak():
			b.WriteString(" softbreak")
		}
	} else {
		for _, f := range nodeFields(n) {
			b.WriteString(" " + f)
		}
	}
	if attrs := n.Attributes(); len(attrs) > 0 {
		b.WriteString(" {")
		for i, a := range attrs {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(a.Name + "=" + formatValue(reflect.ValueOf(a.Value)))
		}
		b.WriteByte('}')
	}
	if s := n.SourceSpan(); n.Type() != TypeInline || s != (Span{}) {
		fmt.Fprintf(&b, " [%d:%d]", s.Start, s.End)
		if n.Type() == TypeBlock && 0 <= s.Start && s.Start <= s.End && s.End <= len(src) {
			b.WriteString(" " + strconv.Quote(src[s.Start:s.End]))
		}
	}
	b.WriteByte('\n')
	if _, err := io.WriteString(w, b.String()); err != nil {
		return err
	}
	for c := range Children(n) {
		if err := dump(w, c, src, depth+1); err != nil {
			return err
		}
	}
	return nil
}

// nodeFields returns the exported fields of the type of node n that are not
// zero, as Name=value, without those of the types it embeds.
func nodeFields(n Node) []string {
	v := reflect.ValueOf(n)
	for v.Kind() == reflect.Pointer && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	var out []string
	for i := range v.NumField() {
		f := v.Type().Field(i)
		if f.Anonymous || !f.IsExported() || v.Field(i).IsZero() {
			continue
		}
		out = append(out, f.Name+"="+formatValue(v.Field(i)))
	}
	return out
}

// formatValue returns v as Dump writes values: strings quoted, bytes as
// quoted characters, values with a String method as it returns them, and
// slices, maps, and structs element by element.
func formatValue(v reflect.Value) string {
	if !v.IsValid() {
		return "nil"
	}
	if v.CanInterface() && (v.Kind() != reflect.Pointer || !v.IsNil()) {
		if s, ok := v.Interface().(fmt.Stringer); ok {
			return s.String()
		}
	}
	switch v.Kind() {
	case reflect.String:
		return strconv.Quote(v.String())
	case reflect.Uint8:
		if v.Type() == reflect.TypeFor[byte]() {
			return strconv.QuoteRune(rune(v.Uint()))
		}
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return "nil"
		}
		return formatValue(v.Elem())
	case reflect.Slice, reflect.Array:
		elems := make([]string, v.Len())
		for i := range elems {
			elems[i] = formatValue(v.Index(i))
		}
		return "[" + strings.Join(elems, " ") + "]"
	case reflect.Struct:
		return "{" + strings.Join(structFields(v), " ") + "}"
	}
	return fmt.Sprint(v.Interface())
}

// structFields returns the exported fields of struct v, with those of the
// structs it embeds, as Name=value.
func structFields(v reflect.Value) []string {
	var out []string
	for i := range v.NumField() {
		f := v.Type().Field(i)
		switch {
		case !f.IsExported():
		case f.Anonymous && v.Field(i).Kind() == reflect.Struct:
			out = append(out, structFields(v.Field(i))...)
		default:
			out = append(out, f.Name+"="+formatValue(v.Field(i)))
		}
	}
	return out
}
