package input

import (
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ---------------------------------------------------------------------------
// Kitty protocol flag constants
// ---------------------------------------------------------------------------

const (
	KittyFlagDisambiguate     uint = 0b00001
	KittyFlagEventTypes       uint = 0b00010
	KittyFlagAlternateKeys    uint = 0b00100
	KittyFlagAllKeysAsEscapes uint = 0b01000
	KittyFlagReportText       uint = 0b10000
)

// ---------------------------------------------------------------------------
// CSI key lookup map
// ---------------------------------------------------------------------------

var csiKeyMap = map[string]string{
	"A":   "up",
	"B":   "down",
	"C":   "right",
	"D":   "left",
	"H":   "home",
	"F":   "end",
	"2~":  "insert",
	"3~":  "delete",
	"5~":  "pageup",
	"6~":  "pagedown",
	"P":   "f1",
	"Q":   "f2",
	"R":   "f3",
	"S":   "f4",
	"15~": "f5",
	"17~": "f6",
	"18~": "f7",
	"19~": "f8",
	"20~": "f9",
	"21~": "f10",
	"23~": "f11",
	"24~": "f12",
}

// csiModifiedKeyMap maps CSI sequences with a leading 1 (modified keys) to
// key names. These follow the pattern CSI 1;mod X.
var csiModifiedKeyMap = map[byte]string{
	'A': "up",
	'B': "down",
	'C': "right",
	'D': "left",
	'H': "home",
	'F': "end",
	'P': "f1",
	'Q': "f2",
	'R': "f3",
	'S': "f4",
}

// ss3KeyMap maps SS3 (ESC O) sequences.
var ss3KeyMap = map[byte]string{
	'A': "up",
	'B': "down",
	'C': "right",
	'D': "left",
	'H': "home",
	'F': "end",
	'P': "f1",
	'Q': "f2",
	'R': "f3",
	'S': "f4",
}

// ctrlKeyMap maps byte values 1-26 to ctrl+letter.
var ctrlKeyMap = map[byte]string{
	1: "a", 2: "b", 3: "c", 4: "d", 5: "e", 6: "f", 7: "g",
	8: "h", 10: "j", 11: "k", 12: "l", 14: "n", 15: "o",
	16: "p", 17: "q", 18: "r", 19: "s", 20: "t", 21: "u",
	22: "v", 23: "w", 24: "x", 25: "y", 26: "z",
}

// kittySpecialKeys maps Kitty protocol codepoints to key names.
var kittySpecialKeys = map[int]string{
	9:     "tab",
	13:    "return",
	27:    "escape",
	32:    "space",
	127:   "backspace",
	57358: "capslock",
	57359: "scrolllock",
	57360: "numlock",
	57361: "printscreen",
	57362: "pause",
	57363: "menu",
	57376: "f13",
	57377: "f14",
	57378: "f15",
	57379: "f16",
	57380: "f17",
	57381: "f18",
	57382: "f19",
	57383: "f20",
	57399: "kp_0",
	57400: "kp_1",
	57401: "kp_2",
	57402: "kp_3",
	57403: "kp_4",
	57404: "kp_5",
	57405: "kp_6",
	57406: "kp_7",
	57407: "kp_8",
	57408: "kp_9",
	57409: "kp_decimal",
	57410: "kp_divide",
	57411: "kp_multiply",
	57412: "kp_subtract",
	57413: "kp_add",
	57414: "kp_enter",
	57415: "kp_equal",
	57416: "kp_separator",
	57417: "kp_left",
	57418: "kp_right",
	57419: "kp_up",
	57420: "kp_down",
	57421: "kp_pageup",
	57422: "kp_pagedown",
	57423: "kp_home",
	57424: "kp_end",
	57425: "kp_insert",
	57426: "kp_delete",
	57427: "kp_begin",
}

// ansiRegex is used to strip ANSI escape sequences from pasted text.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\].*?(\x1b\\|\x07)|\x1b[^[\]()]`)

// ---------------------------------------------------------------------------
// Parser
// ---------------------------------------------------------------------------

// Parser converts raw terminal input into events.
type Parser struct {
	kittyEnabled      bool
	altSendsEscPrefix bool // infer Shift from double-ESC + unmodified nav key
	buf               []byte
}

// NewParser creates a new terminal input parser.
func NewParser() *Parser {
	return &Parser{}
}

// SetKittyEnabled enables or disables Kitty keyboard protocol parsing.
func (p *Parser) SetKittyEnabled(enabled bool) {
	p.kittyEnabled = enabled
}

// SetAltSendsEscPrefix enables a heuristic for terminals where the
// Alt/Option key sends ESC-prefixed sequences for word navigation
// (ESC b / ESC f) but falls back to double-ESC + plain CSI for
// Shift+Alt+Arrow (ESC ESC [ D). When enabled, the parser infers
// Shift on double-ESC + unmodified navigation key sequences.
//
// Enable this when your terminal uses emacs-style ESC b/f for
// Alt+Arrow (e.g. macOS Terminal.app, xterm with altSendsEscape).
// Do NOT enable this for terminals that send ESC ESC [ D for plain
// Alt+Arrow (e.g. iTerm2 with "Esc+" option key).
func (p *Parser) SetAltSendsEscPrefix(enabled bool) {
	p.altSendsEscPrefix = enabled
}

// Parse parses raw input bytes into zero or more events.
// Leftover bytes from incomplete sequences are buffered internally.
func (p *Parser) Parse(data []byte) []TerminalEvent {
	p.buf = append(p.buf, data...)
	var events []TerminalEvent

	for len(p.buf) > 0 {
		ev, consumed := p.parseOne(p.buf)
		if consumed == 0 {
			// Incomplete sequence -- keep in buffer for next call
			break
		}
		p.buf = p.buf[consumed:]
		if ev.Type != EventType(-1) {
			events = append(events, ev)
		}
	}

	return events
}

// parseOne attempts to parse a single event starting at buf[0].
// Returns the event and the number of bytes consumed (0 if incomplete).
func (p *Parser) parseOne(buf []byte) (TerminalEvent, int) {
	if len(buf) == 0 {
		return TerminalEvent{Type: EventType(-1)}, 0
	}

	b := buf[0]

	// ESC sequence
	if b == 0x1b {
		return p.parseEsc(buf)
	}

	// LF (0x0a) — treat the same as CR (0x0d) so pasted newlines
	// produce KeyReturn instead of Ctrl+J.
	if b == 10 {
		return TerminalEvent{
			Type: EventKeyboard,
			Key: &KeyEvent{
				Name:   KeyReturn,
				Type:   KeyPress,
				Raw:    string(buf[:1]),
				Source: "raw",
			},
		}, 1
	}

	// Ctrl keys (0x01-0x1a, excluding 0x09=tab, 0x0a=lf, 0x0d=return, 0x1b=esc)
	if b >= 1 && b <= 26 && b != 9 && b != 10 && b != 13 {
		name, ok := ctrlKeyMap[b]
		if ok {
			return TerminalEvent{
				Type: EventKeyboard,
				Key: &KeyEvent{
					Name:   name,
					Ctrl:   true,
					Type:   KeyPress,
					Raw:    string(buf[:1]),
					Source: "raw",
				},
			}, 1
		}
	}

	// Tab
	if b == 9 {
		return TerminalEvent{
			Type: EventKeyboard,
			Key: &KeyEvent{
				Name:   KeyTab,
				Type:   KeyPress,
				Raw:    string(buf[:1]),
				Source: "raw",
			},
		}, 1
	}

	// Return/Enter
	if b == 13 {
		return TerminalEvent{
			Type: EventKeyboard,
			Key: &KeyEvent{
				Name:   KeyReturn,
				Type:   KeyPress,
				Raw:    string(buf[:1]),
				Source: "raw",
			},
		}, 1
	}

	// Backspace (0x7f)
	if b == 127 {
		return TerminalEvent{
			Type: EventKeyboard,
			Key: &KeyEvent{
				Name:   KeyBackspace,
				Type:   KeyPress,
				Raw:    string(buf[:1]),
				Source: "raw",
			},
		}, 1
	}

	// UTF-8 multibyte or ASCII printable
	return parseUTF8(buf)
}

// parseEsc handles sequences starting with ESC (0x1b).
//
// ESC is treated as an Alt modifier prefix. parseOne is called recursively
// on buf[1:] to parse the inner sequence, then Alt is set on the result.
// This naturally handles:
//   - ESC + printable      → Alt + char
//   - ESC + Ctrl-key       → Alt + Ctrl + key
//   - ESC + CSI sequence   → Alt + arrow/function/etc.  (ESC ESC [ ...)
//   - ESC + SS3 sequence   → Alt + arrow/function/etc.  (ESC ESC O ...)
//   - ESC + ESC            → Alt + Escape
func (p *Parser) parseEsc(buf []byte) (TerminalEvent, int) {
	if len(buf) < 2 {
		// Could be a lone ESC or an incomplete sequence.
		// Treat as lone ESC.
		return TerminalEvent{
			Type: EventKeyboard,
			Key: &KeyEvent{
				Name:   KeyEscape,
				Type:   KeyPress,
				Raw:    string(buf[:1]),
				Source: "raw",
			},
		}, 1
	}

	switch buf[1] {
	case '[':
		return p.parseCSI(buf)
	case 'O':
		return parseSS3(buf)
	default:
		// Treat ESC as an Alt modifier prefix: recursively parse
		// whatever follows and add Alt to the result.
		ev, consumed := p.parseOne(buf[1:])
		if consumed > 0 && ev.Type == EventKeyboard && ev.Key != nil {
			ev.Key.Alt = true
			ev.Key.Raw = string(buf[:1+consumed])

			// macOS double-ESC heuristic (opt-in): Option+Arrow sends
			// ESC b / ESC f (emacs-style), but Shift+Option+Arrow sends
			// ESC ESC [ D (double-ESC + plain CSI). When the inner
			// sequence is an ESC-initiated CSI/SS3 navigation key with
			// no modifier bits, infer Shift.
			if p.altSendsEscPrefix && buf[1] == 0x1b &&
				!ev.Key.Shift && !ev.Key.Ctrl && !ev.Key.Meta {
				if isNavigationKey(ev.Key.Name) {
					ev.Key.Shift = true
				}
			}

			return ev, 1 + consumed
		}
		// Incomplete inner sequence — wait for more data
		if consumed == 0 {
			return TerminalEvent{Type: EventType(-1)}, 0
		}
		// Inner parse produced a non-keyboard event or nil key — standalone ESC
		return TerminalEvent{
			Type: EventKeyboard,
			Key: &KeyEvent{
				Name:   KeyEscape,
				Type:   KeyPress,
				Raw:    string(buf[:1]),
				Source: "raw",
			},
		}, 1
	}
}

// ---------------------------------------------------------------------------
// CSI sequences: ESC [ ...
// ---------------------------------------------------------------------------

func (p *Parser) parseCSI(buf []byte) (TerminalEvent, int) {
	// Need at least ESC [
	if len(buf) < 3 {
		return TerminalEvent{Type: EventType(-1)}, 0
	}

	// Bracketed paste start: ESC [ 200 ~
	if len(buf) >= 6 && string(buf[2:6]) == "200~" {
		return parsePaste(buf)
	}

	// SGR mouse: ESC [ < ...
	if buf[2] == '<' {
		return parseMouse(buf)
	}

	// Collect parameter bytes and find final byte
	paramStart := 2
	i := paramStart
	for i < len(buf) {
		c := buf[i]
		// Parameter bytes: digits, semicolons, colons
		if (c >= '0' && c <= '9') || c == ';' || c == ':' {
			i++
			continue
		}
		// Final byte (uppercase letter, tilde, or lowercase for Kitty)
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '~' {
			paramStr := string(buf[paramStart:i])
			finalByte := c
			consumed := i + 1

			// Kitty CSI u
			if finalByte == 'u' && p.kittyEnabled {
				return parseKittyKey(buf[:consumed], paramStr)
			}

			return parseCSIKey(buf[:consumed], paramStr, finalByte)
		}
		// Unrecognised byte — skip the whole sequence
		return TerminalEvent{Type: EventType(-1)}, i + 1
	}

	// Incomplete sequence
	return TerminalEvent{Type: EventType(-1)}, 0
}

// parseCSIKey resolves a CSI sequence to a KeyEvent.
func parseCSIKey(raw []byte, paramStr string, finalByte byte) (TerminalEvent, int) {
	consumed := len(raw)

	// Try simple lookup: e.g. "A" -> "up", "3~" -> "delete"
	lookupKey := paramStr + string(finalByte)
	if name, ok := csiKeyMap[lookupKey]; ok {
		return TerminalEvent{
			Type: EventKeyboard,
			Key: &KeyEvent{
				Name:   name,
				Type:   KeyPress,
				Raw:    string(raw),
				Source: "raw",
			},
		}, consumed
	}

	// Try just the final byte for simple sequences
	if name, ok := csiKeyMap[string(finalByte)]; ok && paramStr == "" {
		return TerminalEvent{
			Type: EventKeyboard,
			Key: &KeyEvent{
				Name:   name,
				Type:   KeyPress,
				Raw:    string(raw),
				Source: "raw",
			},
		}, consumed
	}

	// Modified keys: CSI 1;mod X or CSI num;mod ~
	parts := strings.Split(paramStr, ";")
	if len(parts) == 2 {
		mod, err := strconv.Atoi(parts[1])
		if err == nil {
			shift, alt, ctrl, meta := decodeXtermModifiers(mod)

			// CSI 1;mod X (modified arrow/function keys)
			if parts[0] == "1" || parts[0] == "" {
				if name, ok := csiModifiedKeyMap[finalByte]; ok {
					return TerminalEvent{
						Type: EventKeyboard,
						Key: &KeyEvent{
							Name:   name,
							Ctrl:   ctrl,
							Shift:  shift,
							Alt:    alt,
							Meta:   meta,
							Type:   KeyPress,
							Raw:    string(raw),
							Source: "raw",
						},
					}, consumed
				}
			}

			// CSI num;mod ~ (modified special keys like insert, delete, pageup, etc.)
			if finalByte == '~' {
				tildeKey := parts[0] + "~"
				if name, ok := csiKeyMap[tildeKey]; ok {
					return TerminalEvent{
						Type: EventKeyboard,
						Key: &KeyEvent{
							Name:   name,
							Ctrl:   ctrl,
							Shift:  shift,
							Alt:    alt,
							Meta:   meta,
							Type:   KeyPress,
							Raw:    string(raw),
							Source: "raw",
						},
					}, consumed
				}
			}
		}
	}

	// Unknown CSI sequence
	return TerminalEvent{Type: EventType(-1)}, consumed
}

// decodeXtermModifiers converts an xterm modifier parameter to individual flags.
// The modifier value is 1 + (shift*1 + alt*2 + ctrl*4 + meta*8).
func decodeXtermModifiers(mod int) (shift, alt, ctrl, meta bool) {
	mod-- // subtract 1 to get the bitmask
	if mod < 0 {
		return
	}
	shift = mod&1 != 0
	alt = mod&2 != 0
	ctrl = mod&4 != 0
	meta = mod&8 != 0
	return
}

// ---------------------------------------------------------------------------
// SS3 sequences: ESC O ...
// ---------------------------------------------------------------------------

func parseSS3(buf []byte) (TerminalEvent, int) {
	if len(buf) < 3 {
		return TerminalEvent{Type: EventType(-1)}, 0
	}

	if name, ok := ss3KeyMap[buf[2]]; ok {
		return TerminalEvent{
			Type: EventKeyboard,
			Key: &KeyEvent{
				Name:   name,
				Type:   KeyPress,
				Raw:    string(buf[:3]),
				Source: "raw",
			},
		}, 3
	}

	// Unknown SS3 — consume 3 bytes
	return TerminalEvent{Type: EventType(-1)}, 3
}

// isNavigationKey returns true for key names that are navigation keys
// (arrows, home, end, page up/down, function keys).
func isNavigationKey(name string) bool {
	switch name {
	case "up", "down", "left", "right",
		"home", "end", "pageup", "pagedown",
		"insert", "delete",
		"f1", "f2", "f3", "f4", "f5", "f6",
		"f7", "f8", "f9", "f10", "f11", "f12":
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// SGR Mouse: ESC [ < Cb ; Cx ; Cy [Mm]
// ---------------------------------------------------------------------------

func parseMouse(buf []byte) (TerminalEvent, int) {
	// Minimum: ESC [ < d ; d ; d M = 10 bytes
	if len(buf) < 10 {
		// Try to find the end
		for i := 3; i < len(buf); i++ {
			if buf[i] == 'M' || buf[i] == 'm' {
				return parseMouseComplete(buf[:i+1])
			}
		}
		return TerminalEvent{Type: EventType(-1)}, 0
	}

	// Find the terminator M or m
	for i := 3; i < len(buf); i++ {
		if buf[i] == 'M' || buf[i] == 'm' {
			return parseMouseComplete(buf[:i+1])
		}
	}

	return TerminalEvent{Type: EventType(-1)}, 0
}

func parseMouseComplete(raw []byte) (TerminalEvent, int) {
	consumed := len(raw)
	terminator := raw[consumed-1] // 'M' = press, 'm' = release

	// Extract "Cb;Cx;Cy" between '<' and terminator
	inner := string(raw[3 : consumed-1])
	parts := strings.Split(inner, ";")
	if len(parts) != 3 {
		return TerminalEvent{Type: EventType(-1)}, consumed
	}

	cb, err1 := strconv.Atoi(parts[0])
	cx, err2 := strconv.Atoi(parts[1])
	cy, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return TerminalEvent{Type: EventType(-1)}, consumed
	}

	// Convert to 0-indexed
	cx--
	cy--

	// Decode modifiers
	mods := MouseModifiers{
		Shift: cb&4 != 0,
		Alt:   cb&8 != 0,
		Ctrl:  cb&16 != 0,
	}

	isMotion := cb&32 != 0
	isWheel := cb&64 != 0

	// Decode button
	btnBits := cb & 3

	me := &MouseEvent{
		X:         cx,
		Y:         cy,
		Modifiers: mods,
	}

	if isWheel {
		me.Type = MouseScroll
		me.Button = ButtonNone
		deltaY := 1
		if btnBits == 0 {
			deltaY = -1 // wheel up
		}
		me.Scroll = &ScrollInfo{DeltaY: deltaY}
	} else if isMotion {
		me.Type = MouseDrag
		me.IsDragging = true
		switch btnBits {
		case 0:
			me.Button = ButtonLeft
		case 1:
			me.Button = ButtonMiddle
		case 2:
			me.Button = ButtonRight
		default:
			me.Button = ButtonNone
		}
		// Pure motion (no button) is MouseMove
		if cb&3 == 3 {
			me.Type = MouseMove
			me.IsDragging = false
			me.Button = ButtonNone
		}
	} else if terminator == 'm' {
		// Release
		me.Type = MouseUp
		switch btnBits {
		case 0:
			me.Button = ButtonLeft
		case 1:
			me.Button = ButtonMiddle
		case 2:
			me.Button = ButtonRight
		default:
			me.Button = ButtonNone
		}
	} else {
		// Press
		me.Type = MouseDown
		switch btnBits {
		case 0:
			me.Button = ButtonLeft
		case 1:
			me.Button = ButtonMiddle
		case 2:
			me.Button = ButtonRight
		default:
			me.Button = ButtonNone
		}
	}

	return TerminalEvent{
		Type:  EventMouse,
		Mouse: me,
	}, consumed
}

// ---------------------------------------------------------------------------
// Kitty CSI u: ESC [ codepoint [; modifiers [: event_type]] u
// ---------------------------------------------------------------------------

func parseKittyKey(raw []byte, paramStr string) (TerminalEvent, int) {
	consumed := len(raw)

	// Split by semicolons: "codepoint", "codepoint;modifiers", "codepoint;modifiers:eventtype"
	parts := strings.Split(paramStr, ";")

	codepoint, err := strconv.Atoi(parts[0])
	if err != nil {
		return TerminalEvent{Type: EventType(-1)}, consumed
	}

	var shift, alt, ctrl, meta bool
	var eventType KeyEventType = KeyPress

	if len(parts) >= 2 {
		modParts := strings.Split(parts[1], ":")
		mod, err := strconv.Atoi(modParts[0])
		if err == nil {
			shift, alt, ctrl, meta = decodeXtermModifiers(mod)
		}
		if len(modParts) >= 2 {
			et, err := strconv.Atoi(modParts[1])
			if err == nil {
				switch et {
				case 1:
					eventType = KeyPress
				case 2:
					eventType = KeyRepeat
				case 3:
					eventType = KeyRelease
				}
			}
		}
	}

	// Resolve key name and printable text
	name := ""
	text := "" // non-empty only for printable characters
	if n, ok := kittySpecialKeys[codepoint]; ok {
		name = n
		// Space is the only kittySpecialKey that produces printable text.
		if codepoint == 32 {
			text = " "
		}
	} else if n, ok := csiKeyMap[strconv.Itoa(codepoint)+"~"]; ok {
		name = n
	} else if codepoint >= 32 && codepoint < 0x110000 {
		name = string(rune(codepoint))
		text = name // printable Unicode character
	} else {
		name = strconv.Itoa(codepoint)
	}

	return TerminalEvent{
		Type: EventKeyboard,
		Key: &KeyEvent{
			Name:     name,
			Text:     text,
			Ctrl:     ctrl,
			Shift:    shift,
			Alt:      alt,
			Meta:     meta,
			Type:     eventType,
			Code:     strconv.Itoa(codepoint),
			Raw:      string(raw),
			Source:   "kitty",
			Repeated: eventType == KeyRepeat,
		},
	}, consumed
}

// ---------------------------------------------------------------------------
// Bracketed paste: ESC [ 200 ~ ... ESC [ 201 ~
// ---------------------------------------------------------------------------

func parsePaste(buf []byte) (TerminalEvent, int) {
	// Start marker is ESC [ 200 ~ (6 bytes, already confirmed)
	// Look for end marker ESC [ 201 ~
	endMarker := "\x1b[201~"
	content := buf[6:]
	idx := strings.Index(string(content), endMarker)
	if idx < 0 {
		// Incomplete paste
		return TerminalEvent{Type: EventType(-1)}, 0
	}

	pasteText := string(content[:idx])
	consumed := 6 + idx + len(endMarker)

	// Normalize line endings: terminals send \r inside bracketed paste,
	// but the text buffer splits on \n.
	cleaned := stripANSI(pasteText)
	cleaned = strings.ReplaceAll(cleaned, "\r\n", "\n")
	cleaned = strings.ReplaceAll(cleaned, "\r", "\n")

	return TerminalEvent{
		Type: EventPaste,
		Paste: &PasteEvent{
			Text: cleaned,
		},
	}, consumed
}

// ---------------------------------------------------------------------------
// UTF-8 character
// ---------------------------------------------------------------------------

func parseUTF8(buf []byte) (TerminalEvent, int) {
	r, size := utf8.DecodeRune(buf)
	if r == utf8.RuneError && size <= 1 {
		// Invalid or incomplete UTF-8
		if size == 0 {
			return TerminalEvent{Type: EventType(-1)}, 0
		}
		// Skip invalid byte
		return TerminalEvent{Type: EventType(-1)}, 1
	}

	text := string(r) // always the literal character for printable input
	name := text
	if r == ' ' {
		name = KeySpace // symbolic name for keybinding matching
	}

	return TerminalEvent{
		Type: EventKeyboard,
		Key: &KeyEvent{
			Name:   name,
			Text:   text,
			Type:   KeyPress,
			Raw:    string(buf[:size]),
			Source: "raw",
		},
	}, size
}

// ---------------------------------------------------------------------------
// ANSI stripping
// ---------------------------------------------------------------------------

// StripANSI removes ANSI escape sequences from text.
func StripANSI(text string) string {
	return stripANSI(text)
}

func stripANSI(text string) string {
	return ansiRegex.ReplaceAllString(text, "")
}
