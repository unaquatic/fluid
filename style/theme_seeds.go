package style

// ThemeSeeds contains the anchor colors used to derive a complete Theme.
//
// ThemeFromSeeds expands these colors into 12-step OKLCH ramps for surface,
// text, border, markdown, diff, and code roles. Callers can override fields on
// the returned Theme.
type ThemeSeeds struct {
	// Neutral seeds surfaces, borders, and muted text. Use a dark color for a
	// dark theme and a light color for a light theme.
	Neutral Color

	// Ink seeds body text. A transparent value uses near-white in dark themes and
	// near-black in light themes.
	Ink Color

	// Primary seeds headings and accent roles.
	Primary Color

	// Interactive seeds links, focus, and selection. A transparent value inherits
	// Primary.
	Interactive Color

	// Semantic colors seed status-specific ramps.
	Success Color
	Warning Color
	Danger  Color
	Info    Color
}

// withDefaults resolves the optional Interactive and Ink seeds.
func (s ThemeSeeds) withDefaults(dark bool) ThemeSeeds {
	out := s
	if out.Interactive.A == 0 {
		out.Interactive = out.Primary
	}
	if out.Ink.A == 0 {
		if dark {
			out.Ink = ColorFromInts(240, 240, 240, 255)
		} else {
			out.Ink = ColorFromInts(20, 20, 20, 255)
		}
	}
	return out
}

// ThemeFromSeeds returns a complete theme derived from OKLCH ramps of seeds.
//
// The default syntax palette uses the neutral ramp and text attributes rather
// than distinct token hues. Callers can replace Theme.Code after construction.
//
// dark selects whether ramps progress from dark to light or light to dark. The
// returned theme retains name unchanged.
func ThemeFromSeeds(name string, seeds ThemeSeeds, dark bool) *Theme {
	seeds = seeds.withDefaults(dark)

	// Each ramp progresses from low-contrast surfaces to high-contrast text.
	neutral := GenerateNeutralScale(seeds.Neutral, dark, seeds.Ink)
	primary := GenerateScale(seeds.Primary, dark)
	interactive := GenerateScale(seeds.Interactive, dark)
	success := GenerateScale(seeds.Success, dark)
	warning := GenerateScale(seeds.Warning, dark)
	danger := GenerateScale(seeds.Danger, dark)
	info := GenerateScale(seeds.Info, dark)

	// Low neutral steps supply surfaces; higher steps supply text and accents.
	bg := neutral[0]
	surface := neutral[0]
	surfaceAlt := neutral[2]
	surfaceRaised := neutral[3]

	// Borders use middle neutral steps between surfaces and text.
	borderWeak := neutral[4]
	border := neutral[5]
	borderStrong := neutral[7]

	// Text roles descend from the highest-contrast neutral step.
	textStrong := neutral[11]
	textBase := neutral[10]
	textWeak := neutral[8]
	textMuted := neutral[7]

	// Accent and interactive roles use prominent chromatic steps.
	accent := primary[10]
	accentStrong := primary[9]
	interactiveBase := interactive[10]
	interactiveStrong := interactive[9]

	// Heading levels descend one primary-ramp step at a time to preserve a
	// monotonic, single-hue hierarchy.
	headings := [6]Entry{
		{Fg: primary[10], Attrs: AttrBold},
		{Fg: primary[9], Attrs: AttrBold},
		{Fg: primary[8], Attrs: AttrBold},
		{Fg: primary[7], Attrs: AttrBold},
		{Fg: primary[6], Attrs: AttrBold},
		{Fg: primary[5], Attrs: AttrBold},
	}

	// Diff surfaces use subtle steps; markers use prominent steps.
	var diffAddSurface, diffRemoveSurface Color
	if dark {
		diffAddSurface = success[2]
		diffRemoveSurface = danger[2]
	} else {
		diffAddSurface = success[1]
		diffRemoveSurface = danger[1]
	}

	// Semantic surfaces sit one step above the diff surfaces, so a diff row
	// stays distinct inside them.
	semanticStep := 2
	if dark {
		semanticStep = 3
	}

	// Selection uses a low interactive step, with the preceding step for its
	// inactive state.
	selectionBg := interactive[3]
	selectionInactiveBg := interactive[2]

	t := &Theme{
		Name: name,

		Bg:            bg,
		Surface:       surface,
		SurfaceAlt:    surfaceAlt,
		SurfaceRaised: surfaceRaised,
		Border:        border,
		BorderWeak:    borderWeak,
		BorderStrong:  borderStrong,
		BorderFocus:   interactive[8],

		TextStrong: textStrong,
		TextBase:   textBase,
		TextWeak:   textWeak,
		TextMuted:  textMuted,
		TextLink:   interactive[10],

		Accent:            accent,
		AccentStrong:      accentStrong,
		Interactive:       interactiveBase,
		InteractiveStrong: interactiveStrong,

		Success: success[10],
		Warning: warning[10],
		Danger:  danger[10],
		Info:    info[10],

		SuccessSurface: success[semanticStep],
		WarningSurface: warning[semanticStep],
		DangerSurface:  danger[semanticStep],
		InfoSurface:    info[semanticStep],

		// Filled roles use an automatically contrasting foreground.
		OnAccent:      ChooseOnColor(accent),
		OnInteractive: ChooseOnColor(interactiveBase),
		OnSuccess:     ChooseOnColor(success[10]),
		OnWarning:     ChooseOnColor(warning[10]),
		OnDanger:      ChooseOnColor(danger[10]),
		OnInfo:        ChooseOnColor(info[10]),

		SelectionBg:         selectionBg,
		SelectionFg:         ChooseOnColor(selectionBg),
		SelectionInactiveBg: selectionInactiveBg,

		DiffAddSurface:    diffAddSurface,
		DiffRemoveSurface: diffRemoveSurface,
		DiffAddFg:         success[10],
		DiffRemoveFg:      danger[10],

		MarkdownHeading:     headings,
		MarkdownText:        Entry{Fg: textBase},
		MarkdownLink:        Entry{Fg: interactive[10], Attrs: AttrUnderline},
		MarkdownLinkText:    Entry{Fg: interactive[8], Attrs: AttrUnderline},
		MarkdownCode:        Entry{Fg: success[10], Bg: surface}, // Distinguish inline code from prose.
		MarkdownEmph:        Entry{Fg: warning[10], Attrs: AttrItalic},
		MarkdownStrong:      Entry{Fg: primary[10], Attrs: AttrBold},
		MarkdownBullet:      Entry{Fg: interactive[10]},
		MarkdownListEnum:    Entry{Fg: info[10]},
		MarkdownQuote:       Entry{Fg: textWeak, Attrs: AttrItalic},
		MarkdownQuoteBorder: Entry{Fg: border},
		MarkdownRule:        Entry{Fg: borderWeak},
		MarkdownImage:       Entry{Fg: interactive[10], Attrs: AttrUnderline},
		MarkdownImageText:   Entry{Fg: interactive[8], Attrs: AttrUnderline},

		CodeFg: textBase,
		CodeBg: surface, // Avoid a separate background chip around code blocks.
		Code:   NeutralCodePalette(neutral),
	}
	return t
}
