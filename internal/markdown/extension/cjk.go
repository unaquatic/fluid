package extension

import "github.com/unaquatic/fluid/internal/markdown/parser"

// cjk is the extension CJK.
type cjk struct{}

// ExtendParser implements [parser.Extension].
func (cjk) ExtendParser(c *parser.Config) { c.Apply(parser.WithEscapedSpace()) }
