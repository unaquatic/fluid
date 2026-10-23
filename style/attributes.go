package style

// TextAttributes bitmask constants.
// The lower 8 bits store text decoration flags.
// The upper 24 bits can store a link ID.
const (
	AttrNone          uint32 = 0
	AttrBold          uint32 = 1 << 0 // 1
	AttrDim           uint32 = 1 << 1 // 2
	AttrItalic        uint32 = 1 << 2 // 4
	AttrUnderline     uint32 = 1 << 3 // 8
	AttrBlink         uint32 = 1 << 4 // 16
	AttrInverse       uint32 = 1 << 5 // 32
	AttrHidden        uint32 = 1 << 6 // 64
	AttrStrikethrough uint32 = 1 << 7 // 128

	AttributeBaseBits uint32 = 8
	AttributeBaseMask uint32 = 0xFF
)

// AttributeOptions provides named booleans for building a text attribute bitmask.
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

// CreateTextAttributes builds a bitmask from named booleans.
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

// GetBaseAttributes extracts the lower 8 attribute bits.
func GetBaseAttributes(attr uint32) uint32 {
	return attr & AttributeBaseMask
}

// AttributesWithLink packs a link ID into the upper 24 bits, preserving the lower 8 attribute bits.
func AttributesWithLink(baseAttrs uint32, linkID uint32) uint32 {
	return (baseAttrs & AttributeBaseMask) | (linkID << AttributeBaseBits)
}

// GetLinkID extracts the link ID from the upper 24 bits.
func GetLinkID(attr uint32) uint32 {
	return attr >> AttributeBaseBits
}

// HasAttribute checks if a specific flag is set in the attribute bitmask.
func HasAttribute(attr, flag uint32) bool {
	return attr&flag != 0
}
