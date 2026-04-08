package input

import (
	"strings"
	"testing"
)

func TestMouseEventType_String(t *testing.T) {
	tests := []struct {
		t    MouseEventType
		want string
	}{
		{MouseDown, "mousedown"},
		{MouseUp, "mouseup"},
		{MouseMove, "mousemove"},
		{MouseDrag, "mousedrag"},
		{MouseDragEnd, "mousedragend"},
		{MouseDrop, "mousedrop"},
		{MouseOver, "mouseover"},
		{MouseOut, "mouseout"},
		{MouseScroll, "mousescroll"},
	}
	for _, tt := range tests {
		if got := tt.t.String(); got != tt.want {
			t.Errorf("MouseEventType(%d).String() = %q, want %q", tt.t, got, tt.want)
		}
	}
}

func TestMouseEvent_PropagationControl(t *testing.T) {
	ev := MouseEvent{Type: MouseDown, Button: ButtonLeft, X: 5, Y: 10}

	if ev.PropagationStopped() {
		t.Error("should not be stopped initially")
	}
	if ev.DefaultPrevented() {
		t.Error("should not be prevented initially")
	}

	ev.StopPropagation()
	if !ev.PropagationStopped() {
		t.Error("should be stopped after StopPropagation")
	}

	ev.PreventDefault()
	if !ev.DefaultPrevented() {
		t.Error("should be prevented after PreventDefault")
	}
}

func TestMouseEvent_String(t *testing.T) {
	ev := MouseEvent{
		Type:   MouseDown,
		Button: ButtonLeft,
		X:      10,
		Y:      20,
	}
	s := ev.String()
	if !strings.Contains(s, "mousedown") {
		t.Errorf("String() = %q, should contain 'mousedown'", s)
	}
	if !strings.Contains(s, "(10,20)") {
		t.Errorf("String() = %q, should contain '(10,20)'", s)
	}
}

func TestMouseEvent_String_WithModifiers(t *testing.T) {
	ev := MouseEvent{
		Type:      MouseDown,
		Button:    ButtonLeft,
		Modifiers: MouseModifiers{Ctrl: true, Alt: true},
	}
	s := ev.String()
	if !strings.Contains(s, "ctrl") {
		t.Errorf("String() = %q, should contain 'ctrl'", s)
	}
	if !strings.Contains(s, "alt") {
		t.Errorf("String() = %q, should contain 'alt'", s)
	}
}

func TestMouseEvent_String_WithScroll(t *testing.T) {
	ev := MouseEvent{
		Type:   MouseScroll,
		Button: ButtonNone,
		Scroll: &ScrollInfo{DeltaX: 0, DeltaY: -1},
	}
	s := ev.String()
	if !strings.Contains(s, "scroll") {
		t.Errorf("String() = %q, should contain 'scroll'", s)
	}
}

func TestMouseButton_Constants(t *testing.T) {
	if ButtonLeft != 0 {
		t.Error("ButtonLeft should be 0")
	}
	if ButtonMiddle != 1 {
		t.Error("ButtonMiddle should be 1")
	}
	if ButtonRight != 2 {
		t.Error("ButtonRight should be 2")
	}
	if ButtonNone != -1 {
		t.Error("ButtonNone should be -1")
	}
}
