package input

import (
	"maps"
	"strings"
)

// ---------------------------------------------------------------------------
// KeyBinding
// ---------------------------------------------------------------------------

// KeyBinding maps a key combination to an action string.
type KeyBinding struct {
	Name   string // Key name (e.g. "up", "a", "return")
	Ctrl   bool
	Shift  bool
	Alt    bool
	Meta   bool   // Meta/Super/Cmd modifier (Kitty keyboard protocol)
	Action string // Action identifier (e.g. "move-up", "submit")
}

// ---------------------------------------------------------------------------
// KeyAliasMap
// ---------------------------------------------------------------------------

// KeyAliasMap maps alternative key names to canonical names.
type KeyAliasMap map[string]string

// DefaultAliases provides standard key name aliases.
var DefaultAliases = KeyAliasMap{
	"enter": "return",
	"esc":   "escape",
}

// ---------------------------------------------------------------------------
// Functions
// ---------------------------------------------------------------------------

// GetKeyBindingKey generates a unique lookup key from a binding.
// Format: "ctrl+alt+meta+shift+name" (only present modifiers included).
func GetKeyBindingKey(binding KeyBinding) string {
	return buildComboString(binding.Name, binding.Ctrl, binding.Shift, binding.Alt, binding.Meta)
}

// KeyBindingToString returns a human-readable form of a binding, e.g. "ctrl+shift+y".
func KeyBindingToString(binding KeyBinding) string {
	return buildComboString(binding.Name, binding.Ctrl, binding.Shift, binding.Alt, binding.Meta)
}

// buildComboString builds the canonical combo string.
func buildComboString(name string, ctrl, shift, alt, meta bool) string {
	var parts []string
	if ctrl {
		parts = append(parts, "ctrl")
	}
	if alt {
		parts = append(parts, "alt")
	}
	if meta {
		parts = append(parts, "meta")
	}
	if shift {
		parts = append(parts, "shift")
	}
	parts = append(parts, name)
	return strings.Join(parts, "+")
}

// MergeKeyBindings merges custom bindings over defaults.
// Custom entries override default entries that map to the same key combo.
func MergeKeyBindings(defaults, custom []KeyBinding) []KeyBinding {
	// Build a set of custom combo keys
	customKeys := make(map[string]struct{}, len(custom))
	for _, c := range custom {
		customKeys[GetKeyBindingKey(c)] = struct{}{}
	}

	// Start with defaults that aren't overridden
	result := make([]KeyBinding, 0, len(defaults)+len(custom))
	for _, d := range defaults {
		if _, overridden := customKeys[GetKeyBindingKey(d)]; !overridden {
			result = append(result, d)
		}
	}

	// Append custom bindings
	result = append(result, custom...)
	return result
}

// MergeKeyAliases merges custom aliases over defaults.
// Custom entries override defaults with the same key.
func MergeKeyAliases(defaults, custom KeyAliasMap) KeyAliasMap {
	merged := make(KeyAliasMap, len(defaults)+len(custom))
	maps.Copy(merged, defaults)
	maps.Copy(merged, custom)
	return merged
}

// BuildKeyBindingsMap builds a fast lookup map from combo string to action.
// Aliases are applied to binding names during construction.
func BuildKeyBindingsMap(bindings []KeyBinding, aliases ...KeyAliasMap) map[string]string {
	// Merge all alias maps
	aliasMap := make(KeyAliasMap)
	for _, a := range aliases {
		maps.Copy(aliasMap, a)
	}

	m := make(map[string]string, len(bindings))
	for _, b := range bindings {
		name := b.Name
		if canonical, ok := aliasMap[name]; ok {
			name = canonical
		}
		key := buildComboString(name, b.Ctrl, b.Shift, b.Alt, b.Meta)
		m[key] = b.Action
	}
	return m
}

// ResolveAction looks up the action for a KeyEvent in a binding map.
func ResolveAction(ev KeyEvent, bindingMap map[string]string) (action string, found bool) {
	key := buildComboString(ev.Name, ev.Ctrl, ev.Shift, ev.Alt, ev.Meta)
	action, found = bindingMap[key]
	return
}
