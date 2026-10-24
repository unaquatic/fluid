package input

import (
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Kitty keyboard protocol feature flags.
const (
	KittyFlagDisambiguate     uint = 0b00001
	KittyFlagEventTypes       uint = 0b00010
	KittyFlagAlternateKeys    uint = 0b00100
	KittyFlagAllKeysAsEscapes uint = 0b01000
	KittyFlagReportText       uint = 0b10000
)

// csiKeyMap maps complete CSI key suffixes to canonical key names.
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

// csiModifiedKeyMap maps the final byte in a CSI 1;modifier key sequence to a
// canonical key name.
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

// ss3KeyMap maps the final byte in an SS3 sequence to a canonical key name.
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

// ctrlKeyMap maps supported C0 control bytes to their Control-letter names.
var ctrlKeyMap = map[byte]string{
	1: "a", 2: "b", 3: "c", 4: "d", 5: "e", 6: "f", 7: "g",
	8: "h", 10: "j", 11: "k", 12: "l", 14: "n", 15: "o",
	16: "p", 17: "q", 18: "r", 19: "s", 20: "t", 21: "u",
	22: "v", 23: "w", 24: "x", 25: "y", 26: "z",
}

// kittySpecialKeys maps Kitty keyboard protocol code points to canonical key
// names.
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

// ansiRegex matches the ANSI escape sequences removed from pasted text.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\].*?(\x1b\\|\x07)|\x1b[^[\]()]`)

// Parser converts terminal input bytes into events and retains incomplete
// control sequences between Parse calls.
type Parser struct {
	kittyEnabled      bool
	altSendsEscPrefix bool // Whether to infer Shift from double-ESC navigation sequences.
	buf               []byte
}

// NewParser returns a terminal input parser with optional protocols disabled.
func NewParser() *Parser {
	return &Parser{}
}

// SetKittyEnabled controls Kitty keyboard protocol parsing.
func (p *Parser) SetKittyEnabled(enabled bool) {
	p.kittyEnabled = enabled
}

// SetAltSendsEscPrefix controls a heuristic for terminals that encode Alt as
// an ESC prefix. When enabled, an unmodified navigation key inside a second ESC
// prefix is interpreted as Shift+Alt, as in ESC ESC [ D.
//
// Enable it when plain Alt+Arrow uses Emacs-style ESC b or ESC f sequences.
// Leave it disabled when plain Alt+Arrow itself uses a double-ESC sequence.
func (p *Parser) SetAltSendsEscPrefix(enabled bool) {
	p.altSendsEscPrefix = enabled
}

// Parse converts data into zero or more terminal events. It buffers an
// incomplete control sequence for the next call.
func (p *Parser) Parse(data []byte) []TerminalEvent {
	p.buf = append(p.buf, data...)
	var events []TerminalEvent

	for len(p.buf) > 0 {
		ev, consumed := p.parseOne(p.buf)
		if consumed == 0 {
			// Retain the incomplete sequence for the next Parse call.
			break
		}
		p.buf = p.buf[consumed:]
		if ev.Type != EventType(-1) {
			events = append(events, ev)
		}
	}

	return events
}

// parseOne parses the event at the start of buf. It returns a zero byte count
// for incomplete input and an EventType of -1 for consumed input that does not
// produce an event.
func (p *Parser) parseOne(buf []byte) (TerminalEvent, int) {
	if len(buf) == 0 {
		return TerminalEvent{Type: EventType(-1)}, 0
	}

	b := buf[0]

	// ESC begins a terminal control sequence or an Alt-prefixed key.
	if b == 0x1b {
		return p.parseEsc(buf)
	}

	// Treat line feed as Return rather than Control-J.
	if b == 10 {
		return TerminalEvent{
			Type: EventKeyboard,
			Key: &KeyEvent{
				Name:   KeyReturn,
				Type:   KeyPress,
				Raw:    string(buf[:1]),
				Source: KeySourceRaw,
			},
		}, 1
	}

	// Map supported C0 bytes to Control-letter events, excluding dedicated keys.
	if b >= 1 && b <= 26 && b != 9 && b != 10 && b != 13 {
		name, ok := ctrlKeyMap[b]
		if ok {
			return TerminalEvent{
				Type: EventKeyboard,
				Key: &KeyEvent{
					Name:   name,
					Mods:   ModCtrl,
					Type:   KeyPress,
					Raw:    string(buf[:1]),
					Source: KeySourceRaw,
				},
			}, 1
		}
	}

	if b == 9 {
		return TerminalEvent{
			Type: EventKeyboard,
			Key: &KeyEvent{
				Name:   KeyTab,
				Type:   KeyPress,
				Raw:    string(buf[:1]),
				Source: KeySourceRaw,
			},
		}, 1
	}

	if b == 13 {
		return TerminalEvent{
			Type: EventKeyboard,
			Key: &KeyEvent{
				Name:   KeyReturn,
				Type:   KeyPress,
				Raw:    string(buf[:1]),
				Source: KeySourceRaw,
			},
		}, 1
	}

	if b == 127 {
		return TerminalEvent{
			Type: EventKeyboard,
			Key: &KeyEvent{
				Name:   KeyBackspace,
				Type:   KeyPress,
				Raw:    string(buf[:1]),
				Source: KeySourceRaw,
			},
		}, 1
	}

	// Remaining input begins a printable ASCII or UTF-8 character.
	return parseUTF8(buf)
}

// parseEsc parses input beginning with ESC.
//
// For input other than CSI or SS3, ESC acts as an Alt prefix around the next
// parsed key. This supports printable keys, Control keys, nested CSI or SS3
// sequences, and Alt+Escape.
func (p *Parser) parseEsc(buf []byte) (TerminalEvent, int) {
	if len(buf) < 2 {
		// Without a following byte, emit a standalone Escape key.
		return TerminalEvent{
			Type: EventKeyboard,
			Key: &KeyEvent{
				Name:   KeyEscape,
				Type:   KeyPress,
				Raw:    string(buf[:1]),
				Source: KeySourceRaw,
			},
		}, 1
	}

	switch buf[1] {
	case '[':
		return p.parseCSI(buf)
	case 'O':
		return parseSS3(buf)
	default:
		// Parse the following key recursively, then add the Alt modifier.
		ev, consumed := p.parseOne(buf[1:])
		if consumed > 0 && ev.Type == EventKeyboard && ev.Key != nil {
			ev.Key.Mods |= ModAlt
			ev.Key.Raw = string(buf[:1+consumed])

			// Some terminals use ESC b/f for Option+Arrow but double ESC for
			// Shift+Option+Arrow. In that mode, infer Shift for an otherwise
			// unmodified navigation sequence nested inside ESC.
			if p.altSendsEscPrefix && buf[1] == 0x1b &&
				ev.Key.Mods&(ModShift|ModCtrl|ModMeta) == 0 {
				if isNavigationKey(ev.Key.Name) {
					ev.Key.Mods |= ModShift
				}
			}

			return ev, 1 + consumed
		}
		// Retain an incomplete nested sequence.
		if consumed == 0 {
			return TerminalEvent{Type: EventType(-1)}, 0
		}
		// A non-key event cannot carry Alt, so consume only the standalone ESC.
		return TerminalEvent{
			Type: EventKeyboard,
			Key: &KeyEvent{
				Name:   KeyEscape,
				Type:   KeyPress,
				Raw:    string(buf[:1]),
				Source: KeySourceRaw,
			},
		}, 1
	}
}

// parseCSI parses a Control Sequence Introducer beginning with ESC [.
func (p *Parser) parseCSI(buf []byte) (TerminalEvent, int) {
	// Wait for a byte after the introducer.
	if len(buf) < 3 {
		return TerminalEvent{Type: EventType(-1)}, 0
	}

	// ESC [ 200 ~ begins bracketed paste.
	if len(buf) >= 6 && string(buf[2:6]) == "200~" {
		return parsePaste(buf)
	}

	// ESC [ < begins an SGR mouse report.
	if buf[2] == '<' {
		return parseMouse(buf)
	}

	// Scan parameter bytes until the sequence's final byte.
	paramStart := 2
	i := paramStart
	for i < len(buf) {
		c := buf[i]
		// This parser accepts numeric parameters separated by semicolons or colons.
		if (c >= '0' && c <= '9') || c == ';' || c == ':' {
			i++
			continue
		}
		// Uppercase letters and tilde identify standard keys; Kitty also uses lowercase.
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '~' {
			paramStr := string(buf[paramStart:i])
			finalByte := c
			consumed := i + 1

			// CSI u is a Kitty key only when Kitty parsing is enabled.
			if finalByte == 'u' && p.kittyEnabled {
				return parseKittyKey(buf[:consumed], paramStr)
			}

			return parseCSIKey(buf[:consumed], paramStr, finalByte)
		}
		// Consume a malformed sequence through its first unrecognized byte.
		return TerminalEvent{Type: EventType(-1)}, i + 1
	}

	// No final byte is available yet.
	return TerminalEvent{Type: EventType(-1)}, 0
}

// parseCSIKey converts a complete standard CSI sequence into a key event.
func parseCSIKey(raw []byte, paramStr string, finalByte byte) (TerminalEvent, int) {
	consumed := len(raw)

	// First match a complete unmodified suffix such as A or 3~.
	lookupKey := paramStr + string(finalByte)
	if name, ok := csiKeyMap[lookupKey]; ok {
		return TerminalEvent{
			Type: EventKeyboard,
			Key: &KeyEvent{
				Name:   name,
				Type:   KeyPress,
				Raw:    string(raw),
				Source: KeySourceRaw,
			},
		}, consumed
	}

	// A parameterless sequence can also be identified by its final byte.
	if name, ok := csiKeyMap[string(finalByte)]; ok && paramStr == "" {
		return TerminalEvent{
			Type: EventKeyboard,
			Key: &KeyEvent{
				Name:   name,
				Type:   KeyPress,
				Raw:    string(raw),
				Source: KeySourceRaw,
			},
		}, consumed
	}

	// Modified keys use CSI 1;modifier X or CSI number;modifier ~.
	parts := strings.Split(paramStr, ";")
	if len(parts) == 2 {
		mod, err := strconv.Atoi(parts[1])
		if err == nil {
			mods := decodeXtermModifiers(mod)

			// CSI 1;modifier X encodes modified arrows and function keys.
			if parts[0] == "1" || parts[0] == "" {
				if name, ok := csiModifiedKeyMap[finalByte]; ok {
					return TerminalEvent{
						Type: EventKeyboard,
						Key: &KeyEvent{
							Name:   name,
							Mods:   mods,
							Type:   KeyPress,
							Raw:    string(raw),
							Source: KeySourceRaw,
						},
					}, consumed
				}
			}

			// CSI number;modifier ~ encodes modified named keys.
			if finalByte == '~' {
				tildeKey := parts[0] + "~"
				if name, ok := csiKeyMap[tildeKey]; ok {
					return TerminalEvent{
						Type: EventKeyboard,
						Key: &KeyEvent{
							Name:   name,
							Mods:   mods,
							Type:   KeyPress,
							Raw:    string(raw),
							Source: KeySourceRaw,
						},
					}, consumed
				}
			}
		}
	}

	// A complete but unsupported sequence produces no event.
	return TerminalEvent{Type: EventType(-1)}, consumed
}

// decodeXtermModifiers decodes an xterm modifier parameter. Its value is one
// plus a bitmask containing Shift, Alt, Control, and Meta in ascending bits.
func decodeXtermModifiers(mod int) Mods {
	mod-- // Remove xterm's one-based offset.
	if mod < 0 {
		return 0
	}
	var mods Mods
	if mod&1 != 0 {
		mods |= ModShift
	}
	if mod&2 != 0 {
		mods |= ModAlt
	}
	if mod&4 != 0 {
		mods |= ModCtrl
	}
	if mod&8 != 0 {
		mods |= ModMeta
	}
	return mods
}

// parseSS3 parses a three-byte SS3 key sequence beginning with ESC O.
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
				Source: KeySourceRaw,
			},
		}, 3
	}

	// Consume a complete but unsupported SS3 sequence.
	return TerminalEvent{Type: EventType(-1)}, 3
}

// isNavigationKey reports whether name identifies a navigation, editing, or
// function key accepted by the double-ESC Shift heuristic.
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

// parseMouse parses an SGR mouse report of the form ESC [ < Cb ; Cx ; Cy M/m.
func parseMouse(buf []byte) (TerminalEvent, int) {
	// A valid report with one-digit fields is at least ten bytes.
	if len(buf) < 10 {
		// Consume an early terminator as a complete malformed report.
		for i := 3; i < len(buf); i++ {
			if buf[i] == 'M' || buf[i] == 'm' {
				return parseMouseComplete(buf[:i+1])
			}
		}
		return TerminalEvent{Type: EventType(-1)}, 0
	}

	// Wait until the press or release terminator arrives.
	for i := 3; i < len(buf); i++ {
		if buf[i] == 'M' || buf[i] == 'm' {
			return parseMouseComplete(buf[:i+1])
		}
	}

	return TerminalEvent{Type: EventType(-1)}, 0
}

func parseMouseComplete(raw []byte) (TerminalEvent, int) {
	consumed := len(raw)
	terminator := raw[consumed-1] // M reports a press or motion; m reports release.

	// Extract the button mask and coordinates between '<' and the terminator.
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

	// SGR reports one-based coordinates; MouseEvent uses zero-based coordinates.
	cx--
	cy--

	// Modifier flags occupy bits 2 through 4 of the button mask.
	var mods Mods
	if cb&4 != 0 {
		mods |= ModShift
	}
	if cb&8 != 0 {
		mods |= ModAlt
	}
	if cb&16 != 0 {
		mods |= ModCtrl
	}

	isMotion := cb&32 != 0
	isWheel := cb&64 != 0

	// The low two bits identify the button or wheel direction.
	btnBits := cb & 3

	me := &MouseEvent{
		X:    cx,
		Y:    cy,
		Mods: mods,
	}

	if isWheel {
		me.Type = MouseScroll
		me.Button = ButtonNone
		deltaY := 1
		if btnBits == 0 {
			deltaY = -1 // Wheel up uses button value zero.
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
		// Motion with button value three means no button is held.
		if cb&3 == 3 {
			me.Type = MouseMove
			me.IsDragging = false
			me.Button = ButtonNone
		}
	} else if terminator == 'm' {
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

// parseKittyKey parses a Kitty CSI u key sequence.
func parseKittyKey(raw []byte, paramStr string) (TerminalEvent, int) {
	consumed := len(raw)

	// Semicolons separate the code point and modifiers; a colon adds event type.
	parts := strings.Split(paramStr, ";")

	codepoint, err := strconv.Atoi(parts[0])
	if err != nil {
		return TerminalEvent{Type: EventType(-1)}, consumed
	}

	var mods Mods
	var eventType KeyEventType = KeyPress

	if len(parts) >= 2 {
		modParts := strings.Split(parts[1], ":")
		mod, err := strconv.Atoi(modParts[0])
		if err == nil {
			mods = decodeXtermModifiers(mod)
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

	// Resolve the protocol code point to a canonical name and printable text.
	name := ""
	text := "" // Non-empty only for printable input.
	if n, ok := kittySpecialKeys[codepoint]; ok {
		name = n
		// Space is the only named special key that also produces text.
		if codepoint == 32 {
			text = " "
		}
	} else if n, ok := csiKeyMap[strconv.Itoa(codepoint)+"~"]; ok {
		name = n
	} else if codepoint >= 32 && codepoint < 0x110000 {
		name = string(rune(codepoint))
		text = name // Preserve printable Unicode input.
	} else {
		name = strconv.Itoa(codepoint)
	}

	return TerminalEvent{
		Type: EventKeyboard,
		Key: &KeyEvent{
			Name:   name,
			Text:   text,
			Mods:   mods,
			Type:   eventType,
			Code:   strconv.Itoa(codepoint),
			Raw:    string(raw),
			Source: KeySourceKitty,
		},
	}, consumed
}

// parsePaste parses text between bracketed paste start and end markers.
func parsePaste(buf []byte) (TerminalEvent, int) {
	// parseCSI already confirmed the six-byte start marker; locate the end marker.
	endMarker := "\x1b[201~"
	content := buf[6:]
	idx := strings.Index(string(content), endMarker)
	if idx < 0 {
		// Retain pasted content until its end marker arrives.
		return TerminalEvent{Type: EventType(-1)}, 0
	}

	pasteText := string(content[:idx])
	consumed := 6 + idx + len(endMarker)

	// Normalize terminal carriage returns for line-oriented text buffers.
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

// parseUTF8 parses one remaining UTF-8-encoded rune.
func parseUTF8(buf []byte) (TerminalEvent, int) {
	r, size := utf8.DecodeRune(buf)
	if r == utf8.RuneError && size <= 1 {
		// DecodeRune reports a one-byte RuneError for invalid or incomplete input.
		if size == 0 {
			return TerminalEvent{Type: EventType(-1)}, 0
		}
		// Consume one invalid byte so parsing can continue.
		return TerminalEvent{Type: EventType(-1)}, 1
	}

	text := string(r) // Text retains the literal printable character.
	name := text
	if r == ' ' {
		name = KeySpace // Key bindings use the symbolic name for space.
	}

	return TerminalEvent{
		Type: EventKeyboard,
		Key: &KeyEvent{
			Name:   name,
			Text:   text,
			Type:   KeyPress,
			Raw:    string(buf[:size]),
			Source: KeySourceRaw,
		},
	}, size
}

// StripANSI returns text with recognized ANSI escape sequences removed.
func StripANSI(text string) string {
	return stripANSI(text)
}

func stripANSI(text string) string {
	return ansiRegex.ReplaceAllString(text, "")
}
