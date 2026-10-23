package style

// SyntaxTokenStyle defines the visual appearance of a named syntax token.
type SyntaxTokenStyle struct {
	Name       string
	Fg         *Color // nil = inherit default
	Bg         *Color // nil = inherit default
	Attributes uint32
}

// SyntaxStyle is a registry of named token styles used for syntax highlighting.
// Each registered style gets a unique numeric ID (starting at 1).
type SyntaxStyle struct {
	styles []SyntaxTokenStyle // index 0 is unused; IDs start at 1
	byName map[string]int     // name → style ID
}

// NewSyntaxStyle creates a new empty SyntaxStyle registry.
func NewSyntaxStyle() *SyntaxStyle {
	return &SyntaxStyle{
		styles: []SyntaxTokenStyle{{}}, // slot 0 reserved (ID 0 = "no style")
		byName: make(map[string]int),
	}
}

// Register adds a named token style and returns its numeric ID.
// If a style with the same name already exists, it is overwritten
// and the existing ID is returned.
func (ss *SyntaxStyle) Register(name string, fg, bg *Color, attributes uint32) int {
	if id, ok := ss.byName[name]; ok {
		// Overwrite existing
		ss.styles[id] = SyntaxTokenStyle{
			Name:       name,
			Fg:         fg,
			Bg:         bg,
			Attributes: attributes,
		}
		return id
	}

	id := len(ss.styles)
	ss.styles = append(ss.styles, SyntaxTokenStyle{
		Name:       name,
		Fg:         fg,
		Bg:         bg,
		Attributes: attributes,
	})
	ss.byName[name] = id
	return id
}

// ResolveByName returns the style ID for the given name, or -1 if not found.
func (ss *SyntaxStyle) ResolveByName(name string) int {
	if id, ok := ss.byName[name]; ok {
		return id
	}
	return -1
}

// StyleCount returns the number of registered styles (excluding the reserved slot 0).
func (ss *SyntaxStyle) StyleCount() int {
	return len(ss.styles) - 1
}

// Style returns the token style for the given ID, or nil if invalid.
func (ss *SyntaxStyle) Style(id int) *SyntaxTokenStyle {
	if id <= 0 || id >= len(ss.styles) {
		return nil
	}
	s := ss.styles[id]
	return &s
}
