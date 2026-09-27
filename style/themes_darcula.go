package style

// DarculaSolid returns the Darcula Solid palette.
func DarculaSolid() *Theme {
	var (
		// Canvas colors.
		editorBg   = ColorFromHex("#1c1c1c")
		popupBg    = ColorFromHex("#252526")
		caretRowBg = ColorFromHex("#28292b")

		// Border colors.
		indentGuide    = ColorFromHex("#2c2c2d")
		selIndentGuide = ColorFromHex("#313132")
		tearline       = ColorFromHex("#3d3e40")
		selTearline    = ColorFromHex("#5e6063")

		// Text colors.
		identifierFg = ColorFromHex("#cfd2d5")
		consoleFg    = ColorFromHex("#ececec")
		commentFg    = ColorFromHex("#8a8a8a")

		// Accent and selection colors.
		keywordOrange = ColorFromHex("#cc7832")
		selectionBg   = ColorFromHex("#176081")
		selectionFg   = identifierFg

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
		Bg:            editorBg,
		Surface:       editorBg,
		SurfaceAlt:    popupBg,
		SurfaceRaised: caretRowBg,
		Border:        tearline,
		BorderWeak:    indentGuide,
		BorderStrong:  selTearline,
		BorderFocus:   selectionBg,

		// Text roles.
		TextStrong: consoleFg,
		TextBase:   identifierFg,
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
		Warning: ColorFromHex("#f3f99c"),
		Danger:  consoleRed,
		Info:    consoleGray,

		// Semantic surfaces.
		SuccessSurface: ColorFromHex("#262b27"),
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
		MarkdownText:        Entry{Fg: identifierFg},
		MarkdownLink:        Entry{Fg: keywordOrange, Attrs: AttrUnderline},
		MarkdownLinkText:    Entry{Fg: ColorFromHex("#4e94ce"), Attrs: AttrUnderline},
		MarkdownCode:        Entry{Fg: ColorFromHex("#8ea765"), Bg: editorBg},
		MarkdownEmph:        Entry{Fg: ColorFromHex("#f3f99c"), Attrs: AttrItalic},
		MarkdownStrong:      Entry{Fg: keywordOrange, Attrs: AttrBold},
		MarkdownBullet:      Entry{Fg: keywordOrange},
		MarkdownListEnum:    Entry{Fg: consoleGray},
		MarkdownQuote:       Entry{Fg: commentFg, Attrs: AttrItalic},
		MarkdownQuoteBorder: Entry{Fg: tearline},
		MarkdownRule:        Entry{Fg: indentGuide},
		MarkdownImage:       Entry{Fg: keywordOrange, Attrs: AttrUnderline},
		MarkdownImageText:   Entry{Fg: ColorFromHex("#4e94ce"), Attrs: AttrUnderline},

		// Code roles.
		CodeFg: identifierFg,
		CodeBg: editorBg,

		Code: CodePalette{
			Keyword:     Entry{Fg: keywordOrange, Attrs: AttrBold},
			Type:        Entry{Fg: ColorFromHex("#5ca4a9")},
			Function:    Entry{Fg: ColorFromHex("#ffc66d")},
			Builtin:     Entry{Fg: ColorFromHex("#9bc1bc")},
			Variable:    Entry{Fg: identifierFg},
			Constant:    Entry{Fg: ColorFromHex("#a782bb"), Attrs: AttrItalic},
			String:      Entry{Fg: ColorFromHex("#8ea765")},
			Number:      Entry{Fg: ColorFromHex("#6897bb")},
			Comment:     Entry{Fg: commentFg, Attrs: AttrItalic},
			Operator:    Entry{Fg: identifierFg},
			Punctuation: Entry{Fg: identifierFg},
			Decorator:   Entry{Fg: ColorFromHex("#bbb529")},
			Error:       Entry{Fg: consoleRed, Attrs: AttrUnderline},

			StringEscape: Entry{Fg: keywordOrange},
			Regex:        Entry{Fg: ColorFromHex("#8ea765")},
			Tag:          Entry{Fg: ColorFromHex("#e8bf6a")},
			Attribute:    Entry{Fg: ColorFromHex("#bababa")},
			Property:     Entry{Fg: ColorFromHex("#a782bb")},
			Primitive:    Entry{Fg: keywordOrange},
			Object:       Entry{Fg: ColorFromHex("#b9becf")},
			Preproc:      Entry{Fg: ColorFromHex("#bbb529")},
			DiffAdded:    Entry{Fg: consoleGreen},
			DiffRemoved:  Entry{Fg: consoleRed},
		},
	}
}
