package style

import (
	"fmt"
	"strings"
)

// Attrs holds text attributes: decoration flags, such as AttrBold, in its low
// eight bits, and a hyperlink ID in its upper 24 bits. Combine flags with |.
type Attrs uint32

// Decoration flags.
const (
	AttrNone          Attrs = 0
	AttrBold          Attrs = 1 << 0
	AttrDim           Attrs = 1 << 1
	AttrItalic        Attrs = 1 << 2
	AttrUnderline     Attrs = 1 << 3
	AttrBlink         Attrs = 1 << 4
	AttrInverse       Attrs = 1 << 5
	AttrHidden        Attrs = 1 << 6
	AttrStrikethrough Attrs = 1 << 7
)

// The layout of Attrs: AttributeBaseMask selects the decoration flags, and a
// link ID sits AttributeBaseBits above them.
const (
	AttributeBaseBits       = 8
	AttributeBaseMask Attrs = 0xFF
)

// Has reports whether a contains any of the flags in flag.
func (a Attrs) Has(flag Attrs) bool {
	return a&flag != 0
}

// Base returns the decoration flags of a, without its link ID.
func (a Attrs) Base() Attrs {
	return a & AttributeBaseMask
}

// LinkID returns the hyperlink ID stored in a, or zero for none.
func (a Attrs) LinkID() uint32 {
	return uint32(a >> AttributeBaseBits)
}

// WithLink returns the decoration flags of a with the hyperlink ID id.
func (a Attrs) WithLink(id uint32) Attrs {
	return a.Base() | Attrs(id)<<AttributeBaseBits
}

// ANSI returns one ANSI SGR sequence for the decoration flags of a. It
// returns an empty string when no supported decorations are set.
func (a Attrs) ANSI() string {
	base := a.Base()
	if base == AttrNone {
		return ""
	}

	var parts []string
	if base&AttrBold != 0 {
		parts = append(parts, "1")
	}
	if base&AttrDim != 0 {
		parts = append(parts, "2")
	}
	if base&AttrItalic != 0 {
		parts = append(parts, "3")
	}
	if base&AttrUnderline != 0 {
		parts = append(parts, "4")
	}
	if base&AttrBlink != 0 {
		parts = append(parts, "5")
	}
	if base&AttrInverse != 0 {
		parts = append(parts, "7")
	}
	if base&AttrHidden != 0 {
		parts = append(parts, "8")
	}
	if base&AttrStrikethrough != 0 {
		parts = append(parts, "9")
	}

	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("\x1b[%sm", strings.Join(parts, ";"))
}
