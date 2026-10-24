package style

// DarculaSolid returns the Darcula Solid palette: Darcula's roles in
// brighter syntax colors, on a canvas a shade darker than Darcula's editor,
// with one color for all text, code included.
func DarculaSolid() *Theme {
	var (
		// Canvas colors: the canvas, behind text and code, and the raised
		// surface of notices, code blocks, and boxes, a shade lighter.
		canvasBg = ColorFromHex("#1a1a1a")
		popupBg  = ColorFromHex("#252526")
		raisedBg = ColorFromHex("#1f1f1f")

		// Border colors.
		indentGuide    = ColorFromHex("#2c2c2d")
		selIndentGuide = ColorFromHex("#313132")
		tearline       = ColorFromHex("#3d3e40")
		selTearline    = ColorFromHex("#5e6063")

		// Text colors: the text of the UI, prose, and code, and comments.
		textFg    = ColorFromHex("#e6e6e6")
		commentFg = ColorFromHex("#8a8a8a")

		// Accent and selection colors. Keywords, the accent, links, bold,
		// and warnings share keywordOrange.
		keywordOrange = ColorFromHex("#eb7513")
		selectionBg   = ColorFromHex("#176081")
		selectionFg   = textFg

		// Syntax colors that more than one role shares.
		stringGreen     = ColorFromHex("#91b557")
		functionYellow  = ColorFromHex("#ffb642")
		constantPurple  = ColorFromHex("#aa77c6")
		decoratorYellow = ColorFromHex("#e4be13")

		// Semantic colors.
		consoleGreen = ColorFromHex("#5af78d")
		consoleRed   = ColorFromHex("#ff5b56")
		consoleGray  = ColorFromHex("#8A8A8A")

		// Diff colors.
		diffAddBg    = ColorFromHex("#444d3b")
		diffRemoveBg = ColorFromHex("#4f2b2b")
	)

	return &Theme{
		Name: "darcula-solid",

		// Core surfaces.
		Bg:            canvasBg,
		Surface:       canvasBg,
		SurfaceAlt:    popupBg,
		SurfaceRaised: raisedBg,
		Border:        tearline,
		BorderWeak:    indentGuide,
		BorderStrong:  selTearline,
		BorderFocus:   selectionBg,

		// Text roles.
		TextStrong: textFg,
		TextBase:   textFg,
		TextWeak:   selIndentGuide,
		TextMuted:  commentFg,
		TextLink:   keywordOrange,

		// Accent and interactive roles.
		Accent:            keywordOrange,
		AccentStrong:      ColorFromHex("#d18616"),
		Interactive:       consoleGray,
		InteractiveStrong: ColorFromHex("#4e94ce"),

		// Semantic roles.
		Success: consoleGreen,
		Warning: keywordOrange,
		Danger:  consoleRed,
		Info:    consoleGray,

		// Semantic surfaces.
		SuccessSurface: raisedBg,
		WarningSurface: ColorFromHex("#2a2a25"),
		DangerSurface:  ColorFromHex("#2e2827"),
		InfoSurface:    ColorFromHex("#292929"),

		// Foregrounds placed on filled roles.
		OnAccent:      ColorWhite,
		OnInteractive: ColorWhite,
		OnSuccess:     ColorWhite,
		OnWarning:     ColorWhite,
		OnDanger:      ColorWhite,
		OnInfo:        ColorWhite,

		// Selection roles.
		SelectionBg:         selectionBg,
		SelectionFg:         selectionFg,
		SelectionInactiveBg: ColorFromHex("#313234"),

		// Diff roles.
		DiffAddSurface:    diffAddBg,
		DiffRemoveSurface: diffRemoveBg,
		DiffAddFg:         consoleGreen,
		DiffRemoveFg:      consoleRed,

		// Markdown roles.
		MarkdownHeading: [6]Entry{
			{Fg: keywordOrange, Attrs: AttrBold},
			{Fg: ColorFromHex("#d18616"), Attrs: AttrBold},
			{Fg: ColorFromHex("#b3632a"), Attrs: AttrBold},
			{Fg: ColorFromHex("#946a4e"), Attrs: AttrBold},
			{Fg: commentFg, Attrs: AttrBold},
			{Fg: ColorFromHex("#a0a0a0"), Attrs: AttrBold},
		},
		MarkdownText:        Entry{Fg: textFg},
		MarkdownLink:        Entry{Fg: keywordOrange, Attrs: AttrUnderline},
		MarkdownLinkText:    Entry{Fg: ColorFromHex("#4e94ce"), Attrs: AttrUnderline},
		MarkdownCode:        Entry{Fg: stringGreen, Bg: canvasBg},
		MarkdownEmph:        Entry{Fg: keywordOrange, Attrs: AttrItalic},
		MarkdownStrong:      Entry{Fg: keywordOrange, Attrs: AttrBold},
		MarkdownBullet:      Entry{Fg: keywordOrange},
		MarkdownListEnum:    Entry{Fg: consoleGray},
		MarkdownQuote:       Entry{Fg: commentFg, Attrs: AttrItalic},
		MarkdownQuoteBorder: Entry{Fg: tearline},
		MarkdownRule:        Entry{Fg: indentGuide},
		MarkdownImage:       Entry{Fg: keywordOrange, Attrs: AttrUnderline},
		MarkdownImageText:   Entry{Fg: ColorFromHex("#4e94ce"), Attrs: AttrUnderline},

		// Code roles.
		CodeFg: textFg,
		CodeBg: canvasBg,

		Code: CodePalette{
			Keyword:     Entry{Fg: keywordOrange, Attrs: AttrBold},
			Type:        Entry{Fg: ColorFromHex("#3dbfc8")},
			Function:    Entry{Fg: functionYellow},
			Builtin:     Entry{Fg: textFg},
			Variable:    Entry{Fg: textFg},
			Constant:    Entry{Fg: constantPurple, Attrs: AttrItalic},
			String:      Entry{Fg: stringGreen},
			Number:      Entry{Fg: ColorFromHex("#5a99c9")},
			Comment:     Entry{Fg: commentFg, Attrs: AttrItalic},
			Operator:    Entry{Fg: textFg},
			Punctuation: Entry{Fg: textFg},
			Decorator:   Entry{Fg: decoratorYellow},
			Error:       Entry{Fg: consoleRed, Attrs: AttrUnderline},

			StringEscape: Entry{Fg: keywordOrange},
			Regex:        Entry{Fg: stringGreen},
			Tag:          Entry{Fg: functionYellow},
			Attribute:    Entry{Fg: ColorFromHex("#bababa")},
			Property:     Entry{Fg: constantPurple},
			Primitive:    Entry{Fg: keywordOrange},
			Object:       Entry{Fg: ColorFromHex("#b9becf")},
			Preproc:      Entry{Fg: decoratorYellow},
			DiffAdded:    Entry{Fg: consoleGreen},
			DiffRemoved:  Entry{Fg: consoleRed},
		},
	}
}
