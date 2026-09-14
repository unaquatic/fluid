package style

// SyntaxTokenStyle contains the optional colors and attributes for a named
// syntax token.
type SyntaxTokenStyle struct {
	Name       string
	Fg         *Color // Nil inherits the default foreground.
	Bg         *Color // Nil inherits the default background.
	Attributes uint32
}

// SyntaxStyle maps syntax token names to stable positive style IDs.
type SyntaxStyle struct {
	styles []SyntaxTokenStyle // Index zero is reserved; IDs begin at one.
	byName map[string]int     // Token name to style ID.
}

// NewSyntaxStyle returns an empty syntax style registry.
func NewSyntaxStyle() *SyntaxStyle {
	return &SyntaxStyle{
		styles: []SyntaxTokenStyle{{}}, // Reserve ID zero to mean no style.
		byName: make(map[string]int),
	}
}

// Register stores a named token style and returns its ID. Registering an
// existing name replaces its style without changing its ID.
func (ss *SyntaxStyle) Register(name string, fg, bg *Color, attributes uint32) int {
	if id, ok := ss.byName[name]; ok {
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

// ResolveByName returns name's style ID, or -1 when name is not registered.
func (ss *SyntaxStyle) ResolveByName(name string) int {
	if id, ok := ss.byName[name]; ok {
		return id
	}
	return -1
}

// StyleCount returns the number of registered token styles.
func (ss *SyntaxStyle) StyleCount() int {
	return len(ss.styles) - 1
}

// Style returns a copy of the token style identified by id, or nil for an
// invalid or reserved ID.
func (ss *SyntaxStyle) Style(id int) *SyntaxTokenStyle {
	if id <= 0 || id >= len(ss.styles) {
		return nil
	}
	s := ss.styles[id]
	return &s
}
