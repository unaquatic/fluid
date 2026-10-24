package terminal

import (
	"bytes"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
)

// The test terminal is 40 columns by 20 rows.
const (
	testTermWidth  = 40
	testTermHeight = 20
)

func newInlineTestRenderer() (*Renderer, *bytes.Buffer) {
	var outBuf bytes.Buffer
	w, h := testTermWidth, testTermHeight
	root := NewRootRenderable(w, h)
	hitGrid := NewHitGrid(w, h)
	cfg := DefaultConfig()
	cfg.InlineMode = true
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
	root.SetHost(r)
	return r, &outBuf
}

// rowsBuffer returns a buffer whose row y is filled with fill glyphs of the
// rune at index y in glyphs.
func rowsBuffer(width int, glyphs string, fill int) *buffer.Buffer {
	runes := []rune(glyphs)
	buf := buffer.New(width, len(runes))
	for y, ch := range runes {
		for x := range fill {
			buf.SetCell(x, y, ch, style.ColorWhite, style.ColorBlack, 0)
		}
	}
	return buf
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

func TestInlineFlush_FirstFrame_KeepsExistingScreen(t *testing.T) {
	r, outBuf := newInlineTestRenderer()

	r.InlineFlush(rowsBuffer(10, "AC", 1))

	// The screen may hold unrelated output, which inline mode preserves.
	if strings.Contains(outBuf.String(), "\x1b[2J") {
		t.Error("first frame should not erase the screen")
	}
	if r.inlineViewportTop != 0 {
		t.Errorf("expected viewportTop=0, got %d", r.inlineViewportTop)
	}
	if r.inlineCursorRow != 1 {
		t.Errorf("expected cursorRow=1, got %d", r.inlineCursorRow)
	}
}

// TestInlineFlush_TallFirstFrame_ScrollsInsteadOfErasing is the property that
// makes long transcripts usable: content past the last row scrolls into the
// scrollback rather than triggering a screen-clearing repaint.
func TestInlineFlush_TallFirstFrame_ScrollsInsteadOfErasing(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	const height = testTermHeight + 5

	r.InlineFlush(rowsBuffer(10, strings.Repeat("A", height), 1))
	output := outBuf.String()

	if strings.Contains(output, "\x1b[2J") || strings.Contains(output, "\x1b[3J") {
		t.Error("tall content should scroll, not erase the screen and scrollback")
	}
	if r.inlineViewportTop != height-testTermHeight {
		t.Errorf("expected viewportTop=%d, got %d", height-testTermHeight, r.inlineViewportTop)
	}
	if r.inlineCursorRow != height-1 {
		t.Errorf("expected cursorRow=%d, got %d", height-1, r.inlineCursorRow)
	}
}

// TestInlineFlush_TallFrame_StaysQuietWhenOnlyTheLastRowChanges covers the
// repaint loop that made long conversations shake: once content exceeds the
// terminal, updates must still be differential.
func TestInlineFlush_TallFrame_StaysQuietWhenOnlyTheLastRowChanges(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	const height = testTermHeight + 5

	r.InlineFlush(rowsBuffer(10, strings.Repeat("A", height), 5))
	firstBytes := outBuf.Len()
	outBuf.Reset()

	next := rowsBuffer(10, strings.Repeat("A", height), 5)
	for x := range 5 {
		next.SetCell(x, height-1, 'Z', style.ColorWhite, style.ColorBlack, 0)
	}
	r.InlineFlush(next)
	output := outBuf.String()

	if strings.Contains(output, "\x1b[2J") {
		t.Errorf("expected a differential update, got a full repaint: %q", output)
	}
	if c := strings.Count(output, inlineEraseRow); c != 1 {
		t.Errorf("expected 1 row erase, got %d", c)
	}
	if outBuf.Len() >= firstBytes {
		t.Errorf("diff wrote %d bytes, want well under the first frame's %d", outBuf.Len(), firstBytes)
	}
}

// TestInlineFlush_ChangeInScrollback_Repaints covers the one case that cannot
// be patched in place: the terminal cannot address rows it has scrolled away.
func TestInlineFlush_ChangeInScrollback_Repaints(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	const height = testTermHeight + 5

	r.InlineFlush(rowsBuffer(10, strings.Repeat("A", height), 5))
	outBuf.Reset()

	next := rowsBuffer(10, strings.Repeat("A", height), 5)
	next.SetCell(0, 0, 'Z', style.ColorWhite, style.ColorBlack, 0)
	r.InlineFlush(next)

	if !strings.Contains(outBuf.String(), inlineEraseAll) {
		t.Error("expected a full repaint when a scrolled-away row changes")
	}
}

func TestInlineFlush_AppendPastLastRow_AdvancesViewport(t *testing.T) {
	r, _ := newInlineTestRenderer()

	r.InlineFlush(rowsBuffer(10, strings.Repeat("A", testTermHeight), 1))
	if r.inlineViewportTop != 0 {
		t.Fatalf("expected viewportTop=0 after filling the screen, got %d", r.inlineViewportTop)
	}

	r.InlineFlush(rowsBuffer(10, strings.Repeat("A", testTermHeight+2), 1))

	if r.inlineViewportTop != 2 {
		t.Errorf("expected viewportTop=2 after two rows scrolled away, got %d", r.inlineViewportTop)
	}
	if r.inlineCursorRow != testTermHeight+1 {
		t.Errorf("expected cursorRow=%d, got %d", testTermHeight+1, r.inlineCursorRow)
	}
}

func TestInlineScroll(t *testing.T) {
	tests := []struct {
		name            string
		viewportTop     int
		cursorRow       int
		target          int
		wantViewportTop int
		wantCursorRow   int
		wantOutput      string
	}{
		{
			name: "target on screen", viewportTop: 0, cursorRow: 4, target: 3,
			wantViewportTop: 0, wantCursorRow: 4, wantOutput: "",
		},
		{
			name: "target on the last row", viewportTop: 0, cursorRow: 0, target: testTermHeight - 1,
			wantViewportTop: 0, wantCursorRow: 0, wantOutput: "",
		},
		{
			name: "target below the last row", viewportTop: 0, cursorRow: 5, target: testTermHeight + 1,
			wantViewportTop: 2, wantCursorRow: testTermHeight + 1,
			wantOutput: "\x1b[14B\r\n\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _ := newInlineTestRenderer()
			r.inlineViewportTop, r.inlineCursorRow = tt.viewportTop, tt.cursorRow

			var out bytes.Buffer
			viewportTop, cursorRow := r.inlineScroll(&out, tt.target, testTermHeight)

			if viewportTop != tt.wantViewportTop || cursorRow != tt.wantCursorRow {
				t.Errorf("got viewportTop=%d cursorRow=%d, want %d and %d",
					viewportTop, cursorRow, tt.wantViewportTop, tt.wantCursorRow)
			}
			if out.String() != tt.wantOutput {
				t.Errorf("output is %q, want %q", out.String(), tt.wantOutput)
			}
		})
	}
}

func TestInlineChangedRows(t *testing.T) {
	tests := []struct {
		name                string
		prev, next          string
		wantFirst, wantLast int
	}{
		{name: "identical", prev: "abc", next: "abc", wantFirst: -1, wantLast: -1},
		{name: "middle row", prev: "abc", next: "aXc", wantFirst: 1, wantLast: 1},
		{name: "appended", prev: "abc", next: "abcde", wantFirst: 3, wantLast: 4},
		{name: "dropped", prev: "abcde", next: "abc", wantFirst: 3, wantLast: 4},
		{name: "changed and appended", prev: "abc", next: "aXcde", wantFirst: 1, wantLast: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first, last := inlineChangedRows(rowsBuffer(10, tt.prev, 5), rowsBuffer(10, tt.next, 5), 0)
			if first != tt.wantFirst || last != tt.wantLast {
				t.Errorf("got first=%d last=%d, want %d and %d", first, last, tt.wantFirst, tt.wantLast)
			}
		})
	}
}

func TestInlineFlush_NothingChanged_ZeroOutput(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	r.InlineFlush(rowsBuffer(10, "Hello", 5))
	outBuf.Reset()

	r.InlineFlush(rowsBuffer(10, "Hello", 5))

	if outBuf.Len() != 0 {
		t.Errorf("expected 0 bytes for an identical frame, got %d", outBuf.Len())
	}
}

func TestInlineFlush_OnlyChangedRowsRewritten(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	r.InlineFlush(rowsBuffer(10, "abcde", 5))
	outBuf.Reset()

	r.InlineFlush(rowsBuffer(10, "abXde", 5))
	output := outBuf.String()

	if c := strings.Count(output, inlineEraseRow); c != 1 {
		t.Errorf("expected 1 erase for 1 dirty row, got %d", c)
	}
	if !strings.Contains(output, "XXXXX") {
		t.Error("expected the changed row content in the output")
	}
	for _, unchanged := range []string{"aaaaa", "bbbbb", "ddddd", "eeeee"} {
		if strings.Contains(output, unchanged) {
			t.Errorf("unchanged row %q should not be rewritten", unchanged)
		}
	}
}

func TestInlineFlush_HeightGrew_AppendsNewRows(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	r.InlineFlush(rowsBuffer(10, "abc", 5))
	outBuf.Reset()

	r.InlineFlush(rowsBuffer(10, "abcde", 5))
	output := outBuf.String()

	if c := strings.Count(output, inlineEraseRow); c != 2 {
		t.Errorf("expected 2 erases for 2 new rows, got %d", c)
	}
	if !strings.Contains(output, "ddddd") || !strings.Contains(output, "eeeee") {
		t.Error("expected both appended rows in the output")
	}
	for _, unchanged := range []string{"aaaaa", "bbbbb", "ccccc"} {
		if strings.Contains(output, unchanged) {
			t.Errorf("unchanged row %q should not be rewritten", unchanged)
		}
	}
	if r.inlineCursorRow != 4 {
		t.Errorf("expected cursorRow=4, got %d", r.inlineCursorRow)
	}
}

func TestInlineFlush_HeightShrunk_ErasesRemovedRows(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	r.InlineFlush(rowsBuffer(10, "abcde", 5))
	outBuf.Reset()

	r.InlineFlush(rowsBuffer(10, "abc", 5))
	output := outBuf.String()

	if c := strings.Count(output, inlineEraseRow); c != 2 {
		t.Errorf("expected 2 erases for 2 removed rows, got %d", c)
	}
	for _, unchanged := range []string{"aaaaa", "bbbbb", "ccccc"} {
		if strings.Contains(output, unchanged) {
			t.Errorf("unchanged row %q should not be rewritten", unchanged)
		}
	}
	if r.inlineCursorRow != 2 {
		t.Errorf("expected cursorRow=2, got %d", r.inlineCursorRow)
	}
}

// filledBuffer returns a buffer with one row for each background in bgs, each
// filled edge to edge with blanks in that background.
func filledBuffer(width int, bgs ...style.Color) *buffer.Buffer {
	buf := buffer.New(width, len(bgs))
	for y, bg := range bgs {
		buf.FillRect(0, y, width, 1, bg)
	}
	return buf
}

// eraseBackgrounds returns the background active at each row erase in
// output, as SGR parameters such as "48;2;48;48;48", or "" for the terminal
// default. Terminals fill erased cells with the active background.
func eraseBackgrounds(output string) []string {
	var bgs []string
	bg := ""
	for i := 0; i < len(output); {
		if strings.HasPrefix(output[i:], inlineEraseRow) {
			bgs = append(bgs, bg)
			i += len(inlineEraseRow)
			continue
		}
		if !strings.HasPrefix(output[i:], "\x1b[") {
			i++
			continue
		}
		end := i + 2
		for end < len(output) && (output[end] < 0x40 || output[end] > 0x7e) {
			end++
		}
		if end < len(output) && output[end] == 'm' {
			bg = sgrBackground(output[i+2:end], bg)
		}
		i = end + 1
	}
	return bgs
}

// sgrBackground returns the background that the SGR parameters params leave
// active when bg was active before them.
func sgrBackground(params, bg string) string {
	p := strings.Split(params, ";")
	for k := 0; k < len(p); k++ {
		switch p[k] {
		case "", "0", "49":
			bg = ""
		case "48":
			n := 3 // 48;5;index
			if k+1 < len(p) && p[k+1] == "2" {
				n = 5 // 48;2;r;g;b
			}
			n = min(n, len(p)-k)
			bg = strings.Join(p[k:k+n], ";")
			k += n - 1
		default:
			// A 16-color background is one code, 40-47 or bright 100-107.
			if n, err := strconv.Atoi(p[k]); err == nil && (40 <= n && n <= 47 || 100 <= n && n <= 107) {
				bg = p[k]
			}
		}
	}
	return bg
}

// bgParams returns the SGR parameters that select c as the background.
func bgParams(r *Renderer, c style.Color) string {
	seq := style.BgColorSeq(c, r.backend.ColorMode())
	return seq[2 : len(seq)-1]
}

// TestInlineFlush_ShrinkErasesInTheDefaultBackground guards against rows that
// a shorter frame drops taking the fill of the last cell written above them,
// as when a filled box moves up into the rows a closed menu left.
func TestInlineFlush_ShrinkErasesInTheDefaultBackground(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	menu, fill := style.ColorFromInts(40, 40, 40, 255), style.ColorFromInts(48, 48, 48, 255)
	r.InlineFlush(filledBuffer(10, menu, menu, fill, fill, fill))
	outBuf.Reset()

	r.InlineFlush(filledBuffer(10, fill, fill, fill))

	got := eraseBackgrounds(outBuf.String())
	want := []string{bgParams(r, fill), bgParams(r, fill), bgParams(r, fill), "", ""}
	if !slices.Equal(got, want) {
		t.Errorf("rows erased in backgrounds %q, want the two dropped rows in the default, %q", got, want)
	}
}

// TestInlineFlush_ErasesEachRowInItsEdgeBackground checks that the terminal
// column layout leaves free continues each row's last cell, rather than the
// row written before it.
func TestInlineFlush_ErasesEachRowInItsEdgeBackground(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	red, blue := style.ColorFromInts(255, 0, 0, 255), style.ColorFromInts(0, 0, 255, 255)

	r.InlineFlush(filledBuffer(10, red, blue, style.ColorTransparent))

	want := []string{bgParams(r, red), bgParams(r, blue), ""}
	if got := eraseBackgrounds(outBuf.String()); !slices.Equal(got, want) {
		t.Errorf("rows erased in backgrounds %q, want %q", got, want)
	}
}

// TestInlineFlush_EdgeBackgroundIgnoresTheCursor checks that a cursor on a
// row's last cell, which swaps the cell's colors, does not widen into the
// free column.
func TestInlineFlush_EdgeBackgroundIgnoresTheCursor(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	fill := style.ColorFromInts(48, 48, 48, 255)
	buf := filledBuffer(10, fill)
	r.SetCursorPosition(9, 0, true)
	r.drawInlineCursor(buf)

	r.InlineFlush(buf)

	if got, want := eraseBackgrounds(outBuf.String()), []string{bgParams(r, fill)}; !slices.Equal(got, want) {
		t.Errorf("row erased in backgrounds %q, want %q", got, want)
	}
}

func TestInlineFlush_WidthChange_Repaints(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	r.InlineFlush(rowsBuffer(10, "abc", 5))
	outBuf.Reset()

	r.InlineFlush(rowsBuffer(15, "abc", 5))
	output := outBuf.String()

	if !strings.Contains(output, inlineEraseAll) {
		t.Error("expected a full repaint after a width change")
	}
	if c := strings.Count(output, inlineEraseRow); c != 3 {
		t.Errorf("expected 3 row erases from the repaint, got %d", c)
	}
}

func TestInlineFlush_HeightChange_Repaints(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	r.InlineFlush(rowsBuffer(10, "abc", 5))
	outBuf.Reset()

	r.height = testTermHeight - 5
	r.InlineFlush(rowsBuffer(10, "abc", 5))

	if !strings.Contains(outBuf.String(), inlineEraseAll) {
		t.Error("expected a full repaint after a terminal height change")
	}
}

func TestInlineFlush_MovesUpToAChangedRow(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	r.InlineFlush(rowsBuffer(10, "abcde", 5))
	outBuf.Reset()

	r.InlineFlush(rowsBuffer(10, "Zbcde", 5))
	output := outBuf.String()

	// The cursor sat on row four and the change is on row zero.
	if !strings.Contains(output, "\x1b[4A") {
		t.Errorf("expected a four-row move up, got %q", output)
	}
	// Every flush parks on the last row of content.
	if !strings.Contains(output, "\x1b[4B") {
		t.Errorf("expected a four-row move back down, got %q", output)
	}
	if r.inlineCursorRow != 4 {
		t.Errorf("expected cursorRow=4, got %d", r.inlineCursorRow)
	}
}

func TestRenderInlineFrame_LayoutWidthMargin(t *testing.T) {
	r, _ := newInlineTestRenderer()
	r.renderInlineFrame()
	if r.root.LayoutNode().Width.Value != testTermWidth-1 {
		t.Errorf("expected layout width=%d, got %v", testTermWidth-1, r.root.LayoutNode().Width.Value)
	}
}

func TestRenderInlineFrame_DynamicHeight(t *testing.T) {
	r, _ := newInlineTestRenderer()
	addRows := func(id string, rows int) {
		r.root.Add(render.NewBaseRenderable(render.WithID(id),
			render.WithWidth(layout.Points(testTermWidth-1)), render.WithHeight(layout.Points(rows))))
	}

	addRows("c1", 3)
	r.renderInlineFrame()
	if r.inlineCursorRow != 2 {
		t.Errorf("three rows: expected cursorRow=2, got %d", r.inlineCursorRow)
	}

	addRows("c2", 5)
	r.renderInlineFrame()
	if r.inlineCursorRow != 7 {
		t.Errorf("eight rows: expected cursorRow=7, got %d", r.inlineCursorRow)
	}
}

func TestProcessResize_InlineMode(t *testing.T) {
	r, _ := newInlineTestRenderer()
	r.InlineFlush(rowsBuffer(10, "abc", 5))

	r.processResize(100, 50)

	if r.width != 100 {
		t.Errorf("expected width=100, got %d", r.width)
	}
	if r.root.LayoutNode().Width.Value != 100 {
		t.Errorf("expected layout width=100, got %v", r.root.LayoutNode().Width.Value)
	}
	// The flush detects the height change and repaints; resize leaves the
	// tracked terminal state alone.
	if r.inlineTermHeight != testTermHeight {
		t.Errorf("expected the tracked height to stay %d, got %d", testTermHeight, r.inlineTermHeight)
	}
}

func TestPrepareLayout_ExecuteRender_Split(t *testing.T) {
	root := NewRootRenderable(40, 20)
	hitGrid := NewHitGrid(40, 20)
	root.SetHost(&mockHost{hitGrid: hitGrid})
	child := render.NewBaseRenderable(render.WithID("child"), render.WithWidth(layout.Points(20)), render.WithHeight(layout.Points(5)))
	root.Add(child)
	root.PrepareLayout(16 * time.Millisecond)
	if child.LayoutNode().Computed.Width != 20 {
		t.Errorf("expected 20, got %d", child.LayoutNode().Computed.Width)
	}
	buf := buffer.New(40, 20)
	execStats := root.ExecuteRender(buf, 16*time.Millisecond)
	if execStats.CmdCount == 0 {
		t.Error("expected at least one render command")
	}
}

func TestDestroy_InlineMode(t *testing.T) {
	r, outBuf := newInlineTestRenderer()
	r.InlineFlush(rowsBuffer(10, "abcde", 5))
	outBuf.Reset()

	r.Destroy()

	// Restoring terminal modes follows, so the newline leads the output.
	if !strings.HasPrefix(outBuf.String(), "\r\n") {
		t.Errorf("expected a trailing newline below inline output, got %q", outBuf.String())
	}
}
