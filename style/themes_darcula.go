package style

// DarculaSolid returns the Darcula Solid palette.
func DarculaSolid() *Theme {
	var (
		// Canvas
		editorBg   = ColorFromHex("#1c1c1c")
		popupBg    = ColorFromHex("#252526")
		caretRowBg = ColorFromHex("#323335")

		// Borders
		indentGuide    = ColorFromHex("#2c2c2d")
		selIndentGuide = ColorFromHex("#313132")
		tearline       = ColorFromHex("#3d3e40")
		selTearline    = ColorFromHex("#5e6063")

		// Text
		identifierFg = ColorFromHex("#cfd2d5")
		consoleFg    = ColorFromHex("#ececec")
		commentFg    = ColorFromHex("#8a8a8a")

		// Accent / selection
		keywordOrange = ColorFromHex("#cc7832")
		selectionBg   = ColorFromHex("#176081")
		selectionFg   = identifierFg

		// Semantic
		consoleGreen = ColorFromHex("#5af78d")
		consoleRed   = ColorFromHex("#ff5b56")
		consoleBlue  = ColorFromHex("#5786fe")

		// Diff
		diffAddBg    = ColorFromHex("#444d3b")
		diffRemoveBg = ColorFromHex("#4f2b2b")
	)

	return &Theme{
		Name: "darcula-solid",

		// Core surfaces
		Bg:            editorBg,
		Surface:       editorBg,
		SurfaceAlt:    popupBg,
		SurfaceRaised: caretRowBg,
		Border:        tearline,
		BorderWeak:    indentGuide,
		BorderStrong:  selTearline,
		BorderFocus:   selectionBg,

		// Text
		TextStrong: consoleFg,
		TextBase:   identifierFg,
		TextWeak:   selIndentGuide,
		TextMuted:  commentFg,
		TextLink:   keywordOrange,

		// Accent / interactive
		Accent:            keywordOrange,
		AccentStrong:      ColorFromHex("#d18616"),
		Interactive:       consoleBlue,
		InteractiveStrong: ColorFromHex("#4e94ce"),

		// Semantic
		Success: consoleGreen,
		Warning: ColorFromHex("#f3f99c"),
		Danger:  consoleRed,
		Info:    consoleBlue,

		// On-color foregrounds
		OnAccent:      ColorWhite,
		OnInteractive: ColorWhite,
		OnSuccess:     ColorWhite,
		OnWarning:     ColorWhite,
		OnDanger:      ColorWhite,
		OnInfo:        ColorWhite,

		// Selection
		SelectionBg:         selectionBg,
		SelectionFg:         selectionFg,
		SelectionInactiveBg: ColorFromHex("#313234"),

		// Diff
		DiffAddSurface:    diffAddBg,
		DiffRemoveSurface: diffRemoveBg,
		DiffAddFg:         consoleGreen,
		DiffRemoveFg:      consoleRed,

		// Markdown
		MarkdownHeading: [6]Color{
			keywordOrange,
			ColorFromHex("#d18616"),
			ColorFromHex("#b3632a"),
			ColorFromHex("#946a4e"),
			commentFg,
			ColorFromHex("#a0a0a0"),
		},
		MarkdownText:      identifierFg,
		MarkdownLink:      keywordOrange,
		MarkdownLinkText:  ColorFromHex("#4e94ce"),
		MarkdownCode:      ColorFromHex("#8ea765"),
		MarkdownEmph:      ColorFromHex("#f3f99c"),
		MarkdownStrong:    keywordOrange,
		MarkdownBullet:    keywordOrange,
		MarkdownListEnum:  consoleBlue,
		MarkdownQuoteFg:   commentFg,
		MarkdownQuoteBd:   tearline,
		MarkdownRule:      indentGuide,
		MarkdownImage:     keywordOrange,
		MarkdownImageText: ColorFromHex("#4e94ce"),

		// Code
		CodeFg: identifierFg,
		CodeBg: editorBg,

		Code: CodePalette{
			Keyword:     keywordOrange,
			Type:        ColorFromHex("#5ca4a9"),
			Function:    ColorFromHex("#ffc66d"),
			Builtin:     ColorFromHex("#9bc1bc"),
			Variable:    identifierFg,
			Constant:    ColorFromHex("#a782bb"),
			String:      ColorFromHex("#8ea765"),
			Number:      ColorFromHex("#6897bb"),
			Comment:     commentFg,
			Operator:    identifierFg,
			Punctuation: identifierFg,
			Decorator:   ColorFromHex("#bbb529"),
			Error:       consoleRed,

			StringEscape: keywordOrange,
			Regex:        ColorFromHex("#8ea765"),
			Tag:          ColorFromHex("#e8bf6a"),
			Attribute:    ColorFromHex("#bababa"),
			Property:     ColorFromHex("#a782bb"),
			Primitive:    keywordOrange,
			Object:       ColorFromHex("#b9becf"),
			Preproc:      ColorFromHex("#bbb529"),
			DiffAdded:    consoleGreen,
			DiffRemoved:  consoleRed,

			KeywordAttrs:  AttrBold,
			ConstantAttrs: AttrItalic,
			CommentAttrs:  AttrItalic,
			ErrorAttrs:    AttrUnderline,
		},
	}
}
