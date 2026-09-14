package terminal

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
)

func newInlineTestRenderer() (*Renderer, *bytes.Buffer) {
	var outBuf bytes.Buffer
	w, h := 40, 20
	root := NewRootRenderable(w, h)
	hitGrid := NewHitGrid(w, h)
	cfg := DefaultConfig()
	cfg.InlineMode = true
	cfg.Fullscreen = false
	r := &Renderer{
		root: root, hitGrid: hitGrid,
		backend:       NewBackend(strings.NewReader(""), &outBuf),
		config:        cfg,
		currentBuffer: buffer.New(w, h),
		nextBuffer:    buffer.New(w, h),
		width:         w,
		height:        h,
		controlState:  StateIdle,
		parser:        input.NewParser(),
		eventBus:      input.NewEventBus(),
		pendingWork:   make(chan func()),
		wake:          make(chan struct{}, 1),
		stopChan:      make(chan struct{}, 1),
		done:          make(chan struct{}),
		sel:           render.NewSelection(),
		linkRegistry:  NewLinkRegistry(),
		stats:         NewStats(60),
		lastFrameTime: time.Now(),
	}
	root.SetContext(r)
	return r, &outBuf
}

func TestInlineMode_ConfigForcesNoAlternateScreen(t *testing.T) {
	cfg := mergeConfig([]Config{{InlineMode: true, Fullscreen: true}})
	if cfg.Fullscreen {
		t.Error("expected Fullscreen=false")
	}
}

func TestComputeContentHeight(t *testing.T) {
	r, _ := newInlineTestRenderer()
	if h := r.computeContentHeight(); h != 0 {
		t.Errorf("empty root: expected 0, got %d", h)
	}
	r.root.Add(render.NewBaseRenderable(render.WithID("c1"), render.WithWidth(layout.Points(40)), render.WithHeight(layout.Points(3))))
	r.root.Add(render.NewBaseRenderable(render.WithID("c2"), render.WithWidth(layout.Points(40)), render.WithHeight(layout.Points(5))))
	r.root.LayoutNode().Width = layout.Points(40)
	r.root.LayoutNode().Height = layout.Points(inlineMaxLayoutHeight)
	r.root.CalculateLayout()
	if h := r.computeContentHeight(); h != 8 {
		t.Errorf("with children: expected 8, got %d", h)
	}
}

func TestInlineFlush_FirstFrame_NoErase(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	buf := buffer.New(5, 2)
	buf.SetCell(0, 0, 'A', style.ColorWhite, style.ColorBlack, 0)
	buf.SetCell(0, 1, 'C', style.ColorWhite, style.ColorBlack, 0)
	r.InlineFlush(buf)
	if strings.Contains(outBuf.String(), "\x1b[2K") {
		t.Error("first frame should not erase")
	}
	// Two content rows plus the trailing cursor row produce three rows to erase.
	if r.lastInlineHeight != 3 {
		t.Errorf("expected lastInlineHeight=3 (2+1), got %d", r.lastInlineHeight)
	}
}

func TestInlineFlush_SecondFrame_Diff(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	// The first frame contains three distinct rows.
	buf1 := buffer.New(5, 3)
	for y := range 3 {
		buf1.SetCell(0, y, rune('A'+y), style.ColorWhite, style.ColorBlack, 0)
	}
	r.InlineFlush(buf1)
	outBuf.Reset()

	// The second frame changes both common rows and removes the third.
	buf2 := buffer.New(5, 2)
	buf2.SetCell(0, 0, 'X', style.ColorWhite, style.ColorBlack, 0)
	buf2.SetCell(0, 1, 'Y', style.ColorWhite, style.ColorBlack, 0)
	r.InlineFlush(buf2)
	output := outBuf.String()

	// Two changed rows and one removed row require three line erases.
	if c := strings.Count(output, "\x1b[2K"); c != 3 {
		t.Errorf("expected 3 erases (2 dirty + 1 removed), got %d", c)
	}
	// Rewritten rows contain the new glyphs.
	if !strings.Contains(output, "X") || !strings.Contains(output, "Y") {
		t.Error("expected new content X and Y in output")
	}
	// Stored height includes the new trailing cursor row.
	if r.lastInlineHeight != 3 {
		t.Errorf("expected lastInlineHeight=3 (2+1), got %d", r.lastInlineHeight)
	}
}

func TestInlineFlush_Overflow_ClearTerminal(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	// Twenty-five rows overflow the 20-row terminal.
	buf := buffer.New(10, 25)
	for y := range 25 {
		buf.SetCell(0, y, rune('a'+y%26), style.ColorWhite, style.ColorBlack, 0)
	}
	r.InlineFlush(buf)
	output := outBuf.String()

	// Overflow clears the screen, scrollback, and cursor position.
	if !strings.Contains(output, "\x1b[2J\x1b[3J\x1b[H") {
		t.Error("overflow should use clearTerminal")
	}
	// Overflow does not walk and erase prior lines individually.
	if strings.Contains(output, "\x1b[2K") {
		t.Error("overflow should not use per-line erase")
	}
	// Twenty-five content rows plus the trailing row produce 26.
	if r.lastInlineHeight != 26 {
		t.Errorf("expected lastInlineHeight=26 (25+1), got %d", r.lastInlineHeight)
	}
}

func TestInlineFlush_Overflow_Subsequent_StillClears(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	// The first frame overflows.
	buf1 := buffer.New(10, 25)
	for y := range 25 {
		buf1.SetCell(0, y, 'A', style.ColorWhite, style.ColorBlack, 0)
	}
	r.InlineFlush(buf1)
	outBuf.Reset()

	// The previous 26-row output keeps the second frame in overflow handling.
	buf2 := buffer.New(10, 22)
	for y := range 22 {
		buf2.SetCell(0, y, 'B', style.ColorWhite, style.ColorBlack, 0)
	}
	r.InlineFlush(buf2)
	output := outBuf.String()

	// Prior overflow again requires a complete clear.
	if !strings.Contains(output, "\x1b[2J\x1b[3J\x1b[H") {
		t.Error("subsequent overflow should still use clearTerminal")
	}
}

func TestInlineFlush_Overflow_Discards_CurrentBuf(t *testing.T) {
	r, _ := newInlineTestRenderer()
	// The first frame fits the terminal.
	buf1 := buffer.New(10, 5)
	buf1.SetCell(0, 0, 'A', style.ColorWhite, style.ColorBlack, 0)
	r.InlineFlush(buf1)
	if r.inlineCurrentBuf == nil {
		t.Fatal("expected inlineCurrentBuf after non-overflow flush")
	}

	// The second frame overflows the terminal.
	buf2 := buffer.New(10, 25)
	for y := range 25 {
		buf2.SetCell(0, y, 'B', style.ColorWhite, style.ColorBlack, 0)
	}
	r.InlineFlush(buf2)
	if r.inlineCurrentBuf != nil {
		t.Error("expected inlineCurrentBuf=nil after overflow")
	}
}

func TestInlineDiff_NothingChanged_ZeroOutput(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	buf := buffer.New(10, 5)
	buf.SetCell(0, 0, 'H', style.ColorWhite, style.ColorBlack, 0)
	buf.SetCell(0, 1, 'e', style.ColorWhite, style.ColorBlack, 0)
	buf.SetCell(0, 2, 'l', style.ColorWhite, style.ColorBlack, 0)
	buf.SetCell(0, 3, 'l', style.ColorWhite, style.ColorBlack, 0)
	buf.SetCell(0, 4, 'o', style.ColorWhite, style.ColorBlack, 0)

	// The first frame uses a complete flush.
	r.InlineFlush(buf)
	outBuf.Reset()

	// The second frame is identical.
	buf2 := buffer.New(10, 5)
	buf2.SetCell(0, 0, 'H', style.ColorWhite, style.ColorBlack, 0)
	buf2.SetCell(0, 1, 'e', style.ColorWhite, style.ColorBlack, 0)
	buf2.SetCell(0, 2, 'l', style.ColorWhite, style.ColorBlack, 0)
	buf2.SetCell(0, 3, 'l', style.ColorWhite, style.ColorBlack, 0)
	buf2.SetCell(0, 4, 'o', style.ColorWhite, style.ColorBlack, 0)
	r.InlineFlush(buf2)

	// Identical output writes no bytes.
	if outBuf.Len() != 0 {
		t.Errorf("expected 0 bytes for identical frame, got %d", outBuf.Len())
	}
}

func TestInlineDiff_OnlyChangedRowsRewritten(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	// Multi-character row tokens avoid confusing content with letters inside
	// cursor escape sequences.
	buf1 := buffer.New(10, 5)
	for y := range 5 {
		for x := range 5 {
			buf1.SetCell(x, y, rune('a'+y), style.ColorWhite, style.ColorBlack, 0)
		}
	}
	r.InlineFlush(buf1)
	outBuf.Reset()

	// The second frame changes only row two.
	buf2 := buffer.New(10, 5)
	for y := range 5 {
		for x := range 5 {
			buf2.SetCell(x, y, rune('a'+y), style.ColorWhite, style.ColorBlack, 0)
		}
	}
	// Replace row two with X glyphs.
	for x := range 5 {
		buf2.SetCell(x, 2, 'X', style.ColorWhite, style.ColorBlack, 0)
	}
	r.InlineFlush(buf2)
	output := outBuf.String()

	// Only the changed row is erased.
	if c := strings.Count(output, "\x1b[2K"); c != 1 {
		t.Errorf("expected 1 erase for 1 dirty row, got %d", c)
	}
	// Output contains the replacement content.
	if !strings.Contains(output, "XXXXX") {
		t.Error("expected changed row content XXXXX in output")
	}
	// Output omits content from unchanged rows.
	if strings.Contains(output, "aaaaa") || strings.Contains(output, "bbbbb") ||
		strings.Contains(output, "ddddd") || strings.Contains(output, "eeeee") {
		t.Error("unchanged rows should not appear in output")
	}
}

func TestInlineDiff_HeightGrew_AppendsNewRows(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	// The first frame contains three rows.
	buf1 := buffer.New(10, 3)
	for y := range 3 {
		buf1.SetCell(0, y, rune('A'+y), style.ColorWhite, style.ColorBlack, 0)
	}
	r.InlineFlush(buf1)
	outBuf.Reset()

	// The second frame preserves three rows and appends two.
	buf2 := buffer.New(10, 5)
	for y := range 3 {
		buf2.SetCell(0, y, rune('A'+y), style.ColorWhite, style.ColorBlack, 0)
	}
	buf2.SetCell(0, 3, 'D', style.ColorWhite, style.ColorBlack, 0)
	buf2.SetCell(0, 4, 'E', style.ColorWhite, style.ColorBlack, 0)
	r.InlineFlush(buf2)
	output := outBuf.String()

	// Only the appended rows require line erases and writes.
	eraseCount := strings.Count(output, "\x1b[2K")
	if eraseCount != 2 {
		t.Errorf("expected 2 erases (2 new rows), got %d", eraseCount)
	}
	// Output contains both appended rows.
	if !strings.Contains(output, "D") || !strings.Contains(output, "E") {
		t.Error("expected new content D and E")
	}
	// Output omits unchanged existing rows.
	if strings.Contains(output, "A") || strings.Contains(output, "B") || strings.Contains(output, "C") {
		t.Error("unchanged rows A, B, C should not be in output")
	}
	// Stored height includes five content rows and one trailing row.
	if r.lastInlineHeight != 6 {
		t.Errorf("expected lastInlineHeight=6 (5+1), got %d", r.lastInlineHeight)
	}
}

func TestInlineDiff_HeightShrunk_ErasesRemovedRows(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	// The first frame contains five identifiable rows.
	buf1 := buffer.New(10, 5)
	for y := range 5 {
		for x := range 5 {
			buf1.SetCell(x, y, rune('a'+y), style.ColorWhite, style.ColorBlack, 0)
		}
	}
	r.InlineFlush(buf1)
	outBuf.Reset()

	// The second frame retains three common rows and removes two.
	buf2 := buffer.New(10, 3)
	for y := range 3 {
		for x := range 5 {
			buf2.SetCell(x, y, rune('a'+y), style.ColorWhite, style.ColorBlack, 0)
		}
	}
	r.InlineFlush(buf2)
	output := outBuf.String()

	// Both removed rows are erased.
	eraseCount := strings.Count(output, "\x1b[2K")
	if eraseCount != 2 {
		t.Errorf("expected 2 erases (2 removed rows), got %d", eraseCount)
	}
	// Unchanged common rows are not rewritten.
	if strings.Contains(output, "aaaaa") || strings.Contains(output, "bbbbb") || strings.Contains(output, "ccccc") {
		t.Error("unchanged common rows should not be rewritten")
	}
	if r.lastInlineHeight != 4 {
		t.Errorf("expected lastInlineHeight=4 (3+1), got %d", r.lastInlineHeight)
	}
}

func TestInlineDiff_HeightGrewAndRowsChanged(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	// The first frame contains three identifiable rows.
	buf1 := buffer.New(10, 3)
	for y := range 3 {
		for x := range 5 {
			buf1.SetCell(x, y, rune('a'+y), style.ColorWhite, style.ColorBlack, 0)
		}
	}
	r.InlineFlush(buf1)
	outBuf.Reset()

	// The second frame changes one row and adds two rows.
	buf2 := buffer.New(10, 5)
	for x := range 5 {
		buf2.SetCell(x, 0, 'a', style.ColorWhite, style.ColorBlack, 0) // Unchanged.
		buf2.SetCell(x, 1, 'X', style.ColorWhite, style.ColorBlack, 0) // Changed.
		buf2.SetCell(x, 2, 'c', style.ColorWhite, style.ColorBlack, 0) // Unchanged.
		buf2.SetCell(x, 3, 'd', style.ColorWhite, style.ColorBlack, 0) // Added.
		buf2.SetCell(x, 4, 'e', style.ColorWhite, style.ColorBlack, 0) // Added.
	}
	r.InlineFlush(buf2)
	output := outBuf.String()

	// One changed row and two added rows require three line erases.
	eraseCount := strings.Count(output, "\x1b[2K")
	if eraseCount != 3 {
		t.Errorf("expected 3 erases (1 dirty + 2 new), got %d", eraseCount)
	}
	if !strings.Contains(output, "XXXXX") || !strings.Contains(output, "ddddd") || !strings.Contains(output, "eeeee") {
		t.Error("expected changed/new content XXXXX, ddddd, eeeee")
	}
	// Unchanged rows are not rewritten.
	if strings.Contains(output, "aaaaa") || strings.Contains(output, "ccccc") {
		t.Error("unchanged rows should not be rewritten")
	}
}

func TestInlineDiff_WidthChange_FallsBackToFull(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	// The first frame has width ten.
	buf1 := buffer.New(10, 3)
	buf1.SetCell(0, 0, 'A', style.ColorWhite, style.ColorBlack, 0)
	r.InlineFlush(buf1)
	outBuf.Reset()

	// The second frame changes width to 15.
	buf2 := buffer.New(15, 3)
	buf2.SetCell(0, 0, 'A', style.ColorWhite, style.ColorBlack, 0)
	r.InlineFlush(buf2)
	output := outBuf.String()

	// A width change forces a complete flush and erases the three prior content
	// rows plus their trailing row.
	if c := strings.Count(output, "\x1b[2K"); c != 4 {
		t.Errorf("expected 4 erases from full flush, got %d", c)
	}
}

func TestInlineDiff_AfterOverflow_FullFlush(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	// The first frame overflows and discards its diff buffer.
	buf1 := buffer.New(10, 25) // Exceeds the 20-row terminal.
	for y := range 25 {
		buf1.SetCell(0, y, 'A', style.ColorWhite, style.ColorBlack, 0)
	}
	r.InlineFlush(buf1)
	if r.inlineCurrentBuf != nil {
		t.Fatal("expected nil inlineCurrentBuf after overflow")
	}

	// The second frame fits, but prior output still exceeds the terminal and must
	// use overflow clearing once more.
	buf2 := buffer.New(10, 5)
	buf2.SetCell(0, 0, 'B', style.ColorWhite, style.ColorBlack, 0)
	r.InlineFlush(buf2)
	// Overflow handling stores six rows and still leaves no diff buffer.

	// The third frame no longer overflows but needs a complete flush to establish
	// a new diff buffer.
	outBuf.Reset()
	buf3 := buffer.New(10, 5)
	buf3.SetCell(0, 0, 'B', style.ColorWhite, style.ColorBlack, 0)
	r.InlineFlush(buf3)
	if r.inlineCurrentBuf == nil {
		t.Error("expected inlineCurrentBuf after non-overflow full flush")
	}

	// The identical fourth frame uses diffing and writes nothing.
	outBuf.Reset()
	buf4 := buffer.New(10, 5)
	buf4.SetCell(0, 0, 'B', style.ColorWhite, style.ColorBlack, 0)
	r.InlineFlush(buf4)
	if outBuf.Len() != 0 {
		t.Errorf("expected 0 bytes for identical frame after recovery, got %d", outBuf.Len())
	}
}

func TestInlineDiff_CursorMovement_Up(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	// The first frame leaves the cursor after row five.
	buf1 := buffer.New(10, 5)
	for y := range 5 {
		buf1.SetCell(0, y, rune('A'+y), style.ColorWhite, style.ColorBlack, 0)
	}
	r.InlineFlush(buf1)
	outBuf.Reset()

	// The second frame changes only row zero.
	buf2 := buffer.New(10, 5)
	buf2.SetCell(0, 0, 'Z', style.ColorWhite, style.ColorBlack, 0) // Changed.
	for y := 1; y < 5; y++ {
		buf2.SetCell(0, y, rune('A'+y), style.ColorWhite, style.ColorBlack, 0)
	}
	r.InlineFlush(buf2)
	output := outBuf.String()

	// Rewriting row zero first moves up five rows from the trailing cursor.
	if !strings.Contains(output, "\x1b[5A") {
		t.Error("expected cursor up 5 to reach row 0")
	}
	// Completion restores the cursor five rows down after the content.
	if !strings.Contains(output, "\x1b[5B") {
		t.Error("expected cursor down 5 to return to trailing position")
	}
}

func TestInlineDiff_MultipleRows_MinimalOutput(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	// The first frame contains ten wide rows so skipped rows dominate diff overhead.
	buf1 := buffer.New(40, 10)
	for y := range 10 {
		for x := range 30 {
			buf1.SetCell(x, y, rune('a'+y), style.ColorWhite, style.ColorBlack, 0)
		}
	}
	r.InlineFlush(buf1)
	firstBytes := outBuf.Len()
	outBuf.Reset()

	// The second frame changes only rows three and seven.
	buf2 := buffer.New(40, 10)
	for y := range 10 {
		for x := range 30 {
			buf2.SetCell(x, y, rune('a'+y), style.ColorWhite, style.ColorBlack, 0)
		}
	}
	for x := range 30 {
		buf2.SetCell(x, 3, 'X', style.ColorWhite, style.ColorBlack, 0)
		buf2.SetCell(x, 7, 'Y', style.ColorWhite, style.ColorBlack, 0)
	}
	r.InlineFlush(buf2)
	diffBytes := outBuf.Len()

	// Two rewritten rows produce substantially less output than a complete flush.
	if diffBytes >= firstBytes {
		t.Errorf("diff (%d bytes) should be much smaller than full flush (%d bytes)", diffBytes, firstBytes)
	}
	// Exactly two rows are erased.
	if c := strings.Count(outBuf.String(), "\x1b[2K"); c != 2 {
		t.Errorf("expected 2 erases, got %d", c)
	}
}

func TestRenderInlineFrame_LayoutWidthMargin(t *testing.T) {
	r, _ := newInlineTestRenderer()
	r.renderInlineFrame()
	if r.root.LayoutNode().Width.Value != 39 {
		t.Errorf("expected layout width=39, got %v", r.root.LayoutNode().Width.Value)
	}
}

func TestRenderInlineFrame_DynamicHeight(t *testing.T) {
	r, _ := newInlineTestRenderer()
	r.root.Add(render.NewBaseRenderable(render.WithID("c1"), render.WithWidth(layout.Points(39)), render.WithHeight(layout.Points(3))))
	r.renderInlineFrame()
	// Three content rows plus the trailing row produce four.
	if r.lastInlineHeight != 4 {
		t.Errorf("expected 4 (3+1), got %d", r.lastInlineHeight)
	}
	r.root.Add(render.NewBaseRenderable(render.WithID("c2"), render.WithWidth(layout.Points(39)), render.WithHeight(layout.Points(5))))
	r.renderInlineFrame()
	// Eight content rows plus the trailing row produce nine.
	if r.lastInlineHeight != 9 {
		t.Errorf("expected 9 (8+1), got %d", r.lastInlineHeight)
	}
}

func TestProcessResize_InlineMode_Initial(t *testing.T) {
	r, _ := newInlineTestRenderer()
	// Before any frame, there is no prior output to erase.
	r.processResize(100, 50)
	if r.width != 100 {
		t.Errorf("expected width=100, got %d", r.width)
	}
	// Resize preserves zero so the first frame does not clear existing terminal output.
	if r.lastInlineHeight != 0 {
		t.Errorf("expected lastInlineHeight=0 (initial), got %d", r.lastInlineHeight)
	}
}

func TestProcessResize_InlineMode_AfterRender(t *testing.T) {
	r, _ := newInlineTestRenderer()
	r.lastInlineHeight = 5 // Simulate prior inline output.
	r.processResize(100, 50)
	// A prior frame forces overflow-style clearing after resize.
	if r.lastInlineHeight != 51 {
		t.Errorf("expected lastInlineHeight=51 (termHeight+1), got %d", r.lastInlineHeight)
	}
}

func TestPrepareLayout_ExecuteRender_Split(t *testing.T) {
	root := NewRootRenderable(40, 20)
	hitGrid := NewHitGrid(40, 20)
	root.SetContext(&mockContext{hitGrid: hitGrid})
	child := render.NewBaseRenderable(render.WithID("child"), render.WithWidth(layout.Points(20)), render.WithHeight(layout.Points(5)))
	root.Add(child)
	root.PrepareLayout(0.016)
	if child.LayoutNode().Computed.Width != 20 {
		t.Errorf("expected 20, got %d", child.LayoutNode().Computed.Width)
	}
	buf := buffer.New(40, 20)
	execStats := root.ExecuteRender(buf, 0.016)
	if execStats.CmdCount == 0 {
		t.Error("expected at least one render command")
	}
}

func TestDestroy_InlineMode(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	r.lastInlineHeight = 5
	r.Destroy()
	if !strings.Contains(outBuf.String(), "\r\n") {
		t.Error("expected \\r\\n on destroy")
	}
}
