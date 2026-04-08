package widget

import (
	"fmt"
	"sync/atomic"

	"github.com/unaquatic/fluid/style"
)

// ---------------------------------------------------------------------------
// TextNode
// ---------------------------------------------------------------------------
//
// TextNode forms a tree of inline styled text. Each node can override
// fg, bg, attributes, and link; children inherit from their parent.
//
// Text modifier elements (<span>, <b>, <i>, <u>, <a>, <br>) are all
// represented as TextNode instances with pre-configured options.

var textNodeNextID uint64

// TextChild is a child of a TextNode: either a plain string or another *TextNode.
type TextChild struct {
	Text string    // non-empty when this child is a plain string
	Node *TextNode // non-nil when this child is a sub-node
}

// IsText returns true if this child is a plain string.
func (tc TextChild) IsText() bool { return tc.Node == nil }

// StringChild creates a TextChild from a plain string.
func StringChild(s string) TextChild {
	return TextChild{Text: s}
}

// NodeChild creates a TextChild from a *TextNode.
func NodeChild(n *TextNode) TextChild {
	return TextChild{Node: n}
}

// TextNode represents an inline text element within a Text widget.
// Each node can have:
//   - Optional fg, bg, attributes, link overrides (nil = inherit from parent)
//   - Children that are either string literals or other TextNodes
//
// Style inheritance works like CSS: child nodes inherit parent styles unless
// they provide their own overrides. Attributes (bold, italic, etc.) are merged
// via bitwise OR with the parent's attributes.
//
// The tree is flattened via GatherChunks() into []style.TextChunk for rendering.
type TextNode struct {
	id         string
	fg         *style.Color // nil = inherit from parent
	bg         *style.Color // nil = inherit
	attributes uint32       // bitmask, OR'd with inherited
	link       string       // URL, empty = inherit

	children []TextChild
	parent   *TextNode
	dirty    bool

	// ownerText is the Text widget that owns the root TextNode.
	// Set only on the root node; used to propagate dirty state.
	ownerText interface{ textNodeDirtied() }
}

// TextNodeOption configures a TextNode during construction.
type TextNodeOption func(*TextNode)

// NewTextNode creates a new TextNode with optional configuration.
func NewTextNode(opts ...TextNodeOption) *TextNode {
	n := &TextNode{
		id:    fmt.Sprintf("textnode-%d", atomic.AddUint64(&textNodeNextID, 1)),
		dirty: true,
	}
	for _, opt := range opts {
		opt(n)
	}
	return n
}

// ---------------------------------------------------------------------------
// TextNode Options
// ---------------------------------------------------------------------------

// TextNodeWithID sets the node ID.
func TextNodeWithID(id string) TextNodeOption {
	return func(n *TextNode) { n.id = id }
}

// TextNodeWithFg sets the foreground color.
func TextNodeWithFg(c style.Color) TextNodeOption {
	return func(n *TextNode) { n.fg = &c }
}

// TextNodeWithBg sets the background color.
func TextNodeWithBg(c style.Color) TextNodeOption {
	return func(n *TextNode) { n.bg = &c }
}

// TextNodeWithAttributes sets text attributes.
func TextNodeWithAttributes(a uint32) TextNodeOption {
	return func(n *TextNode) { n.attributes = a }
}

// TextNodeWithLink sets the hyperlink URL.
func TextNodeWithLink(url string) TextNodeOption {
	return func(n *TextNode) { n.link = url }
}

// ---------------------------------------------------------------------------
// Identity
// ---------------------------------------------------------------------------

// ID returns the node's identifier.
func (n *TextNode) ID() string { return n.id }

// SetID sets the node's identifier.
func (n *TextNode) SetID(id string) { n.id = id }

// ---------------------------------------------------------------------------
// Tree Manipulation
// ---------------------------------------------------------------------------

// Parent returns the parent TextNode, or nil if this is the root.
func (n *TextNode) Parent() *TextNode { return n.parent }

// Children returns the list of children.
func (n *TextNode) ChildrenList() []TextChild { return n.children }

// ChildrenCount returns the number of children.
func (n *TextNode) ChildrenCount() int { return len(n.children) }

// Add appends a child (string or *TextNode). Returns the insertion index.
func (n *TextNode) Add(child TextChild) int {
	idx := len(n.children)
	n.children = append(n.children, child)
	if child.Node != nil {
		if child.Node.parent != nil {
			child.Node.parent.Remove(child.Node.id)
		}
		child.Node.parent = n
	}
	n.markDirty()
	return idx
}

// AddString appends a plain string child. Returns the insertion index.
func (n *TextNode) AddString(s string) int {
	return n.Add(StringChild(s))
}

// AddNode appends a *TextNode child. Returns the insertion index.
func (n *TextNode) AddNode(child *TextNode) int {
	return n.Add(NodeChild(child))
}

// InsertAt inserts a child at the given index.
func (n *TextNode) InsertAt(index int, child TextChild) {
	if index < 0 {
		index = 0
	}
	if index > len(n.children) {
		index = len(n.children)
	}
	n.children = append(n.children, TextChild{})
	copy(n.children[index+1:], n.children[index:])
	n.children[index] = child
	if child.Node != nil {
		if child.Node.parent != nil {
			child.Node.parent.Remove(child.Node.id)
		}
		child.Node.parent = n
	}
	n.markDirty()
}

// Remove removes a child TextNode by ID. Returns true if found.
func (n *TextNode) Remove(id string) bool {
	for i, child := range n.children {
		if child.Node != nil && child.Node.id == id {
			child.Node.parent = nil
			n.children = append(n.children[:i], n.children[i+1:]...)
			n.markDirty()
			return true
		}
	}
	return false
}

// RemoveAt removes a child at the given index.
func (n *TextNode) RemoveAt(index int) {
	if index < 0 || index >= len(n.children) {
		return
	}
	if n.children[index].Node != nil {
		n.children[index].Node.parent = nil
	}
	n.children = append(n.children[:index], n.children[index+1:]...)
	n.markDirty()
}

// Clear removes all children.
func (n *TextNode) Clear() {
	for _, child := range n.children {
		if child.Node != nil {
			child.Node.parent = nil
		}
	}
	n.children = nil
	n.markDirty()
}

// ReplaceChildren replaces all children at once.
func (n *TextNode) ReplaceChildren(children []TextChild) {
	// Detach old children
	for _, child := range n.children {
		if child.Node != nil {
			child.Node.parent = nil
		}
	}
	n.children = children
	// Attach new children
	for _, child := range n.children {
		if child.Node != nil {
			child.Node.parent = n
		}
	}
	n.markDirty()
}

// ReplaceTextAt replaces the child at the given index with a new string.
func (n *TextNode) ReplaceTextAt(text string, index int) {
	if index < 0 || index >= len(n.children) {
		return
	}
	if n.children[index].Node != nil {
		n.children[index].Node.parent = nil
	}
	n.children[index] = TextChild{Text: text}
	n.markDirty()
}

// ReplaceNodeAt replaces the child at the given index with a TextNode.
func (n *TextNode) ReplaceNodeAt(child *TextNode, index int) {
	if index < 0 || index >= len(n.children) {
		return
	}
	if n.children[index].Node != nil {
		n.children[index].Node.parent = nil
	}
	if child.parent != nil {
		child.parent.Remove(child.id)
	}
	child.parent = n
	n.children[index] = TextChild{Node: child}
	n.markDirty()
}

// InsertBefore inserts a child node before the anchor node (by ID).
func (n *TextNode) InsertBefore(child *TextNode, anchorID string) {
	for i, c := range n.children {
		if c.Node != nil && c.Node.id == anchorID {
			n.InsertAt(i, NodeChild(child))
			return
		}
	}
	// Anchor not found, append
	n.AddNode(child)
}

// ---------------------------------------------------------------------------
// Style Getters/Setters
// ---------------------------------------------------------------------------

// Fg returns the foreground color override, or nil.
func (n *TextNode) Fg() *style.Color { return n.fg }

// Bg returns the background color override, or nil.
func (n *TextNode) Bg() *style.Color { return n.bg }

// Attributes returns the attribute bitmask.
func (n *TextNode) Attributes() uint32 { return n.attributes }

// Link returns the link URL, or "".
func (n *TextNode) Link() string { return n.link }

// SetFg sets the foreground color override.
func (n *TextNode) SetFg(c *style.Color) {
	n.fg = c
	n.markDirty()
}

// SetBg sets the background color override.
func (n *TextNode) SetBg(c *style.Color) {
	n.bg = c
	n.markDirty()
}

// SetAttributes sets the attribute bitmask.
func (n *TextNode) SetAttributes(a uint32) {
	n.attributes = a
	n.markDirty()
}

// SetLink sets the link URL.
func (n *TextNode) SetLink(url string) {
	n.link = url
	n.markDirty()
}

// ---------------------------------------------------------------------------
// Dirty Tracking
// ---------------------------------------------------------------------------

// IsDirty returns true if this node or any descendant has been modified.
func (n *TextNode) IsDirty() bool { return n.dirty }

// ClearDirty clears the dirty flag on this node and all descendants.
func (n *TextNode) ClearDirty() {
	n.dirty = false
	for _, child := range n.children {
		if child.Node != nil {
			child.Node.ClearDirty()
		}
	}
}

func (n *TextNode) markDirty() {
	n.dirty = true
	// Propagate up to parent
	if n.parent != nil {
		n.parent.markDirty()
	}
	// Notify owner Text widget via the root node
	if n.ownerText != nil {
		n.ownerText.textNodeDirtied()
	}
	// Walk up to root to find ownerText (for non-root mutations)
	root := n
	for root.parent != nil {
		root = root.parent
	}
	if root.ownerText != nil && root != n {
		root.ownerText.textNodeDirtied()
	}
}

// ---------------------------------------------------------------------------
// Style Gathering / Flattening
// ---------------------------------------------------------------------------

// InheritedStyle is the style context passed down during chunk gathering.
type InheritedStyle struct {
	Fg         style.Color
	Bg         style.Color
	Attributes uint32
	Link       string
}

// GatherChunks recursively collects all text content as styled chunks,
// inheriting styles from ancestors. This is the main method that converts
// the TextNode tree into a flat []style.TextChunk for rendering.
func (n *TextNode) GatherChunks(inherited InheritedStyle) []style.TextChunk {
	// Resolve this node's effective style
	effective := n.resolveStyle(inherited)

	var chunks []style.TextChunk

	for _, child := range n.children {
		if child.IsText() {
			if len(child.Text) > 0 {
				fg := effective.Fg
				bg := effective.Bg
				chunk := style.TextChunk{
					Text:       child.Text,
					Fg:         &fg,
					Bg:         &bg,
					Attributes: effective.Attributes,
				}
				if effective.Link != "" {
					chunk.Link = effective.Link
				}
				chunks = append(chunks, chunk)
			}
		} else if child.Node != nil {
			childChunks := child.Node.GatherChunks(effective)
			chunks = append(chunks, childChunks...)
		}
	}

	return chunks
}

// resolveStyle merges this node's overrides with the inherited style.
func (n *TextNode) resolveStyle(inherited InheritedStyle) InheritedStyle {
	result := inherited

	if n.fg != nil {
		result.Fg = *n.fg
	}
	if n.bg != nil {
		result.Bg = *n.bg
	}
	// Attributes are OR'd (additive)
	result.Attributes = inherited.Attributes | n.attributes
	if n.link != "" {
		result.Link = n.link
	}

	return result
}

// PlainText returns the concatenated plain text of all children.
func (n *TextNode) PlainText() string {
	var sb []byte
	n.plainTextInto(&sb)
	return string(sb)
}

func (n *TextNode) plainTextInto(buf *[]byte) {
	for _, child := range n.children {
		if child.IsText() {
			*buf = append(*buf, child.Text...)
		} else if child.Node != nil {
			child.Node.plainTextInto(buf)
		}
	}
}

// ---------------------------------------------------------------------------
// Text Modifier Nodes — convenience constructors
// ---------------------------------------------------------------------------

// NewSpanNode creates a generic inline styled span.
// Maps to <span> element / SpanRenderable.
func NewSpanNode(opts ...TextNodeOption) *TextNode {
	return NewTextNode(opts...)
}

// NewBoldNode creates a bold text node. Maps to <b> / <strong>.
// Accepts optional initial children.
func NewBoldNode(children ...TextChild) *TextNode {
	n := NewTextNode(TextNodeWithAttributes(style.AttrBold))
	for _, child := range children {
		n.Add(child)
	}
	return n
}

// NewItalicNode creates an italic text node. Maps to <i> / <em>.
// Accepts optional initial children.
func NewItalicNode(children ...TextChild) *TextNode {
	n := NewTextNode(TextNodeWithAttributes(style.AttrItalic))
	for _, child := range children {
		n.Add(child)
	}
	return n
}

// NewUnderlineNode creates an underlined text node. Maps to <u>.
// Accepts optional initial children.
func NewUnderlineNode(children ...TextChild) *TextNode {
	n := NewTextNode(TextNodeWithAttributes(style.AttrUnderline))
	for _, child := range children {
		n.Add(child)
	}
	return n
}

// NewLinkNode creates a hyperlink text node. Maps to <a href="...">.
// Automatically adds underline attribute.
// Accepts optional initial children.
func NewLinkNode(href string, children ...TextChild) *TextNode {
	n := NewTextNode(
		TextNodeWithLink(href),
		TextNodeWithAttributes(style.AttrUnderline),
	)
	for _, child := range children {
		n.Add(child)
	}
	return n
}

// NewLineBreakNode creates a line break. Maps to <br>.
func NewLineBreakNode() *TextNode {
	n := NewTextNode()
	n.AddString("\n")
	return n
}

// ---------------------------------------------------------------------------
// Convenience: Styled text builders
// ---------------------------------------------------------------------------

// SpanText creates a span node containing the given text.
func SpanText(text string, opts ...TextNodeOption) *TextNode {
	n := NewSpanNode(opts...)
	n.AddString(text)
	return n
}

// BoldText creates a bold node containing the given text.
func BoldText(text string) *TextNode {
	return NewBoldNode(StringChild(text))
}

// ItalicText creates an italic node containing the given text.
func ItalicText(text string) *TextNode {
	return NewItalicNode(StringChild(text))
}

// UnderlineText creates an underline node containing the given text.
func UnderlineText(text string) *TextNode {
	return NewUnderlineNode(StringChild(text))
}

// LinkText creates a link node containing the given text.
func LinkText(text string, href string) *TextNode {
	return NewLinkNode(href, StringChild(text))
}
