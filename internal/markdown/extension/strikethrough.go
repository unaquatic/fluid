package extension

import (
	"github.com/unaquatic/fluid/internal/markdown/ast"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// NewStrikethroughDelimiterRule returns the delimiter rule of struck-through
// text: one or two "~" characters at each end.
func NewStrikethroughDelimiterRule() parser.DelimiterRule {
	return parser.DelimiterRule{
		Char:      '~',
		MaxRun:    2,
		Intraword: true,
		New:       func(int) ast.Node { return &ast.Strikethrough{} },
	}
}
