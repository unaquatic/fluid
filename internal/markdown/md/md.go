// Package md converts Markdown to HTML with a parser of package
// [github.com/unaquatic/fluid/internal/markdown/parser] and a renderer of
// package [github.com/unaquatic/fluid/internal/markdown/html], which
// extensions extend together.
//
//	m := md.New(md.WithExtensions(extension.GFM))
//	err := m.Convert(src, w)
package md

import (
	"io"

	"github.com/unaquatic/fluid/internal/markdown/html"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// A Markdown converts Markdown to HTML.
type Markdown interface {
	// Convert parses src and writes its HTML to w.
	Convert(src string, w io.Writer) error

	// Parser returns the parser that Convert parses with.
	Parser() parser.Parser

	// SetParser sets the parser that Convert parses with.
	SetParser(p parser.Parser)

	// Renderer returns the renderer that Convert renders with.
	Renderer() html.Renderer

	// SetRenderer sets the renderer that Convert renders with.
	SetRenderer(r html.Renderer)
}

// An Extender extends the parser and the renderer of a [Markdown].
type Extender interface {
	// Extend adds options to the parser and the renderer of m.
	Extend(m Markdown)
}

// An Option configures a [Markdown] that [New] makes.
type Option func(m *markdown)

// New returns a [Markdown] of CommonMark, configured by opts. The
// extensions that WithExtensions adds extend it after the other options
// apply.
func New(opts ...Option) Markdown {
	m := &markdown{parser: parser.NewParser(), renderer: html.NewRenderer()}
	for _, opt := range opts {
		opt(m)
	}
	for _, e := range m.extensions {
		e.Extend(m)
	}
	return m
}

// WithExtensions adds extensions.
func WithExtensions(exts ...Extender) Option {
	return func(m *markdown) { m.extensions = append(m.extensions, exts...) }
}

// WithParser sets the parser.
func WithParser(p parser.Parser) Option {
	return func(m *markdown) { m.parser = p }
}

// WithParserOptions adds options to the parser.
func WithParserOptions(opts ...parser.Option) Option {
	return func(m *markdown) { m.parser.AddOptions(opts...) }
}

// WithRenderer sets the renderer.
func WithRenderer(r html.Renderer) Option {
	return func(m *markdown) { m.renderer = r }
}

// WithRendererOptions adds options to the renderer.
func WithRendererOptions(opts ...html.Option) Option {
	return func(m *markdown) { m.renderer.AddOptions(opts...) }
}

// markdown is the [Markdown] that New returns.
type markdown struct {
	parser     parser.Parser
	renderer   html.Renderer
	extensions []Extender
}

// Convert implements [Markdown].
func (m *markdown) Convert(src string, w io.Writer) error {
	return m.renderer.Render(w, m.parser.Parse(src))
}

// Parser implements [Markdown].
func (m *markdown) Parser() parser.Parser { return m.parser }

// SetParser implements [Markdown].
func (m *markdown) SetParser(p parser.Parser) { m.parser = p }

// Renderer implements [Markdown].
func (m *markdown) Renderer() html.Renderer { return m.renderer }

// SetRenderer implements [Markdown].
func (m *markdown) SetRenderer(r html.Renderer) { m.renderer = r }

// defaultMarkdown converts CommonMark for Convert.
var defaultMarkdown = New()

// Convert parses CommonMark src and writes its HTML to w.
func Convert(src string, w io.Writer) error { return defaultMarkdown.Convert(src, w) }
