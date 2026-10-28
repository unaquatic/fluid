package extension

import (
	"github.com/unaquatic/fluid/internal/markdown/html"
	"github.com/unaquatic/fluid/internal/markdown/md"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// A CJKOption configures the extension for Chinese, Japanese, and Korean.
type CJKOption func(c *CJKConfig)

// A CJKConfig holds what the extension for Chinese, Japanese, and Korean
// adds.
type CJKConfig struct {
	// EastAsianLineBreaks is how soft line breaks between East Asian
	// characters render.
	EastAsianLineBreaks html.EastAsianLineBreaks

	// EscapedSpace reports whether a backslash before a space escapes it,
	// to stand for nothing.
	EscapedSpace bool
}

// WithEastAsianLineBreaks sets how soft line breaks between East Asian
// characters render, as [html.WithEastAsianLineBreaks] sets it.
func WithEastAsianLineBreaks(style html.EastAsianLineBreaks) CJKOption {
	return func(c *CJKConfig) { c.EastAsianLineBreaks = style }
}

// WithEscapedSpace lets a backslash escape a space, as
// [parser.WithEscapedSpace] lets it.
func WithEscapedSpace() CJKOption { return func(c *CJKConfig) { c.EscapedSpace = true } }

// cjk is an extension for Chinese, Japanese, and Korean.
type cjk struct {
	config CJKConfig
}

// NewCJK returns an extension for Chinese, Japanese, and Korean,
// configured by opts, which adds nothing without them. The renderer of a
// [md.Markdown] gets the option of [WithEastAsianLineBreaks], unless it is
// [html.EastAsianLineBreaksNone], and the parser that of
// [WithEscapedSpace].
func NewCJK(opts ...CJKOption) Extension {
	e := &cjk{}
	for _, opt := range opts {
		opt(&e.config)
	}
	return e
}

// Extend implements [md.Extender].
func (e *cjk) Extend(m md.Markdown) {
	m.Parser().AddOptions(parser.WithExtensions(e))
	if style := e.config.EastAsianLineBreaks; style != html.EastAsianLineBreaksNone {
		m.Renderer().AddOptions(html.WithEastAsianLineBreaks(style))
	}
}

// ExtendParser implements [parser.Extension].
func (e *cjk) ExtendParser(c *parser.Config) {
	if e.config.EscapedSpace {
		c.Apply(parser.WithEscapedSpace())
	}
}
