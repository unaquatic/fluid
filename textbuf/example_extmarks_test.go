package textbuf_test

import (
	"fmt"
	"sort"

	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/textbuf"
)

// newExtmarksSetup is a helper that creates an EditBuffer, EditSurface, and
// ExtmarksController wired together with the given initial text.
func newExtmarksSetup(text string) (*textbuf.EditBuffer, *textbuf.EditSurface, *textbuf.ExtmarksController) {
	tb := textbuf.NewTextBuffer()
	eb := textbuf.NewEditBuffer(tb)
	ev := textbuf.NewEditSurface(eb, 80, 24)
	eb.SetText(text)
	ec := textbuf.NewExtmarksController(eb, ev)
	return eb, ev, ec
}

// ExampleExtmarksController_createAndGet demonstrates creating extmarks and
// retrieving them by ID. Extmarks track arbitrary ranges in the buffer.
func ExampleExtmarksController_createAndGet() {
	_, _, ec := newExtmarksSetup("Hello World")
	defer ec.Destroy()

	// Create an extmark spanning "Hello" [0,5)
	id := ec.Create(textbuf.ExtmarkOptions{Start: 0, End: 5})
	em := ec.Get(id)
	fmt.Printf("extmark %d: [%d,%d)\n", em.ID, em.Start, em.End)

	// Create another spanning "World" [6,11)
	id2 := ec.Create(textbuf.ExtmarkOptions{Start: 6, End: 11})
	em2 := ec.Get(id2)
	fmt.Printf("extmark %d: [%d,%d)\n", em2.ID, em2.Start, em2.End)

	fmt.Printf("total: %d\n", len(ec.All()))

	// Output:
	// extmark 1: [0,5)
	// extmark 2: [6,11)
	// total: 2
}

// ExampleExtmarksController_virtual demonstrates virtual extmarks whose
// ranges cause the cursor to skip over them during navigation.
func ExampleExtmarksController_virtual() {
	eb, ev, ec := newExtmarksSetup("abc[GHOST]xyz")
	defer ec.Destroy()

	// Mark "[GHOST]" at offsets [3,10) as virtual
	ec.Create(textbuf.ExtmarkOptions{
		Start:   3,
		End:     10,
		Virtual: true,
	})

	// Place cursor just before the virtual range (offset 2 = 'c')
	eb.SetCursorByOffset(2)
	fmt.Printf("cursor start:       offset %d\n", ev.VisualCursor().Offset)

	// First right: lands at the boundary (offset 3, right before '[')
	eb.MoveCursorRight()
	fmt.Printf("cursor at boundary: offset %d\n", ev.VisualCursor().Offset)

	// Second right: from boundary, skips the entire virtual range
	eb.MoveCursorRight()
	fmt.Printf("cursor after skip:  offset %d\n", ev.VisualCursor().Offset)

	// Moving left: skips back to the start boundary
	eb.MoveCursorLeft()
	fmt.Printf("cursor after left:  offset %d\n", ev.VisualCursor().Offset)

	// Output:
	// cursor start:       offset 2
	// cursor at boundary: offset 3
	// cursor after skip:  offset 10
	// cursor after left:  offset 3
}

// ExampleExtmarksController_insertShiftsExtmarks shows that when text is
// inserted before an extmark, the extmark's positions shift accordingly.
func ExampleExtmarksController_insertShiftsExtmarks() {
	eb, ev, ec := newExtmarksSetup("Hello World")
	defer ec.Destroy()

	// Mark "World" at [6,11)
	id := ec.Create(textbuf.ExtmarkOptions{Start: 6, End: 11})

	// Move cursor to the beginning and insert ">> "
	eb.SetCursorByOffset(0)
	eb.InsertText(">> ")

	// The extmark shifts right by 3 characters
	em := ec.Get(id)
	fmt.Printf("after insert: [%d,%d)\n", em.Start, em.End)
	fmt.Printf("text: %q\n", eb.Text())
	_ = ev // used for cursor tracking

	// Output:
	// after insert: [9,14)
	// text: ">> Hello World"
}

// ExampleExtmarksController_deleteAdjusts shows extmarks adjusting when
// text is deleted — shifting, shrinking, or being removed as appropriate.
func ExampleExtmarksController_deleteAdjusts() {
	eb, _, ec := newExtmarksSetup("Hello World")
	defer ec.Destroy()

	// Mark "World" at [6,11)
	id := ec.Create(textbuf.ExtmarkOptions{Start: 6, End: 11})

	// Delete "Hello " (6 chars from offset 0)
	pos0 := eb.OffsetToPosition(0)
	pos6 := eb.OffsetToPosition(6)
	eb.DeleteRange(pos0.Row, pos0.Col, pos6.Row, pos6.Col)

	em := ec.Get(id)
	fmt.Printf("after delete: [%d,%d)\n", em.Start, em.End)
	fmt.Printf("text: %q\n", eb.Text())

	// Output:
	// after delete: [0,5)
	// text: "World"
}

// ExampleExtmarksController_typeRegistry demonstrates the type registry
// for categorising extmarks by name.
func ExampleExtmarksController_typeRegistry() {
	_, _, ec := newExtmarksSetup("Some code here")
	defer ec.Destroy()

	// Register named types
	diagID := ec.RegisterType("diagnostic")
	compID := ec.RegisterType("completion")

	// Creating extmarks with type IDs
	ec.Create(textbuf.ExtmarkOptions{Start: 0, End: 4, TypeID: diagID})
	ec.Create(textbuf.ExtmarkOptions{Start: 5, End: 9, TypeID: compID})
	ec.Create(textbuf.ExtmarkOptions{Start: 10, End: 14, TypeID: diagID})

	// Query by type
	diags := ec.AllForTypeID(diagID)
	fmt.Printf("diagnostics: %d\n", len(diags))
	fmt.Printf("completions: %d\n", len(ec.AllForTypeID(compID)))

	// Look up type name
	fmt.Printf("type %d = %q\n", diagID, ec.TypeName(diagID))

	// Output:
	// diagnostics: 2
	// completions: 1
	// type 1 = "diagnostic"
}

// ExampleExtmarksController_metadata demonstrates storing and retrieving
// arbitrary metadata on extmarks, separate from the extmark's Data field.
func ExampleExtmarksController_metadata() {
	_, _, ec := newExtmarksSetup("function main()")
	defer ec.Destroy()

	id := ec.Create(textbuf.ExtmarkOptions{
		Start:    0,
		End:      8,
		Data:     "function-keyword", // inline data
		Metadata: map[string]string{"scope": "global", "lang": "go"},
	})

	em := ec.Get(id)
	fmt.Printf("data: %v\n", em.Data)

	meta := ec.Metadata(id).(map[string]string)
	fmt.Printf("scope: %s\n", meta["scope"])

	// Output:
	// data: function-keyword
	// scope: global
}

// ExampleExtmarksController_highlighting shows that extmarks with Fg/Bg
// colors automatically create TextBuffer highlights.
func ExampleExtmarksController_highlighting() {
	eb, _, ec := newExtmarksSetup("error on this line")
	defer ec.Destroy()

	red := style.ColorRed
	ec.Create(textbuf.ExtmarkOptions{
		Start: 0,
		End:   5,
		Fg:    &red,
	})

	tb := eb.TextBuffer()
	fmt.Printf("highlights: %d\n", tb.HighlightCount())

	// Deleting the extmark removes the highlight
	ec.Clear()
	fmt.Printf("after clear: %d\n", tb.HighlightCount())

	// Output:
	// highlights: 1
	// after clear: 0
}

// ExampleExtmarksController_undoRedo demonstrates that extmark positions
// are restored when the user undoes and redoes edits.
func ExampleExtmarksController_undoRedo() {
	eb, _, ec := newExtmarksSetup("Hello World")
	defer ec.Destroy()

	id := ec.Create(textbuf.ExtmarkOptions{Start: 6, End: 11})

	// Insert text, which shifts the extmark
	eb.SetCursorByOffset(0)
	eb.InsertText(">> ")
	em := ec.Get(id)
	fmt.Printf("after insert: [%d,%d)\n", em.Start, em.End)

	// Undo reverts both text and extmark positions
	eb.Undo()
	em = ec.Get(id)
	fmt.Printf("after undo:   [%d,%d)\n", em.Start, em.End)

	// Redo re-applies
	eb.Redo()
	em = ec.Get(id)
	fmt.Printf("after redo:   [%d,%d)\n", em.Start, em.End)

	// Output:
	// after insert: [9,14)
	// after undo:   [6,11)
	// after redo:   [9,14)
}

// ExampleExtmarksController_getAtOffset shows querying which extmarks
// contain a specific buffer offset.
func ExampleExtmarksController_getAtOffset() {
	_, _, ec := newExtmarksSetup("aaaBBBccc")
	defer ec.Destroy()

	ec.Create(textbuf.ExtmarkOptions{Start: 0, End: 3}) // "aaa"
	ec.Create(textbuf.ExtmarkOptions{Start: 3, End: 6}) // "BBB"
	ec.Create(textbuf.ExtmarkOptions{Start: 6, End: 9}) // "ccc"

	// Offset 4 is inside "BBB" [3,6)
	at4 := ec.AtOffset(4)
	fmt.Printf("at offset 4: %d extmark(s)\n", len(at4))
	if len(at4) > 0 {
		fmt.Printf("  range: [%d,%d)\n", at4[0].Start, at4[0].End)
	}

	// Offset 6 is inside "ccc" [6,9)
	at6 := ec.AtOffset(6)
	fmt.Printf("at offset 6: %d extmark(s)\n", len(at6))

	// Output:
	// at offset 4: 1 extmark(s)
	//   range: [3,6)
	// at offset 6: 1 extmark(s)
}

// ExampleExtmarksController_backspaceDeletesVirtual demonstrates that
// pressing backspace at the end of a virtual extmark deletes the entire
// virtual range atomically (useful for ghost text / inline completions).
func ExampleExtmarksController_backspaceDeletesVirtual() {
	eb, ev, ec := newExtmarksSetup("ab[GHOST]cd")
	defer ec.Destroy()

	id := ec.Create(textbuf.ExtmarkOptions{
		Start:   2,
		End:     9,
		Virtual: true,
	})

	// Place cursor at the end of the virtual range
	eb.SetCursorByOffset(9)
	fmt.Printf("cursor: offset %d\n", ev.VisualCursor().Offset)

	// Backspace deletes the entire "[GHOST]" virtual extmark
	eb.DeleteCharBackward()

	em := ec.Get(id)
	fmt.Printf("extmark exists: %v\n", em != nil)
	fmt.Printf("text: %q\n", eb.Text())

	// Output:
	// cursor: offset 9
	// extmark exists: false
	// text: "abcd"
}

// ExampleExtmarksController_destroy shows that destroying the controller
// removes all overrides, returning the EditBuffer to normal behavior.
func ExampleExtmarksController_destroy() {
	eb, _, ec := newExtmarksSetup("Hello")
	defer ec.Destroy()

	ec.Create(textbuf.ExtmarkOptions{Start: 0, End: 5, Virtual: true})

	virtBefore := len(ec.Virtual())
	fmt.Printf("virtual before destroy: %d\n", virtBefore)

	ec.Destroy()

	// After destroy, Get returns nil and cursor moves normally
	fmt.Printf("virtual after destroy: %v\n", ec.Virtual())

	// EditBuffer still works normally
	eb.SetCursorByOffset(0)
	eb.InsertChar('!')
	fmt.Printf("text: %q\n", eb.Text())

	// Output:
	// virtual before destroy: 1
	// virtual after destroy: []
	// text: "!Hello"
}

// ExampleExtmarksController_multipleInsideRange demonstrates that inserting
// text inside an extmark expands the extmark's end position.
func ExampleExtmarksController_multipleInsideRange() {
	eb, _, ec := newExtmarksSetup("[]")
	defer ec.Destroy()

	// Mark the brackets [0,2)
	id := ec.Create(textbuf.ExtmarkOptions{Start: 0, End: 2})

	// Insert inside the range
	eb.SetCursorByOffset(1)
	eb.InsertText("abc")

	em := ec.Get(id)
	fmt.Printf("extmark: [%d,%d)\n", em.Start, em.End)
	fmt.Printf("text: %q\n", eb.Text())

	// Output:
	// extmark: [0,5)
	// text: "[abc]"
}

// ExampleExtmarksController_deletionRemovesContained shows that deleting
// a range that fully contains an extmark removes that extmark entirely.
func ExampleExtmarksController_deletionRemovesContained() {
	eb, _, ec := newExtmarksSetup("aaaBBBccc")
	defer ec.Destroy()

	ec.Create(textbuf.ExtmarkOptions{Start: 0, End: 3})        // "aaa"
	idB := ec.Create(textbuf.ExtmarkOptions{Start: 3, End: 6}) // "BBB"
	ec.Create(textbuf.ExtmarkOptions{Start: 6, End: 9})        // "ccc"

	// Delete "BBB" — extmark for "BBB" is entirely contained, so it's removed
	posStart := eb.OffsetToPosition(3)
	posEnd := eb.OffsetToPosition(6)
	eb.DeleteRange(posStart.Row, posStart.Col, posEnd.Row, posEnd.Col)

	all := ec.All()
	sort.Slice(all, func(i, j int) bool { return all[i].Start < all[j].Start })

	fmt.Printf("remaining: %d extmarks\n", len(all))
	for _, em := range all {
		fmt.Printf("  [%d,%d)\n", em.Start, em.End)
	}
	fmt.Printf("deleted 'BBB' extmark exists: %v\n", ec.Get(idB) != nil)

	// Output:
	// remaining: 2 extmarks
	//   [0,3)
	//   [3,6)
	// deleted 'BBB' extmark exists: false
}
