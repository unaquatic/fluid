package widget

import (
	"strings"

	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/textbuf"
)

// Input is a single-line editor built on Textarea.
// It removes multiline actions and can enforce a maximum rune count.
type Input struct {
	*Textarea

	maxLength int

	// OnInput is called after the value changes.
	OnInput func(value string)
	// OnChange is called after the value changes.
	OnChange func(value string)
}

// InputOption configures an Input widget during construction.
type InputOption func(*Input)

// NewInput returns a one-row Input with optional configuration.
func NewInput(opts ...InputOption) *Input {
	// Disable wrapping and constrain the embedded editor to one row.
	ta := NewTextarea(
		TextareaWithWrapMode(WrapNone),
		TextareaWithHeight(layout.Points(1)),
	)

	inp := &Input{
		Textarea: ta,
	}

	// Forward Textarea content changes through both Input callbacks.
	ta.OnContentChange = func(text string) {
		if inp.OnInput != nil {
			inp.OnInput(text)
		}
		if inp.OnChange != nil {
			inp.OnChange(text)
		}
	}

	for _, opt := range opts {
		opt(inp)
	}

	return inp
}

// Input value and cursor access.

// Value returns the current input value.
func (inp *Input) Value() string {
	return inp.Text()
}

// SetValue sets the input value.
func (inp *Input) SetValue(v string) {
	// Remove line breaks before updating the single-line editor.
	v = strings.ReplaceAll(v, "\n", "")
	v = strings.ReplaceAll(v, "\r", "")
	inp.SetText(v)
	// Place the cursor after the new value.
	inp.SetCursorPos(len([]rune(v)))
}

// CursorPos returns the cursor position (display column on row 0).
func (inp *Input) CursorPos() int {
	_, col := inp.CursorPosition()
	return col
}

// SetCursorPos sets the cursor position (display column on row 0).
func (inp *Input) SetCursorPos(pos int) {
	inp.SetCursor(0, pos)
}

// blockedInputActions contains multiline actions that Input does not support.
var blockedInputActions = map[string]bool{
	ActionNewline:  true,
	ActionMoveUp:   true,
	ActionMoveDown: true,
	ActionSelectUp: true, ActionSelectDown: true,
	ActionPageUp: true, ActionPageDown: true,
	ActionDeleteLine: true,
	ActionTab:        true,
	ActionVisualHome: true, ActionVisualEnd: true,
	ActionSelectDocStart: true, ActionSelectDocEnd: true,
	ActionMoveDocStart: true, ActionMoveDocEnd: true,
}

// HandleKeyPress processes a key event under Input's single-line constraints.
func (inp *Input) HandleKeyPress(e *input.KeyEvent) bool {
	if e.Type == input.KeyRelease {
		return false
	}

	// Enforce the rune limit before Textarea inserts printable input.
	if e.IsRune() {
		if inp.maxLength > 0 && len([]rune(inp.Value())) >= inp.maxLength {
			return true // Consume input that exceeds the configured limit.
		}
		return inp.Textarea.HandleKeyPress(e)
	}

	// Input has a separate binding table because Enter submits.
	action, found := inp.resolveInputAction(e)
	if !found {
		return false
	}

	// Let unsupported multiline actions propagate to an ancestor.
	if blockedInputActions[action] {
		return false
	}

	// Invoke the callback without inserting a newline.
	if action == ActionSubmit {
		if inp.Textarea.OnSubmit != nil {
			inp.Textarea.OnSubmit(inp.Value())
		}
		e.StopPropagation()
		return true
	}

	// Read-only inputs still accept navigation and selection actions.
	if inp.readOnly && !navActions[action] {
		return false
	}

	// Reuse Textarea's editing and movement handlers.
	if handler, ok := inp.Textarea.actionHandlers[action]; ok {
		handled := handler()
		if handled {
			e.StopPropagation()
		}
		return handled
	}

	return false
}

// HandlePaste inserts pasted text after removing line breaks and applying the rune limit.
func (inp *Input) HandlePaste(e *input.PasteEvent) {
	if inp.readOnly {
		return
	}

	// Normalize the paste to a single line.
	text := strings.ReplaceAll(e.Text, "\n", "")
	text = strings.ReplaceAll(text, "\r", "")

	// Truncate the paste to the remaining rune capacity.
	if inp.maxLength > 0 {
		currentLen := len([]rune(inp.Value()))
		remaining := inp.maxLength - currentLen
		if remaining <= 0 {
			return
		}
		runes := []rune(text)
		if len(runes) > remaining {
			text = string(runes[:remaining])
		}
	}

	inp.deleteSelectedText()
	inp.editBuffer().InsertText(text)
	inp.emitContentChange()
	inp.RequestRender()
}

// Input keybinding resolution.

func (inp *Input) resolveInputAction(e *input.KeyEvent) (string, bool) {
	if inp.bindingsDirty {
		inp.rebuildInputBindingMap()
	}
	return input.ResolveAction(*e, inp.bindingMap)
}

func (inp *Input) rebuildInputBindingMap() {
	defaults := DefaultInputBindings
	merged := input.MergeKeyBindings(defaults, inp.keyBindings)
	aliases := input.MergeKeyAliases(input.DefaultAliases, inp.keyAliases)
	inp.bindingMap = input.BuildKeyBindingsMap(merged, aliases)
	inp.bindingsDirty = false
}

// Input options.

// InputWithValue sets the initial value.
func InputWithValue(v string) InputOption {
	return func(inp *Input) {
		v = strings.ReplaceAll(v, "\n", "")
		v = strings.ReplaceAll(v, "\r", "")
		inp.editBuffer().SetText(v)
		// Place the cursor after the initial value.
		lineW := inp.textBuffer().LineWidthAt(0)
		inp.editBuffer().SetCursor(0, lineW)
	}
}

// InputWithPlaceholder sets the placeholder text.
func InputWithPlaceholder(p string) InputOption {
	return func(inp *Input) {
		inp.editSurface.SetPlaceholderText(p, style.ColorGray)
	}
}

// InputWithMaxLength sets the maximum value length in runes; non-positive values disable the limit.
func InputWithMaxLength(n int) InputOption {
	return func(inp *Input) { inp.maxLength = n }
}

// InputWithTextColor sets the text color.
func InputWithTextColor(c style.Color) InputOption {
	return func(inp *Input) { inp.textColor = c }
}

// InputWithBackgroundColor sets the background color.
func InputWithBackgroundColor(c style.Color) InputOption {
	return func(inp *Input) { inp.backgroundColor = c }
}

// InputWithPlaceholderColor sets the placeholder text color.
func InputWithPlaceholderColor(c style.Color) InputOption {
	return func(inp *Input) {
		phText, _ := inp.editSurface.PlaceholderText()
		inp.editSurface.SetPlaceholderText(phText, c)
	}
}

// InputWithCursorColor sets the cursor color.
func InputWithCursorColor(c style.Color) InputOption {
	return func(inp *Input) { inp.cursorColor = c }
}

// InputWithOnInput sets the callback invoked after each value change.
func InputWithOnInput(fn func(string)) InputOption {
	return func(inp *Input) { inp.OnInput = fn }
}

// InputWithOnChange sets the callback invoked after each value change.
func InputWithOnChange(fn func(string)) InputOption {
	return func(inp *Input) { inp.OnChange = fn }
}

// InputWithOnSubmit sets the callback invoked when the submit action runs.
func InputWithOnSubmit(fn func(string)) InputOption {
	return func(inp *Input) { inp.Textarea.OnSubmit = fn }
}

// InputWithID sets the widget ID.
func InputWithID(id string) InputOption {
	return func(inp *Input) { inp.SetID(id) }
}

// InputWithWidth sets the widget width.
func InputWithWidth(s layout.Scalar) InputOption {
	return func(inp *Input) { inp.LayoutNode().Width = s }
}

// InputWithHeight sets the widget height.
func InputWithHeight(s layout.Scalar) InputOption {
	return func(inp *Input) { inp.LayoutNode().Height = s }
}

// InputWithKeyBindings sets custom keybinding overrides.
func InputWithKeyBindings(bindings []input.KeyBinding) InputOption {
	return func(inp *Input) {
		inp.keyBindings = bindings
		inp.bindingsDirty = true
	}
}

// InputWithKeyAliases sets custom key alias overrides.
func InputWithKeyAliases(aliases input.KeyAliasMap) InputOption {
	return func(inp *Input) {
		inp.keyAliases = aliases
		inp.bindingsDirty = true
	}
}

// InputWithSelectionBg sets the selection background color.
func InputWithSelectionBg(c style.Color) InputOption {
	return func(inp *Input) { inp.selectionBg = c }
}

// InputWithPlaceholderSpans sets a span-based placeholder.
func InputWithPlaceholderSpans(spans []textbuf.Span) InputOption {
	return func(inp *Input) {
		inp.editSurface.SetPlaceholderSpans(spans)
	}
}
