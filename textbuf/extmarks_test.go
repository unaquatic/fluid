package textbuf

import (
	"testing"

	"github.com/unaquatic/fluid/style"
)

func setupExtmarks(text string) (*EditBuffer, *EditSurface, *ExtmarksController) {
	tb := NewTextBuffer()
	eb := NewEditBuffer(tb)
	ev := NewEditSurface(eb, 40, 10)
	eb.SetText(text)
	ec := NewExtmarksController(eb, ev)
	return eb, ev, ec
}

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
	at5 := ec.AtOffset(5) // End offsets are exclusive.
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

func TestExtmarks_TypeRegistry(t *testing.T) {
	_, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	id1 := ec.RegisterType("ghost-text")
	id2 := ec.RegisterType("completion")
	id3 := ec.RegisterType("ghost-text") // Re-register the same name.

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
	ec.Create(ExtmarkOptions{Start: 0, End: 3, TypeID: 0}) // Default type.

	byType := ec.AllForTypeID(tid)
	if len(byType) != 2 {
		t.Errorf("expected 2 for type %d, got %d", tid, len(byType))
	}
}

func TestExtmarks_InsertShiftsAfter(t *testing.T) {
	eb, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	// Track "World" at [6,11).
	id := ec.Create(ExtmarkOptions{Start: 6, End: 11})

	// Insert one column before the extmark.
	eb.SetCursor(0, 0)
	eb.InsertChar('X')

	em := ec.Get(id)
	if em == nil {
		t.Fatal("extmark should still exist")
	}
	// The one-column insertion shifts both endpoints by one.
	if em.Start != 7 || em.End != 12 {
		t.Errorf("expected [7,12), got [%d,%d)", em.Start, em.End)
	}
}

func TestExtmarks_InsertInside(t *testing.T) {
	eb, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	// Track "Hello" at [0,5).
	id := ec.Create(ExtmarkOptions{Start: 0, End: 5})

	// Insert inside the tracked range at column three.
	eb.SetCursor(0, 3)
	eb.InsertChar('X')

	em := ec.Get(id)
	if em == nil {
		t.Fatal("extmark should still exist")
	}
	// An internal insertion extends only the end.
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

	// Insert a newline at global offset five.
	eb.SetCursor(0, 5)
	eb.NewLine()

	em := ec.Get(id)
	// The newline shifts later endpoints by one offset.
	if em.Start != 7 || em.End != 12 {
		t.Errorf("expected [7,12), got [%d,%d)", em.Start, em.End)
	}
}

func TestExtmarks_DeleteShiftsAfter(t *testing.T) {
	eb, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	// Track "World" at [6,11).
	id := ec.Create(ExtmarkOptions{Start: 6, End: 11})

	// Delete one column before the tracked range.
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

	// Track "lo" at [3,5).
	id := ec.Create(ExtmarkOptions{Start: 3, End: 5})

	// Delete the range containing the entire extmark.
	eb.DeleteRange(0, 0, 0, 5)

	// A fully deleted extmark is removed.
	em := ec.Get(id)
	if em != nil {
		t.Error("expected extmark to be removed when fully deleted")
	}
}

func TestExtmarks_DeleteShrinks(t *testing.T) {
	eb, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	// Track the complete text at [0,11).
	id := ec.Create(ExtmarkOptions{Start: 0, End: 11})

	// Delete the internal range "lo W" at [3,7).
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

	// Track "Line2" at [6,11), after "Line1\n".
	id := ec.Create(ExtmarkOptions{Start: 6, End: 11})

	// Delete the tracked logical line.
	eb.SetCursor(1, 0)
	eb.DeleteLine()

	em := ec.Get(id)
	// Deleting "Line2\n" removes the complete extmark.
	if em != nil {
		t.Error("expected extmark to be removed with deleted line")
	}
}

func TestExtmarks_CursorSkipsVirtualLeft(t *testing.T) {
	eb, ev, ec := setupExtmarks("ABCDE")
	defer ec.Destroy()

	// Mark "BCD" at [1,4) as virtual.
	ec.Create(ExtmarkOptions{Start: 1, End: 4, Virtual: true})

	// Start at the range's exclusive end.
	eb.SetCursor(0, 4)
	vc := ev.VisualCursor()
	if vc.Offset != 4 {
		t.Fatalf("setup: expected offset 4, got %d", vc.Offset)
	}

	// Moving left crosses the complete range and stops at its start.
	eb.MoveCursorLeft()
	vc = ev.VisualCursor()
	if vc.Offset != 1 {
		t.Errorf("expected offset 1 after skipping virtual left, got %d", vc.Offset)
	}
}

func TestExtmarks_CursorSkipsVirtualRight(t *testing.T) {
	eb, ev, ec := setupExtmarks("ABCDE")
	defer ec.Destroy()

	// Mark "BCD" at [1,4) as virtual.
	ec.Create(ExtmarkOptions{Start: 1, End: 4, Virtual: true})

	// Start before the virtual range.
	eb.SetCursor(0, 0)

	// The first move reaches the range start.
	eb.MoveCursorRight()
	vc := ev.VisualCursor()
	if vc.Offset != 1 {
		t.Errorf("expected offset 1 (boundary) after first right, got %d", vc.Offset)
	}

	// The second move skips to the range end.
	eb.MoveCursorRight()
	vc = ev.VisualCursor()
	if vc.Offset != 4 {
		t.Errorf("expected offset 4 after skipping virtual right, got %d", vc.Offset)
	}
}

func TestExtmarks_CursorNoSkipNonVirtual(t *testing.T) {
	eb, ev, ec := setupExtmarks("ABCDE")
	defer ec.Destroy()

	// A nonvirtual extmark does not affect cursor movement.
	ec.Create(ExtmarkOptions{Start: 1, End: 4, Virtual: false})

	eb.SetCursor(0, 0)
	eb.MoveCursorRight()
	vc := ev.VisualCursor()
	// One right movement advances normally to offset one.
	if vc.Offset != 1 {
		t.Errorf("expected offset 1, got %d", vc.Offset)
	}
}

func TestExtmarks_BackspaceDeletesVirtual(t *testing.T) {
	eb, ev, ec := setupExtmarks("ABCDE")
	defer ec.Destroy()

	// Mark "BCD" at [1,4) as virtual.
	id := ec.Create(ExtmarkOptions{Start: 1, End: 4, Virtual: true})

	// Place the cursor at the range end.
	eb.SetCursor(0, 4)

	// Backspace at the end removes the complete virtual range.
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

	// Mark "BCD" at [1,4) as virtual.
	id := ec.Create(ExtmarkOptions{Start: 1, End: 4, Virtual: true})

	// Place the cursor at the range start.
	eb.SetCursor(0, 1)

	// Forward delete at the start removes the complete virtual range.
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

func TestExtmarks_UndoRestoresState(t *testing.T) {
	eb, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	id := ec.Create(ExtmarkOptions{Start: 6, End: 11})

	// Insertion shifts the tracked range.
	eb.SetCursor(0, 0)
	eb.InsertChar('X')

	em := ec.Get(id)
	if em.Start != 7 || em.End != 12 {
		t.Fatalf("after insert: expected [7,12), got [%d,%d)", em.Start, em.End)
	}

	// Undo restores its original endpoints.
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
	// Redo reapplies the endpoint shift.
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

	// Undo without edit history reports false.
	ok := eb.Undo()
	if ok {
		t.Error("undo should return false with empty history")
	}
}

func TestExtmarks_UndoHighlightsMatchRestoredText(t *testing.T) {
	// Undo must restore text before remapping global extmarks to line highlights.
	eb, _, ec := setupExtmarks("AB\nCD")
	defer ec.Destroy()

	fg := style.ColorFromHex("#FF0000")

	// Track "CD" at [3,5), after "AB\n".
	ec.Create(ExtmarkOptions{Start: 3, End: 5, Fg: fg})

	tb := ec.editBuffer.TextBuffer()

	// The initial line-local highlight spans [0,2).
	hls := tb.LineHighlights(1)
	if len(hls) != 1 || hls[0].Start != 0 || hls[0].End != 2 {
		t.Fatalf("before edit: expected line 1 highlight [0,2), got %v", hls)
	}

	// Insert X before the first line.
	eb.SetCursor(0, 0)
	eb.InsertChar('X')

	// Global endpoints shift while line-local endpoints remain [0,2).
	hls = tb.LineHighlights(1)
	if len(hls) != 1 || hls[0].Start != 0 || hls[0].End != 2 {
		t.Fatalf("after insert: expected line 1 highlight [0,2), got %v", hls)
	}

	// Undo restores both text and the original line-local highlight.
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
	// Redo also restores text before remapping highlights.
	eb, _, ec := setupExtmarks("AB\nCD")
	defer ec.Destroy()

	fg := style.ColorFromHex("#FF0000")
	ec.Create(ExtmarkOptions{Start: 3, End: 5, Fg: fg})

	tb := ec.editBuffer.TextBuffer()

	// Reapply one insertion through undo and redo.
	eb.SetCursor(0, 0)
	eb.InsertChar('X')
	eb.Undo()
	eb.Redo()

	// The restored global range [4,6) still maps to line-local [0,2).
	hls := tb.LineHighlights(1)
	if len(hls) != 1 {
		t.Fatalf("after redo: expected 1 highlight on line 1, got %d", len(hls))
	}
	if hls[0].Start != 0 || hls[0].End != 2 {
		t.Errorf("after redo: expected line 1 highlight [0,2), got [%d,%d)", hls[0].Start, hls[0].End)
	}
}

func TestExtmarks_BackspaceAtStartNoDesync(t *testing.T) {
	// A no-op backspace must not add an extmark snapshot without a matching text
	// snapshot.
	eb, _, ec := setupExtmarks("AB")
	defer ec.Destroy()

	id := ec.Create(ExtmarkOptions{Start: 0, End: 2})

	// Create one real text and extmark history entry.
	eb.SetCursor(0, 2)
	eb.InsertChar('X') // Text becomes "ABX"; extmark remains [0,2).

	// Backspace at offset zero is a no-op for both histories.
	eb.SetCursor(0, 0)
	eb.DeleteCharBackward()

	// Undo now restores the matching insertion snapshot.
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
	// A no-op forward delete also leaves both histories synchronized.
	eb, _, ec := setupExtmarks("AB")
	defer ec.Destroy()

	id := ec.Create(ExtmarkOptions{Start: 0, End: 2})

	// Create one real text and extmark history entry.
	eb.SetCursor(0, 0)
	eb.InsertChar('X') // Text becomes "XAB"; extmark shifts to [1,3).

	// Forward delete at the buffer end is a no-op for both histories.
	eb.SetCursor(0, 3)
	eb.DeleteChar()

	// Undo restores the matching insertion snapshot.
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
	// Undo rebuilds the secondary type index from restored extmarks.
	eb, _, ec := setupExtmarks("Hello")
	defer ec.Destroy()

	typeID := ec.RegisterType("test")
	ec.Create(ExtmarkOptions{Start: 0, End: 5, TypeID: typeID})

	// Confirm the type index before editing.
	if len(ec.AllForTypeID(typeID)) != 1 {
		t.Fatal("expected 1 extmark for typeID before edit")
	}

	// Shift the range and restore it through undo.
	eb.SetCursor(0, 0)
	eb.InsertChar('X')
	eb.Undo()

	// The rebuilt index returns the restored range.
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

func TestExtmarks_HighlightsApplied(t *testing.T) {
	_, _, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	fg := style.ColorRed
	ec.Create(ExtmarkOptions{
		Start: 0, End: 5,
		Fg: fg,
	})

	// A styled extmark creates a TextBuffer highlight.
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
		Fg: fg,
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

func TestExtmarks_DeleteSelectedText(t *testing.T) {
	_, ev, ec := setupExtmarks("Hello World")
	defer ec.Destroy()

	// Track "World" at [6,11).
	id := ec.Create(ExtmarkOptions{Start: 6, End: 11})

	// Delete the separate selection "Hello" at [0,5).
	bg := style.ColorBlue
	ev.SetSelection(&TextSelection{Start: 0, End: 5, BgColor: bg})
	ev.DeleteSelectedText()

	em := ec.Get(id)
	if em == nil {
		t.Fatal("extmark should survive deletion of other text")
	}
	// Removing five columns shifts "World" to [1,6).
	if em.Start != 1 || em.End != 6 {
		t.Errorf("expected [1,6), got [%d,%d)", em.Start, em.End)
	}
}

func TestExtmarks_Destroy(t *testing.T) {
	eb, _, ec := setupExtmarks("Hello World")

	ec.Create(ExtmarkOptions{Start: 0, End: 5})
	ec.Destroy()

	// After destruction, EditBuffer uses its default implementation.
	eb.SetCursor(0, 5)
	eb.InsertChar('!')
	if eb.Text() != "Hello! World" {
		t.Errorf("expected normal behavior after destroy, got %q", eb.Text())
	}

	// A destroyed controller exposes no extmarks.
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
	ec.Destroy() // Repeated destruction is a safe no-op.
}

func TestExtmarks_MultiLineInsert(t *testing.T) {
	eb, _, ec := setupExtmarks("Line1\nLine2\nLine3")
	defer ec.Destroy()

	// Track "Line3" at offset 12 after two lines and newlines.
	id := ec.Create(ExtmarkOptions{Start: 12, End: 17})

	// Insert one column at the document start.
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

	// Track "Line3" at [12,17).
	id := ec.Create(ExtmarkOptions{Start: 12, End: 17})

	// Delete the six-offset prefix "Line1\n".
	eb.DeleteRange(0, 0, 1, 0)

	em := ec.Get(id)
	if em == nil {
		t.Fatal("extmark should survive")
	}
	if em.Start != 6 || em.End != 11 {
		t.Errorf("expected [6,11), got [%d,%d)", em.Start, em.End)
	}
}

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
	// Move the first snapshot into redo.
	snap := h.Undo()
	h.PushRedo(ExtmarksSnapshot{extmarks: cloneExtmarks(m2), nextID: 2})

	if !h.CanRedo() {
		t.Error("should be able to redo")
	}

	// Saving a new branch clears redo history.
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

func TestCloneExtmarks_CopiesColors(t *testing.T) {
	fg := style.ColorFromHex("#FF0000")
	bg := style.ColorFromHex("#00FF00")
	src := map[int]Extmark{
		1: {ID: 1, Start: 0, End: 5, Fg: fg, Bg: bg},
	}

	dst := cloneExtmarks(src)

	// Cloned colors preserve channel values.
	if dst[1].Fg != src[1].Fg {
		t.Error("cloned Fg value should match original")
	}
	if dst[1].Bg != src[1].Bg {
		t.Error("cloned Bg value should match original")
	}

	// Source mutations do not affect cloned colors.
	e := src[1]
	e.Fg.R = 0
	src[1] = e
	if dst[1].Fg.R == 0 {
		t.Error("mutating source Fg should not affect clone")
	}
}

func TestExtmarks_UndoPreservesColors(t *testing.T) {
	eb, _, ec := setupExtmarks("Hello")
	defer ec.Destroy()

	fg := style.ColorFromHex("#FF0000")
	ec.Create(ExtmarkOptions{Start: 0, End: 5, Fg: fg})

	// Create and restore one history entry.
	eb.InsertChar('X')
	eb.Undo()

	em := ec.Get(1)
	if em == nil {
		t.Fatal("extmark should exist after undo")
	}
	// Restored color values match.
	if em.Fg != fg {
		t.Errorf("Fg value mismatch after undo: got %v, want %v", em.Fg, fg)
	}
}

func TestExtmarks_InsertAtExactStart(t *testing.T) {
	eb, _, ec := setupExtmarks("Hello")
	defer ec.Destroy()

	id := ec.Create(ExtmarkOptions{Start: 0, End: 5})

	eb.SetCursor(0, 0)
	eb.InsertChar('X')

	em := ec.Get(id)
	// Insertion at the exact start shifts both endpoints.
	if em.Start != 1 || em.End != 6 {
		t.Errorf("expected [1,6), got [%d,%d)", em.Start, em.End)
	}
}

func TestExtmarks_SetCursorByOffsetSkipsVirtual(t *testing.T) {
	eb, ev, ec := setupExtmarks("ABCDE")
	defer ec.Destroy()

	ec.Create(ExtmarkOptions{Start: 1, End: 4, Virtual: true})

	// Move forward from zero into the virtual range.
	eb.SetCursor(0, 0)
	eb.SetCursorByOffset(2)

	vc := ev.VisualCursor()
	// Forward movement clamps to the range end.
	if vc.Offset != 4 {
		t.Errorf("expected offset 4 (skipped virtual), got %d", vc.Offset)
	}
}

func TestExtmarks_SetCursorByOffsetSkipsVirtualBackward(t *testing.T) {
	eb, ev, ec := setupExtmarks("ABCDE")
	defer ec.Destroy()

	ec.Create(ExtmarkOptions{Start: 1, End: 4, Virtual: true})

	// Move backward from four into the virtual range.
	eb.SetCursor(0, 4)
	eb.SetCursorByOffset(2)

	vc := ev.VisualCursor()
	// Backward movement clamps to the range start.
	if vc.Offset != 1 {
		t.Errorf("expected offset 1 (skipped virtual backward), got %d", vc.Offset)
	}
}

func TestExtmarks_DeleteLine_MultiByteChars(t *testing.T) {
	// Display-width offsets differ from UTF-8 bytes: "café" spans [0,4),
	// "日本語" spans [5,11), and "end" spans [12,15).
	eb, _, ec := setupExtmarks("café\n日本語\nend")
	defer ec.Destroy()

	// Track "end" at [12,15).
	id := ec.Create(ExtmarkOptions{Start: 12, End: 15})

	// Deleting "日本語\n" removes seven display-width offsets at [5,12).
	eb.SetCursor(1, 0)
	eb.DeleteLine()

	// The later range shifts backward to [5,8).
	em := ec.Get(id)
	if em == nil {
		t.Fatal("expected extmark to survive deletion of different line")
	}
	if em.Start != 5 || em.End != 8 {
		t.Errorf("expected extmark at [5,8) after deleting CJK line, got [%d,%d)", em.Start, em.End)
	}
}

func TestExtmarks_DeleteLine_LastLine_MultiByteChars(t *testing.T) {
	// Deleting a final logical line also removes its preceding newline.
	eb, _, ec := setupExtmarks("café\n日本語")
	defer ec.Destroy()

	// Track "café" at [0,4).
	id := ec.Create(ExtmarkOptions{Start: 0, End: 4})

	// Delete the final seven-offset newline and CJK line.
	eb.SetCursor(1, 0)
	eb.DeleteLine()

	// The preceding extmark remains unchanged.
	em := ec.Get(id)
	if em == nil {
		t.Fatal("expected extmark to survive deletion of different line")
	}
	if em.Start != 0 || em.End != 4 {
		t.Errorf("expected extmark at [0,4), got [%d,%d)", em.Start, em.End)
	}
}

func TestExtmarks_DeleteLine_OnlyLine_MultiByteChars(t *testing.T) {
	// Deleting the sole line clears it while preserving one empty line.
	eb, _, ec := setupExtmarks("日本語")
	defer ec.Destroy()

	// Track the complete six-column content.
	id := ec.Create(ExtmarkOptions{Start: 0, End: 6})

	eb.SetCursor(0, 0)
	eb.DeleteLine()

	// Clearing the tracked content removes its extmark.
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
		{"hello\nworld", 11}, // Two five-column lines and one newline.
		{"\n", 1},
		{"", 0},
		{"a\nb\nc", 5}, // Three columns and two newlines.
	}
	for _, tt := range tests {
		got := DisplayWidthOfText(tt.input)
		if got != tt.expected {
			t.Errorf("DisplayWidthOfText(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}
