package ast

import "strconv"

// Kinds of the inline content of CommonMark, GitHub Flavored Markdown, and
// the other extensions of the parser.
var (
	KindText             = NewNodeKind("Text")
	KindCodeSpan         = NewNodeKind("CodeSpan")
	KindEmphasis         = NewNodeKind("Emphasis")
	KindLink             = NewNodeKind("Link")
	KindImage            = NewNodeKind("Image")
	KindAutoLink         = NewNodeKind("AutoLink")
	KindRawHTML          = NewNodeKind("RawHTML")
	KindStrikethrough    = NewNodeKind("Strikethrough")
	KindFootnoteLink     = NewNodeKind("FootnoteLink")
	KindFootnoteBacklink = NewNodeKind("FootnoteBacklink")
)

// A Text is literal text, with escapes and entities resolved, and the line
// break after it, if any.
type Text struct {
	BaseInline
	Value string
	flags uint8
}

const (
	softLineBreak uint8 = 1 << iota
	hardLineBreak
)

// Kind implements [Node].
func (*Text) Kind() NodeKind { return KindText }

// SoftLineBreak reports whether a line ending that is not a hard line break
// follows the text.
func (t *Text) SoftLineBreak() bool { return t.flags&softLineBreak != 0 }

// SetSoftLineBreak sets whether a soft line break follows the text.
func (t *Text) SetSoftLineBreak(v bool) { t.setFlag(softLineBreak, v) }

// HardLineBreak reports whether a hard line break follows the text.
func (t *Text) HardLineBreak() bool { return t.flags&hardLineBreak != 0 }

// SetHardLineBreak sets whether a hard line break follows the text.
func (t *Text) SetHardLineBreak(v bool) { t.setFlag(hardLineBreak, v) }

func (t *Text) setFlag(f uint8, v bool) {
	if v {
		t.flags |= f
	} else {
		t.flags &^= f
	}
}

// NewText returns text holding value.
func NewText(value string) *Text { return &Text{Value: value} }

// A CodeSpan is a code span, whose children are [*Text] nodes holding its
// code.
type CodeSpan struct {
	BaseInline
}

// Kind implements [Node].
func (*CodeSpan) Kind() NodeKind { return KindCodeSpan }

// An Emphasis is emphasized text, of level 1, or strongly emphasized text,
// of level 2.
type Emphasis struct {
	BaseInline
	Level int
}

// Kind implements [Node].
func (*Emphasis) Kind() NodeKind { return KindEmphasis }

// A Link is an inline or reference link, whose children are its text.
type Link struct {
	BaseInline
	Destination string // with escapes and entities resolved
	Title       string // with escapes and entities resolved
}

// Kind implements [Node].
func (*Link) Kind() NodeKind { return KindLink }

// An Image is an inline or reference image, whose children are its
// description.
type Image struct {
	BaseInline
	Destination string // the source, with escapes and entities resolved
	Title       string // with escapes and entities resolved
}

// Kind implements [Node].
func (*Image) Kind() NodeKind { return KindImage }

// An AutoLinkType is the type of an [AutoLink].
type AutoLinkType int

const (
	AutoLinkURL   AutoLinkType = iota // a URL
	AutoLinkEmail                     // an email address
)

// String returns the name of t: "url" or "email".
func (t AutoLinkType) String() string {
	switch t {
	case AutoLinkURL:
		return "url"
	case AutoLinkEmail:
		return "email"
	}
	return "AutoLinkType(" + strconv.Itoa(int(t)) + ")"
}

// An AutoLink is an autolink: a URI or email address in angle brackets, or,
// with GFM, an extended autolink, which needs none.
type AutoLink struct {
	BaseInline
	AutoLinkType AutoLinkType
	URL          string // the destination, such as "http://" + Label for "www." Label
	Label        string // the link text, as written
}

// Kind implements [Node].
func (*AutoLink) Kind() NodeKind { return KindAutoLink }

// A RawHTML is an inline HTML tag.
type RawHTML struct {
	BaseInline
	Value string // as written
}

// Kind implements [Node].
func (*RawHTML) Kind() NodeKind { return KindRawHTML }

// A Strikethrough is struck-through text, which GFM adds, whose children are
// its text.
type Strikethrough struct {
	BaseInline
}

// Kind implements [Node].
func (*Strikethrough) Kind() NodeKind { return KindStrikethrough }

// A FootnoteLink is a reference to a [*Footnote], such as "[^1]".
type FootnoteLink struct {
	BaseInline

	// Ref is the label of the footnote, as its definition writes it, and the
	// reference too.
	Ref string

	// Index is the number of the footnote.
	Index int

	// RefCount is the number of references to the footnote, and RefIndex
	// the number of those before this one.
	RefCount, RefIndex int
}

// Kind implements [Node].
func (*FootnoteLink) Kind() NodeKind { return KindFootnoteLink }

// A FootnoteBacklink links a [*Footnote] back to one of the references to
// it, at the end of the footnote's last paragraph, or of the footnote.
type FootnoteBacklink struct {
	BaseInline

	// Index is the number of the footnote.
	Index int

	// RefCount is the number of references to the footnote, and RefIndex
	// the number of those before the one that the backlink links to.
	RefCount, RefIndex int
}

// Kind implements [Node].
func (*FootnoteBacklink) Kind() NodeKind { return KindFootnoteBacklink }
