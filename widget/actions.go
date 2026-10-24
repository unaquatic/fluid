package widget

import "github.com/unaquatic/fluid/input"

// Textarea action names.

const (
	// Cursor movement.
	ActionMoveLeft      = "move-left"
	ActionMoveRight     = "move-right"
	ActionMoveUp        = "move-up"
	ActionMoveDown      = "move-down"
	ActionMoveWordLeft  = "word-backward"
	ActionMoveWordRight = "word-forward"
	ActionMoveHome      = "line-home"   // Start of the logical line.
	ActionMoveEnd       = "line-end"    // End of the logical line.
	ActionMoveDocStart  = "buffer-home" // Start of the document.
	ActionMoveDocEnd    = "buffer-end"  // End of the document.
	ActionPageUp        = "page-up"
	ActionPageDown      = "page-down"

	// Movement within the current wrapped line.
	ActionVisualHome = "visual-line-home" // Start of the visual line.
	ActionVisualEnd  = "visual-line-end"  // End of the visual line.

	// Selection movement extends the current selection.
	ActionSelectLeft       = "select-left"
	ActionSelectRight      = "select-right"
	ActionSelectUp         = "select-up"
	ActionSelectDown       = "select-down"
	ActionSelectWordLeft   = "select-word-backward"
	ActionSelectWordRight  = "select-word-forward"
	ActionSelectHome       = "select-line-home"
	ActionSelectEnd        = "select-line-end"
	ActionSelectDocStart   = "select-buffer-home"
	ActionSelectDocEnd     = "select-buffer-end"
	ActionSelectAll        = "select-all"
	ActionSelectVisualHome = "select-visual-line-home"
	ActionSelectVisualEnd  = "select-visual-line-end"

	// Editing
	ActionBackspace     = "backspace"
	ActionDelete        = "delete"
	ActionDeleteLine    = "delete-line"
	ActionDeleteToEnd   = "delete-to-line-end"
	ActionDeleteToStart = "delete-to-line-start"
	ActionDeleteWordFwd = "delete-word-forward"
	ActionDeleteWordBwd = "delete-word-backward"
	ActionNewline       = "newline"
	ActionTab           = "tab"
	ActionUndo          = "undo"
	ActionRedo          = "redo"
	ActionSubmit        = "submit"
)

// navActions is the set of actions allowed in read-only mode.
var navActions = map[string]bool{
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
