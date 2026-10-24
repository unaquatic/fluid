package input

import "testing"

func TestParser_PrintableASCII(t *testing.T) {
	p := NewParser()
	events := p.Parse([]byte("a"))

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	ev := events[0]
	if ev.Type != EventKeyboard || ev.Key == nil {
		t.Fatal("expected keyboard event")
	}
	if ev.Key.Name != "a" {
		t.Errorf("Name = %q, want 'a'", ev.Key.Name)
	}
	if ev.Key.Text != "a" {
		t.Errorf("Text = %q, want 'a'", ev.Key.Text)
	}
	if ev.Key.IsRune() != true {
		t.Error("'a' should be a rune")
	}
}

func TestParser_MultipleChars(t *testing.T) {
	p := NewParser()
	events := p.Parse([]byte("abc"))

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	for i, name := range []string{"a", "b", "c"} {
		if events[i].Key.Name != name {
			t.Errorf("event %d: Name = %q, want %q", i, events[i].Key.Name, name)
		}
	}
}

func TestParser_UTF8(t *testing.T) {
	p := NewParser()
	// 世 is encoded as three UTF-8 bytes.
	events := p.Parse([]byte("\xe4\xb8\x96"))

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Key.Name != "\u4e16" {
		t.Errorf("Name = %q, want '世'", events[0].Key.Name)
	}
	if events[0].Key.Text != "\u4e16" {
		t.Errorf("Text = %q, want '世'", events[0].Key.Text)
	}
}

func TestParser_Space(t *testing.T) {
	p := NewParser()
	events := p.Parse([]byte(" "))

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	ev := events[0].Key
	if ev.Name != KeySpace {
		t.Errorf("Name = %q, want %q", ev.Name, KeySpace)
	}
	if ev.Text != " " {
		t.Errorf("Text = %q, want ' '", ev.Text)
	}
	if !ev.IsRune() {
		t.Error("space should be a rune (via Text field)")
	}
	if ev.Rune() != ' ' {
		t.Errorf("Rune() = %q, want ' '", ev.Rune())
	}
}

func TestParser_Tab(t *testing.T) {
	p := NewParser()
	events := p.Parse([]byte{9})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Key.Name != KeyTab {
		t.Errorf("Name = %q, want 'tab'", events[0].Key.Name)
	}
	if events[0].Key.Text != "" {
		t.Errorf("Text = %q, want empty (tab is not printable input)", events[0].Key.Text)
	}
}

func TestParser_Return(t *testing.T) {
	p := NewParser()
	events := p.Parse([]byte{13})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Key.Name != KeyReturn {
		t.Errorf("Name = %q, want 'return'", events[0].Key.Name)
	}
}

func TestParser_Backspace(t *testing.T) {
	p := NewParser()
	events := p.Parse([]byte{127})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Key.Name != KeyBackspace {
		t.Errorf("Name = %q, want 'backspace'", events[0].Key.Name)
	}
}

func TestParser_Escape(t *testing.T) {
	p := NewParser()
	events := p.Parse([]byte{0x1b})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Key.Name != KeyEscape {
		t.Errorf("Name = %q, want 'escape'", events[0].Key.Name)
	}
}

func TestParser_CtrlKeys(t *testing.T) {
	tests := []struct {
		byte byte
		name string
	}{
		{1, "a"},  // Control-A.
		{3, "c"},  // Control-C.
		{4, "d"},  // Control-D.
		{12, "l"}, // Control-L.
		{26, "z"}, // Control-Z.
	}
	for _, tt := range tests {
		p := NewParser()
		events := p.Parse([]byte{tt.byte})
		if len(events) != 1 {
			t.Fatalf("Ctrl+%s: expected 1 event, got %d", tt.name, len(events))
		}
		ev := events[0].Key
		if ev.Name != tt.name || !ev.Mods.Has(ModCtrl) {
			t.Errorf("byte %d: Name=%q Ctrl=%v, want Name=%q Ctrl=true",
				tt.byte, ev.Name, ev.Mods.Has(ModCtrl), tt.name)
		}
	}
}

func TestParser_ArrowKeys(t *testing.T) {
	tests := []struct {
		seq  string
		name string
	}{
		{"\x1b[A", "up"},
		{"\x1b[B", "down"},
		{"\x1b[C", "right"},
		{"\x1b[D", "left"},
	}
	for _, tt := range tests {
		p := NewParser()
		events := p.Parse([]byte(tt.seq))
		if len(events) != 1 {
			t.Fatalf("%s: expected 1 event, got %d", tt.name, len(events))
		}
		if events[0].Key.Name != tt.name {
			t.Errorf("seq %q: Name = %q, want %q", tt.seq, events[0].Key.Name, tt.name)
		}
	}
}

func TestParser_HomeEnd(t *testing.T) {
	tests := []struct {
		seq  string
		name string
	}{
		{"\x1b[H", "home"},
		{"\x1b[F", "end"},
	}
	for _, tt := range tests {
		p := NewParser()
		events := p.Parse([]byte(tt.seq))
		if len(events) != 1 {
			t.Fatalf("%s: expected 1 event, got %d", tt.name, len(events))
		}
		if events[0].Key.Name != tt.name {
			t.Errorf("Name = %q, want %q", events[0].Key.Name, tt.name)
		}
	}
}

func TestParser_FunctionKeys(t *testing.T) {
	tests := []struct {
		seq  string
		name string
	}{
		{"\x1b[P", "f1"},
		{"\x1b[Q", "f2"},
		{"\x1b[R", "f3"},
		{"\x1b[S", "f4"},
		{"\x1b[15~", "f5"},
		{"\x1b[17~", "f6"},
		{"\x1b[18~", "f7"},
		{"\x1b[19~", "f8"},
		{"\x1b[20~", "f9"},
		{"\x1b[21~", "f10"},
		{"\x1b[23~", "f11"},
		{"\x1b[24~", "f12"},
	}
	for _, tt := range tests {
		p := NewParser()
		events := p.Parse([]byte(tt.seq))
		if len(events) != 1 {
			t.Fatalf("%s: expected 1 event, got %d", tt.name, len(events))
		}
		if events[0].Key.Name != tt.name {
			t.Errorf("seq %q: Name = %q, want %q", tt.seq, events[0].Key.Name, tt.name)
		}
	}
}

func TestParser_InsertDelete(t *testing.T) {
	tests := []struct {
		seq  string
		name string
	}{
		{"\x1b[2~", "insert"},
		{"\x1b[3~", "delete"},
	}
	for _, tt := range tests {
		p := NewParser()
		events := p.Parse([]byte(tt.seq))
		if len(events) != 1 {
			t.Fatalf("%s: expected 1 event, got %d", tt.name, len(events))
		}
		if events[0].Key.Name != tt.name {
			t.Errorf("Name = %q, want %q", events[0].Key.Name, tt.name)
		}
	}
}

func TestParser_PageUpDown(t *testing.T) {
	tests := []struct {
		seq  string
		name string
	}{
		{"\x1b[5~", "pageup"},
		{"\x1b[6~", "pagedown"},
	}
	for _, tt := range tests {
		p := NewParser()
		events := p.Parse([]byte(tt.seq))
		if len(events) != 1 {
			t.Fatalf("%s: expected 1 event, got %d", tt.name, len(events))
		}
		if events[0].Key.Name != tt.name {
			t.Errorf("Name = %q, want %q", events[0].Key.Name, tt.name)
		}
	}
}

func TestParser_ModifiedArrowKeys(t *testing.T) {
	tests := []struct {
		seq   string
		name  string
		ctrl  bool
		shift bool
		alt   bool
	}{
		{"\x1b[1;2A", "up", false, true, false},    // Shift+Up.
		{"\x1b[1;3B", "down", false, false, true},  // Alt+Down.
		{"\x1b[1;5C", "right", true, false, false}, // Control+Right.
		{"\x1b[1;6D", "left", true, true, false},   // Control+Shift+Left.
		{"\x1b[1;7H", "home", true, false, true},   // Control+Alt+Home.
	}
	for _, tt := range tests {
		p := NewParser()
		events := p.Parse([]byte(tt.seq))
		if len(events) != 1 {
			t.Fatalf("%q: expected 1 event, got %d", tt.seq, len(events))
		}
		ev := events[0].Key
		if ev.Name != tt.name || ev.Mods.Has(ModCtrl) != tt.ctrl || ev.Mods.Has(ModShift) != tt.shift || ev.Mods.Has(ModAlt) != tt.alt {
			t.Errorf("seq %q: got Name=%q Ctrl=%v Shift=%v Alt=%v, want Name=%q Ctrl=%v Shift=%v Alt=%v",
				tt.seq, ev.Name, ev.Mods.Has(ModCtrl), ev.Mods.Has(ModShift), ev.Mods.Has(ModAlt),
				tt.name, tt.ctrl, tt.shift, tt.alt)
		}
	}
}

func TestParser_ModifiedTildeKeys(t *testing.T) {
	// CSI 3;2~ encodes Shift+Delete.
	p := NewParser()
	events := p.Parse([]byte("\x1b[3;2~"))
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	ev := events[0].Key
	if ev.Name != "delete" || !ev.Mods.Has(ModShift) {
		t.Errorf("got Name=%q Shift=%v, want delete Shift=true", ev.Name, ev.Mods.Has(ModShift))
	}
}

func TestParser_SS3_ArrowKeys(t *testing.T) {
	tests := []struct {
		seq  string
		name string
	}{
		{"\x1bOA", "up"},
		{"\x1bOB", "down"},
		{"\x1bOC", "right"},
		{"\x1bOD", "left"},
		{"\x1bOH", "home"},
		{"\x1bOF", "end"},
		{"\x1bOP", "f1"},
		{"\x1bOQ", "f2"},
	}
	for _, tt := range tests {
		p := NewParser()
		events := p.Parse([]byte(tt.seq))
		if len(events) != 1 {
			t.Fatalf("SS3 %s: expected 1 event, got %d", tt.name, len(events))
		}
		if events[0].Key.Name != tt.name {
			t.Errorf("SS3 %q: Name = %q, want %q", tt.seq, events[0].Key.Name, tt.name)
		}
	}
}

func TestParser_AltKeys(t *testing.T) {
	tests := []struct {
		seq  string
		name string
	}{
		{"\x1ba", "a"},
		{"\x1bz", "z"},
		{"\x1b1", "1"},
	}
	for _, tt := range tests {
		p := NewParser()
		events := p.Parse([]byte(tt.seq))
		if len(events) != 1 {
			t.Fatalf("Alt+%s: expected 1 event, got %d", tt.name, len(events))
		}
		ev := events[0].Key
		if ev.Name != tt.name || !ev.Mods.Has(ModAlt) {
			t.Errorf("Alt+%s: Name=%q Alt=%v, want Alt=true", tt.name, ev.Name, ev.Mods.Has(ModAlt))
		}
	}
}

func TestParser_AltBackspace(t *testing.T) {
	p := NewParser()
	events := p.Parse([]byte{0x1b, 127})
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	ev := events[0].Key
	if ev.Name != KeyBackspace || !ev.Mods.Has(ModAlt) {
		t.Errorf("got Name=%q Alt=%v, want backspace Alt=true", ev.Name, ev.Mods.Has(ModAlt))
	}
}

func TestParser_AltReturn(t *testing.T) {
	p := NewParser()
	events := p.Parse([]byte{0x1b, 13})
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	ev := events[0].Key
	if ev.Name != KeyReturn || !ev.Mods.Has(ModAlt) {
		t.Errorf("got Name=%q Alt=%v, want return Alt=true", ev.Name, ev.Mods.Has(ModAlt))
	}
}

func TestParser_MouseDown_Left(t *testing.T) {
	// SGR coordinates 10,20 become zero-based coordinates 9,19.
	p := NewParser()
	events := p.Parse([]byte("\x1b[<0;10;20M"))
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	ev := events[0]
	if ev.Type != EventMouse || ev.Mouse == nil {
		t.Fatal("expected mouse event")
	}
	m := ev.Mouse
	if m.Type != MouseDown {
		t.Errorf("Type = %v, want MouseDown", m.Type)
	}
	if m.Button != ButtonLeft {
		t.Errorf("Button = %d, want ButtonLeft", m.Button)
	}
	if m.X != 9 || m.Y != 19 {
		t.Errorf("position = (%d,%d), want (9,19)", m.X, m.Y)
	}
}

func TestParser_MouseUp_Left(t *testing.T) {
	// A lowercase SGR terminator reports button release.
	p := NewParser()
	events := p.Parse([]byte("\x1b[<0;5;5m"))
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	m := events[0].Mouse
	if m.Type != MouseUp {
		t.Errorf("Type = %v, want MouseUp", m.Type)
	}
	if m.Button != ButtonLeft {
		t.Errorf("Button = %d, want ButtonLeft", m.Button)
	}
}

func TestParser_MouseDown_Right(t *testing.T) {
	// Button value two reports a right-button press.
	p := NewParser()
	events := p.Parse([]byte("\x1b[<2;1;1M"))
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	m := events[0].Mouse
	if m.Button != ButtonRight {
		t.Errorf("Button = %d, want ButtonRight", m.Button)
	}
}

func TestParser_MouseDown_Middle(t *testing.T) {
	p := NewParser()
	events := p.Parse([]byte("\x1b[<1;1;1M"))
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	m := events[0].Mouse
	if m.Button != ButtonMiddle {
		t.Errorf("Button = %d, want ButtonMiddle", m.Button)
	}
}

func TestParser_MouseScroll_Up(t *testing.T) {
	// Button mask 64 reports upward wheel movement.
	p := NewParser()
	events := p.Parse([]byte("\x1b[<64;1;1M"))
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	m := events[0].Mouse
	if m.Type != MouseScroll {
		t.Errorf("Type = %v, want MouseScroll", m.Type)
	}
	if m.Scroll == nil || m.Scroll.DeltaY != -1 {
		t.Errorf("Scroll.DeltaY = %v, want -1", m.Scroll)
	}
}

func TestParser_MouseScroll_Down(t *testing.T) {
	// Button mask 65 reports downward wheel movement.
	p := NewParser()
	events := p.Parse([]byte("\x1b[<65;1;1M"))
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	m := events[0].Mouse
	if m.Type != MouseScroll {
		t.Errorf("Type = %v, want MouseScroll", m.Type)
	}
	if m.Scroll == nil || m.Scroll.DeltaY != 1 {
		t.Errorf("Scroll.DeltaY = %v, want 1", m.Scroll)
	}
}

func TestParser_MouseDrag(t *testing.T) {
	// Button mask 32 reports motion while the left button is held.
	p := NewParser()
	events := p.Parse([]byte("\x1b[<32;5;5M"))
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	m := events[0].Mouse
	if m.Type != MouseDrag {
		t.Errorf("Type = %v, want MouseDrag", m.Type)
	}
	if !m.IsDragging {
		t.Error("IsDragging should be true")
	}
	if m.Button != ButtonLeft {
		t.Errorf("Button = %d, want ButtonLeft", m.Button)
	}
}

func TestParser_MouseMove(t *testing.T) {
	// Button mask 35 reports motion without a held button.
	p := NewParser()
	events := p.Parse([]byte("\x1b[<35;5;5M"))
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	m := events[0].Mouse
	if m.Type != MouseMove {
		t.Errorf("Type = %v, want MouseMove", m.Type)
	}
	if m.IsDragging {
		t.Error("IsDragging should be false for move")
	}
}

func TestParser_MouseModifiers(t *testing.T) {
	// Bit 2 reports Shift.
	p := NewParser()
	events := p.Parse([]byte("\x1b[<4;1;1M"))
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	m := events[0].Mouse
	if !m.Mods.Has(ModShift) {
		t.Error("Shift should be true")
	}

	// Bit 3 reports Alt.
	p = NewParser()
	events = p.Parse([]byte("\x1b[<8;1;1M"))
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	m = events[0].Mouse
	if !m.Mods.Has(ModAlt) {
		t.Error("Alt should be true")
	}

	// Bit 4 reports Control.
	p = NewParser()
	events = p.Parse([]byte("\x1b[<16;1;1M"))
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	m = events[0].Mouse
	if !m.Mods.Has(ModCtrl) {
		t.Error("Ctrl should be true")
	}
}

func TestParser_BracketedPaste(t *testing.T) {
	p := NewParser()
	paste := "\x1b[200~Hello World\x1b[201~"
	events := p.Parse([]byte(paste))

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	ev := events[0]
	if ev.Type != EventPaste || ev.Paste == nil {
		t.Fatal("expected paste event")
	}
	if ev.Paste.Text != "Hello World" {
		t.Errorf("Text = %q, want 'Hello World'", ev.Paste.Text)
	}
}

func TestParser_BracketedPaste_WithANSI(t *testing.T) {
	p := NewParser()
	// ANSI styling embedded in pasted text is removed.
	paste := "\x1b[200~Hello \x1b[31mRed\x1b[0m World\x1b[201~"
	events := p.Parse([]byte(paste))

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Paste.Text != "Hello Red World" {
		t.Errorf("Text = %q, want 'Hello Red World'", events[0].Paste.Text)
	}
}

func TestParser_BracketedPaste_Incomplete(t *testing.T) {
	p := NewParser()
	events := p.Parse([]byte("\x1b[200~Hello"))
	if len(events) != 0 {
		t.Errorf("incomplete paste should return 0 events, got %d", len(events))
	}

	// The next call completes the buffered paste.
	events = p.Parse([]byte(" World\x1b[201~"))
	if len(events) != 1 {
		t.Fatalf("completed paste: expected 1 event, got %d", len(events))
	}
	if events[0].Paste.Text != "Hello World" {
		t.Errorf("Text = %q, want 'Hello World'", events[0].Paste.Text)
	}
}

func TestParser_Kitty_SimpleKey(t *testing.T) {
	p := NewParser()
	p.SetKittyEnabled(true)
	// Kitty CSI 97u reports Unicode code point 97, "a".
	events := p.Parse([]byte("\x1b[97u"))

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	ev := events[0].Key
	if ev.Name != "a" {
		t.Errorf("Name = %q, want 'a'", ev.Name)
	}
	if ev.Source != KeySourceKitty {
		t.Errorf("Source = %v, want kitty", ev.Source)
	}
}

func TestParser_Kitty_WithModifiers(t *testing.T) {
	p := NewParser()
	p.SetKittyEnabled(true)
	// Kitty modifier value 5 encodes Control-A as 1 plus the Control bit.
	events := p.Parse([]byte("\x1b[97;5u"))

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	ev := events[0].Key
	if ev.Name != "a" || !ev.Mods.Has(ModCtrl) {
		t.Errorf("Name=%q Ctrl=%v, want 'a' Ctrl=true", ev.Name, ev.Mods.Has(ModCtrl))
	}
}

func TestParser_Kitty_EventType(t *testing.T) {
	p := NewParser()
	p.SetKittyEnabled(true)

	// Kitty event type 2 reports a repeated key.
	events := p.Parse([]byte("\x1b[97;1:2u"))
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	ev := events[0].Key
	if ev.Type != KeyRepeat {
		t.Errorf("Type = %v, want KeyRepeat", ev.Type)
	}

	// Kitty event type 3 reports a released key.
	events = p.Parse([]byte("\x1b[97;1:3u"))
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Key.Type != KeyRelease {
		t.Errorf("Type = %v, want KeyRelease", events[0].Key.Type)
	}
}

func TestParser_Kitty_SpecialKeys(t *testing.T) {
	p := NewParser()
	p.SetKittyEnabled(true)

	tests := []struct {
		seq      string
		name     string
		wantText string // The expected printable text, if any.
	}{
		{"\x1b[13u", "return", ""},
		{"\x1b[27u", "escape", ""},
		{"\x1b[9u", "tab", ""},
		{"\x1b[127u", "backspace", ""},
		{"\x1b[32u", "space", " "},
	}
	for _, tt := range tests {
		events := p.Parse([]byte(tt.seq))
		if len(events) != 1 {
			t.Fatalf("Kitty %s: expected 1 event, got %d", tt.name, len(events))
		}
		if events[0].Key.Name != tt.name {
			t.Errorf("Kitty %q: Name = %q, want %q", tt.seq, events[0].Key.Name, tt.name)
		}
		if events[0].Key.Text != tt.wantText {
			t.Errorf("Kitty %q: Text = %q, want %q", tt.seq, events[0].Key.Text, tt.wantText)
		}
	}
}

func TestParser_Kitty_Disabled(t *testing.T) {
	p := NewParser()
	// CSI u must not produce a Kitty event while the protocol is disabled.
	events := p.Parse([]byte("\x1b[97u"))
	if len(events) == 1 && events[0].Key != nil && events[0].Key.Source == KeySourceKitty {
		t.Error("kitty parsing should be disabled by default")
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello", "Hello"},
		{"\x1b[31mRed\x1b[0m", "Red"},
		{"\x1b[1;32mBold Green\x1b[0m text", "Bold Green text"},
		{"no escapes here", "no escapes here"},
		{"", ""},
	}
	for _, tt := range tests {
		got := StripANSI(tt.input)
		if got != tt.want {
			t.Errorf("StripANSI(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParser_EmptyInput(t *testing.T) {
	p := NewParser()
	events := p.Parse([]byte{})
	if len(events) != 0 {
		t.Errorf("empty input should return 0 events, got %d", len(events))
	}
}

func TestParser_IncrementalInput(t *testing.T) {
	p := NewParser()

	// Buffer an incomplete CSI sequence.
	events := p.Parse([]byte{0x1b, '['})
	if len(events) != 0 {
		t.Errorf("incomplete CSI: expected 0 events, got %d", len(events))
	}

	// Complete the buffered sequence in the next call.
	events = p.Parse([]byte{'A'})
	if len(events) != 1 {
		t.Fatalf("completed CSI: expected 1 event, got %d", len(events))
	}
	if events[0].Key.Name != "up" {
		t.Errorf("Name = %q, want 'up'", events[0].Key.Name)
	}
}

func TestParser_MixedInput(t *testing.T) {
	p := NewParser()
	// Parse printable keys on both sides of an Up sequence.
	events := p.Parse([]byte("a\x1b[Ab"))

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].Key.Name != "a" {
		t.Errorf("event 0: Name = %q, want 'a'", events[0].Key.Name)
	}
	if events[1].Key.Name != "up" {
		t.Errorf("event 1: Name = %q, want 'up'", events[1].Key.Name)
	}
	if events[2].Key.Name != "b" {
		t.Errorf("event 2: Name = %q, want 'b'", events[2].Key.Name)
	}
}

func TestParser_DoubleEsc_CSI_AltArrows(t *testing.T) {
	// With the heuristic disabled, double ESC adds Alt without inferring Shift.
	tests := []struct {
		seq  string
		name string
		desc string
	}{
		{"\x1b\x1b[D", "left", "Option+Left"},
		{"\x1b\x1b[C", "right", "Option+Right"},
		{"\x1b\x1b[A", "up", "Option+Up"},
		{"\x1b\x1b[B", "down", "Option+Down"},
	}
	for _, tt := range tests {
		p := NewParser()
		events := p.Parse([]byte(tt.seq))
		if len(events) != 1 {
			t.Fatalf("%s (%q): expected 1 event, got %d", tt.desc, tt.seq, len(events))
		}
		ev := events[0].Key
		if ev.Name != tt.name {
			t.Errorf("%s: Name = %q, want %q", tt.desc, ev.Name, tt.name)
		}
		if !ev.Mods.Has(ModAlt) {
			t.Errorf("%s: Alt = false, want true", tt.desc)
		}
		if ev.Mods.Has(ModShift) || ev.Mods.Has(ModCtrl) || ev.Mods.Has(ModMeta) {
			t.Errorf("%s: unexpected modifiers Shift=%v Ctrl=%v Meta=%v",
				tt.desc, ev.Mods.Has(ModShift), ev.Mods.Has(ModCtrl), ev.Mods.Has(ModMeta))
		}
	}
}

func TestParser_DoubleEsc_CSI_AltSendsEscPrefix(t *testing.T) {
	// With the heuristic enabled, double-ESC navigation infers Shift because
	// plain Alt+Arrow is expected to use ESC b or ESC f instead.
	tests := []struct {
		seq  string
		name string
		desc string
	}{
		{"\x1b\x1b[D", "left", "Shift+Option+Left"},
		{"\x1b\x1b[C", "right", "Shift+Option+Right"},
		{"\x1b\x1b[A", "up", "Shift+Option+Up"},
		{"\x1b\x1b[B", "down", "Shift+Option+Down"},
	}
	for _, tt := range tests {
		p := NewParser()
		p.SetAltSendsEscPrefix(true)
		events := p.Parse([]byte(tt.seq))
		if len(events) != 1 {
			t.Fatalf("%s (%q): expected 1 event, got %d", tt.desc, tt.seq, len(events))
		}
		ev := events[0].Key
		if ev.Name != tt.name {
			t.Errorf("%s: Name = %q, want %q", tt.desc, ev.Name, tt.name)
		}
		if !ev.Mods.Has(ModAlt) {
			t.Errorf("%s: Alt = false, want true", tt.desc)
		}
		if !ev.Mods.Has(ModShift) {
			t.Errorf("%s: Shift = false, want true (macOS heuristic)", tt.desc)
		}
		if ev.Mods.Has(ModCtrl) || ev.Mods.Has(ModMeta) {
			t.Errorf("%s: unexpected modifiers Ctrl=%v Meta=%v", tt.desc, ev.Mods.Has(ModCtrl), ev.Mods.Has(ModMeta))
		}
	}
}

func TestParser_DoubleEsc_CSI_ShiftAltArrows(t *testing.T) {
	// In ESC ESC [1;2X, the inner modifier explicitly sets Shift and the outer
	// ESC adds Alt.
	tests := []struct {
		seq   string
		name  string
		shift bool
		desc  string
	}{
		{"\x1b\x1b[1;2D", "left", true, "Shift+Option+Left"},
		{"\x1b\x1b[1;2C", "right", true, "Shift+Option+Right"},
		{"\x1b\x1b[1;2A", "up", true, "Shift+Option+Up"},
		{"\x1b\x1b[1;2B", "down", true, "Shift+Option+Down"},
	}
	for _, tt := range tests {
		p := NewParser()
		events := p.Parse([]byte(tt.seq))
		if len(events) != 1 {
			t.Fatalf("%s (%q): expected 1 event, got %d", tt.desc, tt.seq, len(events))
		}
		ev := events[0].Key
		if ev.Name != tt.name {
			t.Errorf("%s: Name = %q, want %q", tt.desc, ev.Name, tt.name)
		}
		if !ev.Mods.Has(ModAlt) {
			t.Errorf("%s: Alt = false, want true", tt.desc)
		}
		if ev.Mods.Has(ModShift) != tt.shift {
			t.Errorf("%s: Shift = %v, want %v", tt.desc, ev.Mods.Has(ModShift), tt.shift)
		}
	}
}

func TestParser_DoubleEsc_SS3(t *testing.T) {
	// With the heuristic disabled, a nested SS3 key gains Alt but not Shift.
	tests := []struct {
		seq  string
		name string
		desc string
	}{
		{"\x1b\x1bOA", "up", "Alt+SS3 Up"},
		{"\x1b\x1bOD", "left", "Alt+SS3 Left"},
		{"\x1b\x1bOH", "home", "Alt+SS3 Home"},
	}
	for _, tt := range tests {
		p := NewParser()
		events := p.Parse([]byte(tt.seq))
		if len(events) != 1 {
			t.Fatalf("%s (%q): expected 1 event, got %d", tt.desc, tt.seq, len(events))
		}
		ev := events[0].Key
		if ev.Name != tt.name {
			t.Errorf("%s: Name = %q, want %q", tt.desc, ev.Name, tt.name)
		}
		if !ev.Mods.Has(ModAlt) {
			t.Errorf("%s: Alt = false, want true", tt.desc)
		}
		if ev.Mods.Has(ModShift) {
			t.Errorf("%s: Shift = true, want false (heuristic off by default)", tt.desc)
		}
	}
}

func TestParser_DoubleEsc_SS3_AltSendsEscPrefix(t *testing.T) {
	// With the heuristic enabled, nested SS3 navigation also infers Shift.
	p := NewParser()
	p.SetAltSendsEscPrefix(true)
	events := p.Parse([]byte("\x1b\x1bOD"))
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	ev := events[0].Key
	if ev.Name != "left" || !ev.Mods.Has(ModAlt) || !ev.Mods.Has(ModShift) {
		t.Errorf("got Name=%q Alt=%v Shift=%v, want left Alt=true Shift=true",
			ev.Name, ev.Mods.Has(ModAlt), ev.Mods.Has(ModShift))
	}
}

func TestParser_DoubleEsc_InStream(t *testing.T) {
	// A nested CSI sequence remains one event within a larger input stream.
	p := NewParser()
	events := p.Parse([]byte("a\x1b\x1b[Db"))
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].Key.Name != "a" {
		t.Errorf("event 0: Name = %q, want 'a'", events[0].Key.Name)
	}
	if events[1].Key.Name != "left" || !events[1].Key.Mods.Has(ModAlt) {
		t.Errorf("event 1: Name=%q Alt=%v, want left Alt=true",
			events[1].Key.Name, events[1].Key.Mods.Has(ModAlt))
	}
	if events[2].Key.Name != "b" {
		t.Errorf("event 2: Name = %q, want 'b'", events[2].Key.Name)
	}
}

func TestParser_DoubleEsc_NonNavKey_NoShift(t *testing.T) {
	// The Shift heuristic applies only to double-ESC navigation, not Alt+A.
	p := NewParser()
	events := p.Parse([]byte("\x1ba"))
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	ev := events[0].Key
	if ev.Name != "a" || !ev.Mods.Has(ModAlt) {
		t.Errorf("got Name=%q Alt=%v, want a Alt=true", ev.Name, ev.Mods.Has(ModAlt))
	}
	if ev.Mods.Has(ModShift) {
		t.Error("Shift should be false for Alt+a (not a double-ESC nav key)")
	}
}

func TestParser_DoubleEsc_StandaloneEscEsc(t *testing.T) {
	// With no following key, the first ESC adds Alt to the second standalone
	// Escape key.
	p := NewParser()
	events := p.Parse([]byte("\x1b\x1b"))
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Key.Name != "escape" {
		t.Errorf("event 0: Name = %q, want 'escape'", events[0].Key.Name)
	}
	if !events[0].Key.Mods.Has(ModAlt) {
		t.Errorf("event 0: Alt = false, want true")
	}
}
