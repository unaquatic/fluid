package widget

import (
	"fmt"
	"sync/atomic"

	"github.com/unaquatic/fluid/style"
)

var textNodeNextID atomic.Uint64

// TextChild contains either plain text or a nested TextNode.
type TextChild struct {
	Text string    // Used when Node is nil.
	Node *TextNode // Non-nil for a nested node.
}

// IsText reports whether the child contains text instead of a nested node.
func (tc TextChild) IsText() bool { return tc.Node == nil }

// StringChild returns a plain-text child containing s.
func StringChild(s string) TextChild {
	return TextChild{Text: s}
}

// NodeChild returns a nested-node child containing n.
func NodeChild(n *TextNode) TextChild {
	return TextChild{Node: n}
}

// TextNode contains ordered text and node children with inheritable style overrides.
// Foreground, background, and link values replace inherited values. Attributes
// combine with inherited attributes. GatherChunks flattens the tree for rendering.
type TextNode struct {
	id         string
	fg         *style.Color // Nil inherits the parent foreground.
	bg         *style.Color // Nil inherits the parent background.
	attributes uint32       // Combined with inherited attributes.
	link       string       // An empty value inherits the parent link.

	children []TextChild
	parent   *TextNode
	dirty    bool

	// ownerText is set on the root node to propagate mutations to its Text widget.
	ownerText interface{ textNodeDirtied() }
}

// TextNodeOption configures a TextNode during construction.
type TextNodeOption func(*TextNode)

// NewTextNode returns an empty, dirty TextNode with optional configuration.
func NewTextNode(opts ...TextNodeOption) *TextNode {
	n := &TextNode{
		id:    fmt.Sprintf("textnode-%d", textNodeNextID.Add(1)),
		dirty: true,
	}
	for _, opt := range opts {
		opt(n)
	}
	return n
}

// TextNode options.

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

// ID returns the node's identifier.
func (n *TextNode) ID() string { return n.id }

// SetID sets the node's identifier.
func (n *TextNode) SetID(id string) { n.id = id }

// Tree manipulation.

// Parent returns the parent TextNode, or nil if this is the root.
func (n *TextNode) Parent() *TextNode { return n.parent }

// ChildrenList returns the node's ordered children.
func (n *TextNode) ChildrenList() []TextChild { return n.children }

// ChildrenCount returns the number of children.
func (n *TextNode) ChildrenCount() int { return len(n.children) }

// Add appends child, reparents a nested node, and returns its insertion index.
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

// AddString appends plain text and returns its insertion index.
func (n *TextNode) AddString(s string) int {
	return n.Add(StringChild(s))
}

// AddNode appends and reparents child, then returns its insertion index.
func (n *TextNode) AddNode(child *TextNode) int {
	return n.Add(NodeChild(child))
}

// InsertAt inserts child at a clamped index and reparents nested nodes.
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

// Remove detaches the direct child node with id and reports whether it was found.
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

// RemoveAt removes and detaches the child at index when it exists.
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

// ReplaceChildren detaches existing nodes and adopts children.
func (n *TextNode) ReplaceChildren(children []TextChild) {
	// Remove parent links from the previous child nodes.
	for _, child := range n.children {
		if child.Node != nil {
			child.Node.parent = nil
		}
	}
	n.children = children
	// Point each new child node back to this parent.
	for _, child := range n.children {
		if child.Node != nil {
			child.Node.parent = n
		}
	}
	n.markDirty()
}

// ReplaceTextAt replaces an existing child with plain text.
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

// ReplaceNodeAt replaces an existing child and reparents the new node.
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

// InsertBefore inserts child before the direct child named by anchorID.
func (n *TextNode) InsertBefore(child *TextNode, anchorID string) {
	for i, c := range n.children {
		if c.Node != nil && c.Node.id == anchorID {
			n.InsertAt(i, NodeChild(child))
			return
		}
	}
	// Append when no direct child has the requested ID.
	n.AddNode(child)
}

// Style access.

// Fg returns the foreground override, or nil when it is inherited.
func (n *TextNode) Fg() *style.Color { return n.fg }

// Bg returns the background override, or nil when it is inherited.
func (n *TextNode) Bg() *style.Color { return n.bg }

// Attributes returns the attribute bitmask.
func (n *TextNode) Attributes() uint32 { return n.attributes }

// Link returns the link override, or an empty string when it is inherited.
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

// Dirty tracking.

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
	// Mark ancestors so root state reflects descendant mutations.
	if n.parent != nil {
		n.parent.markDirty()
	}
	// Notify immediately when this node is the owned root.
	if n.ownerText != nil {
		n.ownerText.textNodeDirtied()
	}
	// Descendant mutations find the owner through their root.
	root := n
	for root.parent != nil {
		root = root.parent
	}
	if root.ownerText != nil && root != n {
		root.ownerText.textNodeDirtied()
	}
}

// Style inheritance and flattening.

// InheritedStyle contains the effective parent style passed to child nodes.
type InheritedStyle struct {
	Fg         style.Color
	Bg         style.Color
	Attributes uint32
	Link       string
}

// GatherChunks flattens descendant text into chunks with inherited styles resolved.
func (n *TextNode) GatherChunks(inherited InheritedStyle) []style.TextChunk {
	// Resolve once and pass the effective style to every child.
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
	// Attributes are additive across the ancestry chain.
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

// Styled-node constructors.

// NewSpanNode returns a generic inline style node.
func NewSpanNode(opts ...TextNodeOption) *TextNode {
	return NewTextNode(opts...)
}

// NewBoldNode returns a bold node containing children.
func NewBoldNode(children ...TextChild) *TextNode {
	n := NewTextNode(TextNodeWithAttributes(style.AttrBold))
	for _, child := range children {
		n.Add(child)
	}
	return n
}

// NewItalicNode returns an italic node containing children.
func NewItalicNode(children ...TextChild) *TextNode {
	n := NewTextNode(TextNodeWithAttributes(style.AttrItalic))
	for _, child := range children {
		n.Add(child)
	}
	return n
}

// NewUnderlineNode returns an underlined node containing children.
func NewUnderlineNode(children ...TextChild) *TextNode {
	n := NewTextNode(TextNodeWithAttributes(style.AttrUnderline))
	for _, child := range children {
		n.Add(child)
	}
	return n
}

// NewLinkNode returns an underlined hyperlink node containing children.
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

// NewLineBreakNode returns a node containing one newline.
func NewLineBreakNode() *TextNode {
	n := NewTextNode()
	n.AddString("\n")
	return n
}

// Text-containing node constructors.

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
