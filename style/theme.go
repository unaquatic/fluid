package style

// Theme is a named palette that widgets and applications share so text,
// surfaces, accents, and markdown elements stay visually coherent.
//
// Theme is orthogonal to per-widget styling: every widget that supports a
// theme lets callers substitute individual fields via its regular With...
// options, and falls back to the widget's own defaults when no theme is
// provided.
type Theme struct {
	Name string

	// --- Core surfaces ---------------------------------------------------
	// Bg is the application's root background.
	Bg Color
	// Surface is the message/panel background (sits on top of Bg).
	Surface Color
	// SurfaceAlt is a secondary surface for nested panels, tables, etc.
	SurfaceAlt Color
	// SurfaceRaised is a lifted surface for floating UI (tooltips,
	// menus, toasts). Zero → SurfaceAlt.
	SurfaceRaised Color
	// Border is the default border/separator color.
	Border Color
	// BorderStrong is the emphasized border color.
	BorderStrong Color
	// BorderWeak is the softest divider (between list rows, etc.).
	// Zero → a blend toward Surface.
	BorderWeak Color
	// BorderFocus is the border color for focused inputs and selected
	// items. Zero → Interactive (or Accent).
	BorderFocus Color

	// --- Text ------------------------------------------------------------
	// TextStrong is the primary foreground color.
	TextStrong Color
	// TextBase is the default body foreground. Zero → TextStrong.
	// Kept distinct from TextStrong so themes can separate "emphasised
	// body" from "regular body" — e.g. a slightly lifted TextStrong
	// for headings and bold spans while TextBase stays neutral for
	// prose.
	TextBase Color
	// TextWeak is the secondary (subtle) foreground color.
	TextWeak Color
	// TextMuted is the lowest-contrast foreground color (hints, footers).
	TextMuted Color
	// TextLink is the foreground for clickable text. Zero → MarkdownLink
	// (or Interactive / Accent if that's also unset).
	TextLink Color

	// --- Accent / interactive roles --------------------------------------
	// Accent is the primary highlight color (headings, emphasis, brand).
	Accent Color
	// AccentStrong is a more saturated accent for emphasis.
	AccentStrong Color
	// Interactive drives selection, focus rings, active menu items,
	// clickable elements. Zero → Accent.
	Interactive Color
	// InteractiveStrong is the pressed / active state for Interactive.
	// Zero → AccentStrong (or Interactive).
	InteractiveStrong Color

	// --- Semantic roles --------------------------------------------------
	Success Color
	Warning Color
	Danger  Color
	Info    Color

	// --- On-color foregrounds --------------------------------------------
	// OnAccent is the text color to use on top of Accent-filled surfaces
	// (buttons, badges). Zero → auto-pick black or white via
	// ChooseOnColor(Accent).
	OnAccent      Color
	OnInteractive Color // text on Interactive fills
	OnSuccess     Color
	OnWarning     Color
	OnDanger      Color
	OnInfo        Color

	// --- Selection ------------------------------------------------------
	// SelectionBg fills selected text ranges, highlighted menu rows,
	// and the "current" state for focusable widgets. Zero → a low-
	// alpha blend of Interactive with Surface, which stays readable
	// over any foreground.
	SelectionBg Color
	// SelectionFg is the text color over SelectionBg. Zero →
	// ChooseOnColor(SelectionBg), which picks black/white via
	// luminance so callers don't have to think about contrast.
	SelectionFg Color
	// SelectionInactiveBg is the desaturated variant used when the
	// widget has focus elsewhere (e.g. a menu row that was the most
	// recently active but isn't currently receiving input). Zero →
	// SelectionBg darkened toward Bg.
	SelectionInactiveBg Color

	// --- Diff surfaces ---------------------------------------------------
	// DiffAddSurface / DiffRemoveSurface tint full rows in diff views.
	// Zero → a low-alpha blend of Success/Danger with Bg.
	DiffAddSurface    Color
	DiffRemoveSurface Color
	// DiffAddFg / DiffRemoveFg are the foregrounds for diff markers and
	// intra-line highlights. Zero → Success / Danger.
	DiffAddFg    Color
	DiffRemoveFg Color

	// --- Markdown roles --------------------------------------------------
	//
	// Each role is an Entry so a theme states a span's color and its
	// weight as one decision. Unset roles inherit as documented below.
	MarkdownHeading [6]Entry // H1..H6
	// MarkdownText is the body text inside rendered markdown.
	// Zero → TextBase.
	MarkdownText Entry
	// MarkdownLink styles a URL; MarkdownLinkText the displayed label
	// when it differs from the URL.
	// Zero → TextLink / MarkdownLink.
	MarkdownLink     Entry
	MarkdownLinkText Entry
	// MarkdownCode is the inline-code span inside prose. A set Bg chips
	// the span against the surrounding text.
	// Zero → CodeFg on CodeBg.
	MarkdownCode Entry
	// MarkdownEmph and MarkdownStrong style *italic* and **bold** spans.
	// Zero → TextWeak italic / TextStrong bold.
	MarkdownEmph   Entry
	MarkdownStrong Entry
	// MarkdownBullet styles unordered list markers, MarkdownListEnum the
	// numbers of an ordered list.
	// Zero → MarkdownBullet.
	MarkdownBullet   Entry
	MarkdownListEnum Entry
	// MarkdownQuote styles blockquote prose, MarkdownQuoteBorder the
	// gutter glyph drawn beside it.
	MarkdownQuote       Entry
	MarkdownQuoteBorder Entry
	MarkdownRule        Entry
	// MarkdownImage and MarkdownImageText style image URLs and alt text.
	// Zero → MarkdownLink / MarkdownLinkText.
	MarkdownImage     Entry
	MarkdownImageText Entry

	// --- Code ------------------------------------------------------------
	// CodeFg / CodeBg are the default monochrome colors for inline code
	// spans and unlanguaged fenced blocks. CodeBg is also used as the
	// default background for syntax-highlighted tokens that do not
	// specify their own.
	CodeFg Color
	CodeBg Color

	// Code is the per-token palette used to build a Chroma style for
	// syntax highlighting. Fields left as the zero Color fall back to
	// sensible parents (see OrInherit and the Chroma mapping in the
	// widget package).
	Code CodePalette
}

// CodePalette holds semantic roles that map onto Chroma's ~80 token
// types. The fields are the buckets most syntax themes actually paint;
// unset fields inherit from close parents so a caller can specify just a
// handful of roles and still produce a coherent theme.
//
// Every role carries its attributes alongside its color, so a theme can
// give any token bold, italic or underline — not just the handful that
// happen to be wired. Themes like IntelliJ Darcula italicise constants
// and bold keywords; monochrome themes lean on attributes hard because
// they have no hue to separate roles with.
type CodePalette struct {
	// Core roles — cover ~95% of everyday code.
	Keyword     Entry // if, for, func, class, return, import
	Type        Entry // int, string, MyClass, custom types
	Function    Entry // function / method names (def + call site)
	Builtin     Entry // println, len, true, false, nil, this, self
	Variable    Entry // identifiers that aren't in any role above
	Constant    Entry // SCREAMING_CASE, None, numeric constants by name
	String      Entry // string literals, backticks
	Number      Entry // int / float / hex / binary literals
	Comment     Entry // // ... /* ... */ # ...
	Operator    Entry // + - * / = && ||
	Punctuation Entry // , ; { } ( )
	Decorator   Entry // @decorator, [Attribute], annotations
	Error       Entry // tokenization errors, Chroma's Error token

	// Optional extras — zero means "inherit from a parent role".
	StringEscape Entry // \n, \x1b etc. inside strings    (zero → Number)
	Regex        Entry // regex literals /foo/            (zero → String)
	Tag          Entry // HTML/XML element names           (zero → Keyword)
	Attribute    Entry // HTML attribute names             (zero → Variable)
	Property     Entry // object/struct property access    (zero → Variable)
	Primitive    Entry // true/false/null/undefined        (zero → Constant)
	Object       Entry // class/module identifiers         (zero → Variable)
	Preproc      Entry // #include, #define                (zero → Comment)
	DiffAdded    Entry // GenericInserted (+ lines)        (zero → theme Success)
	DiffRemoved  Entry // GenericDeleted (- lines)         (zero → theme Danger)
}

// Entry describes how one styled role is drawn: a foreground, an
// optional background, and a mask of text attributes (AttrBold,
// AttrItalic, AttrUnderline, ...).
//
// Keeping the color and the attributes in one value is deliberate. A
// role's weight is part of its identity — "strong is orange and bold"
// is a single decision, and splitting it across a Color field and a
// parallel attribute field lets one half be wired up while the other
// is forgotten.
//
// The zero Entry means "unset, inherit the parent role". Inheritance is
// all-or-nothing, so a theme that wants color without any attributes
// writes Entry{Fg: c} and gets exactly that.
type Entry struct {
	Fg    Color
	Bg    Color
	Attrs uint32
}

// IsZero reports whether e is unset and should inherit.
func (e Entry) IsZero() bool { return e == Entry{} }

// Or returns e when it is set, otherwise fallback.
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

// Chunk returns text styled by e. A transparent Fg or Bg is left unset
// on the chunk so the surrounding widget's own colors show through.
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

// Merge layers over on top of e: over's colors win wherever they are
// set, and the attribute masks union so a code span inside **strong**
// keeps the code color and gains the bold bit.
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

// Apply merges e onto an existing chunk, leaving any color the chunk
// already carries in place. Used when an inline element nests inside
// another — a code span inside **strong** keeps the code colors and
// gains the bold bit.
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

// OrInherit returns c when it has an opaque alpha; otherwise fallback.
// Callers use this to express "if the user didn't set this role, use its
// default parent role" without making inheritance implicit throughout
// the code.
func OrInherit(c, fallback Color) Color {
	if c.A == 0 {
		return fallback
	}
	return c
}

// ChooseOnColor picks black or white, whichever has higher BT.709
// luminance contrast against fill. Used as the default "on-color"
// foreground when a theme hasn't specified one.
func ChooseOnColor(fill Color) Color {
	// Contrast ratio is monotonic in |L₁ - L₂| for these two choices,
	// so comparing luminances directly is sufficient.
	if fill.Luminance() < 0.5 {
		return ColorWhite
	}
	return ColorBlack
}

// --- Resolved accessors ----------------------------------------------------
//
// These helpers return the effective color for a role after applying the
// documented inheritance rules, so widgets can read a single field
// without replicating the "zero → parent" logic everywhere.

// ResolveInteractive returns the interactive color (selection/focus),
// falling back to Accent → AccentStrong → TextStrong.
func (t *Theme) ResolveInteractive() Color {
	return OrInherit(t.Interactive, OrInherit(t.Accent, OrInherit(t.AccentStrong, t.TextStrong)))
}

// ResolveInteractiveStrong returns the pressed/active interactive color.
func (t *Theme) ResolveInteractiveStrong() Color {
	return OrInherit(t.InteractiveStrong, OrInherit(t.AccentStrong, t.ResolveInteractive()))
}

// ResolveTextBase returns the default body text color.
func (t *Theme) ResolveTextBase() Color { return OrInherit(t.TextBase, t.TextStrong) }

// ResolveTextLink returns the color for clickable text.
func (t *Theme) ResolveTextLink() Color {
	return OrInherit(t.TextLink, OrInherit(t.MarkdownLink.Fg, t.ResolveInteractive()))
}

// ResolveSurfaceRaised returns the surface for floating UI.
func (t *Theme) ResolveSurfaceRaised() Color { return OrInherit(t.SurfaceRaised, t.SurfaceAlt) }

// ResolveBorderWeak returns the softest divider color.
func (t *Theme) ResolveBorderWeak() Color {
	if t.BorderWeak.A != 0 {
		return t.BorderWeak
	}
	// Blend Border toward Surface so it reads as a hairline divider.
	return t.Border.Blend(t.Surface, 0.5)
}

// ResolveBorderFocus returns the border used for focused inputs.
func (t *Theme) ResolveBorderFocus() Color {
	return OrInherit(t.BorderFocus, t.ResolveInteractive())
}

// ResolveSelectionBg returns the background color for selected ranges
// and highlighted menu rows. When unset, blends Interactive toward
// Surface at 70% so the selection sits legibly over any foreground.
func (t *Theme) ResolveSelectionBg() Color {
	if t.SelectionBg.A != 0 {
		return t.SelectionBg
	}
	return t.ResolveInteractive().Blend(t.Surface, 0.7)
}

// ResolveSelectionFg returns the text color drawn on SelectionBg.
func (t *Theme) ResolveSelectionFg() Color {
	return OrInherit(t.SelectionFg, ChooseOnColor(t.ResolveSelectionBg()))
}

// ResolveSelectionInactiveBg returns the desaturated selection tint
// used when focus is elsewhere. Darkens the active selection toward
// Bg by 40% so it reads as "dimmed but still highlighted".
func (t *Theme) ResolveSelectionInactiveBg() Color {
	if t.SelectionInactiveBg.A != 0 {
		return t.SelectionInactiveBg
	}
	return t.ResolveSelectionBg().Blend(t.Bg, 0.4)
}

// ResolveOnAccent returns the fg for text drawn on Accent surfaces.
func (t *Theme) ResolveOnAccent() Color {
	return OrInherit(t.OnAccent, ChooseOnColor(OrInherit(t.Accent, t.TextStrong)))
}

// ResolveOnInteractive returns the fg for text drawn on Interactive.
func (t *Theme) ResolveOnInteractive() Color {
	return OrInherit(t.OnInteractive, ChooseOnColor(t.ResolveInteractive()))
}

// ResolveOnSuccess returns the fg for text drawn on Success surfaces.
func (t *Theme) ResolveOnSuccess() Color {
	return OrInherit(t.OnSuccess, ChooseOnColor(t.Success))
}

// ResolveOnWarning returns the fg for text drawn on Warning surfaces.
func (t *Theme) ResolveOnWarning() Color {
	return OrInherit(t.OnWarning, ChooseOnColor(t.Warning))
}

// ResolveOnDanger returns the fg for text drawn on Danger surfaces.
func (t *Theme) ResolveOnDanger() Color {
	return OrInherit(t.OnDanger, ChooseOnColor(t.Danger))
}

// ResolveOnInfo returns the fg for text drawn on Info surfaces.
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

// ResolveMarkdownHeading returns the style for a heading level, 1-6.
// Headings are bold unless the theme says otherwise.
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

// NeutralCodePalette derives a monochromatic syntax palette from a
// 12-step neutral ramp. Token differentiation leans on text attributes
// (bold, italic, underline) rather than hue — each role gets a
// luminance step from the ramp, and categories that share a color
// remain distinguishable through bold/italic.
//
// The index map is:
//
//	Error (Keyword Function)   → ramp[11]  // brightest; carry most weight
//	Type Constant Decorator   → ramp[10]
//	Variable Number Builtin   → ramp[9]
//	Operator Punctuation       → ramp[8]
//	String                     → ramp[8]  (+ italic for emphasis)
//	Comment                    → ramp[6]  (+ italic)
//
// Use this with any neutral ramp — Dark, Light, Sepia, Solarized-zero-
// chroma, etc. — to get a reading-friendly code palette that matches
// the rest of the theme without any taste-driven color choices.
func NeutralCodePalette(ramp [12]Color) CodePalette {
	// Without hue to separate roles, attributes are the primary
	// readability cue — hence the bold/italic on the carrying roles.
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
		Keyword:  Entry{Fg: ColorFromInts(249, 38, 114, 255)},  // pink
		Type:     Entry{Fg: ColorFromInts(102, 217, 239, 255)}, // cyan
		Function: Entry{Fg: ColorFromInts(166, 226, 46, 255)},  // green
		Builtin:  Entry{Fg: ColorFromInts(102, 217, 239, 255)}, // = Type
		Variable: Entry{Fg: ColorFromInts(248, 248, 242, 255)}, // default code fg
		Constant: Entry{Fg: ColorFromInts(174, 129, 255, 255)}, // purple
		String:   Entry{Fg: ColorFromInts(230, 219, 116, 255)}, // yellow
		Number:   Entry{Fg: ColorFromInts(174, 129, 255, 255)}, // = Constant
		Comment: Entry{
			Fg:    ColorFromInts(117, 113, 94, 255), // comment grey
			Attrs: AttrItalic,
		},
		Operator:    Entry{Fg: ColorFromInts(249, 38, 114, 255)},  // = Keyword
		Punctuation: Entry{Fg: ColorFromInts(248, 248, 242, 255)}, // default
		Decorator:   Entry{Fg: ColorFromInts(166, 226, 46, 255)},  // = Function
		Error:       Entry{Fg: ColorFromInts(255, 85, 85, 255)},   // brightish red
	}
}

// DefaultDark is the built-in dark theme: seed-derived palette with
// Monokai syntax highlighting. Mutate fields on the returned Theme to
// taste-tune.
func DefaultDark() *Theme {
	t := ThemeFromSeeds("default-dark", ThemeSeeds{
		Neutral:     ColorFromInts(20, 20, 20, 255),
		Ink:         ColorWhite,
		Primary:     ColorBrightCyan, // heading / brand
		Interactive: ColorBrightBlue, // links / selection
		Success:     ColorBrightGreen,
		Warning:     ColorBrightYellow,
		Danger:      ColorBrightRed,
		Info:        ColorBrightCyan,
	}, true)
	t.Code = MonokaiCodePalette()
	return t
}
