package ast

import (
	"iter"
	"slices"
	"sync"
)

// A NodeType is the type of a [Node]: a block, inline content, or a
// document.
type NodeType int

const (
	TypeBlock    NodeType = iota + 1 // a block, such as a paragraph
	TypeInline                       // inline content, such as text
	TypeDocument                     // the root of a document's tree
)

// A NodeKind identifies the kind of a [Node], such as [KindParagraph].
type NodeKind int

var (
	kindsMu   sync.Mutex
	kindNames []string
)

// NewNodeKind returns a new kind of node with name, which [NodeKind.String]
// returns.
func NewNodeKind(name string) NodeKind {
	kindsMu.Lock()
	defer kindsMu.Unlock()
	kindNames = append(kindNames, name)
	return NodeKind(len(kindNames) - 1)
}

// String returns the name of k.
func (k NodeKind) String() string {
	kindsMu.Lock()
	defer kindsMu.Unlock()
	if k < 0 || int(k) >= len(kindNames) {
		return "NodeKind(?)"
	}
	return kindNames[k]
}

// A Node is a node of a document's tree.
//
// The methods that change a node's children take the node itself as self,
// which they make the parent of the children they add, as the methods of
// [BaseNode] cannot find the node that embeds it.
type Node interface {
	// Type returns the type of the node.
	Type() NodeType

	// Kind returns the kind of the node.
	Kind() NodeKind

	// Parent returns the node's parent, or nil for a root or a node that
	// is not in a tree.
	Parent() Node

	// SetParent sets the node's parent. The methods that change children
	// set it; it does not change the children of v.
	SetParent(v Node)

	// PreviousSibling returns the child of the node's parent before it, or
	// nil.
	PreviousSibling() Node

	// SetPreviousSibling sets the sibling before the node, as the methods
	// that change children set it.
	SetPreviousSibling(v Node)

	// NextSibling returns the child of the node's parent after it, or nil.
	NextSibling() Node

	// SetNextSibling sets the sibling after the node, as the methods that
	// change children set it.
	SetNextSibling(v Node)

	// FirstChild returns the node's first child, or nil.
	FirstChild() Node

	// LastChild returns the node's last child, or nil.
	LastChild() Node

	// HasChildren reports whether the node has children.
	HasChildren() bool

	// ChildCount returns the number of the node's children.
	ChildCount() int

	// AppendChild adds child as the last child of self, removing it from
	// its parent first.
	AppendChild(self, child Node)

	// RemoveChild removes child from self, if it is a child of self.
	RemoveChild(self, child Node)

	// RemoveChildren removes the children of self.
	RemoveChildren(self Node)

	// ReplaceChild puts insertee in the place of child old of self.
	ReplaceChild(self, old, insertee Node)

	// InsertBefore adds insertee as a child of self before child before, or
	// as the last child if before is nil.
	InsertBefore(self, before, insertee Node)

	// InsertAfter adds insertee as a child of self after child after.
	InsertAfter(self, after, insertee Node)

	// Attributes returns the node's attributes, in the order they were set.
	Attributes() []Attribute

	// Attribute returns the value of the attribute named name.
	Attribute(name string) (any, bool)

	// SetAttribute sets the attribute named name to value.
	SetAttribute(name string, value any)

	// RemoveAttribute removes the attribute named name.
	RemoveAttribute(name string)

	// SourceSpan returns the range of the source that the node comes from.
	// The parser records it for blocks.
	SourceSpan() Span

	// SetSourceSpan sets the range of the source that the node comes from.
	SetSourceSpan(Span)

	// Lines returns the lines of source text that a block took, which hold
	// its inline content or code; nil for inline content.
	Lines() []Line

	// SetLines sets the lines of source text of a block.
	SetLines([]Line)
}

// An Attribute is an attribute of a [Node].
type Attribute struct {
	Name  string
	Value any
}

// BaseNode implements the parts of [Node] that every node shares: its
// links, attributes, and span. A node type embeds [BaseBlock] or
// [BaseInline], which embed BaseNode.
type BaseNode struct {
	parent, prev, next    Node
	firstChild, lastChild Node
	childCount            int
	attributes            []Attribute
	span                  Span
}

// Parent implements [Node].
func (n *BaseNode) Parent() Node { return n.parent }

// SetParent implements [Node].
func (n *BaseNode) SetParent(v Node) { n.parent = v }

// PreviousSibling implements [Node].
func (n *BaseNode) PreviousSibling() Node { return n.prev }

// SetPreviousSibling implements [Node].
func (n *BaseNode) SetPreviousSibling(v Node) { n.prev = v }

// NextSibling implements [Node].
func (n *BaseNode) NextSibling() Node { return n.next }

// SetNextSibling implements [Node].
func (n *BaseNode) SetNextSibling(v Node) { n.next = v }

// FirstChild implements [Node].
func (n *BaseNode) FirstChild() Node { return n.firstChild }

// LastChild implements [Node].
func (n *BaseNode) LastChild() Node { return n.lastChild }

// HasChildren implements [Node].
func (n *BaseNode) HasChildren() bool { return n.firstChild != nil }

// ChildCount implements [Node].
func (n *BaseNode) ChildCount() int { return n.childCount }

// AppendChild implements [Node].
func (n *BaseNode) AppendChild(self, child Node) {
	isolate(child)
	child.SetParent(self)
	child.SetPreviousSibling(n.lastChild)
	child.SetNextSibling(nil)
	if n.lastChild != nil {
		n.lastChild.SetNextSibling(child)
	} else {
		n.firstChild = child
	}
	n.lastChild = child
	n.childCount++
}

// RemoveChild implements [Node].
func (n *BaseNode) RemoveChild(self, child Node) {
	if child.Parent() != self {
		return
	}
	prev, next := child.PreviousSibling(), child.NextSibling()
	if prev != nil {
		prev.SetNextSibling(next)
	} else {
		n.firstChild = next
	}
	if next != nil {
		next.SetPreviousSibling(prev)
	} else {
		n.lastChild = prev
	}
	child.SetParent(nil)
	child.SetPreviousSibling(nil)
	child.SetNextSibling(nil)
	n.childCount--
}

// RemoveChildren implements [Node].
func (n *BaseNode) RemoveChildren(self Node) {
	for c := n.firstChild; c != nil; {
		next := c.NextSibling()
		c.SetParent(nil)
		c.SetPreviousSibling(nil)
		c.SetNextSibling(nil)
		c = next
	}
	n.firstChild, n.lastChild, n.childCount = nil, nil, 0
}

// ReplaceChild implements [Node].
func (n *BaseNode) ReplaceChild(self, old, insertee Node) {
	if old.Parent() != self {
		return
	}
	n.InsertBefore(self, old, insertee)
	n.RemoveChild(self, old)
}

// InsertBefore implements [Node].
func (n *BaseNode) InsertBefore(self, before, insertee Node) {
	if before == nil {
		n.AppendChild(self, insertee)
		return
	}
	if before.Parent() != self || before == insertee {
		return
	}
	isolate(insertee)
	prev := before.PreviousSibling()
	insertee.SetParent(self)
	insertee.SetPreviousSibling(prev)
	insertee.SetNextSibling(before)
	before.SetPreviousSibling(insertee)
	if prev != nil {
		prev.SetNextSibling(insertee)
	} else {
		n.firstChild = insertee
	}
	n.childCount++
}

// InsertAfter implements [Node].
func (n *BaseNode) InsertAfter(self, after, insertee Node) {
	if after.Parent() != self || after == insertee {
		return
	}
	n.InsertBefore(self, after.NextSibling(), insertee)
}

// isolate removes v from its parent.
func isolate(v Node) {
	if p := v.Parent(); p != nil {
		p.RemoveChild(p, v)
	}
}

// Attributes implements [Node].
func (n *BaseNode) Attributes() []Attribute { return n.attributes }

// Attribute implements [Node].
func (n *BaseNode) Attribute(name string) (any, bool) {
	for _, a := range n.attributes {
		if a.Name == name {
			return a.Value, true
		}
	}
	return nil, false
}

// SetAttribute implements [Node].
func (n *BaseNode) SetAttribute(name string, value any) {
	for i, a := range n.attributes {
		if a.Name == name {
			n.attributes[i].Value = value
			return
		}
	}
	n.attributes = append(n.attributes, Attribute{Name: name, Value: value})
}

// RemoveAttribute implements [Node].
func (n *BaseNode) RemoveAttribute(name string) {
	n.attributes = slices.DeleteFunc(n.attributes, func(a Attribute) bool { return a.Name == name })
}

// SourceSpan implements [Node].
func (n *BaseNode) SourceSpan() Span { return n.span }

// SetSourceSpan implements [Node].
func (n *BaseNode) SetSourceSpan(s Span) { n.span = s }

// BaseBlock implements [Node] for a block. A block type embeds it.
type BaseBlock struct {
	BaseNode
	lines []Line
}

// Type implements [Node]; it returns [TypeBlock].
func (*BaseBlock) Type() NodeType { return TypeBlock }

// Lines implements [Node].
func (b *BaseBlock) Lines() []Line { return b.lines }

// SetLines implements [Node].
func (b *BaseBlock) SetLines(lines []Line) { b.lines = lines }

// BaseInline implements [Node] for inline content. An inline type embeds
// it.
type BaseInline struct {
	BaseNode
}

// Type implements [Node]; it returns [TypeInline].
func (*BaseInline) Type() NodeType { return TypeInline }

// Lines implements [Node]; inline content has no lines.
func (*BaseInline) Lines() []Line { return nil }

// SetLines implements [Node]; it does nothing for inline content.
func (*BaseInline) SetLines([]Line) {}

// Children returns the children of n, in order. Removing the child that it
// returns while ranging over them ends the range.
func Children(n Node) iter.Seq[Node] {
	return func(yield func(Node) bool) {
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if !yield(c) {
				return
			}
		}
	}
}

// A WalkStatus tells [Walk] how to go on after a [Walker] visits a node.
type WalkStatus int

const (
	WalkStop         WalkStatus = iota + 1 // stop walking
	WalkSkipChildren                       // skip the node's children
	WalkContinue                           // go on
)

// A Walker visits a node as [Walk] enters it and as it leaves it, after its
// children.
type Walker func(n Node, entering bool) (WalkStatus, error)

// Walk visits n and its descendants, in depth-first order, with walker,
// until walker returns [WalkStop] or an error, which Walk returns.
func Walk(n Node, walker Walker) error {
	_, err := walk(n, walker)
	return err
}

// walk visits n and its descendants with walker, and reports whether to
// stop.
func walk(n Node, walker Walker) (WalkStatus, error) {
	status, err := walker(n, true)
	if err != nil || status == WalkStop {
		return WalkStop, err
	}
	if status != WalkSkipChildren {
		for c := n.FirstChild(); c != nil; {
			next := c.NextSibling()
			if st, err := walk(c, walker); err != nil || st == WalkStop {
				return WalkStop, err
			}
			c = next
		}
	}
	if status, err = walker(n, false); err != nil || status == WalkStop {
		return WalkStop, err
	}
	return WalkContinue, nil
}
