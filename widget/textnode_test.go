package widget

import (
	"testing"

	"github.com/unaquatic/fluid/style"
)

// TextNode construction.

func TestNewTextNode(t *testing.T) {
	n := NewTextNode()
	if n.ID() == "" {
		t.Error("expected auto-generated ID")
	}
	if n.Fg() != nil {
		t.Error("expected nil Fg by default")
	}
	if n.Bg() != nil {
		t.Error("expected nil Bg by default")
	}
	if n.Attributes() != 0 {
		t.Error("expected 0 attributes by default")
	}
}

func TestTextNode_WithOptions(t *testing.T) {
	fg := style.ColorRed
	n := NewTextNode(
		TextNodeWithID("test-span"),
		TextNodeWithFg(fg),
		TextNodeWithAttributes(style.AttrBold),
	)
	if n.ID() != "test-span" {
		t.Errorf("expected id 'test-span', got %q", n.ID())
	}
	if n.Fg() == nil || !n.Fg().Equals(fg) {
		t.Error("expected red Fg")
	}
	if n.Attributes() != style.AttrBold {
		t.Errorf("expected AttrBold, got %d", n.Attributes())
	}
}

// Plain-text flattening.

func TestTextNode_PlainText(t *testing.T) {
	node := NewTextNode()
	node.AddString("Hello")
	node.AddString(" World")

	if got := node.PlainText(); got != "Hello World" {
		t.Errorf("expected 'Hello World', got %q", got)
	}
}

func TestTextNode_NestedPlainText(t *testing.T) {
	root := NewTextNode()
	root.AddString("A")

	child := NewTextNode()
	child.AddString("B")
	root.AddNode(child)

	root.AddString("C")

	if got := root.PlainText(); got != "ABC" {
		t.Errorf("expected 'ABC', got %q", got)
	}
}

// Basic chunk gathering.

func defaultInherited() InheritedStyle {
	return InheritedStyle{Fg: style.ColorWhite, Bg: style.ColorTransparent}
}

func TestTextNode_GatherChunks_Simple(t *testing.T) {
	n := NewTextNode()
	n.AddString("hello world")

	chunks := n.GatherChunks(defaultInherited())
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Text != "hello world" {
		t.Errorf("expected 'hello world', got %q", chunks[0].Text)
	}
	if chunks[0].Fg == nil || !chunks[0].Fg.Equals(style.ColorWhite) {
		t.Error("expected white Fg")
	}
}

func TestTextNode_GatherChunks_WithFg(t *testing.T) {
	node := NewTextNode(TextNodeWithFg(style.ColorRed))
	node.AddString("hello")

	chunks := node.GatherChunks(defaultInherited())
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Text != "hello" {
		t.Errorf("expected 'hello', got %q", chunks[0].Text)
	}
	if chunks[0].Fg == nil || !chunks[0].Fg.Equals(style.ColorRed) {
		t.Errorf("expected Red fg, got %v", chunks[0].Fg)
	}
}

// Style inheritance and overrides.

func TestTextNode_GatherChunks_Inheritance(t *testing.T) {
	// A child without a foreground inherits red from its parent.
	root := NewTextNode(TextNodeWithFg(style.ColorRed))
	child := NewTextNode()
	child.AddString("inherited")
	root.AddNode(child)

	chunks := root.GatherChunks(defaultInherited())
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Fg == nil || !chunks[0].Fg.Equals(style.ColorRed) {
		t.Errorf("expected inherited Red fg, got %v", chunks[0].Fg)
	}
}

func TestTextNode_GatherChunks_Override(t *testing.T) {
	// A child foreground overrides its parent's red.
	root := NewTextNode(TextNodeWithFg(style.ColorRed))
	child := NewTextNode(TextNodeWithFg(style.ColorBlue))
	child.AddString("overridden")
	root.AddNode(child)

	chunks := root.GatherChunks(defaultInherited())
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Fg == nil || !chunks[0].Fg.Equals(style.ColorBlue) {
		t.Errorf("expected Blue fg override, got %v", chunks[0].Fg)
	}
}

func TestTextNode_StyleInheritance_Fg(t *testing.T) {
	root := NewTextNode(TextNodeWithFg(style.ColorRed))
	root.AddString("red text")

	child := NewTextNode(TextNodeWithFg(style.ColorBlue))
	child.AddString("blue text")
	root.AddNode(child)

	chunks := root.GatherChunks(defaultInherited())
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if !chunks[0].Fg.Equals(style.ColorRed) {
		t.Error("chunk[0] expected red Fg")
	}
	if !chunks[1].Fg.Equals(style.ColorBlue) {
		t.Error("chunk[1] expected blue Fg")
	}
}

func TestTextNode_AttributesMerge(t *testing.T) {
	// Child attributes combine with inherited parent attributes.
	root := NewTextNode(TextNodeWithAttributes(style.AttrBold))
	child := NewTextNode(TextNodeWithAttributes(style.AttrItalic))
	child.AddString("merged")
	root.AddNode(child)

	chunks := root.GatherChunks(defaultInherited())
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Attributes&style.AttrBold == 0 {
		t.Error("expected bold attribute from parent")
	}
	if chunks[0].Attributes&style.AttrItalic == 0 {
		t.Error("expected italic attribute from child")
	}
}

func TestTextNode_LinkInheritance(t *testing.T) {
	root := NewTextNode(TextNodeWithLink("https://example.com"))
	child := NewTextNode()
	child.AddString("link text")
	root.AddNode(child)

	chunks := root.GatherChunks(defaultInherited())
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Link != "https://example.com" {
		t.Errorf("expected link inheritance, got %q", chunks[0].Link)
	}
}

// Nested chunk gathering.

func TestTextNode_GatherChunks_Nested(t *testing.T) {
	// Build plain, bold, nested bold-italic, and plain spans.
	root := NewTextNode()
	root.AddString("Hello ")

	bold := NewBoldNode()
	bold.AddString("bold ")
	italic := NewItalicNode(StringChild("bolditalic"))
	bold.AddNode(italic)

	root.AddNode(bold)
	root.AddString(" world")

	chunks := root.GatherChunks(defaultInherited())

	// Each flattened chunk must retain the effective attributes at its depth.
	if len(chunks) != 4 {
		t.Fatalf("expected 4 chunks, got %d", len(chunks))
	}

	if chunks[0].Text != "Hello " || chunks[0].Attributes != 0 {
		t.Errorf("chunk[0]: text=%q attrs=%d", chunks[0].Text, chunks[0].Attributes)
	}
	if chunks[1].Text != "bold " || chunks[1].Attributes != style.AttrBold {
		t.Errorf("chunk[1]: text=%q attrs=%d", chunks[1].Text, chunks[1].Attributes)
	}
	if chunks[2].Text != "bolditalic" || chunks[2].Attributes != style.AttrBold|style.AttrItalic {
		t.Errorf("chunk[2]: text=%q attrs=%d, want attrs=%d",
			chunks[2].Text, chunks[2].Attributes, style.AttrBold|style.AttrItalic)
	}
	if chunks[3].Text != " world" || chunks[3].Attributes != 0 {
		t.Errorf("chunk[3]: text=%q attrs=%d", chunks[3].Text, chunks[3].Attributes)
	}
}

func TestTextNode_MultipleChildren(t *testing.T) {
	root := NewTextNode()
	root.AddString("plain ")

	bold := NewBoldNode()
	bold.AddString("bold ")
	root.AddNode(bold)

	italic := NewItalicNode()
	italic.AddString("italic")
	root.AddNode(italic)

	chunks := root.GatherChunks(defaultInherited())
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}

	if chunks[0].Text != "plain " || chunks[0].Attributes != 0 {
		t.Errorf("chunk 0: text=%q attrs=%d", chunks[0].Text, chunks[0].Attributes)
	}
	if chunks[1].Text != "bold " || chunks[1].Attributes&style.AttrBold == 0 {
		t.Errorf("chunk 1: text=%q attrs=%d", chunks[1].Text, chunks[1].Attributes)
	}
	if chunks[2].Text != "italic" || chunks[2].Attributes&style.AttrItalic == 0 {
		t.Errorf("chunk 2: text=%q attrs=%d", chunks[2].Text, chunks[2].Attributes)
	}
}

func TestTextNode_DeepNesting(t *testing.T) {
	// Nested emphasis inherits red and combines bold with italic.
	root := NewTextNode(TextNodeWithFg(style.ColorRed))
	bold := NewBoldNode()
	italic := NewItalicNode()
	italic.AddString("styled")
	bold.AddNode(italic)
	root.AddNode(bold)

	chunks := root.GatherChunks(defaultInherited())
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Fg == nil || !chunks[0].Fg.Equals(style.ColorRed) {
		t.Errorf("expected Red fg from root, got %v", chunks[0].Fg)
	}
	if chunks[0].Attributes&style.AttrBold == 0 {
		t.Error("expected bold attribute from bold node")
	}
	if chunks[0].Attributes&style.AttrItalic == 0 {
		t.Error("expected italic attribute from italic node")
	}
}

func TestTextNode_ComplexTree(t *testing.T) {
	// Build mixed text through the convenience constructors.
	root := NewTextNode()
	root.AddString("Hello ")

	bold := NewBoldNode()
	bold.AddString("bold ")
	boldItalic := NewItalicNode()
	boldItalic.AddString("bold-italic")
	bold.AddNode(boldItalic)
	root.AddNode(bold)

	root.AddString(" world")

	chunks := root.GatherChunks(defaultInherited())
	if len(chunks) != 4 {
		t.Fatalf("expected 4 chunks, got %d", len(chunks))
	}

	if chunks[0].Text != "Hello " || chunks[0].Attributes != 0 {
		t.Errorf("chunk 0: text=%q attrs=%d", chunks[0].Text, chunks[0].Attributes)
	}
	if chunks[1].Text != "bold " || chunks[1].Attributes != style.AttrBold {
		t.Errorf("chunk 1: text=%q attrs=%d", chunks[1].Text, chunks[1].Attributes)
	}
	expected := style.AttrBold | style.AttrItalic
	if chunks[2].Text != "bold-italic" || chunks[2].Attributes != expected {
		t.Errorf("chunk 2: text=%q attrs=%d, want attrs=%d", chunks[2].Text, chunks[2].Attributes, expected)
	}
	if chunks[3].Text != " world" || chunks[3].Attributes != 0 {
		t.Errorf("chunk 3: text=%q attrs=%d", chunks[3].Text, chunks[3].Attributes)
	}
}

// Styled-node constructors.

func TestNewBoldNode(t *testing.T) {
	n := NewBoldNode(StringChild("bold"))
	if n.Attributes() != style.AttrBold {
		t.Errorf("expected AttrBold, got %d", n.Attributes())
	}
	if n.ChildrenCount() != 1 {
		t.Errorf("expected 1 child, got %d", n.ChildrenCount())
	}
}

func TestNewItalicNode(t *testing.T) {
	n := NewItalicNode(StringChild("italic"))
	if n.Attributes() != style.AttrItalic {
		t.Errorf("expected AttrItalic, got %d", n.Attributes())
	}
}

func TestNewUnderlineNode(t *testing.T) {
	n := NewUnderlineNode(StringChild("underline"))
	if n.Attributes() != style.AttrUnderline {
		t.Errorf("expected AttrUnderline, got %d", n.Attributes())
	}
}

func TestNewSpanNode(t *testing.T) {
	red := style.ColorRed
	n := NewSpanNode(TextNodeWithFg(red))
	n.AddString("colored")
	if n.Fg() == nil || !n.Fg().Equals(red) {
		t.Error("expected red Fg on span")
	}
}

func TestBoldText(t *testing.T) {
	node := BoldText("bold")
	chunks := node.GatherChunks(defaultInherited())
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Attributes&style.AttrBold == 0 {
		t.Error("expected bold attribute")
	}
}

func TestItalicText(t *testing.T) {
	node := ItalicText("italic")
	chunks := node.GatherChunks(defaultInherited())
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Attributes&style.AttrItalic == 0 {
		t.Error("expected italic attribute")
	}
}

func TestUnderlineText(t *testing.T) {
	node := UnderlineText("underline")
	chunks := node.GatherChunks(defaultInherited())
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Attributes&style.AttrUnderline == 0 {
		t.Error("expected underline attribute")
	}
}

func TestSpanText(t *testing.T) {
	node := SpanText("hello", TextNodeWithFg(style.ColorGreen))
	chunks := node.GatherChunks(defaultInherited())
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Text != "hello" {
		t.Errorf("expected 'hello', got %q", chunks[0].Text)
	}
	if chunks[0].Fg == nil || !chunks[0].Fg.Equals(style.ColorGreen) {
		t.Errorf("expected Green fg, got %v", chunks[0].Fg)
	}
}

func TestTextNode_LinkNode(t *testing.T) {
	link := NewLinkNode("https://example.com", StringChild("click here"))

	chunks := link.GatherChunks(defaultInherited())
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Link != "https://example.com" {
		t.Errorf("expected link URL, got %q", chunks[0].Link)
	}
	if chunks[0].Attributes&style.AttrUnderline == 0 {
		t.Error("expected underline attribute on link")
	}
}

func TestLinkText(t *testing.T) {
	node := LinkText("click me", "https://example.com")
	chunks := node.GatherChunks(defaultInherited())
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Link != "https://example.com" {
		t.Errorf("expected link URL, got %q", chunks[0].Link)
	}
}

func TestTextNode_LineBreak(t *testing.T) {
	root := NewTextNode()
	root.AddString("line 1")
	root.AddNode(NewLineBreakNode())
	root.AddString("line 2")

	chunks := root.GatherChunks(defaultInherited())
	// The line-break node contributes a newline between its siblings.
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}
	if chunks[1].Text != "\n" {
		t.Errorf("expected newline chunk, got %q", chunks[1].Text)
	}
}

func TestNewLineBreakNode_PlainText(t *testing.T) {
	node := NewLineBreakNode()
	if node.PlainText() != "\n" {
		t.Errorf("expected newline, got %q", node.PlainText())
	}
}

func TestTextNode_WithLineBreak_PlainText(t *testing.T) {
	root := NewTextNode()
	root.AddString("line1")
	root.AddNode(NewLineBreakNode())
	root.AddString("line2")

	if got := root.PlainText(); got != "line1\nline2" {
		t.Errorf("expected 'line1\\nline2', got %q", got)
	}
}

// Tree manipulation.

func TestTextNode_AddChildren(t *testing.T) {
	n := NewTextNode()
	n.AddString("hello ")
	child := NewBoldNode(StringChild("world"))
	n.AddNode(child)

	if n.ChildrenCount() != 2 {
		t.Errorf("expected 2 children, got %d", n.ChildrenCount())
	}
	if child.Parent() != n {
		t.Error("expected parent to be set")
	}
}

func TestTextNode_Remove(t *testing.T) {
	n := NewTextNode()
	child := NewTextNode(TextNodeWithID("remove-me"))
	child.AddString("hello")
	n.AddNode(child)

	if n.ChildrenCount() != 1 {
		t.Fatalf("expected 1 child, got %d", n.ChildrenCount())
	}

	removed := n.Remove("remove-me")
	if !removed {
		t.Error("expected Remove to return true")
	}
	if n.ChildrenCount() != 0 {
		t.Errorf("expected 0 children after remove, got %d", n.ChildrenCount())
	}
	if child.Parent() != nil {
		t.Error("expected parent to be nil after remove")
	}
}

func TestTextNode_RemoveChild(t *testing.T) {
	root := NewTextNode()
	child := NewTextNode(TextNodeWithID("child1"))
	child.AddString("A")
	root.AddNode(child)
	root.AddString("B")

	if root.PlainText() != "AB" {
		t.Fatalf("expected 'AB', got %q", root.PlainText())
	}

	root.Remove("child1")
	if root.PlainText() != "B" {
		t.Errorf("expected 'B' after remove, got %q", root.PlainText())
	}
}

func TestTextNode_Clear(t *testing.T) {
	n := NewTextNode()
	n.AddString("a")
	n.AddString("b")
	n.AddNode(NewBoldNode(StringChild("c")))

	n.Clear()
	if n.ChildrenCount() != 0 {
		t.Errorf("expected 0 children after clear, got %d", n.ChildrenCount())
	}
	if n.PlainText() != "" {
		t.Errorf("expected empty plain text, got %q", n.PlainText())
	}
}

func TestTextNode_InsertAt(t *testing.T) {
	n := NewTextNode()
	n.AddString("first")
	n.AddString("third")
	n.InsertAt(1, StringChild("second"))

	if n.ChildrenCount() != 3 {
		t.Fatalf("expected 3 children, got %d", n.ChildrenCount())
	}

	chunks := n.GatherChunks(defaultInherited())
	texts := make([]string, len(chunks))
	for i, c := range chunks {
		texts[i] = c.Text
	}
	if texts[0] != "first" || texts[1] != "second" || texts[2] != "third" {
		t.Errorf("unexpected chunk order: %v", texts)
	}
}

func TestTextNode_InsertNodeAt(t *testing.T) {
	root := NewTextNode()
	root.AddString("A")
	root.AddString("C")

	child := NewTextNode()
	child.AddString("B")
	root.InsertAt(1, NodeChild(child))

	if got := root.PlainText(); got != "ABC" {
		t.Errorf("expected 'ABC', got %q", got)
	}
}

func TestTextNode_InsertBefore(t *testing.T) {
	root := NewTextNode()
	a := NewTextNode(TextNodeWithID("a"))
	a.AddString("A")
	root.AddNode(a)

	c := NewTextNode(TextNodeWithID("c"))
	c.AddString("C")
	root.AddNode(c)

	b := NewTextNode(TextNodeWithID("b"))
	b.AddString("B")
	root.InsertBefore(b, "c")

	if got := root.PlainText(); got != "ABC" {
		t.Errorf("expected 'ABC', got %q", got)
	}
}

func TestTextNode_ReplaceChildren(t *testing.T) {
	n := NewTextNode()
	n.AddString("old")

	n.ReplaceChildren([]TextChild{
		StringChild("new1"),
		StringChild("new2"),
	})

	if n.ChildrenCount() != 2 {
		t.Fatalf("expected 2 children, got %d", n.ChildrenCount())
	}

	chunks := n.GatherChunks(defaultInherited())
	if chunks[0].Text != "new1" || chunks[1].Text != "new2" {
		t.Error("unexpected chunk content after replace")
	}
}

// Dirty tracking.

func TestTextNode_DirtyTracking(t *testing.T) {
	root := NewTextNode()
	root.ClearDirty()
	if root.IsDirty() {
		t.Error("should be clean after ClearDirty")
	}

	root.AddString("change")
	if !root.IsDirty() {
		t.Error("should be dirty after AddString")
	}
}

func TestTextNode_DirtyPropagation(t *testing.T) {
	root := NewTextNode()
	child := NewBoldNode(StringChild("hello"))
	root.AddNode(child)
	root.ClearDirty()

	if root.IsDirty() {
		t.Error("expected clean after ClearDirty")
	}

	// A descendant style mutation dirties its ancestors.
	child.SetAttributes(style.AttrItalic)
	if !root.IsDirty() {
		t.Error("expected root dirty after child modification")
	}
}

func TestTextNode_DirtyBubbles_ViaOwnerText(t *testing.T) {
	// Root ownership propagates descendant mutations to Text.
	var dirtyCalled bool
	mock := &mockOwnerText{onDirty: func() { dirtyCalled = true }}

	root := NewTextNode()
	root.ownerText = mock
	root.ClearDirty()

	child := NewTextNode()
	root.AddNode(child)
	dirtyCalled = false
	root.ClearDirty()

	child.AddString("change")
	if !dirtyCalled {
		t.Error("ownerText.textNodeDirtied should be called when child is modified")
	}
}

// mockOwnerText records TextNode dirty notifications.
type mockOwnerText struct {
	onDirty func()
}

func (m *mockOwnerText) textNodeDirtied() {
	if m.onDirty != nil {
		m.onDirty()
	}
}
