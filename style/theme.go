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
	MarkdownHeading [6]Color // H1..H6
	// MarkdownText is the body text color inside rendered markdown.
	// Zero → TextBase.
	MarkdownText    Color
	MarkdownLink    Color // URL color
	// MarkdownLinkText is the displayed label for a link when distinct
	// from the URL. Zero → TextLink (or MarkdownLink).
	MarkdownLinkText Color
	// MarkdownCode is the inline-code foreground inside prose.
	// Zero → CodeFg.
	MarkdownCode     Color
	// MarkdownEmph / MarkdownStrong tint italic and bold spans.
	// Zero → TextStrong (italic inherits from weak).
	MarkdownEmph    Color
	MarkdownStrong  Color
	MarkdownBullet  Color
	// MarkdownListEnum colors ordered-list numbers. Zero → MarkdownBullet.
	MarkdownListEnum Color
	MarkdownQuoteFg Color
	MarkdownQuoteBd Color
	MarkdownRule    Color
	// MarkdownImage / MarkdownImageText color image URLs and alt text.
	// Zero → MarkdownLink / MarkdownLinkText.
	MarkdownImage     Color
	MarkdownImageText Color

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

// CodePalette holds semantic color roles that map onto Chroma's ~80 token
// types. The fields are the buckets most syntax themes actually paint;
// unset fields inherit from close parents so a caller can specify just a
// handful of roles and still produce a coherent theme.
//
// Attribute fields (masks of AttrBold / AttrItalic / AttrUnderline) apply
// alongside the color. Common defaults: Comment uses AttrItalic, and
// Keyword occasionally uses AttrBold.
type CodePalette struct {
	// Core roles — cover ~95% of everyday code.
	Keyword     Color // if, for, func, class, return, import
	Type        Color // int, string, MyClass, custom types
	Function    Color // function / method names (def + call site)
	Builtin     Color // println, len, true, false, nil, this, self
	Variable    Color // identifiers that aren't in any role above
	Constant    Color // SCREAMING_CASE, None, numeric constants by name
	String      Color // string literals, regex, backticks
	Number      Color // int / float / hex / binary literals
	Comment     Color // // ... /* ... */ # ...
	Operator    Color // + - * / = && ||
	Punctuation Color // , ; { } ( )
	Decorator   Color // @decorator, [Attribute], annotations
	Error       Color // tokenization errors, Chroma's Error token

	// Optional extras — zero means "inherit from a parent role".
	StringEscape Color // \n, \x1b etc. inside strings    (zero → Number)
	Regex        Color // regex literals /foo/            (zero → String)
	Tag          Color // HTML/XML element names           (zero → Keyword)
	Attribute    Color // HTML attribute names             (zero → Variable)
	Property     Color // object/struct property access    (zero → Variable)
	Primitive    Color // true/false/null/undefined        (zero → Constant)
	Object       Color // class/module identifiers         (zero → Type)
	Preproc      Color // #include, #define                (zero → Comment)
	DiffAdded    Color // GenericInserted (+ lines)        (zero → theme Success)
	DiffRemoved  Color // GenericDeleted (- lines)         (zero → theme Danger)

	// Attribute masks (AttrBold | AttrItalic | AttrUnderline) applied
	// alongside the color. Zero = no attributes.
	//
	// Themes like IntelliJ Darcula use italic for constants and bold
	// for keywords; monochrome themes lean on these fields hard because
	// they have no hue to distinguish roles with.
	KeywordAttrs     uint32
	TypeAttrs        uint32
	FunctionAttrs    uint32
	ConstantAttrs    uint32
	StringAttrs      uint32
	OperatorAttrs    uint32
	PunctuationAttrs uint32
	CommentAttrs     uint32
	ErrorAttrs       uint32
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
	return OrInherit(t.TextLink, OrInherit(t.MarkdownLink, t.ResolveInteractive()))
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

// ResolveMarkdownText returns the default body color for markdown prose.
func (t *Theme) ResolveMarkdownText() Color { return OrInherit(t.MarkdownText, t.ResolveTextBase()) }

// ResolveMarkdownLinkText returns the displayed label color for links.
func (t *Theme) ResolveMarkdownLinkText() Color {
	return OrInherit(t.MarkdownLinkText, t.ResolveTextLink())
}

// ResolveMarkdownCode returns the inline-code foreground inside prose.
func (t *Theme) ResolveMarkdownCode() Color { return OrInherit(t.MarkdownCode, t.CodeFg) }

// ResolveMarkdownEmph returns the italic/emphasis text color.
func (t *Theme) ResolveMarkdownEmph() Color { return OrInherit(t.MarkdownEmph, t.TextWeak) }

// ResolveMarkdownStrong returns the bold/strong text color.
func (t *Theme) ResolveMarkdownStrong() Color { return OrInherit(t.MarkdownStrong, t.TextStrong) }

// ResolveMarkdownListEnum returns the ordered-list number color.
func (t *Theme) ResolveMarkdownListEnum() Color {
	return OrInherit(t.MarkdownListEnum, t.MarkdownBullet)
}

// ResolveMarkdownImage returns the image-URL color.
func (t *Theme) ResolveMarkdownImage() Color { return OrInherit(t.MarkdownImage, t.MarkdownLink) }

// ResolveMarkdownImageText returns the image alt-text color.
func (t *Theme) ResolveMarkdownImageText() Color {
	return OrInherit(t.MarkdownImageText, t.ResolveMarkdownLinkText())
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
	return CodePalette{
		Keyword:     ramp[11],
		Type:        ramp[10],
		Function:    ramp[11],
		Builtin:     ramp[9],
		Variable:    ramp[9],
		Constant:    ramp[10],
		String:      ramp[8],
		Number:      ramp[9],
		Comment:     ramp[6],
		Operator:    ramp[8],
		Punctuation: ramp[8],
		Decorator:   ramp[10],
		Error:       ramp[11],

		// Attribute-based differentiation: without color to separate
		// roles, these are the primary readability cues.
		KeywordAttrs:  AttrBold,
		FunctionAttrs: AttrBold,
		TypeAttrs:     AttrBold,
		CommentAttrs:  AttrItalic,
		ErrorAttrs:    AttrBold | AttrUnderline,
	}
}

// MonokaiCodePalette returns the classic Monokai syntax coloring.
func MonokaiCodePalette() CodePalette {
	return CodePalette{
		Keyword:     ColorFromInts(249, 38, 114, 255),  // pink
		Type:        ColorFromInts(102, 217, 239, 255), // cyan
		Function:    ColorFromInts(166, 226, 46, 255),  // green
		Builtin:     ColorFromInts(102, 217, 239, 255), // = Type
		Variable:    ColorFromInts(248, 248, 242, 255), // default code fg
		Constant:    ColorFromInts(174, 129, 255, 255), // purple
		String:      ColorFromInts(230, 219, 116, 255), // yellow
		Number:      ColorFromInts(174, 129, 255, 255), // = Constant
		Comment:     ColorFromInts(117, 113, 94, 255),  // comment grey
		Operator:    ColorFromInts(249, 38, 114, 255),  // = Keyword
		Punctuation: ColorFromInts(248, 248, 242, 255), // default
		Decorator:   ColorFromInts(166, 226, 46, 255),  // = Function
		Error:       ColorFromInts(255, 85, 85, 255),   // brightish red

		CommentAttrs: AttrItalic,
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

