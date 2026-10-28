package extension

import (
	"github.com/unaquatic/fluid/internal/markdown"
	"github.com/unaquatic/fluid/internal/markdown/html"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// NewStrikethroughDelimiterRule returns the delimiter rule of struck-through
// text: one or two "~" characters at each end.
func NewStrikethroughDelimiterRule() parser.DelimiterRule {
	return parser.DelimiterRule{
		Char:      '~',
		MaxRun:    2,
		Intraword: true,
		New:       func(int) markdown.Node { return &markdown.Strikethrough{} },
	}
}

// StrikethroughAttributeFilter allows the attributes of del elements.
var StrikethroughAttributeFilter = html.GlobalAttributeFilter

// strikethroughHTMLRenderer renders struck-through text.
type strikethroughHTMLRenderer struct{}

// NewStrikethroughHTMLRenderer returns the renderer of struck-through text,
// as a del element.
func NewStrikethroughHTMLRenderer() html.NodeRenderer { return strikethroughHTMLRenderer{} }

// RegisterFuncs implements [html.NodeRenderer].
func (strikethroughHTMLRenderer) RegisterFuncs(reg html.NodeRendererFuncRegisterer) {
	reg.Register(markdown.KindStrikethrough, func(w *html.Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error) {
		if entering {
			w.WriteString("<del")
			w.WriteAttributes(n, StrikethroughAttributeFilter)
			w.WriteString(">")
		} else {
			w.WriteString("</del>")
		}
		return markdown.WalkContinue, nil
	})
}
