package style

// Text attribute values use the low eight bits for decoration flags and the
// upper 24 bits for a link ID.
const (
	AttrNone          uint32 = 0
	AttrBold          uint32 = 1 << 0
	AttrDim           uint32 = 1 << 1
	AttrItalic        uint32 = 1 << 2
	AttrUnderline     uint32 = 1 << 3
	AttrBlink         uint32 = 1 << 4
	AttrInverse       uint32 = 1 << 5
	AttrHidden        uint32 = 1 << 6
	AttrStrikethrough uint32 = 1 << 7

	AttributeBaseBits uint32 = 8
	AttributeBaseMask uint32 = 0xFF
)

// AttributeOptions selects text decorations for CreateTextAttributes.
type AttributeOptions struct {
	Bold          bool
	Dim           bool
	Italic        bool
	Underline     bool
	Blink         bool
	Inverse       bool
	Hidden        bool
	Strikethrough bool
}

// CreateTextAttributes returns the decoration bitmask selected by opts.
func CreateTextAttributes(opts AttributeOptions) uint32 {
	var attr uint32
	if opts.Bold {
		attr |= AttrBold
	}
	if opts.Dim {
		attr |= AttrDim
	}
	if opts.Italic {
		attr |= AttrItalic
	}
	if opts.Underline {
		attr |= AttrUnderline
	}
	if opts.Blink {
		attr |= AttrBlink
	}
	if opts.Inverse {
		attr |= AttrInverse
	}
	if opts.Hidden {
		attr |= AttrHidden
	}
	if opts.Strikethrough {
		attr |= AttrStrikethrough
	}
	return attr
}

// GetBaseAttributes returns the decoration bits from attr.
func GetBaseAttributes(attr uint32) uint32 {
	return attr & AttributeBaseMask
}

// AttributesWithLink combines baseAttrs with linkID in the upper 24 bits.
func AttributesWithLink(baseAttrs uint32, linkID uint32) uint32 {
	return (baseAttrs & AttributeBaseMask) | (linkID << AttributeBaseBits)
}

// GetLinkID returns the link ID stored in attr.
func GetLinkID(attr uint32) uint32 {
	return attr >> AttributeBaseBits
}

// HasAttribute reports whether attr contains any bit in flag.
func HasAttribute(attr, flag uint32) bool {
	return attr&flag != 0
}
