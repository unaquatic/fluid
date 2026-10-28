// Package html renders the tree of a Markdown document, which package
// [github.com/unaquatic/fluid/internal/markdown] defines, as HTML.
//
// A [Renderer] walks the tree and calls, for each node, the function that a
// [NodeRenderer] registers for the node's kind. The renderer of CommonMark
// registers the kinds of CommonMark; extensions register their own.
//
// By default a Renderer leaves out raw HTML, and the destinations of links
// and images that could run script, as fits a document of unknown origin.
// [WithUnsafe] keeps them.
//
// The elements of nodes with attributes, such as the "id" of a heading, get
// those that their [AttributeFilter] allows, which [Writer.WriteAttributes]
// writes. Code blocks, HTML, and text get none.
//
// A soft line break renders as a line ending, which browsers show as a
// space, unless [WithEastAsianLineBreaks] leaves it out between East Asian
// characters, which their scripts do not part with spaces.
package html

import (
	"bufio"
	"cmp"
	"io"
	"slices"
	"strings"

	"github.com/unaquatic/fluid/internal/markdown"
)

// A Renderer renders documents as HTML.
type Renderer interface {
	// Render writes the HTML of n, usually a document, to w. It is safe for
	// concurrent use, but not with AddOptions.
	Render(w io.Writer, n markdown.Node) error

	// AddOptions applies opts.
	AddOptions(opts ...Option)
}

// NewRenderer returns a [Renderer] of the kinds of node of CommonMark, and
// of those that opts add.
func NewRenderer(opts ...Option) Renderer {
	r := &renderer{config: Config{NodeRenderers: []markdown.Prioritized[NodeRenderer]{{Value: commonMark{}, Priority: 1000}}}}
	r.AddOptions(opts...)
	return r
}

// An Option configures a [Renderer].
type Option func(c *Config)

// A Config holds how a [Renderer] renders.
type Config struct {
	// HardWraps renders soft line breaks as hard ones.
	HardWraps bool

	// XHTML renders empty elements as XHTML does, such as "<br />".
	XHTML bool

	// Unsafe renders raw HTML, and the destinations of links and images
	// that could run script, as they are written.
	Unsafe bool

	// EastAsianLineBreaks is how soft line breaks between East Asian
	// characters render.
	EastAsianLineBreaks EastAsianLineBreaks

	// NodeRenderers holds the renderers of kinds of node. Of several that
	// render a kind, the one of the lowest priority renders it.
	NodeRenderers []markdown.Prioritized[NodeRenderer]
}

// WithHardWraps renders soft line breaks as hard ones.
func WithHardWraps() Option { return func(c *Config) { c.HardWraps = true } }

// WithXHTML renders empty elements as XHTML does, such as "<br />".
func WithXHTML() Option { return func(c *Config) { c.XHTML = true } }

// WithUnsafe renders raw HTML, and the destinations of links and images
// that could run script, as they are written.
func WithUnsafe() Option { return func(c *Config) { c.Unsafe = true } }

// WithEastAsianLineBreaks sets how soft line breaks between East Asian
// characters render, such as [EastAsianLineBreaksSimple], which leaves out
// those between two wide characters.
func WithEastAsianLineBreaks(style EastAsianLineBreaks) Option {
	return func(c *Config) { c.EastAsianLineBreaks = style }
}

// WithNodeRenderer adds node renderer r with priority. The renderer of
// CommonMark has priority 1000.
func WithNodeRenderer(r NodeRenderer, priority int) Option {
	return func(c *Config) {
		c.NodeRenderers = append(c.NodeRenderers, markdown.Prioritized[NodeRenderer]{Value: r, Priority: priority})
	}
}

// A NodeRenderer renders some kinds of node.
type NodeRenderer interface {
	// RegisterFuncs registers the function that renders each kind of node
	// that the renderer renders.
	RegisterFuncs(reg NodeRendererFuncRegisterer)
}

// A NodeRendererFuncRegisterer registers the functions of a [NodeRenderer].
type NodeRendererFuncRegisterer interface {
	// Register registers f as the function that renders nodes of kind.
	Register(kind markdown.NodeKind, f NodeRendererFunc)
}

// A NodeRendererFunc renders node n, as a [Renderer] enters it and as it
// leaves it, after its children, and returns how the walk of the tree goes
// on, such as [markdown.WalkSkipChildren] for a node that renders its
// children itself.
type NodeRendererFunc func(w *Writer, n markdown.Node, entering bool) (markdown.WalkStatus, error)

// renderer is the [Renderer] that NewRenderer returns.
type renderer struct {
	config Config
	funcs  map[markdown.NodeKind]NodeRendererFunc
}

// AddOptions implements [Renderer].
func (r *renderer) AddOptions(opts ...Option) {
	for _, opt := range opts {
		opt(&r.config)
	}
	nrs := slices.Clone(r.config.NodeRenderers)
	slices.SortStableFunc(nrs, func(a, b markdown.Prioritized[NodeRenderer]) int { return cmp.Compare(a.Priority, b.Priority) })
	r.funcs = make(map[markdown.NodeKind]NodeRendererFunc)
	for _, nr := range slices.Backward(nrs) {
		nr.Value.RegisterFuncs(r)
	}
}

// Register implements [NodeRendererFuncRegisterer].
func (r *renderer) Register(kind markdown.NodeKind, f NodeRendererFunc) { r.funcs[kind] = f }

// Render implements [Renderer]. A node of a kind that no renderer renders
// writes nothing of its own, but its children render.
func (r *renderer) Render(w io.Writer, n markdown.Node) error {
	bw := bufio.NewWriter(w)
	hw := &Writer{w: bw, config: &r.config}
	err := markdown.Walk(n, func(n markdown.Node, entering bool) (markdown.WalkStatus, error) {
		if f := r.funcs[n.Kind()]; f != nil {
			return f(hw, n, entering)
		}
		return markdown.WalkContinue, nil
	})
	if err != nil {
		return err
	}
	return bw.Flush()
}

// A Writer writes the HTML of a document for the functions of a
// [NodeRenderer].
type Writer struct {
	w      *bufio.Writer
	config *Config
	last   byte // the last byte written, or 0
}

// WriteString writes s as it is.
func (w *Writer) WriteString(s string) {
	if s == "" {
		return
	}
	w.w.WriteString(s)
	w.last = s[len(s)-1]
}

// WriteText writes text s, escaping the characters that HTML gives meaning
// to.
func (w *Writer) WriteText(s string) { w.WriteString(escaper.Replace(s)) }

// WriteURL writes URL s as an attribute value: with the bytes that a URL
// cannot hold percent-encoded, and escaped as text.
func (w *Writer) WriteURL(s string) { w.WriteText(escapeURL(s)) }

// CR starts a new line, unless nothing is written or a line just ended.
func (w *Writer) CR() {
	if w.last != 0 && w.last != '\n' {
		w.WriteString("\n")
	}
}

// Config returns the configuration of the [Renderer] that w writes for.
func (w *Writer) Config() Config { return *w.config }

// emptyEnd returns the end of an empty element: " />" with XHTML and ">"
// without.
func (w *Writer) emptyEnd() string {
	if w.config.XHTML {
		return " />"
	}
	return ">"
}

var escaper = strings.NewReplacer(`&`, "&amp;", `<`, "&lt;", `>`, "&gt;", `"`, "&quot;")

// escapeURL percent-encodes the bytes of s that a URL cannot hold, keeping
// percent-encoded bytes as they are.
func escapeURL(s string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case isAlnum(c) || strings.IndexByte(";/?:@&=+$,-_.!~*'()#", c) >= 0:
			b.WriteByte(c)
		case c == '%' && i+2 < len(s) && isHex(s[i+1]) && isHex(s[i+2]):
			b.WriteByte(c)
		default:
			b.WriteByte('%')
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&15])
		}
	}
	return b.String()
}

// isDangerousURL reports whether url could run script: one with the
// scheme javascript, vbscript, or file, or data unless it is an image of
// type gif, png, jpeg, or webp.
func isDangerousURL(url string) bool {
	lower := strings.ToLower(strings.TrimLeft(url, " \t\n\r\f"))
	for _, scheme := range []string{"javascript:", "vbscript:", "file:"} {
		if strings.HasPrefix(lower, scheme) {
			return true
		}
	}
	if !strings.HasPrefix(lower, "data:") {
		return false
	}
	for _, safe := range []string{"data:image/png", "data:image/gif", "data:image/jpeg", "data:image/webp"} {
		if strings.HasPrefix(lower, safe) {
			return false
		}
	}
	return true
}

func isAlnum(c byte) bool { return '0' <= c && c <= '9' || 'a' <= c|0x20 && c|0x20 <= 'z' }
func isHex(c byte) bool   { return '0' <= c && c <= '9' || 'a' <= c|0x20 && c|0x20 <= 'f' }
