package widget

import "github.com/unaquatic/fluid/input"

// Textarea actions.
const (
	// Cursor movement.
	ActionMoveLeft      input.Action = "move-left"
	ActionMoveRight     input.Action = "move-right"
	ActionMoveUp        input.Action = "move-up"
	ActionMoveDown      input.Action = "move-down"
	ActionMoveWordLeft  input.Action = "word-backward"
	ActionMoveWordRight input.Action = "word-forward"
	ActionMoveHome      input.Action = "line-home"   // Start of the logical line.
	ActionMoveEnd       input.Action = "line-end"    // End of the logical line.
	ActionMoveDocStart  input.Action = "buffer-home" // Start of the document.
	ActionMoveDocEnd    input.Action = "buffer-end"  // End of the document.
	ActionPageUp        input.Action = "page-up"
	ActionPageDown      input.Action = "page-down"

	// Movement within the current wrapped line.
	ActionVisualHome input.Action = "visual-line-home" // Start of the visual line.
	ActionVisualEnd  input.Action = "visual-line-end"  // End of the visual line.

	// Selection movement extends the current selection.
	ActionSelectLeft       input.Action = "select-left"
	ActionSelectRight      input.Action = "select-right"
	ActionSelectUp         input.Action = "select-up"
	ActionSelectDown       input.Action = "select-down"
	ActionSelectWordLeft   input.Action = "select-word-backward"
	ActionSelectWordRight  input.Action = "select-word-forward"
	ActionSelectHome       input.Action = "select-line-home"
	ActionSelectEnd        input.Action = "select-line-end"
	ActionSelectDocStart   input.Action = "select-buffer-home"
	ActionSelectDocEnd     input.Action = "select-buffer-end"
	ActionSelectAll        input.Action = "select-all"
	ActionSelectVisualHome input.Action = "select-visual-line-home"
	ActionSelectVisualEnd  input.Action = "select-visual-line-end"

	// Editing
	ActionBackspace     input.Action = "backspace"
	ActionDelete        input.Action = "delete"
	ActionDeleteLine    input.Action = "delete-line"
	ActionDeleteToEnd   input.Action = "delete-to-line-end"
	ActionDeleteToStart input.Action = "delete-to-line-start"
	ActionDeleteWordFwd input.Action = "delete-word-forward"
	ActionDeleteWordBwd input.Action = "delete-word-backward"
	ActionNewline       input.Action = "newline"
	ActionTab           input.Action = "tab"
	ActionUndo          input.Action = "undo"
	ActionRedo          input.Action = "redo"
	ActionSubmit        input.Action = "submit"
)

// navActions is the set of actions allowed in read-only mode.
var navActions = map[input.Action]bool{
	ActionMoveLeft: true, ActionMoveRight: true,
	ActionMoveUp: true, ActionMoveDown: true,
	ActionMoveWordLeft: true, ActionMoveWordRight: true,
	ActionMoveHome: true, ActionMoveEnd: true,
	ActionMoveDocStart: true, ActionMoveDocEnd: true,
	ActionPageUp: true, ActionPageDown: true,
	ActionVisualHome: true, ActionVisualEnd: true,
	ActionSelectLeft: true, ActionSelectRight: true,
	ActionSelectUp: true, ActionSelectDown: true,
	ActionSelectWordLeft: true, ActionSelectWordRight: true,
	ActionSelectHome: true, ActionSelectEnd: true,
	ActionSelectDocStart: true, ActionSelectDocEnd: true,
	ActionSelectAll:        true,
	ActionSelectVisualHome: true,
	ActionSelectVisualEnd:  true,
}

// DefaultTextareaBindings contains the standard multiline editor bindings.
var DefaultTextareaBindings = []input.KeyBinding{
	// Arrow-key movement.
	{Name: input.KeyLeft, Action: ActionMoveLeft},
	{Name: input.KeyRight, Action: ActionMoveRight},
	{Name: input.KeyUp, Action: ActionMoveUp},
	{Name: input.KeyDown, Action: ActionMoveDown},

	// Emacs-style character movement.
	{Name: "f", Mods: input.ModCtrl, Action: ActionMoveRight},
	{Name: "b", Mods: input.ModCtrl, Action: ActionMoveLeft},

	// Arrow-key selection.
	{Name: input.KeyLeft, Mods: input.ModShift, Action: ActionSelectLeft},
	{Name: input.KeyRight, Mods: input.ModShift, Action: ActionSelectRight},
	{Name: input.KeyUp, Mods: input.ModShift, Action: ActionSelectUp},
	{Name: input.KeyDown, Mods: input.ModShift, Action: ActionSelectDown},

	// Emacs-style logical line movement.
	{Name: "a", Mods: input.ModCtrl, Action: ActionMoveHome},
	{Name: "e", Mods: input.ModCtrl, Action: ActionMoveEnd},
	{Name: "a", Mods: input.ModCtrl | input.ModShift, Action: ActionSelectHome},
	{Name: "e", Mods: input.ModCtrl | input.ModShift, Action: ActionSelectEnd},

	// Wrapped-line movement.
	{Name: "a", Mods: input.ModAlt, Action: ActionVisualHome},
	{Name: "e", Mods: input.ModAlt, Action: ActionVisualEnd},
	{Name: "a", Mods: input.ModAlt | input.ModShift, Action: ActionSelectVisualHome},
	{Name: "e", Mods: input.ModAlt | input.ModShift, Action: ActionSelectVisualEnd},

	// Document movement.
	{Name: input.KeyHome, Action: ActionMoveDocStart},
	{Name: input.KeyEnd, Action: ActionMoveDocEnd},
	{Name: input.KeyHome, Mods: input.ModShift, Action: ActionSelectDocStart},
	{Name: input.KeyEnd, Mods: input.ModShift, Action: ActionSelectDocEnd},

	// Page movement.
	{Name: input.KeyPageUp, Action: ActionPageUp},
	{Name: input.KeyPageDown, Action: ActionPageDown},

	// Word movement.
	{Name: input.KeyLeft, Mods: input.ModCtrl, Action: ActionMoveWordLeft},
	{Name: input.KeyRight, Mods: input.ModCtrl, Action: ActionMoveWordRight},
	{Name: input.KeyLeft, Mods: input.ModAlt, Action: ActionMoveWordLeft},
	{Name: input.KeyRight, Mods: input.ModAlt, Action: ActionMoveWordRight},
	{Name: "b", Mods: input.ModAlt, Action: ActionMoveWordLeft},
	{Name: "f", Mods: input.ModAlt, Action: ActionMoveWordRight},

	// Word selection.
	{Name: input.KeyLeft, Mods: input.ModCtrl | input.ModShift, Action: ActionSelectWordLeft},
	{Name: input.KeyRight, Mods: input.ModCtrl | input.ModShift, Action: ActionSelectWordRight},
	{Name: input.KeyLeft, Mods: input.ModAlt | input.ModShift, Action: ActionSelectWordLeft},
	{Name: input.KeyRight, Mods: input.ModAlt | input.ModShift, Action: ActionSelectWordRight},
	{Name: "b", Mods: input.ModAlt | input.ModShift, Action: ActionSelectWordLeft},
	{Name: "f", Mods: input.ModAlt | input.ModShift, Action: ActionSelectWordRight},

	// Editing
	{Name: input.KeyBackspace, Action: ActionBackspace},
	{Name: input.KeyBackspace, Mods: input.ModShift, Action: ActionBackspace},
	{Name: "d", Mods: input.ModCtrl, Action: ActionDelete},
	{Name: input.KeyDelete, Action: ActionDelete},
	{Name: input.KeyDelete, Mods: input.ModShift, Action: ActionDelete},
	{Name: input.KeyReturn, Action: ActionNewline},
	{Name: input.KeyTab, Action: ActionTab},
	{Name: input.KeyReturn, Mods: input.ModAlt, Action: ActionSubmit},

	// Word deletion.
	{Name: "w", Mods: input.ModCtrl, Action: ActionDeleteWordBwd},
	{Name: input.KeyBackspace, Mods: input.ModCtrl, Action: ActionDeleteWordBwd},
	{Name: input.KeyBackspace, Mods: input.ModAlt, Action: ActionDeleteWordBwd},
	{Name: "d", Mods: input.ModAlt, Action: ActionDeleteWordFwd},
	{Name: input.KeyDelete, Mods: input.ModAlt, Action: ActionDeleteWordFwd},
	{Name: input.KeyDelete, Mods: input.ModCtrl, Action: ActionDeleteWordFwd},

	// Line deletion.
	{Name: "d", Mods: input.ModCtrl | input.ModShift, Action: ActionDeleteLine},
	{Name: "k", Mods: input.ModCtrl, Action: ActionDeleteToEnd},
	{Name: "u", Mods: input.ModCtrl, Action: ActionDeleteToStart},

	// Undo and redo.
	{Name: "-", Mods: input.ModCtrl, Action: ActionUndo},
	{Name: ".", Mods: input.ModCtrl, Action: ActionRedo},

	// Command-key bindings reported through the Kitty keyboard protocol.
	{Name: input.KeyLeft, Mods: input.ModMeta, Action: ActionVisualHome},
	{Name: input.KeyRight, Mods: input.ModMeta, Action: ActionVisualEnd},
	{Name: input.KeyUp, Mods: input.ModMeta, Action: ActionMoveDocStart},
	{Name: input.KeyDown, Mods: input.ModMeta, Action: ActionMoveDocEnd},
	{Name: input.KeyLeft, Mods: input.ModMeta | input.ModShift, Action: ActionSelectVisualHome},
	{Name: input.KeyRight, Mods: input.ModMeta | input.ModShift, Action: ActionSelectVisualEnd},
	{Name: input.KeyUp, Mods: input.ModMeta | input.ModShift, Action: ActionSelectDocStart},
	{Name: input.KeyDown, Mods: input.ModMeta | input.ModShift, Action: ActionSelectDocEnd},
	{Name: "a", Mods: input.ModMeta, Action: ActionSelectAll},
	{Name: "z", Mods: input.ModMeta, Action: ActionUndo},
	{Name: "z", Mods: input.ModMeta | input.ModShift, Action: ActionRedo},
}

// DefaultInputBindings contains the standard single-line editor bindings.
// Enter submits instead of inserting a newline, and multiline movement is omitted.
var DefaultInputBindings = []input.KeyBinding{
	// Navigation
	{Name: input.KeyLeft, Action: ActionMoveLeft},
	{Name: input.KeyRight, Action: ActionMoveRight},
	{Name: "f", Mods: input.ModCtrl, Action: ActionMoveRight},
	{Name: "b", Mods: input.ModCtrl, Action: ActionMoveLeft},

	// Logical line movement.
	{Name: "a", Mods: input.ModCtrl, Action: ActionMoveHome},
	{Name: "e", Mods: input.ModCtrl, Action: ActionMoveEnd},

	// Word navigation
	{Name: input.KeyLeft, Mods: input.ModCtrl, Action: ActionMoveWordLeft},
	{Name: input.KeyRight, Mods: input.ModCtrl, Action: ActionMoveWordRight},
	{Name: input.KeyLeft, Mods: input.ModAlt, Action: ActionMoveWordLeft},
	{Name: input.KeyRight, Mods: input.ModAlt, Action: ActionMoveWordRight},
	{Name: "b", Mods: input.ModAlt, Action: ActionMoveWordLeft},
	{Name: "f", Mods: input.ModAlt, Action: ActionMoveWordRight},

	// Document movement.
	{Name: input.KeyHome, Action: ActionMoveDocStart},
	{Name: input.KeyEnd, Action: ActionMoveDocEnd},

	// Selection
	{Name: input.KeyLeft, Mods: input.ModShift, Action: ActionSelectLeft},
	{Name: input.KeyRight, Mods: input.ModShift, Action: ActionSelectRight},
	{Name: "a", Mods: input.ModCtrl | input.ModShift, Action: ActionSelectHome},
	{Name: "e", Mods: input.ModCtrl | input.ModShift, Action: ActionSelectEnd},
	{Name: input.KeyLeft, Mods: input.ModCtrl | input.ModShift, Action: ActionSelectWordLeft},
	{Name: input.KeyRight, Mods: input.ModCtrl | input.ModShift, Action: ActionSelectWordRight},
	{Name: input.KeyLeft, Mods: input.ModAlt | input.ModShift, Action: ActionSelectWordLeft},
	{Name: input.KeyRight, Mods: input.ModAlt | input.ModShift, Action: ActionSelectWordRight},
	{Name: "b", Mods: input.ModAlt | input.ModShift, Action: ActionSelectWordLeft},
	{Name: "f", Mods: input.ModAlt | input.ModShift, Action: ActionSelectWordRight},
	{Name: input.KeyHome, Mods: input.ModShift, Action: ActionSelectDocStart},
	{Name: input.KeyEnd, Mods: input.ModShift, Action: ActionSelectDocEnd},

	// Editing
	{Name: input.KeyBackspace, Action: ActionBackspace},
	{Name: input.KeyBackspace, Mods: input.ModShift, Action: ActionBackspace},
	{Name: "d", Mods: input.ModCtrl, Action: ActionDelete},
	{Name: input.KeyDelete, Action: ActionDelete},
	{Name: input.KeyDelete, Mods: input.ModShift, Action: ActionDelete},
	{Name: input.KeyReturn, Action: ActionSubmit},
	{Name: input.KeyReturn, Mods: input.ModAlt, Action: ActionSubmit},

	// Word deletion.
	{Name: "w", Mods: input.ModCtrl, Action: ActionDeleteWordBwd},
	{Name: input.KeyBackspace, Mods: input.ModCtrl, Action: ActionDeleteWordBwd},
	{Name: input.KeyBackspace, Mods: input.ModAlt, Action: ActionDeleteWordBwd},
	{Name: "d", Mods: input.ModAlt, Action: ActionDeleteWordFwd},
	{Name: input.KeyDelete, Mods: input.ModAlt, Action: ActionDeleteWordFwd},
	{Name: input.KeyDelete, Mods: input.ModCtrl, Action: ActionDeleteWordFwd},

	// Delete to a line boundary.
	{Name: "k", Mods: input.ModCtrl, Action: ActionDeleteToEnd},
	{Name: "u", Mods: input.ModCtrl, Action: ActionDeleteToStart},

	// Command+A reported through the Kitty keyboard protocol.
	{Name: "a", Mods: input.ModMeta, Action: ActionSelectAll},

	// Undo and redo.
	{Name: "-", Mods: input.ModCtrl, Action: ActionUndo},
	{Name: ".", Mods: input.ModCtrl, Action: ActionRedo},
	{Name: "z", Mods: input.ModMeta, Action: ActionUndo},
	{Name: "z", Mods: input.ModMeta | input.ModShift, Action: ActionRedo},
}
