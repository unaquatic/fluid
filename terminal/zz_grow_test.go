package terminal

import "testing"

// TestGrowAcrossWindow streams one message that grows past the window, logging
// the flush kind at each step.
func TestGrowAcrossWindow(t *testing.T) {
	r, _ := newInlineTestRenderer() // 10-row terminal
	msg := newGlyphRow("msg", 'a', 39, 2)
	r.root.Add(msg)
	prompt := newGlyphRow("prompt", 'p', 39, 2)
	r.root.Add(prompt)
	r.renderInlineFrame()

	for rows := 3; rows <= 16; rows++ {
		msg.resize(rows) // one streaming step: the message gets one row taller
		r.renderInlineFrame()
		kind := "diff"
		if r.root.LastFrameStats.FlushBytes > 1000 {
			kind = "FULL"
		}
		t.Logf("msg rows=%-3d content=%-3d vpTop=%-3d msgRect={Y:%d H:%d}  damage={Y:%d H:%d}  bytes=%-6d %s",
			rows, r.nextBuffer.Height(), r.inlineViewportTop, msg.Y(), msg.Height(),
			r.inlineDamaged.Y, r.inlineDamaged.Height, r.root.LastFrameStats.FlushBytes, kind)
	}
}
