package textbuf

import (
	"testing"

	"github.com/unaquatic/fluid/style"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func setupExtmarks(text string) (*EditBuffer, *EditSurface, *ExtmarksController) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	ev := NewEditSurface(eb, 40, 10)
	eb.SetText(text)
	ec := NewExtmarksController(eb, ev)
	return eb, ev, ec
}

// ---------------------------------------------------------------------------
// Creation and basic operations
// ---------------------------------------------------------------------------

func TestExtmarks_Create(t *testing.T) {
	_, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	id := ec.Create(ExtmarkOptions{Start: 0, End: 5})
	if id != 1 {
		t.Errorf("expected id 1, got %d", id)
	}

	em := ec.Get(id)
	if em == nil {
		t.Fatal("expected non-nil extmark")
	}
	if em.Start != 0 || em.End != 5 {
		t.Errorf("expected [0,5), got [%d,%d)", em.Start, em.End)
	}
	if em.Virtual {
		t.Error("expected non-virtual by default")
	}
}

func TestExtmarks_CreateVirtual(t *testing.T) {
	_, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	id := ec.Create(ExtmarkOptions{Start: 6, End: 11, Virtual: true})
	em := ec.Get(id)
	if em == nil || !em.Virtual {
		t.Error("expected virtual extmark")
	}
}

func TestExtmarks_MultipleIDs(t *testing.T) {
	_, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	id1 := ec.Create(ExtmarkOptions{Start: 0, End: 5})
	id2 := ec.Create(ExtmarkOptions{Start: 6, End: 11})
	if id1 == id2 {
		t.Error("IDs should be unique")
	}
	all := ec.All()
	if len(all) != 2 {
		t.Errorf("expected 2 extmarks, got %d", len(all))
	}
}

func TestExtmarks_Data(t *testing.T) {
	_, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	id := ec.Create(ExtmarkOptions{
		Start: 0, End: 5,
		Data: "test-data",
	})
	em := ec.Get(id)
	if em.Data != "test-data" {
		t.Errorf("expected data %q, got %v", "test-data", em.Data)
	}
}

func TestExtmarks_Metadata(t *testing.T) {
	_, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	id := ec.Create(ExtmarkOptions{
		Start: 0, End: 5,
		Metadata: map[string]string{"key": "val"},
	})
	meta := ec.Metadata(id)
	m, ok := meta.(map[string]string)
	if !ok || m["key"] != "val" {
		t.Error("metadata mismatch")
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestExtmarks_Delete(t *testing.T) {
	_, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	id := ec.Create(ExtmarkOptions{Start: 0, End: 5})
	ok := ec.Delete(id)
	if !ok {
		t.Error("expected delete to succeed")
	}
	if ec.Get(id) != nil {
		t.Error("expected nil after delete")
	}
}

func TestExtmarks_DeleteNonExistent(t *testing.T) {
	_, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	ok := ec.Delete(999)
	if ok {
		t.Error("expected false for non-existent ID")
	}
}

// ---------------------------------------------------------------------------
// Query helpers
// ---------------------------------------------------------------------------

func TestExtmarks_GetVirtual(t *testing.T) {
	_, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	ec.Create(ExtmarkOptions{Start: 0, End: 5, Virtual: false})
	ec.Create(ExtmarkOptions{Start: 6, End: 11, Virtual: true})
	v := ec.Virtual()
	if len(v) != 1 {
		t.Errorf("expected 1 virtual, got %d", len(v))
	}
}

func TestExtmarks_GetAtOffset(t *testing.T) {
	_, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	ec.Create(ExtmarkOptions{Start: 0, End: 5})
	ec.Create(ExtmarkOptions{Start: 6, End: 11})

	at2 := ec.AtOffset(2)
	if len(at2) != 1 {
		t.Errorf("expected 1 at offset 2, got %d", len(at2))
	}
	at7 := ec.AtOffset(7)
	if len(at7) != 1 {
		t.Errorf("expected 1 at offset 7, got %d", len(at7))
	}
	at5 := ec.AtOffset(5) // end is exclusive
	if len(at5) != 0 {
		t.Errorf("expected 0 at offset 5 (exclusive), got %d", len(at5))
	}
}

func TestExtmarks_Clear(t *testing.T) {
	_, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	ec.Create(ExtmarkOptions{Start: 0, End: 5})
	ec.Create(ExtmarkOptions{Start: 6, End: 11})
	ec.Clear()
	if len(ec.All()) != 0 {
		t.Error("expected empty after clear")
	}
}

// ---------------------------------------------------------------------------
// Type registry
// ---------------------------------------------------------------------------

func TestExtmarks_TypeRegistry(t *testing.T) {
	_, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	id1 := ec.RegisterType("ghost-text")
	id2 := ec.RegisterType("completion")
	id3 := ec.RegisterType("ghost-text") // same name

	if id1 == id2 {
		t.Error("different names should get different IDs")
	}
	if id1 != id3 {
		t.Error("same name should return same ID")
	}
	if ec.TypeName(id1) != "ghost-text" {
		t.Error("type name mismatch")
	}
	if ec.TypeID("completion") != id2 {
		t.Error("type ID lookup mismatch")
	}
	if ec.TypeID("nonexistent") != -1 {
		t.Error("expected -1 for unknown type name")
	}
}

func TestExtmarks_GetAllForTypeID(t *testing.T) {
	_, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	tid := ec.RegisterType("test")
	ec.Create(ExtmarkOptions{Start: 0, End: 5, TypeID: tid})
	ec.Create(ExtmarkOptions{Start: 6, End: 11, TypeID: tid})
	ec.Create(ExtmarkOptions{Start: 0, End: 3, TypeID: 0}) // default type

	byType := ec.AllForTypeID(tid)
	if len(byType) != 2 {
		t.Errorf("expected 2 for type %d, got %d", tid, len(byType))
	}
}

// ---------------------------------------------------------------------------
// Position adjustment — insertion
// ---------------------------------------------------------------------------

func TestExtmarks_InsertShiftsAfter(t *testing.T) {
	eb, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	// Mark "World" [6,11)
	id := ec.Create(ExtmarkOptions{Start: 6, End: 11})

	// Insert "X" at cursor 0 (before the extmark)
	eb.SetCursor(0, 0)
	eb.InsertChar('X')

	em := ec.Get(id)
	if em == nil {
		t.Fatal("extmark should still exist")
	}
	// "X" has width 1, so the extmark should shift by 1
	if em.Start != 7 || em.End != 12 {
		t.Errorf("expected [7,12), got [%d,%d)", em.Start, em.End)
	}
}

func TestExtmarks_InsertInside(t *testing.T) {
	eb, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	// Mark "Hello" [0,5)
	id := ec.Create(ExtmarkOptions{Start: 0, End: 5})

	// Insert at col 3 (inside "Hello")
	eb.SetCursor(0, 3)
	eb.InsertChar('X')

	em := ec.Get(id)
	if em == nil {
		t.Fatal("extmark should still exist")
	}
	// Insertion inside: start unchanged, end grows
	if em.Start != 0 || em.End != 6 {
		t.Errorf("expected [0,6), got [%d,%d)", em.Start, em.End)
	}
}

func TestExtmarks_InsertBefore(t *testing.T) {
	eb, _, ec := setupExtmarks("AB")
	defer ec.Destroy()

	id := ec.Create(ExtmarkOptions{Start: 1, End: 2})

	eb.SetCursor(0, 0)
	eb.InsertChar('X')

	em := ec.Get(id)
	if em.Start != 2 || em.End != 3 {
		t.Errorf("expected [2,3), got [%d,%d)", em.Start, em.End)
	}
}

func TestExtmarks_InsertText(t *testing.T) {
	eb, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	id := ec.Create(ExtmarkOptions{Start: 6, End: 11})

	eb.SetCursor(0, 0)
	eb.InsertText("XYZ")

	em := ec.Get(id)
	if em.Start != 9 || em.End != 14 {
		t.Errorf("expected [9,14), got [%d,%d)", em.Start, em.End)
	}
}

func TestExtmarks_NewLineShifts(t *testing.T) {
	eb, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	id := ec.Create(ExtmarkOptions{Start: 6, End: 11})

	// Insert newline at col 5 (between "Hello" and " World")
	eb.SetCursor(0, 5)
	eb.NewLine()

	em := ec.Get(id)
	// Newline at offset 5 shifts everything after by +1
	if em.Start != 7 || em.End != 12 {
		t.Errorf("expected [7,12), got [%d,%d)", em.Start, em.End)
	}
}

// ---------------------------------------------------------------------------
// Position adjustment — deletion
// ---------------------------------------------------------------------------

func TestExtmarks_DeleteShiftsAfter(t *testing.T) {
	eb, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	// Mark "World" [6,11)
	id := ec.Create(ExtmarkOptions{Start: 6, End: 11})

	// Delete first char
	eb.SetCursor(0, 1)
	eb.DeleteCharBackward()

	em := ec.Get(id)
	if em.Start != 5 || em.End != 10 {
		t.Errorf("expected [5,10), got [%d,%d)", em.Start, em.End)
	}
}

func TestExtmarks_DeleteRemovesContained(t *testing.T) {
	eb, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	// Mark "lo" [3,5)
	id := ec.Create(ExtmarkOptions{Start: 3, End: 5})

	// Delete "Hello" (5 chars)
	eb.DeleteRange(0, 0, 0, 5)

	// Extmark was entirely within deleted range
	em := ec.Get(id)
	if em != nil {
		t.Error("expected extmark to be removed when fully deleted")
	}
}

func TestExtmarks_DeleteShrinks(t *testing.T) {
	eb, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	// Mark "Hello World" [0,11)
	id := ec.Create(ExtmarkOptions{Start: 0, End: 11})

	// Delete "lo W" [3,7)
	eb.DeleteRange(0, 3, 0, 7)

	em := ec.Get(id)
	if em == nil {
		t.Fatal("expected extmark to survive partial deletion")
	}
	if em.Start != 0 || em.End != 7 {
		t.Errorf("expected [0,7), got [%d,%d)", em.Start, em.End)
	}
}

func TestExtmarks_DeleteLine(t *testing.T) {
	eb, _, ec := setupExtmarks("Line1\nLine2\nLine3")
	defer ec.Destroy()

	// Mark "Line2" — offset: "Line1\n" = 6, so "Line2" = [6,11)
	id := ec.Create(ExtmarkOptions{Start: 6, End: 11})

	// Position cursor on Line2 and delete it
	eb.SetCursor(1, 0)
	eb.DeleteLine()

	em := ec.Get(id)
	// "Line2\n" = 6 chars deleted, extmark was entirely within
	if em != nil {
		t.Error("expected extmark to be removed with deleted line")
	}
}

// ---------------------------------------------------------------------------
// Virtual extmark — cursor skipping
// ---------------------------------------------------------------------------

func TestExtmarks_CursorSkipsVirtualLeft(t *testing.T) {
	eb, ev, ec := setupExtmarks("ABCDE")
	defer ec.Destroy()

	// Make "BCD" [1,4) virtual
	ec.Create(ExtmarkOptions{Start: 1, End: 4, Virtual: true})

	// Position cursor at offset 4 (after the virtual range)
	eb.SetCursor(0, 4)
	vc := ev.VisualCursor()
	if vc.Offset != 4 {
		t.Fatalf("setup: expected offset 4, got %d", vc.Offset)
	}

	// Move left should skip over [1,4) and land at offset 1 (the boundary
	// right before the virtual range, not one further at offset 0).
	eb.MoveCursorLeft()
	vc = ev.VisualCursor()
	if vc.Offset != 1 {
		t.Errorf("expected offset 1 after skipping virtual left, got %d", vc.Offset)
	}
}

func TestExtmarks_CursorSkipsVirtualRight(t *testing.T) {
	eb, ev, ec := setupExtmarks("ABCDE")
	defer ec.Destroy()

	// Make "BCD" [1,4) virtual
	ec.Create(ExtmarkOptions{Start: 1, End: 4, Virtual: true})

	// Position cursor at offset 0
	eb.SetCursor(0, 0)

	// First right: lands at boundary offset 1 (right before virtual range)
	eb.MoveCursorRight()
	vc := ev.VisualCursor()
	if vc.Offset != 1 {
		t.Errorf("expected offset 1 (boundary) after first right, got %d", vc.Offset)
	}

	// Second right: from boundary, skips over [1,4) to offset 4
	eb.MoveCursorRight()
	vc = ev.VisualCursor()
	if vc.Offset != 4 {
		t.Errorf("expected offset 4 after skipping virtual right, got %d", vc.Offset)
	}
}

func TestExtmarks_CursorNoSkipNonVirtual(t *testing.T) {
	eb, ev, ec := setupExtmarks("ABCDE")
	defer ec.Destroy()

	// Non-virtual extmark
	ec.Create(ExtmarkOptions{Start: 1, End: 4, Virtual: false})

	eb.SetCursor(0, 0)
	eb.MoveCursorRight()
	vc := ev.VisualCursor()
	// Should move to offset 1 normally (no skipping for non-virtual)
	if vc.Offset != 1 {
		t.Errorf("expected offset 1, got %d", vc.Offset)
	}
}

func TestExtmarks_BackspaceDeletesVirtual(t *testing.T) {
	eb, ev, ec := setupExtmarks("ABCDE")
	defer ec.Destroy()

	// Make "BCD" [1,4) virtual
	id := ec.Create(ExtmarkOptions{Start: 1, End: 4, Virtual: true})

	// Position cursor at offset 4 (end of virtual range)
	eb.SetCursor(0, 4)

	// Backspace should delete the entire virtual extmark
	eb.DeleteCharBackward()

	text := eb.Text()
	if text != "AE" {
		t.Errorf("expected %q after virtual delete, got %q", "AE", text)
	}

	em := ec.Get(id)
	if em != nil {
		t.Error("expected virtual extmark to be removed")
	}

	vc := ev.VisualCursor()
	if vc.Offset != 1 {
		t.Errorf("expected cursor at offset 1, got %d", vc.Offset)
	}
}

func TestExtmarks_ForwardDeleteVirtual(t *testing.T) {
	eb, ev, ec := setupExtmarks("ABCDE")
	defer ec.Destroy()

	// Make "BCD" [1,4) virtual
	id := ec.Create(ExtmarkOptions{Start: 1, End: 4, Virtual: true})

	// Position cursor at offset 1 (start of virtual range)
	eb.SetCursor(0, 1)

	// Forward delete at start should delete entire virtual
	eb.DeleteChar()

	text := eb.Text()
	if text != "AE" {
		t.Errorf("expected %q, got %q", "AE", text)
	}

	if ec.Get(id) != nil {
		t.Error("expected virtual extmark to be removed")
	}

	vc := ev.VisualCursor()
	if vc.Offset != 1 {
		t.Errorf("expected cursor at offset 1, got %d", vc.Offset)
	}
}

// ---------------------------------------------------------------------------
// Undo / Redo
// ---------------------------------------------------------------------------

func TestExtmarks_UndoRestoresState(t *testing.T) {
	eb, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	id := ec.Create(ExtmarkOptions{Start: 6, End: 11})

	// Insert shifts extmark
	eb.SetCursor(0, 0)
	eb.InsertChar('X')

	em := ec.Get(id)
	if em.Start != 7 || em.End != 12 {
		t.Fatalf("after insert: expected [7,12), got [%d,%d)", em.Start, em.End)
	}

	// Undo should restore extmark position
	eb.Undo()

	em = ec.Get(id)
	if em == nil {
		t.Fatal("extmark should exist after undo")
	}
	if em.Start != 6 || em.End != 11 {
		t.Errorf("after undo: expected [6,11), got [%d,%d)", em.Start, em.End)
	}
}

func TestExtmarks_RedoRestoresState(t *testing.T) {
	eb, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	id := ec.Create(ExtmarkOptions{Start: 6, End: 11})

	eb.SetCursor(0, 0)
	eb.InsertChar('X')

	eb.Undo()
	// Now redo
	eb.Redo()

	em := ec.Get(id)
	if em == nil {
		t.Fatal("extmark should exist after redo")
	}
	if em.Start != 7 || em.End != 12 {
		t.Errorf("after redo: expected [7,12), got [%d,%d)", em.Start, em.End)
	}
}

func TestExtmarks_UndoWithoutHistory(t *testing.T) {
	eb, _, ec := setupExtmarks("Hello")
	defer ec.Destroy()

	ec.Create(ExtmarkOptions{Start: 0, End: 5})

	// Undo with no edit history should be a no-op
	ok := eb.Undo()
	if ok {
		t.Error("undo should return false with empty history")
	}
}

func TestExtmarks_UndoHighlightsMatchRestoredText(t *testing.T) {
	// Regression: updateHighlights was called before doUndo restored the
	// text, so per-line highlights were mapped against wrong line widths.
	// After inserting on line 0 and undoing, highlights on line 1 would
	// shift left by the inserted width.
	eb, _, ec := setupExtmarks("AB\nCD")
	defer ec.Destroy()

	fg := style.ColorFromHex("#FF0000")

	// Extmark on "CD" — line 1, offsets [3,5)  (AB=2, \n=1, CD starts at 3)
	ec.Create(ExtmarkOptions{Start: 3, End: 5, Fg: &fg})

	tb := ec.editBuffer.TextBuffer()

	// Before edit: line 1 highlight should span [0,2) within the line.
	hls := tb.LineHighlights(1)
	if len(hls) != 1 || hls[0].Start != 0 || hls[0].End != 2 {
		t.Fatalf("before edit: expected line 1 highlight [0,2), got %v", hls)
	}

	// Insert "X" at start of line 0 → text becomes "XAB\nCD"
	eb.SetCursor(0, 0)
	eb.InsertChar('X')

	// Extmark shifted to [4,6), line 1 highlight should still be [0,2).
	hls = tb.LineHighlights(1)
	if len(hls) != 1 || hls[0].Start != 0 || hls[0].End != 2 {
		t.Fatalf("after insert: expected line 1 highlight [0,2), got %v", hls)
	}

	// Undo → text returns to "AB\nCD", extmark returns to [3,5).
	// Line 1 highlight must be [0,2) — NOT shifted.
	eb.Undo()

	hls = tb.LineHighlights(1)
	if len(hls) != 1 {
		t.Fatalf("after undo: expected 1 highlight on line 1, got %d", len(hls))
	}
	if hls[0].Start != 0 || hls[0].End != 2 {
		t.Errorf("after undo: expected line 1 highlight [0,2), got [%d,%d)", hls[0].Start, hls[0].End)
	}
}

func TestExtmarks_RedoHighlightsMatchRestoredText(t *testing.T) {
	// Same issue as undo but for redo path.
	eb, _, ec := setupExtmarks("AB\nCD")
	defer ec.Destroy()

	fg := style.ColorFromHex("#FF0000")
	ec.Create(ExtmarkOptions{Start: 3, End: 5, Fg: &fg})

	tb := ec.editBuffer.TextBuffer()

	// Insert, undo, then redo.
	eb.SetCursor(0, 0)
	eb.InsertChar('X')
	eb.Undo()
	eb.Redo()

	// After redo text is "XAB\nCD", extmark at [4,6).
	// Line 1 highlight should be [0,2).
	hls := tb.LineHighlights(1)
	if len(hls) != 1 {
		t.Fatalf("after redo: expected 1 highlight on line 1, got %d", len(hls))
	}
	if hls[0].Start != 0 || hls[0].End != 2 {
		t.Errorf("after redo: expected line 1 highlight [0,2), got [%d,%d)", hls[0].Start, hls[0].End)
	}
}

func TestExtmarks_BackspaceAtStartNoDesync(t *testing.T) {
	// Regression: overrideDeleteCharBackward called saveSnapshot before
	// checking if the deletion would actually happen. At offset 0,
	// doDeleteCharBackward is a no-op (no saveUndo), creating an orphan
	// extmarks snapshot that desynchronized the undo stacks.
	eb, _, ec := setupExtmarks("AB")
	defer ec.Destroy()

	id := ec.Create(ExtmarkOptions{Start: 0, End: 2})

	// Type a character so there's one real undo entry.
	eb.SetCursor(0, 2)
	eb.InsertChar('X') // text "ABX", extmark [0,2)

	// Backspace at offset 0 — should be a no-op, must NOT push an
	// orphan extmarks snapshot.
	eb.SetCursor(0, 0)
	eb.DeleteCharBackward()

	// Undo should revert the InsertChar, NOT apply a stale extmark snapshot.
	eb.Undo()

	em := ec.Get(id)
	if em == nil {
		t.Fatal("extmark should exist after undo")
	}
	if em.Start != 0 || em.End != 2 {
		t.Errorf("expected extmark [0,2) after undo, got [%d,%d)", em.Start, em.End)
	}
	if eb.Text() != "AB" {
		t.Errorf("expected text 'AB' after undo, got %q", eb.Text())
	}
}

func TestExtmarks_DeleteCharAtEndNoDesync(t *testing.T) {
	// Same issue as backspace-at-start but for forward delete at end of buffer.
	eb, _, ec := setupExtmarks("AB")
	defer ec.Destroy()

	id := ec.Create(ExtmarkOptions{Start: 0, End: 2})

	// Type a character.
	eb.SetCursor(0, 0)
	eb.InsertChar('X') // text "XAB", extmark [1,3)

	// Forward delete at end of buffer — should be a no-op.
	eb.SetCursor(0, 3)
	eb.DeleteChar()

	// Undo should revert the InsertChar cleanly.
	eb.Undo()

	em := ec.Get(id)
	if em == nil {
		t.Fatal("extmark should exist after undo")
	}
	if em.Start != 0 || em.End != 2 {
		t.Errorf("expected extmark [0,2) after undo, got [%d,%d)", em.Start, em.End)
	}
	if eb.Text() != "AB" {
		t.Errorf("expected text 'AB' after undo, got %q", eb.Text())
	}
}

func TestExtmarks_UndoRestoresTypeIndex(t *testing.T) {
	// Regression: restoreSnapshot did not rebuild extmarksByType, so
	// AllForTypeID returned stale results after undo.
	eb, _, ec := setupExtmarks("Hello")
	defer ec.Destroy()

	typeID := ec.RegisterType("test")
	ec.Create(ExtmarkOptions{Start: 0, End: 5, TypeID: typeID})

	// Verify the type index works before any edits.
	if len(ec.AllForTypeID(typeID)) != 1 {
		t.Fatal("expected 1 extmark for typeID before edit")
	}

	// Edit then undo.
	eb.SetCursor(0, 0)
	eb.InsertChar('X')
	eb.Undo()

	// After undo the type index must still return the extmark.
	ems := ec.AllForTypeID(typeID)
	if len(ems) != 1 {
		t.Fatalf("expected 1 extmark for typeID after undo, got %d", len(ems))
	}
	if ems[0].Start != 0 || ems[0].End != 5 {
		t.Errorf("expected extmark [0,5) after undo, got [%d,%d)", ems[0].Start, ems[0].End)
	}
}

func TestExtmarks_RedoRestoresTypeIndex(t *testing.T) {
	eb, _, ec := setupExtmarks("Hello")
	defer ec.Destroy()

	typeID := ec.RegisterType("test")
	ec.Create(ExtmarkOptions{Start: 0, End: 5, TypeID: typeID})

	eb.SetCursor(0, 0)
	eb.InsertChar('X')
	eb.Undo()
	eb.Redo()

	ems := ec.AllForTypeID(typeID)
	if len(ems) != 1 {
		t.Fatalf("expected 1 extmark for typeID after redo, got %d", len(ems))
	}
	if ems[0].Start != 1 || ems[0].End != 6 {
		t.Errorf("expected extmark [1,6) after redo, got [%d,%d)", ems[0].Start, ems[0].End)
	}
}

// ---------------------------------------------------------------------------
// Highlights
// ---------------------------------------------------------------------------

func TestExtmarks_HighlightsApplied(t *testing.T) {
	_, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	fg := style.ColorRed
	ec.Create(ExtmarkOptions{
		Start: 0, End: 5,
		Fg: &fg,
	})

	// Check that the TextBuffer has highlights
	tb := ec.editBuffer.TextBuffer()
	hls := tb.LineHighlights(0)
	if len(hls) == 0 {
		t.Error("expected highlights on line 0")
	}
}

func TestExtmarks_HighlightsClearedOnDelete(t *testing.T) {
	_, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	fg := style.ColorRed
	id := ec.Create(ExtmarkOptions{
		Start: 0, End: 5,
		Fg: &fg,
	})
	ec.Delete(id)

	tb := ec.editBuffer.TextBuffer()
	hls := tb.LineHighlights(0)
	if len(hls) != 0 {
		t.Error("expected no highlights after deleting extmark")
	}
}

func TestExtmarks_NoHighlightWithoutColors(t *testing.T) {
	_, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	ec.Create(ExtmarkOptions{Start: 0, End: 5})

	tb := ec.editBuffer.TextBuffer()
	hls := tb.LineHighlights(0)
	if len(hls) != 0 {
		t.Error("expected no highlights when no colors specified")
	}
}

// ---------------------------------------------------------------------------
// SetText / ReplaceText / Clear — extmarks cleared
// ---------------------------------------------------------------------------

func TestExtmarks_SetTextClearsExtmarks(t *testing.T) {
	eb, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	ec.Create(ExtmarkOptions{Start: 0, End: 5})
	eb.SetText("New text")

	if len(ec.All()) != 0 {
		t.Error("SetText should clear all extmarks")
	}
}

func TestExtmarks_ReplaceTextClearsExtmarks(t *testing.T) {
	eb, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	ec.Create(ExtmarkOptions{Start: 0, End: 5})
	eb.ReplaceText("New text")

	if len(ec.All()) != 0 {
		t.Error("ReplaceText should clear all extmarks")
	}
}

func TestExtmarks_ClearClearsExtmarks(t *testing.T) {
	eb, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	ec.Create(ExtmarkOptions{Start: 0, End: 5})
	eb.Clear()

	if len(ec.All()) != 0 {
		t.Error("Clear should clear all extmarks")
	}
}

// ---------------------------------------------------------------------------
// DeleteSelectedText
// ---------------------------------------------------------------------------

func TestExtmarks_DeleteSelectedText(t *testing.T) {
	_, ev, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	// Mark "World" [6,11)
	id := ec.Create(ExtmarkOptions{Start: 6, End: 11})

	// Select "Hello" [0,5) and delete it
	bg := style.ColorBlue
	ev.SetSelection(&TextSelection{Start: 0, End: 5, BgColor: &bg})
	ev.DeleteSelectedText()

	em := ec.Get(id)
	if em == nil {
		t.Fatal("extmark should survive deletion of other text")
	}
	// After deleting 5 chars at offset 0, "World" shifts from [6,11) to [1,6)
	if em.Start != 1 || em.End != 6 {
		t.Errorf("expected [1,6), got [%d,%d)", em.Start, em.End)
	}
}

// ---------------------------------------------------------------------------
// Destroy
// ---------------------------------------------------------------------------

func TestExtmarks_Destroy(t *testing.T) {
	eb, _, ec := setupExtmarks("Hello World")

	ec.Create(ExtmarkOptions{Start: 0, End: 5})
	ec.Destroy()

	// After destroy, operations should fall through to default behavior
	eb.SetCursor(0, 5)
	eb.InsertChar('!')
	if eb.Text() != "Hello! World" {
		t.Errorf("expected normal behavior after destroy, got %q", eb.Text())
	}

	// Accessing destroyed controller should return nil/empty
	if ec.Get(1) != nil {
		t.Error("Get should return nil on destroyed controller")
	}
	if len(ec.All()) != 0 {
		t.Error("All should return nil on destroyed controller")
	}
}

func TestExtmarks_DestroyIdempotent(t *testing.T) {
	_, _, ec := setupExtmarks("Hello")
	ec.Destroy()
	ec.Destroy() // should not panic
}

// ---------------------------------------------------------------------------
// Multi-line extmarks
// ---------------------------------------------------------------------------

func TestExtmarks_MultiLineInsert(t *testing.T) {
	eb, _, ec := setupExtmarks("Line1\nLine2\nLine3")
	defer ec.Destroy()

	// "Line3" starts at offset 12 (5+1+5+1=12)
	id := ec.Create(ExtmarkOptions{Start: 12, End: 17})

	// Insert at start of Line1
	eb.SetCursor(0, 0)
	eb.InsertText("X")

	em := ec.Get(id)
	if em.Start != 13 || em.End != 18 {
		t.Errorf("expected [13,18), got [%d,%d)", em.Start, em.End)
	}
}

func TestExtmarks_MultiLineDelete(t *testing.T) {
	eb, _, ec := setupExtmarks("Line1\nLine2\nLine3")
	defer ec.Destroy()

	// "Line3" starts at offset 12
	id := ec.Create(ExtmarkOptions{Start: 12, End: 17})

	// Delete "Line1\n" (6 chars)
	eb.DeleteRange(0, 0, 1, 0)

	em := ec.Get(id)
	if em == nil {
		t.Fatal("extmark should survive")
	}
	if em.Start != 6 || em.End != 11 {
		t.Errorf("expected [6,11), got [%d,%d)", em.Start, em.End)
	}
}

// ---------------------------------------------------------------------------
// ExtmarksHistory
// ---------------------------------------------------------------------------

func TestExtmarksHistory_Basic(t *testing.T) {
	h := &ExtmarksHistory{}

	m := map[int]Extmark{
		1: {ID: 1, Start: 0, End: 5},
	}
	h.SaveSnapshot(m, 2)

	if !h.CanUndo() {
		t.Error("should be able to undo")
	}
	if h.CanRedo() {
		t.Error("should not be able to redo initially")
	}

	snap := h.Undo()
	if snap == nil {
		t.Fatal("undo should return snapshot")
	}
	em := snap.extmarks[1]
	if em.Start != 0 || em.End != 5 {
		t.Error("snapshot data mismatch")
	}

	if h.CanUndo() {
		t.Error("should not be able to undo after popping last")
	}
}

func TestExtmarksHistory_ClearOnSave(t *testing.T) {
	h := &ExtmarksHistory{}

	m1 := map[int]Extmark{1: {ID: 1, Start: 0, End: 5}}
	m2 := map[int]Extmark{1: {ID: 1, Start: 1, End: 6}}

	h.SaveSnapshot(m1, 2)
	// Simulate undo
	snap := h.Undo()
	h.PushRedo(ExtmarksSnapshot{extmarks: cloneExtmarks(m2), nextID: 2})

	if !h.CanRedo() {
		t.Error("should be able to redo")
	}

	// New save should clear redo
	h.SaveSnapshot(snap.extmarks, snap.nextID)
	if h.CanRedo() {
		t.Error("redo should be cleared after new save")
	}
}

func TestExtmarksHistory_Clear(t *testing.T) {
	h := &ExtmarksHistory{}
	h.SaveSnapshot(map[int]Extmark{}, 1)
	h.PushRedo(ExtmarksSnapshot{})
	h.Clear()
	if h.CanUndo() || h.CanRedo() {
		t.Error("both stacks should be empty after clear")
	}
}

func TestCloneExtmarks_DeepCopiesColors(t *testing.T) {
	fg := style.ColorFromHex("#FF0000")
	bg := style.ColorFromHex("#00FF00")
	src := map[int]Extmark{
		1: {ID: 1, Start: 0, End: 5, Fg: &fg, Bg: &bg},
	}

	dst := cloneExtmarks(src)

	// Pointers must differ (independent allocations).
	if dst[1].Fg == src[1].Fg {
		t.Error("cloneExtmarks should allocate a new Fg pointer")
	}
	if dst[1].Bg == src[1].Bg {
		t.Error("cloneExtmarks should allocate a new Bg pointer")
	}

	// Values must be equal.
	if *dst[1].Fg != *src[1].Fg {
		t.Error("cloned Fg value should match original")
	}
	if *dst[1].Bg != *src[1].Bg {
		t.Error("cloned Bg value should match original")
	}

	// Mutating the source must not affect the clone.
	src[1].Fg.R = 0
	if dst[1].Fg.R == 0 {
		t.Error("mutating source Fg should not affect clone")
	}
}

func TestExtmarks_UndoPreservesColors(t *testing.T) {
	eb, _, ec := setupExtmarks("Hello")
	defer ec.Destroy()

	fg := style.ColorFromHex("#FF0000")
	ec.Create(ExtmarkOptions{Start: 0, End: 5, Fg: &fg})

	// Edit to push an undo entry, then undo.
	eb.InsertChar('X')
	eb.Undo()

	em := ec.Get(1)
	if em == nil {
		t.Fatal("extmark should exist after undo")
	}
	if em.Fg == nil {
		t.Fatal("Fg should not be nil after undo")
	}

	// The restored color must match the original value but be an
	// independent pointer (not aliased with the local `fg` variable).
	if *em.Fg != fg {
		t.Errorf("Fg value mismatch after undo: got %v, want %v", *em.Fg, fg)
	}
	if em.Fg == &fg {
		t.Error("restored Fg should be an independent pointer, not the original")
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestExtmarks_InsertAtExactStart(t *testing.T) {
	eb, _, ec := setupExtmarks("Hello")
	defer ec.Destroy()

	id := ec.Create(ExtmarkOptions{Start: 0, End: 5})

	eb.SetCursor(0, 0)
	eb.InsertChar('X')

	em := ec.Get(id)
	// Insertion at the exact start: the extmark starts at insertOffset,
	// so start should shift
	if em.Start != 1 || em.End != 6 {
		t.Errorf("expected [1,6), got [%d,%d)", em.Start, em.End)
	}
}

func TestExtmarks_SetCursorByOffsetSkipsVirtual(t *testing.T) {
	eb, ev, ec := setupExtmarks("ABCDE")
	defer ec.Destroy()

	ec.Create(ExtmarkOptions{Start: 1, End: 4, Virtual: true})

	// Position at offset 0, then set offset to 2 (inside virtual)
	eb.SetCursor(0, 0)
	eb.SetCursorByOffset(2)

	vc := ev.VisualCursor()
	// Should have skipped to end of virtual range (4)
	if vc.Offset != 4 {
		t.Errorf("expected offset 4 (skipped virtual), got %d", vc.Offset)
	}
}

func TestExtmarks_SetCursorByOffsetSkipsVirtualBackward(t *testing.T) {
	eb, ev, ec := setupExtmarks("ABCDE")
	defer ec.Destroy()

	ec.Create(ExtmarkOptions{Start: 1, End: 4, Virtual: true})

	// Position at offset 4, then set offset to 2 (inside virtual, going backward)
	eb.SetCursor(0, 4)
	eb.SetCursorByOffset(2)

	vc := ev.VisualCursor()
	// Should have skipped to start of virtual range (1), not start-1 (0)
	if vc.Offset != 1 {
		t.Errorf("expected offset 1 (skipped virtual backward), got %d", vc.Offset)
	}
}

// ---------------------------------------------------------------------------
// DeleteLine with non-ASCII / multi-byte characters
// ---------------------------------------------------------------------------

func TestExtmarks_DeleteLine_MultiByteChars(t *testing.T) {
	// "café" uses multi-byte UTF-8 (é = 2 bytes) but display-width 4.
	// Line layout in offset space:
	//   Line 0: "café"  offsets [0,4)   width 4
	//   newline          offset 4       width 1
	//   Line 1: "日本語" offsets [5,11)  width 6 (each CJK char = 2)
	//   newline          offset 11      width 1
	//   Line 2: "end"   offsets [12,15) width 3
	eb, _, ec := setupExtmarks("café\n日本語\nend")
	defer ec.Destroy()

	// Place extmark on "end" — offsets [12,15)
	id := ec.Create(ExtmarkOptions{Start: 12, End: 15})

	// Delete line 1 ("日本語") — removes offsets [5,12): 6 for content + 1 for newline = 7
	eb.SetCursor(1, 0)
	eb.DeleteLine()

	// "end" extmark should shift from [12,15) to [5,8)
	em := ec.Get(id)
	if em == nil {
		t.Fatal("expected extmark to survive deletion of different line")
	}
	if em.Start != 5 || em.End != 8 {
		t.Errorf("expected extmark at [5,8) after deleting CJK line, got [%d,%d)", em.Start, em.End)
	}
}

func TestExtmarks_DeleteLine_LastLine_MultiByteChars(t *testing.T) {
	// Deleting the last line when there are multiple lines removes the
	// preceding newline + the line content.
	eb, _, ec := setupExtmarks("café\n日本語")
	defer ec.Destroy()

	// Place extmark on "café" — offsets [0,4)
	id := ec.Create(ExtmarkOptions{Start: 0, End: 4})

	// Delete line 1 ("日本語", the last line) — removes preceding \n + content = 1 + 6 = 7
	eb.SetCursor(1, 0)
	eb.DeleteLine()

	// "café" extmark should be unchanged at [0,4)
	em := ec.Get(id)
	if em == nil {
		t.Fatal("expected extmark to survive deletion of different line")
	}
	if em.Start != 0 || em.End != 4 {
		t.Errorf("expected extmark at [0,4), got [%d,%d)", em.Start, em.End)
	}
}

func TestExtmarks_DeleteLine_OnlyLine_MultiByteChars(t *testing.T) {
	// Single line: content is cleared but the line remains (empty).
	eb, _, ec := setupExtmarks("日本語")
	defer ec.Destroy()

	// Extmark on the full content — offsets [0,6)
	id := ec.Create(ExtmarkOptions{Start: 0, End: 6})

	eb.SetCursor(0, 0)
	eb.DeleteLine()

	// Extmark was entirely within the deleted content — should be removed
	em := ec.Get(id)
	if em != nil {
		t.Error("expected extmark to be removed when only line is cleared")
	}
}

func TestExtmarks_DisplayWidthOfText(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"hello", 5},
		{"hello\nworld", 11}, // 5 + 1 + 5
		{"\n", 1},
		{"", 0},
		{"a\nb\nc", 5}, // 1+1+1+1+1
	}
	for _, tt := range tests {
		got := DisplayWidthOfText(tt.input)
		if got != tt.expected {
			t.Errorf("DisplayWidthOfText(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}
