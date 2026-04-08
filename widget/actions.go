package widget

import "github.com/unaquatic/fluid/input"

// ---------------------------------------------------------------------------
// TextAction constants for Textarea actions.
// ---------------------------------------------------------------------------

const (
	// Navigation
	ActionMoveLeft      = "move-left"
	ActionMoveRight     = "move-right"
	ActionMoveUp        = "move-up"
	ActionMoveDown      = "move-down"
	ActionMoveWordLeft  = "word-backward"
	ActionMoveWordRight = "word-forward"
	ActionMoveHome      = "line-home"   // logical line start (Ctrl+A)
	ActionMoveEnd       = "line-end"    // logical line end (Ctrl+E)
	ActionMoveDocStart  = "buffer-home" // document start (Home)
	ActionMoveDocEnd    = "buffer-end"  // document end (End)
	ActionPageUp        = "page-up"
	ActionPageDown      = "page-down"

	// Visual-line navigation (wrap-aware)
	ActionVisualHome = "visual-line-home" // visual (wrapped) line start
	ActionVisualEnd  = "visual-line-end"  // visual (wrapped) line end

	// Selection — navigation that extends selection
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

// ---------------------------------------------------------------------------
// Default Textarea Keybindings.
// ---------------------------------------------------------------------------

var DefaultTextareaBindings = []input.KeyBinding{
	// Basic cursor movement
	{Name: input.KeyLeft, Action: ActionMoveLeft},
	{Name: input.KeyRight, Action: ActionMoveRight},
	{Name: input.KeyUp, Action: ActionMoveUp},
	{Name: input.KeyDown, Action: ActionMoveDown},

	// Emacs char navigation
	{Name: "f", Ctrl: true, Action: ActionMoveRight},
	{Name: "b", Ctrl: true, Action: ActionMoveLeft},

	// Shift+Arrow selection
	{Name: input.KeyLeft, Shift: true, Action: ActionSelectLeft},
	{Name: input.KeyRight, Shift: true, Action: ActionSelectRight},
	{Name: input.KeyUp, Shift: true, Action: ActionSelectUp},
	{Name: input.KeyDown, Shift: true, Action: ActionSelectDown},

	// Line start/end (Ctrl+A / Ctrl+E — emacs style)
	{Name: "a", Ctrl: true, Action: ActionMoveHome},
	{Name: "e", Ctrl: true, Action: ActionMoveEnd},
	{Name: "a", Ctrl: true, Shift: true, Action: ActionSelectHome},
	{Name: "e", Ctrl: true, Shift: true, Action: ActionSelectEnd},

	// Visual-line start/end (Alt+A / Alt+E)
	{Name: "a", Alt: true, Action: ActionVisualHome},
	{Name: "e", Alt: true, Action: ActionVisualEnd},
	{Name: "a", Alt: true, Shift: true, Action: ActionSelectVisualHome},
	{Name: "e", Alt: true, Shift: true, Action: ActionSelectVisualEnd},

	// Document start/end (Home / End)
	{Name: input.KeyHome, Action: ActionMoveDocStart},
	{Name: input.KeyEnd, Action: ActionMoveDocEnd},
	{Name: input.KeyHome, Shift: true, Action: ActionSelectDocStart},
	{Name: input.KeyEnd, Shift: true, Action: ActionSelectDocEnd},

	// Page up/down
	{Name: input.KeyPageUp, Action: ActionPageUp},
	{Name: input.KeyPageDown, Action: ActionPageDown},

	// Word navigation (Ctrl+Arrow, Alt+Arrow, Alt+B/F)
	{Name: input.KeyLeft, Ctrl: true, Action: ActionMoveWordLeft},
	{Name: input.KeyRight, Ctrl: true, Action: ActionMoveWordRight},
	{Name: input.KeyLeft, Alt: true, Action: ActionMoveWordLeft},
	{Name: input.KeyRight, Alt: true, Action: ActionMoveWordRight},
	{Name: "b", Alt: true, Action: ActionMoveWordLeft},
	{Name: "f", Alt: true, Action: ActionMoveWordRight},

	// Word selection (Ctrl+Shift+Arrow, Alt+Shift+Arrow, Alt+Shift+B/F)
	{Name: input.KeyLeft, Ctrl: true, Shift: true, Action: ActionSelectWordLeft},
	{Name: input.KeyRight, Ctrl: true, Shift: true, Action: ActionSelectWordRight},
	{Name: input.KeyLeft, Alt: true, Shift: true, Action: ActionSelectWordLeft},
	{Name: input.KeyRight, Alt: true, Shift: true, Action: ActionSelectWordRight},
	{Name: "b", Alt: true, Shift: true, Action: ActionSelectWordLeft},
	{Name: "f", Alt: true, Shift: true, Action: ActionSelectWordRight},

	// Editing
	{Name: input.KeyBackspace, Action: ActionBackspace},
	{Name: input.KeyBackspace, Shift: true, Action: ActionBackspace},
	{Name: "d", Ctrl: true, Action: ActionDelete},
	{Name: input.KeyDelete, Action: ActionDelete},
	{Name: input.KeyDelete, Shift: true, Action: ActionDelete},
	{Name: input.KeyReturn, Action: ActionNewline},
	{Name: input.KeyTab, Action: ActionTab},
	{Name: input.KeyReturn, Alt: true, Action: ActionSubmit},

	// Delete word
	{Name: "w", Ctrl: true, Action: ActionDeleteWordBwd},
	{Name: input.KeyBackspace, Ctrl: true, Action: ActionDeleteWordBwd},
	{Name: input.KeyBackspace, Alt: true, Action: ActionDeleteWordBwd},
	{Name: "d", Alt: true, Action: ActionDeleteWordFwd},
	{Name: input.KeyDelete, Alt: true, Action: ActionDeleteWordFwd},
	{Name: input.KeyDelete, Ctrl: true, Action: ActionDeleteWordFwd},

	// Delete line / kill
	{Name: "d", Ctrl: true, Shift: true, Action: ActionDeleteLine},
	{Name: "k", Ctrl: true, Action: ActionDeleteToEnd},
	{Name: "u", Ctrl: true, Action: ActionDeleteToStart},

	// Undo/Redo
	{Name: "-", Ctrl: true, Action: ActionUndo},
	{Name: ".", Ctrl: true, Action: ActionRedo},

	// Super (Cmd) keys — Kitty keyboard mode
	{Name: input.KeyLeft, Meta: true, Action: ActionVisualHome},
	{Name: input.KeyRight, Meta: true, Action: ActionVisualEnd},
	{Name: input.KeyUp, Meta: true, Action: ActionMoveDocStart},
	{Name: input.KeyDown, Meta: true, Action: ActionMoveDocEnd},
	{Name: input.KeyLeft, Meta: true, Shift: true, Action: ActionSelectVisualHome},
	{Name: input.KeyRight, Meta: true, Shift: true, Action: ActionSelectVisualEnd},
	{Name: input.KeyUp, Meta: true, Shift: true, Action: ActionSelectDocStart},
	{Name: input.KeyDown, Meta: true, Shift: true, Action: ActionSelectDocEnd},
	{Name: "a", Meta: true, Action: ActionSelectAll},
	{Name: "z", Meta: true, Action: ActionUndo},
	{Name: "z", Meta: true, Shift: true, Action: ActionRedo},
}

// ---------------------------------------------------------------------------
// Default Input Keybindings
// ---------------------------------------------------------------------------

// DefaultInputBindings defines the standard keybindings for an Input.
// Newline is replaced with submit; no vertical movement or multi-line actions.
var DefaultInputBindings = []input.KeyBinding{
	// Navigation
	{Name: input.KeyLeft, Action: ActionMoveLeft},
	{Name: input.KeyRight, Action: ActionMoveRight},
	{Name: "f", Ctrl: true, Action: ActionMoveRight},
	{Name: "b", Ctrl: true, Action: ActionMoveLeft},

	// Line start/end
	{Name: "a", Ctrl: true, Action: ActionMoveHome},
	{Name: "e", Ctrl: true, Action: ActionMoveEnd},

	// Word navigation
	{Name: input.KeyLeft, Ctrl: true, Action: ActionMoveWordLeft},
	{Name: input.KeyRight, Ctrl: true, Action: ActionMoveWordRight},
	{Name: input.KeyLeft, Alt: true, Action: ActionMoveWordLeft},
	{Name: input.KeyRight, Alt: true, Action: ActionMoveWordRight},
	{Name: "b", Alt: true, Action: ActionMoveWordLeft},
	{Name: "f", Alt: true, Action: ActionMoveWordRight},

	// Document start/end
	{Name: input.KeyHome, Action: ActionMoveDocStart},
	{Name: input.KeyEnd, Action: ActionMoveDocEnd},

	// Selection
	{Name: input.KeyLeft, Shift: true, Action: ActionSelectLeft},
	{Name: input.KeyRight, Shift: true, Action: ActionSelectRight},
	{Name: "a", Ctrl: true, Shift: true, Action: ActionSelectHome},
	{Name: "e", Ctrl: true, Shift: true, Action: ActionSelectEnd},
	{Name: input.KeyLeft, Ctrl: true, Shift: true, Action: ActionSelectWordLeft},
	{Name: input.KeyRight, Ctrl: true, Shift: true, Action: ActionSelectWordRight},
	{Name: input.KeyLeft, Alt: true, Shift: true, Action: ActionSelectWordLeft},
	{Name: input.KeyRight, Alt: true, Shift: true, Action: ActionSelectWordRight},
	{Name: "b", Alt: true, Shift: true, Action: ActionSelectWordLeft},
	{Name: "f", Alt: true, Shift: true, Action: ActionSelectWordRight},
	{Name: input.KeyHome, Shift: true, Action: ActionSelectDocStart},
	{Name: input.KeyEnd, Shift: true, Action: ActionSelectDocEnd},

	// Editing
	{Name: input.KeyBackspace, Action: ActionBackspace},
	{Name: input.KeyBackspace, Shift: true, Action: ActionBackspace},
	{Name: "d", Ctrl: true, Action: ActionDelete},
	{Name: input.KeyDelete, Action: ActionDelete},
	{Name: input.KeyDelete, Shift: true, Action: ActionDelete},
	{Name: input.KeyReturn, Action: ActionSubmit},
	{Name: input.KeyReturn, Alt: true, Action: ActionSubmit},

	// Delete word
	{Name: "w", Ctrl: true, Action: ActionDeleteWordBwd},
	{Name: input.KeyBackspace, Ctrl: true, Action: ActionDeleteWordBwd},
	{Name: input.KeyBackspace, Alt: true, Action: ActionDeleteWordBwd},
	{Name: "d", Alt: true, Action: ActionDeleteWordFwd},
	{Name: input.KeyDelete, Alt: true, Action: ActionDeleteWordFwd},
	{Name: input.KeyDelete, Ctrl: true, Action: ActionDeleteWordFwd},

	// Kill
	{Name: "k", Ctrl: true, Action: ActionDeleteToEnd},
	{Name: "u", Ctrl: true, Action: ActionDeleteToStart},

	// Select all (Cmd+A — Kitty keyboard mode only)
	{Name: "a", Meta: true, Action: ActionSelectAll},

	// Undo/Redo
	{Name: "-", Ctrl: true, Action: ActionUndo},
	{Name: ".", Ctrl: true, Action: ActionRedo},
	{Name: "z", Meta: true, Action: ActionUndo},
	{Name: "z", Meta: true, Shift: true, Action: ActionRedo},
}
