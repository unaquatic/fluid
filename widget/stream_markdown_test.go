package widget

import (
	"fmt"
	"strings"
	"testing"

	"github.com/unaquatic/fluid/style"
)

func TestStreamMarkdown_Empty(t *testing.T) {
	sm := NewStreamMarkdown(StreamMarkdownWithFg(style.ColorWhite))
	sm.SetContent("")
	if len(sm.children) != 1 {
		t.Errorf("expected 1 child for empty content, got %d", len(sm.children))
	}
}

func TestStreamMarkdown_ShortContent(t *testing.T) {
	sm := NewStreamMarkdown(StreamMarkdownWithFg(style.ColorWhite))
	sm.SetContent("# Hello\n\nWorld")
	// One child per parsed block.
	if len(sm.children) != len(sm.allBlocks) {
		t.Errorf("expected %d children (one per block), got %d",
			len(sm.allBlocks), len(sm.children))
	}
}

func TestStreamMarkdown_OneBlockPerChild(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 25; i++ {
		sb.WriteString(fmt.Sprintf("Paragraph %d\n\n", i))
	}
	sb.WriteString("tail text")

	sm := NewStreamMarkdown(StreamMarkdownWithFg(style.ColorWhite))
	sm.SetContent(sb.String())

	// One child per block.
	if len(sm.children) != len(sm.allBlocks) {
		t.Fatalf("expected %d children (one per block), got %d",
			len(sm.allBlocks), len(sm.children))
	}

	// Every child holds exactly one block.
	for i, child := range sm.children {
		if len(child.blocks) != 1 {
			t.Errorf("child %d: expected 1 block, got %d", i, len(child.blocks))
		}
	}
}

func TestStreamMarkdown_IncrementalAppend(t *testing.T) {
	sm := NewStreamMarkdown(StreamMarkdownWithFg(style.ColorWhite))

	// Simulate streaming: append content incrementally.
	var content strings.Builder
	for i := 0; i < 25; i++ {
		content.WriteString(fmt.Sprintf("Line %d\n\n", i))
		sm.SetContent(content.String())
	}

	// Child count should match block count.
	if len(sm.children) != len(sm.allBlocks) {
		t.Fatalf("expected %d children, got %d", len(sm.allBlocks), len(sm.children))
	}

	// Append more content — child count should not decrease (grow-only).
	prevChildCount := len(sm.children)
	content.WriteString("one more line")
	sm.SetContent(content.String())

	if len(sm.children) < prevChildCount {
		t.Errorf("children count decreased: %d < %d", len(sm.children), prevChildCount)
	}
}

func TestStreamMarkdown_CodeBlockNotSplit(t *testing.T) {
	// A large code block is a single MdBlockCode. Since we split at
	// block boundaries, the code block always stays in one child.
	var sb strings.Builder
	sb.WriteString("```go\n")
	for i := 0; i < 30; i++ {
		sb.WriteString(fmt.Sprintf("line %d\n", i))
	}
	sb.WriteString("```\n\ntail")

	sm := NewStreamMarkdown(StreamMarkdownWithFg(style.ColorWhite))
	sm.SetContent(sb.String())

	// The code block should be entirely within one child.
	for _, child := range sm.children {
		codeCount := 0
		for _, blk := range child.blocks {
			if blk.Kind == MdBlockCode {
				codeCount++
			}
		}
		// If this child has a code block, it should be complete
		// (the parser always produces complete code blocks).
		if codeCount > 1 {
			t.Errorf("child has %d code blocks — expected at most 1", codeCount)
		}
	}
}

func TestStreamMarkdown_EarlyChildStableBlockCount(t *testing.T) {
	sm := NewStreamMarkdown(StreamMarkdownWithFg(style.ColorWhite))

	// Build content large enough to produce multiple blocks.
	var sb strings.Builder
	for i := 0; i < 25; i++ {
		sb.WriteString(fmt.Sprintf("Para %d\n\n", i))
	}
	base := sb.String()
	sm.SetContent(base + "tail")

	if len(sm.children) < 2 {
		t.Fatalf("expected multiple children, got %d", len(sm.children))
	}

	// Record first child's single block.
	if len(sm.children[0].blocks) != 1 {
		t.Fatalf("expected 1 block in first child, got %d", len(sm.children[0].blocks))
	}
	firstBlock := sm.children[0].blocks[0]

	// Append more content — earlier children's blocks should be unchanged
	// since new content only extends the tail.
	sm.SetContent(base + "tail extended with more text")
	if len(sm.children[0].blocks) != 1 {
		t.Errorf("first child block count changed: got %d", len(sm.children[0].blocks))
	}
	if sm.children[0].blocks[0].Kind != firstBlock.Kind {
		t.Errorf("first child block kind changed")
	}
}

func TestStreamMarkdown_ContentDedup(t *testing.T) {
	sm := NewStreamMarkdown(StreamMarkdownWithFg(style.ColorWhite))
	sm.SetContent("hello")

	// Setting the same content should be a no-op.
	childCountBefore := len(sm.children)
	sm.SetContent("hello")
	if len(sm.children) != childCountBefore {
		t.Errorf("child count changed on duplicate SetContent: %d → %d",
			childCountBefore, len(sm.children))
	}
}

func TestStreamMarkdown_BlocksDistributed(t *testing.T) {
	sm := NewStreamMarkdown(StreamMarkdownWithFg(style.ColorWhite))

	// Build enough paragraphs to trigger a split.
	var sb strings.Builder
	for i := 0; i < 30; i++ {
		sb.WriteString(fmt.Sprintf("Block %d\n\n", i))
	}
	sm.SetContent(sb.String())

	// Total blocks across all children should equal allBlocks.
	totalChildBlocks := 0
	for _, child := range sm.children {
		totalChildBlocks += len(child.blocks)
	}
	if totalChildBlocks != len(sm.allBlocks) {
		t.Errorf("block distribution mismatch: children have %d blocks, allBlocks has %d",
			totalChildBlocks, len(sm.allBlocks))
	}
}

func TestMarkdown_SetBlocks(t *testing.T) {
	md := NewMarkdown("", MarkdownWithFg(style.ColorWhite))

	// Parse some content to get blocks.
	parser := NewGoldmarkParser()
	styles := MdStyles{Fg: style.ColorWhite}
	blocks := parser.Parse("# Hello\n\nWorld\n\n- item", styles)

	if len(blocks) == 0 {
		t.Fatal("parser returned no blocks")
	}

	// SetBlocks should replace blocks and mark wrap dirty.
	md.SetBlocks(blocks)
	if len(md.blocks) != len(blocks) {
		t.Errorf("expected %d blocks, got %d", len(blocks), len(md.blocks))
	}
}
