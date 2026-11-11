package terminal

import (
	"testing"
	"unicode/utf16"
	"unsafe"
)

// keyRecord returns a record of a key event of the UTF-16 unit c, a press
// when down is set and a release otherwise.
func keyRecord(c uint16, down bool) ConsoleRecord {
	r := ConsoleRecord{EventType: ConsoleKeyEvent}
	if down {
		r.Event[0] = 1
	}
	r.Event[2] = uint32(c) << 16
	return r
}

// typed returns the key press records of s, one for each UTF-16 unit, as a
// console in virtual terminal input mode sends them.
func typed(s string) []ConsoleRecord {
	var records []ConsoleRecord
	for _, c := range utf16.Encode([]rune(s)) {
		records = append(records, keyRecord(c, true))
	}
	return records
}

func TestConsoleDecoder(t *testing.T) {
	smile := utf16.Encode([]rune("😀"))
	tests := []struct {
		name    string
		records []ConsoleRecord
		want    string
		resized bool
	}{
		{name: "text", records: typed("hé"), want: "hé"},
		{name: "escape sequence", records: typed("\x1b[A"), want: "\x1b[A"},
		{name: "ctrl-z", records: typed("a\x1ab"), want: "a\x1ab"},
		{name: "ctrl-space", records: typed("\x00"), want: "\x00"},
		{name: "ctrl-c", records: typed("\x03"), want: "\x03"},
		{name: "releases", records: []ConsoleRecord{keyRecord('a', true), keyRecord('a', false)}, want: "a"},
		{name: "surrogate pair", records: typed("x😀y"), want: "x😀y"},
		{name: "surrogate pair released", records: []ConsoleRecord{keyRecord(smile[0], false), keyRecord(smile[1], false)}, want: "😀"},
		{name: "lone high surrogate", records: []ConsoleRecord{keyRecord(smile[0], true), keyRecord('a', true)}, want: "\uFFFDa"},
		{name: "lone low surrogate", records: []ConsoleRecord{keyRecord(smile[1], true), keyRecord('a', true)}, want: "\uFFFDa"},
		{name: "two high surrogates", records: []ConsoleRecord{keyRecord(smile[0], true), keyRecord(smile[0], true), keyRecord(smile[1], true)}, want: "\uFFFD😀"},
		{name: "resize", records: []ConsoleRecord{{EventType: ConsoleBufferSizeEvent}}, resized: true},
		{name: "other events", records: []ConsoleRecord{{EventType: 0x0002}, {EventType: 0x0010}, keyRecord('a', true)}, want: "a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d ConsoleDecoder
			got, resized := d.Decode(nil, tt.records)
			if string(got) != tt.want || resized != tt.resized {
				t.Errorf("Decode = %q, %v; want %q, %v", got, resized, tt.want, tt.resized)
			}
		})
	}
}

// TestConsoleDecoderJoinsAcrossCalls checks that a surrogate pair whose
// halves come in separate reads decodes once, when its second half comes.
func TestConsoleDecoderJoinsAcrossCalls(t *testing.T) {
	records := typed("😀")
	var d ConsoleDecoder
	got, _ := d.Decode(nil, records[:1])
	if len(got) != 0 {
		t.Fatalf("first half decoded to %q; want nothing yet", got)
	}
	got, _ = d.Decode(got, records[1:])
	if string(got) != "😀" {
		t.Errorf("second half decoded to %q; want %q", got, "😀")
	}
}

// TestConsoleRecordLayout checks that a ConsoleRecord is laid out as the
// INPUT_RECORD that ReadConsoleInputW fills: 20 bytes, aligned to 4, with
// its event at offset 4.
func TestConsoleRecordLayout(t *testing.T) {
	var r ConsoleRecord
	if size, align, off := unsafe.Sizeof(r), unsafe.Alignof(r), unsafe.Offsetof(r.Event); size != 20 || align != 4 || off != 4 {
		t.Errorf("size, alignment, Event offset = %d, %d, %d; want 20, 4, 4", size, align, off)
	}
}
