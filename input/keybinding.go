package input

import (
	"maps"
	"strings"
)

// KeyBinding associates a key combination with an action.
type KeyBinding struct {
	// Name is the canonical key name, such as "up", "a", or "return".
	Name string
	// Ctrl reports whether Ctrl is required.
	Ctrl bool
	// Shift reports whether Shift is required.
	Shift bool
	// Alt reports whether Alt or Option is required.
	Alt bool
	// Meta reports whether Meta, Super, or Command is required.
	Meta bool
	// Action is the action identifier, such as "move-up" or "submit".
	Action string
}

// KeyAliasMap maps alternative key names to canonical names.
type KeyAliasMap map[string]string

// DefaultAliases contains the standard key name aliases.
var DefaultAliases = KeyAliasMap{
	"enter": "return",
	"esc":   "escape",
}

// GetKeyBindingKey returns binding's lookup string. It includes only active
// modifiers in Control, Alt, Meta, Shift order.
func GetKeyBindingKey(binding KeyBinding) string {
	return buildComboString(binding.Name, binding.Ctrl, binding.Shift, binding.Alt, binding.Meta)
}

// KeyBindingToString returns binding in canonical modifier order, such as
// "ctrl+shift+y".
func KeyBindingToString(binding KeyBinding) string {
	return buildComboString(binding.Name, binding.Ctrl, binding.Shift, binding.Alt, binding.Meta)
}

// buildComboString returns a canonical key combination string.
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

// MergeKeyBindings returns defaults followed by custom bindings. A custom key
// combination replaces every default with the same combination.
func MergeKeyBindings(defaults, custom []KeyBinding) []KeyBinding {
	// Index custom combinations before filtering the defaults.
	customKeys := make(map[string]struct{}, len(custom))
	for _, c := range custom {
		customKeys[GetKeyBindingKey(c)] = struct{}{}
	}

	// Preserve the order of defaults that were not replaced.
	result := make([]KeyBinding, 0, len(defaults)+len(custom))
	for _, d := range defaults {
		if _, overridden := customKeys[GetKeyBindingKey(d)]; !overridden {
			result = append(result, d)
		}
	}

	// Custom bindings retain their original order.
	result = append(result, custom...)
	return result
}

// MergeKeyAliases returns a new alias map in which custom entries override
// defaults with the same alternative name.
func MergeKeyAliases(defaults, custom KeyAliasMap) KeyAliasMap {
	merged := make(KeyAliasMap, len(defaults)+len(custom))
	maps.Copy(merged, defaults)
	maps.Copy(merged, custom)
	return merged
}

// BuildKeyBindingsMap returns a map from canonical key combinations to actions.
// Alias maps are applied in argument order, with later maps taking precedence.
func BuildKeyBindingsMap(bindings []KeyBinding, aliases ...KeyAliasMap) map[string]string {
	// Merge aliases once before resolving binding names.
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

// ResolveAction returns the action mapped to ev's canonical key combination.
func ResolveAction(ev KeyEvent, bindingMap map[string]string) (action string, found bool) {
	key := buildComboString(ev.Name, ev.Ctrl, ev.Shift, ev.Alt, ev.Meta)
	action, found = bindingMap[key]
	return
}
