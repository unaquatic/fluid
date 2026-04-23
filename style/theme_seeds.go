package style

// ThemeSeeds is the minimum set of colors needed to derive a full Theme.
//
// A seeds-based theme trades precise control for concision: give 7
// anchor colors and the whole surface / text / border / markdown /
// diff vocabulary is filled in automatically from 12-step OKLCH ramps.
// Fields you still want to override after construction can be set
// directly on the returned Theme.
//
// The seed vocabulary is tuned to avoid two common failure modes:
// with fewer seeds, semantic roles get synthesised from the primary
// (e.g. success = hue-rotated brand) which produces muddy results;
// with more seeds, authors specify ceremony without payoff. Seven
// seeds is the sweet spot for general-purpose themes.
//
// Every seed field must be an opaque color; transparent seeds are
// rejected at construction time.
type ThemeSeeds struct {
	// Neutral is the canvas hue. In dark mode it should be a very dark
	// color (near black); in light mode, near white. A slight tint is
	// welcome and will propagate to every surface, border, and muted
	// text color — this is how themes like Nord or Rose Pine stay
	// harmonious even in "gray" areas.
	Neutral Color

	// Ink is the body text color. If zero, a default is derived from
	// Neutral (near-white for dark, near-black for light). Providing
	// Ink explicitly lets authors give body text a measurable tint of
	// its own, independent of the neutral canvas.
	Ink Color

	// Primary is the brand color. Used for headings, markdown bold,
	// and whatever a theme considers "the" accent. Distinct from
	// Interactive so themes can have a separate hue for
	// links/selection (e.g. purple headings + blue links).
	Primary Color

	// Interactive drives clickable things: links, focus rings, menu
	// selection, cursor. This is what should stand out as the user's
	// attention cue. If zero, falls back to Primary.
	Interactive Color

	// Semantic hues. These are deliberately explicit rather than
	// derived from Primary because "success looks green" / "danger
	// looks red" is a strong UX convention that hue-rotating a brand
	// color fails to satisfy reliably.
	Success Color
	Warning Color
	Danger  Color
	Info    Color
}

// withDefaults fills zero fields with sensible fallbacks so callers
// can omit Ink and still get a usable theme. Primary-or-Interactive
// collapse is handled here to keep the main constructor readable.
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

// ThemeFromSeeds constructs a full Theme from seed colors alone. Every
// role — surfaces, borders, text, accents, markdown, diff, and the
// code-highlighting palette — is derived from OKLCH ramps built off
// the seeds. Nothing is hand-picked.
//
// The default code palette is NeutralCodePalette(neutralRamp), which
// distinguishes tokens via bold/italic/underline rather than hue. This
// is a safe baseline that matches any theme. For chromatic syntax
// highlighting (Monokai, Dracula, One Dark, …) mutate t.Code after
// construction; syntax palettes encode taste and don't derive cleanly
// from seeds.
//
// dark selects the perceptual direction of the ramps: dark themes use
// low-L bg and high-L fg; light themes invert.
//
// The name is passed through verbatim; use something stable so test
// diffs stay readable.
func ThemeFromSeeds(name string, seeds ThemeSeeds, dark bool) *Theme {
	seeds = seeds.withDefaults(dark)

	// 12-step ramps. Indices 0..11 go from near-bg to near-fg.
	neutral := GenerateNeutralScale(seeds.Neutral, dark, seeds.Ink)
	primary := GenerateScale(seeds.Primary, dark)
	interactive := GenerateScale(seeds.Interactive, dark)
	success := GenerateScale(seeds.Success, dark)
	warning := GenerateScale(seeds.Warning, dark)
	danger := GenerateScale(seeds.Danger, dark)
	info := GenerateScale(seeds.Info, dark)

	// Indexing helpers: the typical "use this shade for X" lookups.
	// For most chromatic roles we use step 9 or 10 (the most visible
	// tint in the ramp); for surfaces we use low steps (close to bg).
	// Inside helpers we keep dark/light branches minimal so the ramps
	// remain the source of truth.
	bg := neutral[0]
	surface := neutral[0]
	surfaceAlt := neutral[2]
	surfaceRaised := neutral[3]

	// Borders: cooler than text, warmer than bg. Step 5 reads as a
	// hairline; step 7 is the "emphasised" divider.
	borderWeak := neutral[4]
	border := neutral[5]
	borderStrong := neutral[7]

	// Text: step 11 is the highest-contrast reading, step 8 the secondary.
	textStrong := neutral[11]
	textBase := neutral[10]
	textWeak := neutral[8]
	textMuted := neutral[7]

	// Accent / interactive: the saturated seed sits at step 10.
	accent := primary[10]
	accentStrong := primary[9]
	interactiveBase := interactive[10]
	interactiveStrong := interactive[9]

	// Heading gradient: descend the primary ramp so H1 is the boldest
	// and H6 fades toward the body. This stays within the primary hue
	// family — no rainbow — which reads as "serious" rather than
	// "playful". Theme authors who want color-per-heading can overwrite.
	//
	// We sample [10, 9, 8, 7, 6, 5] rather than mixing in textWeak at
	// the end so lightness stays strictly monotonic: textWeak comes
	// from the neutral ramp and is typically brighter than primary[6],
	// which would invert the H5→H6 step.
	headings := [6]Color{
		primary[10], primary[9], primary[8], primary[7], primary[6], primary[5],
	}

	// Diff surfaces: low-step ramp entries (barely perceptible tint).
	// Diff markers: saturated mid-ramp entries.
	var diffAddSurface, diffRemoveSurface Color
	if dark {
		diffAddSurface = success[2]
		diffRemoveSurface = danger[2]
	} else {
		diffAddSurface = success[1]
		diffRemoveSurface = danger[1]
	}

	// Selection: step 3 of the interactive ramp sits close enough to
	// bg that it doesn't overpower selected text, but reads as a
	// clear "this row is selected" signal. Inactive = one step
	// darker. Fg auto-picks black/white via ChooseOnColor.
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

		// On-colors pick black/white against the semantic fill.
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

		MarkdownHeading:   headings,
		MarkdownText:      textBase,
		MarkdownLink:      interactive[10],
		MarkdownLinkText:  interactive[8],
		MarkdownCode:      success[10],   // green reads as "safe to execute"
		MarkdownEmph:      warning[10],
		MarkdownStrong:    primary[10],
		MarkdownBullet:    interactive[10],
		MarkdownListEnum:  info[10],
		MarkdownQuoteFg:   textWeak,
		MarkdownQuoteBd:   border,
		MarkdownRule:      borderWeak,
		MarkdownImage:     interactive[10],
		MarkdownImageText: interactive[8],

		CodeFg: textBase,
		CodeBg: surface, // match surface so code blocks read as no-chip
		Code:   NeutralCodePalette(neutral),
	}
	return t
}
