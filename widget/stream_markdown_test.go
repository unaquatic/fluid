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
	// Each parsed block receives one child.
	if len(sm.children) != len(sm.allBlocks) {
		t.Errorf("expected %d children (one per block), got %d",
			len(sm.allBlocks), len(sm.children))
	}
}

func TestStreamMarkdown_OneBlockPerChild(t *testing.T) {
	var sb strings.Builder
	for i := range 25 {
		sb.WriteString(fmt.Sprintf("Paragraph %d\n\n", i))
	}
	sb.WriteString("tail text")

	sm := NewStreamMarkdown(StreamMarkdownWithFg(style.ColorWhite))
	sm.SetContent(sb.String())

	// Child count follows parsed block count.
	if len(sm.children) != len(sm.allBlocks) {
		t.Fatalf("expected %d children (one per block), got %d",
			len(sm.allBlocks), len(sm.children))
	}

	// Each child owns exactly one block.
	for i, child := range sm.children {
		if len(child.blocks) != 1 {
			t.Errorf("child %d: expected 1 block, got %d", i, len(child.blocks))
		}
	}
}

func TestStreamMarkdown_IncrementalAppend(t *testing.T) {
	sm := NewStreamMarkdown(StreamMarkdownWithFg(style.ColorWhite))

	// Supply progressively longer complete content values.
	var content strings.Builder
	for i := range 25 {
		content.WriteString(fmt.Sprintf("Line %d\n\n", i))
		sm.SetContent(content.String())
	}

	// The first update creates one child per block.
	if len(sm.children) != len(sm.allBlocks) {
		t.Fatalf("expected %d children, got %d", len(sm.allBlocks), len(sm.children))
	}

	// Later updates retain existing children and add any required children.
	prevChildCount := len(sm.children)
	content.WriteString("one more line")
	sm.SetContent(content.String())

	if len(sm.children) < prevChildCount {
		t.Errorf("children count decreased: %d < %d", len(sm.children), prevChildCount)
	}
}

func TestStreamMarkdown_CodeBlockNotSplit(t *testing.T) {
	// A fenced code block remains one mdBlockCode regardless of its line count.
	var sb strings.Builder
	sb.WriteString("```go\n")
	for i := range 30 {
		sb.WriteString(fmt.Sprintf("line %d\n", i))
	}
	sb.WriteString("```\n\ntail")

	sm := NewStreamMarkdown(StreamMarkdownWithFg(style.ColorWhite))
	sm.SetContent(sb.String())

	// Exactly one child should contain the complete code block.
	for _, child := range sm.children {
		codeCount := 0
		for _, blk := range child.blocks {
			if blk.Kind == mdBlockCode {
				codeCount++
			}
		}
		// A code-block child contains all source lines from that block.
		if codeCount > 1 {
			t.Errorf("child has %d code blocks — expected at most 1", codeCount)
		}
	}
}

func TestStreamMarkdown_EarlyChildStableBlockCount(t *testing.T) {
	sm := NewStreamMarkdown(StreamMarkdownWithFg(style.ColorWhite))

	// Use multiple blocks so unchanged prefix children can be observed.
	var sb strings.Builder
	for i := range 25 {
		sb.WriteString(fmt.Sprintf("Para %d\n\n", i))
	}
	base := sb.String()
	sm.SetContent(base + "tail")

	if len(sm.children) < 2 {
		t.Fatalf("expected multiple children, got %d", len(sm.children))
	}

	// Save the first child's block before extending the tail.
	if len(sm.children[0].blocks) != 1 {
		t.Fatalf("expected 1 block in first child, got %d", len(sm.children[0].blocks))
	}
	firstBlock := sm.children[0].blocks[0]

	// Extending only the tail must not change earlier child blocks.
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

	// Reapplying identical content must preserve child state.
	childCountBefore := len(sm.children)
	sm.SetContent("hello")
	if len(sm.children) != childCountBefore {
		t.Errorf("child count changed on duplicate SetContent: %d → %d",
			childCountBefore, len(sm.children))
	}
}

func TestStreamMarkdown_BlocksDistributed(t *testing.T) {
	sm := NewStreamMarkdown(StreamMarkdownWithFg(style.ColorWhite))

	// Multiple paragraphs produce multiple block children.
	var sb strings.Builder
	for i := range 30 {
		sb.WriteString(fmt.Sprintf("Block %d\n\n", i))
	}
	sm.SetContent(sb.String())

	// Child block counts must account for every parsed block.
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

	// Reuse blocks produced by the configured parser.
	parser := gfmParser{}
	styles := mdStyles{Fg: style.ColorWhite}
	blocks := parser.Parse("# Hello\n\nWorld\n\n- item", styles)

	if len(blocks) == 0 {
		t.Fatal("parser returned no blocks")
	}

	// setBlocks replaces parsed state and schedules wrapping.
	md.setBlocks(blocks)
	if len(md.blocks) != len(blocks) {
		t.Errorf("expected %d blocks, got %d", len(blocks), len(md.blocks))
	}
}
