package textbuf_test

import (
	"fmt"
	"sort"

	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/textbuf"
)

// newExtmarksSetup returns connected editing, surface, and extmark components.
func newExtmarksSetup(text string) (*textbuf.EditBuffer, *textbuf.EditSurface, *textbuf.ExtmarksController) {
	tb := textbuf.NewTextBuffer()
	eb := textbuf.NewEditBuffer(tb)
	ev := textbuf.NewEditSurface(eb, 80, 24)
	eb.SetText(text)
	ec := textbuf.NewExtmarksController(eb, ev)
	return eb, ev, ec
}

// ExampleExtmarksController_createAndGet demonstrates creation and ID lookup.
func ExampleExtmarksController_createAndGet() {
	_, _, ec := newExtmarksSetup("Hello World")
	defer ec.Destroy()

	// Track "Hello" at [0,5).
	id := ec.Create(textbuf.ExtmarkOptions{Start: 0, End: 5})
	em := ec.Get(id)
	fmt.Printf("extmark %d: [%d,%d)\n", em.ID, em.Start, em.End)

	// Track "World" at [6,11).
	id2 := ec.Create(textbuf.ExtmarkOptions{Start: 6, End: 11})
	em2 := ec.Get(id2)
	fmt.Printf("extmark %d: [%d,%d)\n", em2.ID, em2.Start, em2.End)

	fmt.Printf("total: %d\n", len(ec.All()))

	// Output:
	// extmark 1: [0,5)
	// extmark 2: [6,11)
	// total: 2
}

// ExampleExtmarksController_virtual demonstrates cursor movement across a
// virtual range.
func ExampleExtmarksController_virtual() {
	eb, ev, ec := newExtmarksSetup("abc[GHOST]xyz")
	defer ec.Destroy()

	// Mark "[GHOST]" at [3,10) as virtual.
	ec.Create(textbuf.ExtmarkOptions{
		Start:   3,
		End:     10,
		Virtual: true,
	})

	// Start one column before the virtual range.
	eb.SetCursorByOffset(2)
	fmt.Printf("cursor start:       offset %d\n", ev.VisualCursor().Offset)

	// The first right movement reaches the range start.
	eb.MoveCursorRight()
	fmt.Printf("cursor at boundary: offset %d\n", ev.VisualCursor().Offset)

	// The second right movement skips to the range end.
	eb.MoveCursorRight()
	fmt.Printf("cursor after skip:  offset %d\n", ev.VisualCursor().Offset)

	// Moving left skips back to the range start.
	eb.MoveCursorLeft()
	fmt.Printf("cursor after left:  offset %d\n", ev.VisualCursor().Offset)

	// Output:
	// cursor start:       offset 2
	// cursor at boundary: offset 3
	// cursor after skip:  offset 10
	// cursor after left:  offset 3
}

// ExampleExtmarksController_insertShiftsExtmarks demonstrates endpoint movement
// after insertion before a range.
func ExampleExtmarksController_insertShiftsExtmarks() {
	eb, ev, ec := newExtmarksSetup("Hello World")
	defer ec.Destroy()

	// Track "World" at [6,11).
	id := ec.Create(textbuf.ExtmarkOptions{Start: 6, End: 11})

	// Insert three columns at the document start.
	eb.SetCursorByOffset(0)
	eb.InsertText(">> ")

	// Both endpoints shift right by three.
	em := ec.Get(id)
	fmt.Printf("after insert: [%d,%d)\n", em.Start, em.End)
	fmt.Printf("text: %q\n", eb.Text())
	_ = ev // Keep the connected surface alive for cursor notifications.

	// Output:
	// after insert: [9,14)
	// text: ">> Hello World"
}

// ExampleExtmarksController_deleteAdjusts demonstrates endpoint movement after
// deletion before a range.
func ExampleExtmarksController_deleteAdjusts() {
	eb, _, ec := newExtmarksSetup("Hello World")
	defer ec.Destroy()

	// Track "World" at [6,11).
	id := ec.Create(textbuf.ExtmarkOptions{Start: 6, End: 11})

	// Delete the six-offset prefix "Hello ".
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

// ExampleExtmarksController_typeRegistry demonstrates named extmark categories.
func ExampleExtmarksController_typeRegistry() {
	_, _, ec := newExtmarksSetup("Some code here")
	defer ec.Destroy()

	// Register category names.
	diagID := ec.RegisterType("diagnostic")
	compID := ec.RegisterType("completion")

	// Assign categories while creating extmarks.
	ec.Create(textbuf.ExtmarkOptions{Start: 0, End: 4, TypeID: diagID})
	ec.Create(textbuf.ExtmarkOptions{Start: 5, End: 9, TypeID: compID})
	ec.Create(textbuf.ExtmarkOptions{Start: 10, End: 14, TypeID: diagID})

	// Query extmarks by category ID.
	diags := ec.AllForTypeID(diagID)
	fmt.Printf("diagnostics: %d\n", len(diags))
	fmt.Printf("completions: %d\n", len(ec.AllForTypeID(compID)))

	// Resolve a category ID back to its name.
	fmt.Printf("type %d = %q\n", diagID, ec.TypeName(diagID))

	// Output:
	// diagnostics: 2
	// completions: 1
	// type 1 = "diagnostic"
}

// ExampleExtmarksController_metadata demonstrates inline Data and separately
// indexed Metadata.
func ExampleExtmarksController_metadata() {
	_, _, ec := newExtmarksSetup("function main()")
	defer ec.Destroy()

	id := ec.Create(textbuf.ExtmarkOptions{
		Start:    0,
		End:      8,
		Data:     "function-keyword", // Data stored on the Extmark.
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

// ExampleExtmarksController_highlighting demonstrates highlights derived from
// styled extmarks.
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

	// Clearing extmarks removes their highlights.
	ec.Clear()
	fmt.Printf("after clear: %d\n", tb.HighlightCount())

	// Output:
	// highlights: 1
	// after clear: 0
}

// ExampleExtmarksController_undoRedo demonstrates synchronized text and extmark
// history.
func ExampleExtmarksController_undoRedo() {
	eb, _, ec := newExtmarksSetup("Hello World")
	defer ec.Destroy()

	id := ec.Create(textbuf.ExtmarkOptions{Start: 6, End: 11})

	// Insertion shifts the tracked range.
	eb.SetCursorByOffset(0)
	eb.InsertText(">> ")
	em := ec.Get(id)
	fmt.Printf("after insert: [%d,%d)\n", em.Start, em.End)

	// Undo restores both text and extmark positions.
	eb.Undo()
	em = ec.Get(id)
	fmt.Printf("after undo:   [%d,%d)\n", em.Start, em.End)

	// Redo reapplies both histories.
	eb.Redo()
	em = ec.Get(id)
	fmt.Printf("after redo:   [%d,%d)\n", em.Start, em.End)

	// Output:
	// after insert: [9,14)
	// after undo:   [6,11)
	// after redo:   [9,14)
}

// ExampleExtmarksController_getAtOffset demonstrates half-open range queries.
func ExampleExtmarksController_getAtOffset() {
	_, _, ec := newExtmarksSetup("aaaBBBccc")
	defer ec.Destroy()

	ec.Create(textbuf.ExtmarkOptions{Start: 0, End: 3}) // Track "aaa".
	ec.Create(textbuf.ExtmarkOptions{Start: 3, End: 6}) // Track "BBB".
	ec.Create(textbuf.ExtmarkOptions{Start: 6, End: 9}) // Track "ccc".

	// Offset four belongs to [3,6).
	at4 := ec.AtOffset(4)
	fmt.Printf("at offset 4: %d extmark(s)\n", len(at4))
	if len(at4) > 0 {
		fmt.Printf("  range: [%d,%d)\n", at4[0].Start, at4[0].End)
	}

	// Offset six belongs to [6,9), not the preceding range.
	at6 := ec.AtOffset(6)
	fmt.Printf("at offset 6: %d extmark(s)\n", len(at6))

	// Output:
	// at offset 4: 1 extmark(s)
	//   range: [3,6)
	// at offset 6: 1 extmark(s)
}

// ExampleExtmarksController_backspaceDeletesVirtual demonstrates atomic
// backspace deletion at a virtual range boundary.
func ExampleExtmarksController_backspaceDeletesVirtual() {
	eb, ev, ec := newExtmarksSetup("ab[GHOST]cd")
	defer ec.Destroy()

	id := ec.Create(textbuf.ExtmarkOptions{
		Start:   2,
		End:     9,
		Virtual: true,
	})

	// Place the cursor at the range end.
	eb.SetCursorByOffset(9)
	fmt.Printf("cursor: offset %d\n", ev.VisualCursor().Offset)

	// Backspace removes the complete virtual range.
	eb.DeleteCharBackward()

	em := ec.Get(id)
	fmt.Printf("extmark exists: %v\n", em != nil)
	fmt.Printf("text: %q\n", eb.Text())

	// Output:
	// cursor: offset 9
	// extmark exists: false
	// text: "abcd"
}

// ExampleExtmarksController_destroy demonstrates restoration of default editing
// after controller destruction.
func ExampleExtmarksController_destroy() {
	eb, _, ec := newExtmarksSetup("Hello")
	defer ec.Destroy()

	ec.Create(textbuf.ExtmarkOptions{Start: 0, End: 5, Virtual: true})

	virtBefore := len(ec.Virtual())
	fmt.Printf("virtual before destroy: %d\n", virtBefore)

	ec.Destroy()

	// A destroyed controller exposes no virtual ranges.
	fmt.Printf("virtual after destroy: %v\n", ec.Virtual())

	// The EditBuffer continues with its default methods.
	eb.SetCursorByOffset(0)
	eb.InsertChar('!')
	fmt.Printf("text: %q\n", eb.Text())

	// Output:
	// virtual before destroy: 1
	// virtual after destroy: []
	// text: "!Hello"
}

// ExampleExtmarksController_multipleInsideRange demonstrates insertion within a
// tracked range.
func ExampleExtmarksController_multipleInsideRange() {
	eb, _, ec := newExtmarksSetup("[]")
	defer ec.Destroy()

	// Track the brackets at [0,2).
	id := ec.Create(textbuf.ExtmarkOptions{Start: 0, End: 2})

	// Insert three columns inside the range.
	eb.SetCursorByOffset(1)
	eb.InsertText("abc")

	em := ec.Get(id)
	fmt.Printf("extmark: [%d,%d)\n", em.Start, em.End)
	fmt.Printf("text: %q\n", eb.Text())

	// Output:
	// extmark: [0,5)
	// text: "[abc]"
}

// ExampleExtmarksController_deletionRemovesContained demonstrates removal of a
// fully deleted extmark.
func ExampleExtmarksController_deletionRemovesContained() {
	eb, _, ec := newExtmarksSetup("aaaBBBccc")
	defer ec.Destroy()

	ec.Create(textbuf.ExtmarkOptions{Start: 0, End: 3})        // Track "aaa".
	idB := ec.Create(textbuf.ExtmarkOptions{Start: 3, End: 6}) // Track "BBB".
	ec.Create(textbuf.ExtmarkOptions{Start: 6, End: 9})        // Track "ccc".

	// Delete exactly the range tracked for "BBB".
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
