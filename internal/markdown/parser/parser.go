// Package parser parses Markdown into the tree of package
// [github.com/unaquatic/fluid/internal/markdown], as the CommonMark
// specification, version 0.31.2, defines it, with the syntax that
// extensions add.
//
// A [Parser] parses in two phases. The first builds the tree of blocks one
// line at a time: at each line, every open block that the line continues
// consumes its markers, as its [BlockParser] decides, and then each block
// parser in turn can open a new block where the markers end. Each
// paragraph, as it closes, loses the link reference definitions at its
// start, and then [ParagraphTransformer] values can change it. The second
// phase parses the inline content of paragraphs, headings, table cells, and
// definition terms:
// [InlineParser] values parse elements that start with the bytes that
// trigger them, and runs of delimiter characters delimit spans, such as
// emphasis, which the parser pairs by the rules CommonMark gives for
// emphasis, as a [DelimiterProcessor] allows. Links, images, and line breaks
// are built in. [ASTTransformer] values then change the document.
//
// With [WithAttribute], a heading can end with attributes, such as
// {#id .class}, and with [WithAutoHeadingID], each heading gets an ID that
// is unique in its document, which [IDs] make.
//
// [NewParser] makes a Parser with options, such as [WithExtensions] and
// [WithBlockParser], that add syntax to CommonMark's. A parser's priority
// orders it among the others of its kind: lower priorities come first.
package parser

import (
	"cmp"
	"slices"

	"github.com/unaquatic/fluid/internal/markdown"
)

// A Parser parses Markdown.
type Parser interface {
	// Parse parses src as a Markdown document. It is safe for concurrent
	// use, but not with AddOptions.
	Parse(src string) *markdown.Document

	// AddOptions adds the syntax that opts add.
	AddOptions(opts ...Option)
}

// NewParser returns a [Parser] of CommonMark with the syntax that opts add.
func NewParser(opts ...Option) Parser {
	p := &parser{config: commonMark()}
	p.AddOptions(opts...)
	return p
}

// parser is the [Parser] that NewParser returns.
type parser struct {
	config *Config
	syntax *syntax
}

// AddOptions implements [Parser].
func (p *parser) AddOptions(opts ...Option) {
	p.config.Apply(opts...)
	p.syntax = newSyntax(p.config)
}

// Parse implements [Parser].
func (p *parser) Parse(src string) *markdown.Document {
	syn := p.syntax
	s := newBlockState(syn, src)
	for off, line := range lines(src) {
		s.incorporateLine(off, line)
	}
	for len(s.open) > 0 {
		s.Close(s.Tip(), s.lineNumber)
	}
	in := newInlineState(s)
	_ = markdown.Walk(s.doc, func(n markdown.Node, entering bool) (markdown.WalkStatus, error) {
		if !entering {
			return markdown.WalkContinue, nil
		}
		switch n.(type) {
		case *markdown.Paragraph, *markdown.Heading, *markdown.TableCell, *markdown.DefinitionTerm:
			in.parse(n, joinText(n.Lines(), ""))
			return markdown.WalkSkipChildren, nil
		}
		return markdown.WalkContinue, nil
	})
	for _, t := range syn.astTransformers {
		t.Transform(s.doc)
	}
	return s.doc
}

// An Option adds syntax to the [Config] of a [Parser].
type Option func(c *Config)

// An Extension adds syntax to the [Config] of a [Parser].
type Extension interface {
	// ExtendParser changes c, which holds the syntax of CommonMark and of
	// the options before the extension's. It runs once, as the Parser is
	// made or its options are added, whose parsers then parse every
	// document that the Parser does, concurrently too: they keep the state
	// of one document in the values of a [BlockState] or [InlineState].
	ExtendParser(c *Config)
}

// An ExtensionFunc is a function that is an [Extension].
type ExtensionFunc func(c *Config)

// ExtendParser implements [Extension]; it calls f(c).
func (f ExtensionFunc) ExtendParser(c *Config) { f(c) }

// A ParagraphTransformer changes each paragraph as it closes, after the
// paragraph loses the link reference definitions at its start, and before
// its inline content is parsed. It may remove the paragraph from its
// parent, which ends the transformers that come after it.
type ParagraphTransformer interface {
	Transform(p *markdown.Paragraph, s *BlockState)
}

// An ASTTransformer changes a document after a [Parser] parses it.
type ASTTransformer interface {
	Transform(doc *markdown.Document)
}

// A Prioritized is a parser or transformer with the priority that orders
// it among the others of its kind: lower priorities come first, and equal
// ones in the order in which they were added.
type Prioritized[T any] = markdown.Prioritized[T]

// A Config holds the syntax that a [Parser] recognizes, which options and
// extensions change. A new config holds the syntax of CommonMark:
// [DefaultBlockParsers], [DefaultInlineParsers], and the delimiter rules of
// "*" and "_".
type Config struct {
	// BlockParsers holds the block parsers, which each position of a line
	// where a block can start tries in order, until one opens a block. The
	// order is the precedence of the blocks' syntax.
	BlockParsers []Prioritized[BlockParser]

	// InlineParsers holds the inline parsers, which each byte that
	// triggers them tries in order, before the built-in links, images, line
	// breaks, and delimiters.
	InlineParsers []Prioritized[InlineParser]

	// WordParsers holds inline parsers that text tries at the start of
	// each word, where the byte before is not an ASCII letter or digit, if
	// the byte there triggers them.
	WordParsers []Prioritized[InlineParser]

	// DelimiterRules holds the characters whose runs delimit spans of
	// inline text, as "*" delimits emphasis, one for each character.
	DelimiterRules []DelimiterRule

	// DelimiterProcessors holds the processors of delimiter characters, in
	// place of the rules of the characters they process.
	DelimiterProcessors []DelimiterProcessor

	// ParagraphTransformers holds the transformers of paragraphs, in the
	// order in which they run.
	ParagraphTransformers []Prioritized[ParagraphTransformer]

	// ASTTransformers holds the transformers of documents, in the order in
	// which they run.
	ASTTransformers []Prioritized[ASTTransformer]

	// Attribute reports whether a heading can end with attributes in
	// braces, such as {#id .class}, which [ParseAttributes] parses and the
	// heading gets.
	Attribute bool

	// AutoHeadingID reports whether each heading without an attribute
	// "id" gets one, which the IDs of its document generate from the
	// source of its last line.
	AutoHeadingID bool

	// NewIDs returns the IDs of a document, for each document that the
	// parser parses. Nil means [NewIDs].
	NewIDs func() IDs

	// EscapedSpace reports whether a backslash before a space escapes it,
	// to stand for nothing, as text that its script does not part with
	// spaces needs, to delimit emphasis with punctuation that does not
	// show.
	EscapedSpace bool
}

// Apply applies opts to c, in order.
func (c *Config) Apply(opts ...Option) {
	for _, opt := range opts {
		opt(c)
	}
}

// WithExtensions adds the syntax of exts, in order.
func WithExtensions(exts ...Extension) Option {
	return func(c *Config) {
		for _, e := range exts {
			e.ExtendParser(c)
		}
	}
}

// WithBlockParser adds block parser p with priority.
func WithBlockParser(p BlockParser, priority int) Option {
	return func(c *Config) {
		c.BlockParsers = append(c.BlockParsers, Prioritized[BlockParser]{Value: p, Priority: priority})
	}
}

// WithInlineParser adds inline parser p with priority.
func WithInlineParser(p InlineParser, priority int) Option {
	return func(c *Config) {
		c.InlineParsers = append(c.InlineParsers, Prioritized[InlineParser]{Value: p, Priority: priority})
	}
}

// WithWordParser adds p, with priority, as an inline parser that text tries
// at the start of each word.
func WithWordParser(p InlineParser, priority int) Option {
	return func(c *Config) {
		c.WordParsers = append(c.WordParsers, Prioritized[InlineParser]{Value: p, Priority: priority})
	}
}

// WithDelimiterRule adds delimiter rule r, in place of a rule of the same
// character.
func WithDelimiterRule(r DelimiterRule) Option {
	return func(c *Config) {
		if i := slices.IndexFunc(c.DelimiterRules, func(e DelimiterRule) bool { return e.Char == r.Char }); i >= 0 {
			c.DelimiterRules[i] = r
			return
		}
		c.DelimiterRules = append(c.DelimiterRules, r)
	}
}

// WithDelimiterProcessor adds delimiter processor p, in place of the rules
// and the earlier processors of the characters that it processes.
func WithDelimiterProcessor(p DelimiterProcessor) Option {
	return func(c *Config) { c.DelimiterProcessors = append(c.DelimiterProcessors, p) }
}

// WithParagraphTransformer adds paragraph transformer t with priority.
func WithParagraphTransformer(t ParagraphTransformer, priority int) Option {
	return func(c *Config) {
		c.ParagraphTransformers = append(c.ParagraphTransformers, Prioritized[ParagraphTransformer]{Value: t, Priority: priority})
	}
}

// WithASTTransformer adds document transformer t with priority.
func WithASTTransformer(t ASTTransformer, priority int) Option {
	return func(c *Config) {
		c.ASTTransformers = append(c.ASTTransformers, Prioritized[ASTTransformer]{Value: t, Priority: priority})
	}
}

// WithAttribute lets a heading end with attributes in braces, such as
// {#id .class key=value}, which [ParseAttributes] parses and the heading
// gets.
func WithAttribute() Option { return func(c *Config) { c.Attribute = true } }

// WithAutoHeadingID gives each heading without an attribute "id" one, which
// the IDs of its document generate from the source of its last line, as
// "hello-world" for "# Hello, *World*!".
func WithAutoHeadingID() Option { return func(c *Config) { c.AutoHeadingID = true } }

// WithIDs sets the function that returns the IDs of each document that the
// parser parses, in place of [NewIDs].
func WithIDs(newIDs func() IDs) Option { return func(c *Config) { c.NewIDs = newIDs } }

// WithEscapedSpace lets a backslash escape a space, so that the two stand
// for nothing, as in "**“日本”**\ 語", in which they let the "**" close
// emphasis, as punctuation after it does, and do not show. A word after an
// escaped space does not start one. The spaces at the end of a line, which
// line breaks take, are not escaped.
func WithEscapedSpace() Option { return func(c *Config) { c.EscapedSpace = true } }

// DefaultBlockParsers returns the block parsers of CommonMark, with their
// priorities: [NewBlockquoteParser] 100, [NewATXHeadingParser] 200,
// [NewFencedCodeBlockParser] 300, [NewHTMLBlockParser] 400,
// [NewSetextHeadingParser] 500, [NewThematicBreakParser] 600,
// [NewListParser] 700, and [NewCodeBlockParser] 800. Paragraphs, which
// [NewParagraphParser] parses, start where no other block does.
func DefaultBlockParsers() []Prioritized[BlockParser] {
	return []Prioritized[BlockParser]{
		{Value: NewBlockquoteParser(), Priority: 100},
		{Value: NewATXHeadingParser(), Priority: 200},
		{Value: NewFencedCodeBlockParser(), Priority: 300},
		{Value: NewHTMLBlockParser(), Priority: 400},
		{Value: NewSetextHeadingParser(), Priority: 500},
		{Value: NewThematicBreakParser(), Priority: 600},
		{Value: NewListParser(), Priority: 700},
		{Value: NewCodeBlockParser(), Priority: 800},
	}
}

// DefaultInlineParsers returns the inline parsers of CommonMark, with their
// priorities: [NewEscapeParser] 100, [NewCodeSpanParser] 200,
// [NewAutoLinkParser] 300, [NewRawHTMLParser] 400, and [NewEntityParser]
// 500.
func DefaultInlineParsers() []Prioritized[InlineParser] {
	return []Prioritized[InlineParser]{
		{Value: NewEscapeParser(), Priority: 100},
		{Value: NewCodeSpanParser(), Priority: 200},
		{Value: NewAutoLinkParser(), Priority: 300},
		{Value: NewRawHTMLParser(), Priority: 400},
		{Value: NewEntityParser(), Priority: 500},
	}
}

// commonMark returns the config of CommonMark.
func commonMark() *Config {
	return &Config{
		BlockParsers:  DefaultBlockParsers(),
		InlineParsers: DefaultInlineParsers(),
		DelimiterRules: []DelimiterRule{
			{Char: '*', Intraword: true, New: emphasis},
			{Char: '_', New: emphasis},
		},
	}
}

// emphasis returns the emphasis that a span of text makes, which n
// delimiter characters at each end make strong if there are two.
func emphasis(n int) markdown.Node { return &markdown.Emphasis{Level: n} }

// syntax is the syntax of a Parser, which its config sets: the parsers in
// order, and the tables of the bytes that start blocks and inline elements.
type syntax struct {
	paragraph             LineAdderParser
	starts                [256][]BlockParser // block parsers by the byte that starts a block
	words                 []InlineParser
	paragraphTransformers []ParagraphTransformer
	astTransformers       []ASTTransformer

	attribute     bool       // headings can end with attributes
	autoHeadingID bool       // headings get IDs
	newIDs        func() IDs // the IDs of a document
	escapedSpace  bool       // a backslash escapes a space

	triggers  [256][]InlineParser // inline parsers by trigger
	wordStart [256]bool           // bytes that trigger words
	delims    []delimClass        // the classes of delimiter characters
	delimIdx  [256]int            // index in delims, plus 1, of each delimiter character
	special   [256]bool           // bytes that can start anything but text
}

// A LineAdderParser is a [BlockParser] that is also a [LineAdder].
type LineAdderParser interface {
	BlockParser
	LineAdder
}

// delimClass is how runs of a delimiter character are scanned and paired.
type delimClass struct {
	processor DelimiterProcessor
	maxRun    int
	intraword bool
}

// newSyntax returns the syntax that c sets.
func newSyntax(c *Config) *syntax {
	s := &syntax{
		paragraph:             NewParagraphParser().(LineAdderParser),
		words:                 sorted(c.WordParsers),
		paragraphTransformers: sorted(c.ParagraphTransformers),
		astTransformers:       sorted(c.ASTTransformers),
		attribute:             c.Attribute,
		autoHeadingID:         c.AutoHeadingID,
		newIDs:                c.NewIDs,
		escapedSpace:          c.EscapedSpace,
	}
	if s.newIDs == nil {
		s.newIDs = NewIDs
	}
	for _, bp := range sorted(c.BlockParsers) {
		trigger := bp.Trigger()
		for b := range 256 {
			if trigger == nil || slices.Contains(trigger, byte(b)) {
				s.starts[b] = append(s.starts[b], bp)
			}
		}
	}
	for _, b := range []byte("\n[]!") {
		s.special[b] = true
	}
	for _, ip := range sorted(c.InlineParsers) {
		for _, b := range ip.Trigger() {
			s.triggers[b] = append(s.triggers[b], ip)
			s.special[b] = true
		}
	}
	for _, w := range s.words {
		for _, b := range w.Trigger() {
			s.wordStart[b] = true
		}
	}
	setClass := func(b byte, class delimClass) {
		if s.delimIdx[b] == 0 {
			s.delims = append(s.delims, class)
			s.delimIdx[b] = len(s.delims)
		} else {
			s.delims[s.delimIdx[b]-1] = class
		}
		s.special[b] = true
	}
	for _, r := range c.DelimiterRules {
		setClass(r.Char, delimClass{processor: r, maxRun: r.MaxRun, intraword: r.Intraword})
	}
	for _, p := range c.DelimiterProcessors {
		for b := range 256 {
			if p.IsDelimiter(byte(b)) {
				setClass(byte(b), delimClass{processor: p, intraword: true})
			}
		}
	}
	return s
}

// sorted returns the values of ps in the order of their priorities.
func sorted[T any](ps []Prioritized[T]) []T {
	ps = slices.Clone(ps)
	slices.SortStableFunc(ps, func(a, b Prioritized[T]) int { return cmp.Compare(a.Priority, b.Priority) })
	out := make([]T, len(ps))
	for i, p := range ps {
		out[i] = p.Value
	}
	return out
}
