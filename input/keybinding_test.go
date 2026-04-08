package input

import "testing"

func TestGetKeyBindingKey(t *testing.T) {
	tests := []struct {
		binding KeyBinding
		want    string
	}{
		{KeyBinding{Name: "a"}, "a"},
		{KeyBinding{Name: "a", Ctrl: true}, "ctrl+a"},
		{KeyBinding{Name: "a", Alt: true}, "alt+a"},
		{KeyBinding{Name: "a", Shift: true}, "shift+a"},
		{KeyBinding{Name: "up", Ctrl: true, Shift: true}, "ctrl+shift+up"},
		{KeyBinding{Name: "x", Ctrl: true, Alt: true, Shift: true}, "ctrl+alt+shift+x"},
	}
	for _, tt := range tests {
		got := GetKeyBindingKey(tt.binding)
		if got != tt.want {
			t.Errorf("GetKeyBindingKey(%+v) = %q, want %q", tt.binding, got, tt.want)
		}
	}
}

func TestKeyBindingToString(t *testing.T) {
	b := KeyBinding{Name: "y", Ctrl: true, Shift: true, Action: "save"}
	got := KeyBindingToString(b)
	if got != "ctrl+shift+y" {
		t.Errorf("KeyBindingToString = %q, want 'ctrl+shift+y'", got)
	}
}

func TestMergeKeyBindings_NoOverlap(t *testing.T) {
	defaults := []KeyBinding{
		{Name: "up", Action: "move-up"},
		{Name: "down", Action: "move-down"},
	}
	custom := []KeyBinding{
		{Name: "a", Ctrl: true, Action: "select-all"},
	}

	merged := MergeKeyBindings(defaults, custom)
	if len(merged) != 3 {
		t.Fatalf("expected 3 bindings, got %d", len(merged))
	}
}

func TestMergeKeyBindings_Override(t *testing.T) {
	defaults := []KeyBinding{
		{Name: "up", Action: "move-up"},
		{Name: "down", Action: "move-down"},
	}
	custom := []KeyBinding{
		{Name: "up", Action: "custom-up"}, // overrides default
	}

	merged := MergeKeyBindings(defaults, custom)
	if len(merged) != 2 {
		t.Fatalf("expected 2 bindings, got %d", len(merged))
	}

	// Find the "up" binding — should be the custom one
	for _, b := range merged {
		if b.Name == "up" {
			if b.Action != "custom-up" {
				t.Errorf("up action = %q, want 'custom-up'", b.Action)
			}
			return
		}
	}
	t.Error("merged should contain 'up' binding")
}

func TestMergeKeyBindings_EmptyCustom(t *testing.T) {
	defaults := []KeyBinding{
		{Name: "a", Action: "act-a"},
	}
	merged := MergeKeyBindings(defaults, nil)
	if len(merged) != 1 || merged[0].Action != "act-a" {
		t.Error("empty custom should return defaults unchanged")
	}
}

func TestMergeKeyBindings_EmptyDefaults(t *testing.T) {
	custom := []KeyBinding{
		{Name: "x", Action: "act-x"},
	}
	merged := MergeKeyBindings(nil, custom)
	if len(merged) != 1 || merged[0].Action != "act-x" {
		t.Error("empty defaults should return custom only")
	}
}

func TestMergeKeyAliases(t *testing.T) {
	defaults := KeyAliasMap{"enter": "return", "esc": "escape"}
	custom := KeyAliasMap{"cr": "return", "esc": "esc-custom"} // overrides esc

	merged := MergeKeyAliases(defaults, custom)

	if merged["enter"] != "return" {
		t.Error("should preserve default 'enter'")
	}
	if merged["cr"] != "return" {
		t.Error("should include custom 'cr'")
	}
	if merged["esc"] != "esc-custom" {
		t.Errorf("esc = %q, should be overridden to 'esc-custom'", merged["esc"])
	}
}

func TestBuildKeyBindingsMap(t *testing.T) {
	bindings := []KeyBinding{
		{Name: "up", Action: "move-up"},
		{Name: "a", Ctrl: true, Action: "select-all"},
		{Name: "return", Action: "submit"},
	}

	m := BuildKeyBindingsMap(bindings)

	if m["up"] != "move-up" {
		t.Errorf("up -> %q, want 'move-up'", m["up"])
	}
	if m["ctrl+a"] != "select-all" {
		t.Errorf("ctrl+a -> %q, want 'select-all'", m["ctrl+a"])
	}
	if m["return"] != "submit" {
		t.Errorf("return -> %q, want 'submit'", m["return"])
	}
}

func TestBuildKeyBindingsMap_WithAliases(t *testing.T) {
	bindings := []KeyBinding{
		{Name: "enter", Action: "submit"}, // alias for "return"
	}
	aliases := KeyAliasMap{"enter": "return"}

	m := BuildKeyBindingsMap(bindings, aliases)

	// Should be stored under canonical name "return"
	if m["return"] != "submit" {
		t.Errorf("return -> %q, want 'submit'", m["return"])
	}
}

func TestResolveAction(t *testing.T) {
	m := map[string]string{
		"up":     "move-up",
		"ctrl+a": "select-all",
		"return": "submit",
	}

	tests := []struct {
		ev     KeyEvent
		action string
		found  bool
	}{
		{KeyEvent{Name: "up"}, "move-up", true},
		{KeyEvent{Name: "a", Ctrl: true}, "select-all", true},
		{KeyEvent{Name: "return"}, "submit", true},
		{KeyEvent{Name: "x"}, "", false},
		{KeyEvent{Name: "a"}, "", false}, // no ctrl
	}

	for _, tt := range tests {
		action, found := ResolveAction(tt.ev, m)
		if action != tt.action || found != tt.found {
			t.Errorf("ResolveAction(%q): action=%q found=%v, want action=%q found=%v",
				tt.ev.String(), action, found, tt.action, tt.found)
		}
	}
}

func TestDefaultAliases(t *testing.T) {
	if DefaultAliases["enter"] != "return" {
		t.Error("DefaultAliases should map 'enter' to 'return'")
	}
	if DefaultAliases["esc"] != "escape" {
		t.Error("DefaultAliases should map 'esc' to 'escape'")
	}
}
