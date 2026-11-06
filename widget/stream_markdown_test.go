package widget

import (
	"fmt"
	"reflect"
	"slices"
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
	parser := defaultParser{}
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

// heldRow is the row that stands for the held block in the tests.
var heldRow = style.NewStyledText(style.Plain("…"))

// shownKinds returns the kind of each block sm's children show, and the text
// of a paragraph, in order, skipping children that show none.
func shownKinds(sm *StreamMarkdown) []string {
	var out []string
	for _, child := range sm.children {
		for _, b := range child.blocks {
			switch b.Kind {
			case mdBlockParagraph:
				out = append(out, "p:"+chunksText(b.Body))
			case mdBlockBlank:
				out = append(out, "blank")
			case mdBlockTable:
				out = append(out, "table")
			case mdBlockCode:
				out = append(out, "code")
			default:
				out = append(out, fmt.Sprint(b.Kind))
			}
		}
	}
	return out
}

// TestStreamMarkdown_StreamingHoldsLastBlock checks that a streaming
// StreamMarkdown holds back its last block, whatever it is, and a last line
// that holds no text yet, and what stands for the block.
func TestStreamMarkdown_StreamingHoldsLastBlock(t *testing.T) {
	const table = "| a | b |\n|---|---|\n| 1 | 2 |\n| 3 | 4 |"
	tests := []struct {
		name    string
		content string
		noRow   bool // no StreamMarkdownWithPending
		want    []string
	}{
		{name: "prose", content: "Intro\n\nMore", want: []string{"p:Intro", "blank", "p:…"}},
		{name: "one block", content: "Intro", want: []string{"p:…"}},
		{name: "table", content: "Intro\n\n" + table, want: []string{"p:Intro", "blank", "p:…"}},
		{name: "code", content: "Intro\n\n```go\nx := 1\n", want: []string{"p:Intro", "blank", "p:…"}},
		{name: "table before prose", content: table + "\n\nAfter", want: []string{"table", "blank", "p:…"}},
		// The pipe that starts the next row does not end the table.
		{name: "the pipe of a table's next row", content: "Intro\n\n" + table + "\n| ", want: []string{"p:Intro", "blank", "p:…"}},
		// The dash that starts a list does not start a block yet.
		{name: "the dash of a list item", content: "Intro\n\n- ", want: []string{"p:…"}},
		{name: "no row stands for it", content: "Intro\n\nMore", noRow: true, want: []string{"p:Intro"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts []StreamMarkdownOption
			if !tt.noRow {
				opts = append(opts, StreamMarkdownWithPending(heldRow))
			}
			sm := NewStreamMarkdown(opts...)
			sm.SetStreaming(true)
			sm.SetContent(tt.content)
			if got := shownKinds(sm); !slices.Equal(got, tt.want) {
				t.Errorf("streaming shows %q, want %q", got, tt.want)
			}

			// Once streaming ends, every block shows.
			sm.SetStreaming(false)
			whole := NewStreamMarkdown(opts...)
			whole.SetContent(tt.content)
			if got, want := shownKinds(sm), shownKinds(whole); !slices.Equal(got, want) {
				t.Errorf("after streaming shows %q, want %q", got, want)
			}
		})
	}
}

// streamLines returns the lines sm shows at width w, its children's in
// order.
func streamLines(sm *StreamMarkdown, w int) []mdLine {
	var lines []mdLine
	for _, child := range sm.children {
		child.wrapToWidth(w)
		lines = append(lines, child.renderedLines...)
	}
	return lines
}

// TestStreamMarkdown_ShrinkClearsChildren checks that content parsed into
// fewer blocks than before leaves the children past them showing nothing.
func TestStreamMarkdown_ShrinkClearsChildren(t *testing.T) {
	sm := NewStreamMarkdown()
	sm.SetContent("a\n\nb\n\nc")
	sm.SetContent("a\n\nb")
	if got, want := shownKinds(sm), []string{"p:a", "blank", "p:b"}; !slices.Equal(got, want) {
		t.Errorf("shows %q, want %q", got, want)
	}
}

// TestStreamMarkdown_StreamingKeepsShownLines streams a table and a fitted
// code block a word at a time, as a model streams them, each with a line
// wider than those before it, and checks that while streaming each update
// changes no line but the last one shown, and adds lines after it. Without
// streaming, the wider lines change the lines of the table and the code
// above them.
func TestStreamMarkdown_StreamingKeepsShownLines(t *testing.T) {
	src := "Counts:\n\n| file | n |\n|---|---|\n| a | 1 |\n| a_much_longer_name | 12345 |\n| b | 2 |\n\n" +
		"Code:\n\n```go\nx := 1\nresult := computeSomething(withArguments)\n```\n\nDone."
	for _, streaming := range []bool{true, false} {
		t.Run(fmt.Sprintf("streaming=%v", streaming), func(t *testing.T) {
			sm := NewStreamMarkdown(StreamMarkdownWithCodeBlockFit(true), StreamMarkdownWithPending(heldRow))
			sm.SetStreaming(streaming)
			var content string
			var prev []mdLine
			rewrote := false
			for _, word := range strings.SplitAfter(src, " ") {
				content += word
				sm.SetContent(content)
				lines := streamLines(sm, 60)
				kept := max(len(prev)-1, 0)
				if len(lines) < kept || !slices.EqualFunc(lines[:kept], prev[:kept], func(a, b mdLine) bool { return reflect.DeepEqual(a, b) }) {
					rewrote = true
				}
				prev = lines
			}
			if rewrote == streaming {
				t.Errorf("an update rewrote a line above the last = %v, want %v", rewrote, !streaming)
			}
		})
	}
}

// TestStreamMarkdown_DefinitionsAndFootnotesAreText checks that definition
// lists and footnotes show as the paragraphs they are written in, and that
// streaming them a word at a time changes no line shown but the last: a
// later term no longer joins the list above it, and a footnote's definition
// no longer changes the references before it.
func TestStreamMarkdown_DefinitionsAndFootnotesAreText(t *testing.T) {
	const src = "Divide\n: Returns the quotient.\n\nZero divisor\n: Reported as an error.[^1]\n\n[^1]: Go reports an error value."
	sm := NewStreamMarkdown(StreamMarkdownWithPending(heldRow))
	sm.SetContent(src)
	if got, want := shownKinds(sm), []string{
		"p:Divide : Returns the quotient.", "blank",
		"p:Zero divisor : Reported as an error.[^1]", "blank",
		"p:[^1]: Go reports an error value.",
	}; !slices.Equal(got, want) {
		t.Errorf("shows %q, want %q", got, want)
	}

	sm = NewStreamMarkdown(StreamMarkdownWithPending(heldRow))
	sm.SetStreaming(true)
	var content string
	var prev []mdLine
	for _, word := range strings.SplitAfter(src, " ") {
		content += word
		sm.SetContent(content)
		lines := streamLines(sm, 60)
		kept := max(len(prev)-1, 0)
		if len(lines) < kept || !slices.EqualFunc(lines[:kept], prev[:kept], func(a, b mdLine) bool { return reflect.DeepEqual(a, b) }) {
			t.Fatalf("after %q, a line above the last changed", content)
		}
		prev = lines
	}
}
