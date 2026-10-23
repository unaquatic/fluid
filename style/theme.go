package style

// Theme is a named palette of shared interface, text, markdown, diff, and code
// roles.
//
// Widgets can override individual theme roles through their own styling APIs.
type Theme struct {
	// Name identifies the theme, such as "darcula-solid".
	Name string

	// Core surfaces.

	// Bg is the application's root background.
	Bg Color
	// Surface is the panel background above Bg.
	Surface Color
	// SurfaceAlt is a secondary surface for nested regions.
	SurfaceAlt Color
	// SurfaceRaised is the floating-interface surface. A transparent value
	// inherits SurfaceAlt.
	SurfaceRaised Color
	// Border is the default border and separator color.
	Border Color
	// BorderStrong is the emphasized border color.
	BorderStrong Color
	// BorderWeak is the softest divider. A transparent value derives from Border
	// blended toward Surface.
	BorderWeak Color
	// BorderFocus is the focused border color. A transparent value inherits the
	// resolved interactive color.
	BorderFocus Color

	// Text roles.

	// TextStrong is the primary foreground color.
	TextStrong Color
	// TextBase is the body foreground. A transparent value inherits TextStrong.
	TextBase Color
	// TextWeak is the secondary foreground color.
	TextWeak Color
	// TextMuted is the lowest-contrast foreground color.
	TextMuted Color
	// TextLink is the clickable-text foreground. A transparent value inherits
	// MarkdownLink or the resolved interactive color.
	TextLink Color

	// Accent and interactive roles.

	// Accent is the primary highlight color.
	Accent Color
	// AccentStrong is a more saturated accent for emphasis.
	AccentStrong Color
	// Interactive colors selections, focus indicators, and clickable elements. A
	// transparent value inherits Accent.
	Interactive Color
	// InteractiveStrong is the pressed or active interactive color. A transparent
	// value inherits AccentStrong or Interactive.
	InteractiveStrong Color

	// Semantic roles.

	// Success marks successful states.
	Success Color
	// Warning marks states that need attention.
	Warning Color
	// Danger marks errors and destructive actions.
	Danger Color
	// Info marks informational states.
	Info Color

	// Semantic surfaces. Each is a low-contrast background tinted toward its
	// semantic role, with no fallback: a transparent value stays transparent.

	// SuccessSurface is tinted toward Success.
	SuccessSurface Color
	// WarningSurface is tinted toward Warning.
	WarningSurface Color
	// DangerSurface is tinted toward Danger.
	DangerSurface Color
	// InfoSurface is tinted toward Info.
	InfoSurface Color

	// Foregrounds for filled roles.

	// OnAccent is text drawn on Accent. A transparent value uses ChooseOnColor.
	OnAccent Color
	// OnInteractive is text drawn on Interactive. A transparent value uses
	// ChooseOnColor.
	OnInteractive Color
	// OnSuccess is text drawn on Success. A transparent value uses
	// ChooseOnColor.
	OnSuccess Color
	// OnWarning is text drawn on Warning. A transparent value uses
	// ChooseOnColor.
	OnWarning Color
	// OnDanger is text drawn on Danger. A transparent value uses ChooseOnColor.
	OnDanger Color
	// OnInfo is text drawn on Info. A transparent value uses ChooseOnColor.
	OnInfo Color

	// Selection roles.

	// SelectionBg fills selected text and active rows. A transparent value derives
	// from Interactive blended toward Surface.
	SelectionBg Color
	// SelectionFg is text drawn on SelectionBg. A transparent value uses
	// ChooseOnColor.
	SelectionFg Color
	// SelectionInactiveBg fills a selection without focus. A transparent value
	// derives from SelectionBg blended toward Bg.
	SelectionInactiveBg Color

	// Diff roles.

	// DiffAddSurface tints complete added rows. A transparent value derives
	// from Success blended toward Bg.
	DiffAddSurface Color
	// DiffRemoveSurface tints complete removed rows. A transparent value
	// derives from Danger blended toward Bg.
	DiffRemoveSurface Color
	// DiffAddFg colors added-line markers and intra-line highlights. A
	// transparent value inherits Success.
	DiffAddFg Color
	// DiffRemoveFg colors removed-line markers and intra-line highlights. A
	// transparent value inherits Danger.
	DiffRemoveFg Color

	// Markdown roles. Zero entries inherit as described below.

	// MarkdownHeading styles heading levels H1 through H6. A zero entry
	// inherits MarkdownText's foreground in bold.
	MarkdownHeading [6]Entry
	// MarkdownText is the body text inside rendered markdown.
	// A zero entry inherits TextBase.
	MarkdownText Entry
	// MarkdownLink styles a link's URL. A zero entry inherits TextLink.
	MarkdownLink Entry
	// MarkdownLinkText styles a link's displayed label. A zero entry inherits
	// MarkdownLink.
	MarkdownLinkText Entry
	// MarkdownCode styles inline code. A zero entry inherits CodeFg on CodeBg.
	MarkdownCode Entry
	// MarkdownEmph styles emphasis. A zero entry inherits italic TextWeak.
	MarkdownEmph Entry
	// MarkdownStrong styles strong spans. A zero entry inherits bold
	// TextStrong.
	MarkdownStrong Entry
	// MarkdownBullet styles unordered list markers.
	MarkdownBullet Entry
	// MarkdownListEnum styles ordered list markers. A zero entry inherits
	// MarkdownBullet.
	MarkdownListEnum Entry
	// MarkdownQuote styles blockquote prose.
	MarkdownQuote Entry
	// MarkdownQuoteBorder styles a blockquote's gutter.
	MarkdownQuoteBorder Entry
	// MarkdownRule styles horizontal rules. A zero entry inherits BorderWeak as
	// its foreground.
	MarkdownRule Entry
	// MarkdownImage styles image URLs. A zero entry inherits MarkdownLink.
	MarkdownImage Entry
	// MarkdownImageText styles an image's alternative text. A zero entry
	// inherits MarkdownLinkText.
	MarkdownImageText Entry

	// Code roles.

	// CodeFg is the default foreground of code without token styling.
	CodeFg Color
	// CodeBg is the default background of code without token styling.
	CodeBg Color

	// Code contains per-token syntax styles. Zero roles inherit from
	// related roles when converted to a syntax highlighter style.
	Code CodePalette
}

// CodePalette groups semantic syntax roles. Each Entry keeps a role's colors
// and text attributes together; conversion code resolves unset optional roles
// from related core roles.
type CodePalette struct {
	// Core roles.

	// Keyword styles language keywords.
	Keyword Entry
	// Type styles built-in and user-defined types.
	Type Entry
	// Function styles function and method names.
	Function Entry
	// Builtin styles built-in values and functions.
	Builtin Entry
	// Variable styles identifiers without a more specific role.
	Variable Entry
	// Constant styles named constants.
	Constant Entry
	// String styles string literals.
	String Entry
	// Number styles numeric literals.
	Number Entry
	// Comment styles line and block comments.
	Comment Entry
	// Operator styles operators.
	Operator Entry
	// Punctuation styles delimiters and separators.
	Punctuation Entry
	// Decorator styles decorators, attributes, and annotations.
	Decorator Entry
	// Error styles tokenization errors.
	Error Entry

	// Optional roles. A zero entry inherits the role its doc names.

	// StringEscape styles escape sequences in strings. A zero entry inherits
	// Number.
	StringEscape Entry
	// Regex styles regular expression literals. A zero entry inherits String.
	Regex Entry
	// Tag styles markup tag names. A zero entry inherits Keyword.
	Tag Entry
	// Attribute styles markup attribute names. A zero entry inherits Variable.
	Attribute Entry
	// Property styles property names. A zero entry inherits Variable.
	Property Entry
	// Primitive styles pseudo-keywords, such as true, false, and nil. A zero
	// entry inherits Constant.
	Primitive Entry
	// Object styles other names the lexer marks. A zero entry inherits
	// Variable.
	Object Entry
	// Preproc styles preprocessor directives. A zero entry inherits Comment.
	Preproc Entry
	// DiffAdded styles added lines in diff code. A zero entry inherits the
	// theme's Success role.
	DiffAdded Entry
	// DiffRemoved styles removed lines in diff code. A zero entry inherits the
	// theme's Danger role.
	DiffRemoved Entry
}

// Entry contains the foreground, background, and attributes for one styled
// role.
//
// A zero Entry is unset and inherits its complete fallback entry.
type Entry struct {
	// Fg is the foreground color.
	Fg Color
	// Bg is the background color.
	Bg Color
	// Attrs is a bitmask of the Attr flags, such as AttrBold.
	Attrs uint32
}

// IsZero reports whether e contains no colors or attributes.
func (e Entry) IsZero() bool { return e == Entry{} }

// Or returns e unless it is zero, in which case Or returns fallback.
func (e Entry) Or(fallback Entry) Entry {
	if e.IsZero() {
		return fallback
	}
	return e
}

// WithFg returns a copy of e painted with fg.
func (e Entry) WithFg(c Color) Entry {
	e.Fg = c
	return e
}

// WithBg returns a copy of e painted with bg.
func (e Entry) WithBg(c Color) Entry {
	e.Bg = c
	return e
}

// Chunk returns text styled by e. Transparent colors remain nil so the chunk
// inherits its surrounding colors.
func (e Entry) Chunk(text string) TextChunk {
	c := TextChunk{Text: text, Attributes: e.Attrs}
	if e.Fg.A != 0 {
		fg := e.Fg
		c.Fg = &fg
	}
	if e.Bg.A != 0 {
		bg := e.Bg
		c.Bg = &bg
	}
	return c
}

// Merge overlays nontransparent colors from over on e and unions their
// attribute masks.
func (e Entry) Merge(over Entry) Entry {
	e.Attrs |= over.Attrs
	if over.Fg.A != 0 {
		e.Fg = over.Fg
	}
	if over.Bg.A != 0 {
		e.Bg = over.Bg
	}
	return e
}

// Apply adds e's attributes and fills only colors missing from c.
func (e Entry) Apply(c TextChunk) TextChunk {
	c.Attributes |= e.Attrs
	if c.Fg == nil && e.Fg.A != 0 {
		fg := e.Fg
		c.Fg = &fg
	}
	if c.Bg == nil && e.Bg.A != 0 {
		bg := e.Bg
		c.Bg = &bg
	}
	return c
}

// OrInherit returns c when its alpha is nonzero, otherwise fallback.
func OrInherit(c, fallback Color) Color {
	if c.A == 0 {
		return fallback
	}
	return c
}

// ChooseOnColor returns white for a fill below 0.5 BT.709 luma and black for a
// fill at or above that threshold.
func ChooseOnColor(fill Color) Color {
	if fill.Luminance() < 0.5 {
		return ColorWhite
	}
	return ColorBlack
}

// ResolveInteractive returns Interactive, falling back through Accent,
// AccentStrong, and TextStrong.
func (t *Theme) ResolveInteractive() Color {
	return OrInherit(t.Interactive, OrInherit(t.Accent, OrInherit(t.AccentStrong, t.TextStrong)))
}

// ResolveInteractiveStrong returns the pressed/active interactive color.
func (t *Theme) ResolveInteractiveStrong() Color {
	return OrInherit(t.InteractiveStrong, OrInherit(t.AccentStrong, t.ResolveInteractive()))
}

// ResolveTextBase returns TextBase, falling back to TextStrong.
func (t *Theme) ResolveTextBase() Color { return OrInherit(t.TextBase, t.TextStrong) }

// ResolveTextLink returns TextLink, falling back through MarkdownLink and the
// resolved interactive color.
func (t *Theme) ResolveTextLink() Color {
	return OrInherit(t.TextLink, OrInherit(t.MarkdownLink.Fg, t.ResolveInteractive()))
}

// ResolveSurfaceRaised returns SurfaceRaised, falling back to SurfaceAlt.
func (t *Theme) ResolveSurfaceRaised() Color { return OrInherit(t.SurfaceRaised, t.SurfaceAlt) }

// ResolveBorderWeak returns BorderWeak or a blend from Border toward Surface.
func (t *Theme) ResolveBorderWeak() Color {
	if t.BorderWeak.A != 0 {
		return t.BorderWeak
	}
	return t.Border.Blend(t.Surface, 0.5)
}

// ResolveBorderFocus returns BorderFocus or the resolved interactive color.
func (t *Theme) ResolveBorderFocus() Color {
	return OrInherit(t.BorderFocus, t.ResolveInteractive())
}

// ResolveSelectionBg returns SelectionBg or the resolved interactive color
// blended 70 percent toward Surface.
func (t *Theme) ResolveSelectionBg() Color {
	if t.SelectionBg.A != 0 {
		return t.SelectionBg
	}
	return t.ResolveInteractive().Blend(t.Surface, 0.7)
}

// ResolveSelectionFg returns SelectionFg or a black-or-white foreground for
// the resolved selection background.
func (t *Theme) ResolveSelectionFg() Color {
	return OrInherit(t.SelectionFg, ChooseOnColor(t.ResolveSelectionBg()))
}

// ResolveSelectionInactiveBg returns SelectionInactiveBg or the active
// selection background blended 40 percent toward Bg.
func (t *Theme) ResolveSelectionInactiveBg() Color {
	if t.SelectionInactiveBg.A != 0 {
		return t.SelectionInactiveBg
	}
	return t.ResolveSelectionBg().Blend(t.Bg, 0.4)
}

// ResolveOnAccent returns the foreground for text drawn on Accent.
func (t *Theme) ResolveOnAccent() Color {
	return OrInherit(t.OnAccent, ChooseOnColor(OrInherit(t.Accent, t.TextStrong)))
}

// ResolveOnInteractive returns the foreground for text drawn on Interactive.
func (t *Theme) ResolveOnInteractive() Color {
	return OrInherit(t.OnInteractive, ChooseOnColor(t.ResolveInteractive()))
}

// ResolveOnSuccess returns the foreground for text drawn on Success.
func (t *Theme) ResolveOnSuccess() Color {
	return OrInherit(t.OnSuccess, ChooseOnColor(t.Success))
}

// ResolveOnWarning returns the foreground for text drawn on Warning.
func (t *Theme) ResolveOnWarning() Color {
	return OrInherit(t.OnWarning, ChooseOnColor(t.Warning))
}

// ResolveOnDanger returns the foreground for text drawn on Danger.
func (t *Theme) ResolveOnDanger() Color {
	return OrInherit(t.OnDanger, ChooseOnColor(t.Danger))
}

// ResolveOnInfo returns the foreground for text drawn on Info.
func (t *Theme) ResolveOnInfo() Color {
	return OrInherit(t.OnInfo, ChooseOnColor(t.Info))
}

// ResolveDiffAddSurface returns the row tint for added diff lines.
func (t *Theme) ResolveDiffAddSurface() Color {
	if t.DiffAddSurface.A != 0 {
		return t.DiffAddSurface
	}
	return t.Success.Blend(t.Bg, 0.82)
}

// ResolveDiffRemoveSurface returns the row tint for removed diff lines.
func (t *Theme) ResolveDiffRemoveSurface() Color {
	if t.DiffRemoveSurface.A != 0 {
		return t.DiffRemoveSurface
	}
	return t.Danger.Blend(t.Bg, 0.82)
}

// ResolveDiffAddFg returns the foreground for added diff markers.
func (t *Theme) ResolveDiffAddFg() Color { return OrInherit(t.DiffAddFg, t.Success) }

// ResolveDiffRemoveFg returns the foreground for removed diff markers.
func (t *Theme) ResolveDiffRemoveFg() Color { return OrInherit(t.DiffRemoveFg, t.Danger) }

// ResolveMarkdownText returns the default body style for markdown prose.
func (t *Theme) ResolveMarkdownText() Entry {
	return t.MarkdownText.Or(Entry{Fg: t.ResolveTextBase()})
}

// ResolveMarkdownHeading returns the style for heading levels 1 through 6. An
// invalid or unset level uses bold markdown body text.
func (t *Theme) ResolveMarkdownHeading(level int) Entry {
	fallback := Entry{Fg: t.ResolveMarkdownText().Fg, Attrs: AttrBold}
	if level < 1 || level > len(t.MarkdownHeading) {
		return fallback
	}
	return t.MarkdownHeading[level-1].Or(fallback)
}

// ResolveMarkdownLink returns the style for a link URL.
func (t *Theme) ResolveMarkdownLink() Entry {
	return t.MarkdownLink.Or(Entry{Fg: t.ResolveTextLink(), Attrs: AttrUnderline})
}

// ResolveMarkdownLinkText returns the displayed label style for links.
func (t *Theme) ResolveMarkdownLinkText() Entry {
	return t.MarkdownLinkText.Or(t.ResolveMarkdownLink())
}

// ResolveMarkdownCode returns the inline-code style inside prose.
func (t *Theme) ResolveMarkdownCode() Entry {
	return t.MarkdownCode.Or(Entry{Fg: t.CodeFg, Bg: t.CodeBg})
}

// ResolveMarkdownEmph returns the style for *emphasis* spans.
func (t *Theme) ResolveMarkdownEmph() Entry {
	return t.MarkdownEmph.Or(Entry{Fg: t.TextWeak, Attrs: AttrItalic})
}

// ResolveMarkdownStrong returns the style for **strong** spans.
func (t *Theme) ResolveMarkdownStrong() Entry {
	return t.MarkdownStrong.Or(Entry{Fg: t.TextStrong, Attrs: AttrBold})
}

// ResolveMarkdownBullet returns the style for unordered list markers.
func (t *Theme) ResolveMarkdownBullet() Entry {
	return t.MarkdownBullet.Or(Entry{Fg: t.ResolveTextBase()})
}

// ResolveMarkdownListEnum returns the style for ordered-list numbers.
func (t *Theme) ResolveMarkdownListEnum() Entry {
	return t.MarkdownListEnum.Or(t.ResolveMarkdownBullet())
}

// ResolveMarkdownQuote returns the style for blockquote prose.
func (t *Theme) ResolveMarkdownQuote() Entry {
	return t.MarkdownQuote.Or(Entry{Fg: t.TextMuted, Attrs: AttrItalic})
}

// ResolveMarkdownQuoteBorder returns the style for the blockquote gutter.
func (t *Theme) ResolveMarkdownQuoteBorder() Entry {
	return t.MarkdownQuoteBorder.Or(Entry{Fg: t.Border})
}

// ResolveMarkdownRule returns the style for a horizontal rule.
func (t *Theme) ResolveMarkdownRule() Entry {
	return t.MarkdownRule.Or(Entry{Fg: t.ResolveBorderWeak()})
}

// ResolveMarkdownImage returns the image-URL style.
func (t *Theme) ResolveMarkdownImage() Entry {
	return t.MarkdownImage.Or(t.ResolveMarkdownLink())
}

// ResolveMarkdownImageText returns the image alt-text style.
func (t *Theme) ResolveMarkdownImageText() Entry {
	return t.MarkdownImageText.Or(t.ResolveMarkdownLinkText())
}

// NeutralCodePalette derives syntax roles from a 12-step neutral ramp. It uses
// brighter steps and text attributes for stronger roles instead of assigning
// distinct hues.
func NeutralCodePalette(ramp [12]Color) CodePalette {
	// Attributes distinguish roles that share a ramp color.
	return CodePalette{
		Keyword:     Entry{Fg: ramp[11], Attrs: AttrBold},
		Type:        Entry{Fg: ramp[10], Attrs: AttrBold},
		Function:    Entry{Fg: ramp[11], Attrs: AttrBold},
		Builtin:     Entry{Fg: ramp[9]},
		Variable:    Entry{Fg: ramp[9]},
		Constant:    Entry{Fg: ramp[10]},
		String:      Entry{Fg: ramp[8]},
		Number:      Entry{Fg: ramp[9]},
		Comment:     Entry{Fg: ramp[6], Attrs: AttrItalic},
		Operator:    Entry{Fg: ramp[8]},
		Punctuation: Entry{Fg: ramp[8]},
		Decorator:   Entry{Fg: ramp[10]},
		Error:       Entry{Fg: ramp[11], Attrs: AttrBold | AttrUnderline},
	}
}

// MonokaiCodePalette returns the classic Monokai syntax coloring.
func MonokaiCodePalette() CodePalette {
	return CodePalette{
		Keyword:  Entry{Fg: ColorFromInts(249, 38, 114, 255)},
		Type:     Entry{Fg: ColorFromInts(102, 217, 239, 255)},
		Function: Entry{Fg: ColorFromInts(166, 226, 46, 255)},
		Builtin:  Entry{Fg: ColorFromInts(102, 217, 239, 255)},
		Variable: Entry{Fg: ColorFromInts(248, 248, 242, 255)},
		Constant: Entry{Fg: ColorFromInts(174, 129, 255, 255)},
		String:   Entry{Fg: ColorFromInts(230, 219, 116, 255)},
		Number:   Entry{Fg: ColorFromInts(174, 129, 255, 255)},
		Comment: Entry{
			Fg:    ColorFromInts(117, 113, 94, 255),
			Attrs: AttrItalic,
		},
		Operator:    Entry{Fg: ColorFromInts(249, 38, 114, 255)},
		Punctuation: Entry{Fg: ColorFromInts(248, 248, 242, 255)},
		Decorator:   Entry{Fg: ColorFromInts(166, 226, 46, 255)},
		Error:       Entry{Fg: ColorFromInts(255, 85, 85, 255)},
	}
}

// DefaultDark returns the built-in seed-derived dark theme with Monokai syntax
// highlighting.
func DefaultDark() *Theme {
	t := ThemeFromSeeds("default-dark", ThemeSeeds{
		Neutral:     ColorFromInts(20, 20, 20, 255),
		Ink:         ColorWhite,
		Primary:     ColorBrightCyan, // Headings and accents.
		Interactive: ColorBrightBlue, // Links, focus, and selection.
		Success:     ColorBrightGreen,
		Warning:     ColorBrightYellow,
		Danger:      ColorBrightRed,
		Info:        ColorBrightCyan,
	}, true)
	t.Code = MonokaiCodePalette()
	return t
}
