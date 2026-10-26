package widget

import (
	"regexp"
	"strconv"
	"strings"
)

// structuredPatch contains the headers and hunks parsed from a unified diff.
type structuredPatch struct {
	// OldFileName is the text after "--- ", such as "a/main.go".
	OldFileName string
	// NewFileName is the text after "+++ ", such as "b/main.go".
	NewFileName string
	// OldHeader is not set by parsePatch, which keeps the raw lines in
	// FileHeaders.
	OldHeader string
	// NewHeader is not set by parsePatch, which keeps the raw lines in
	// FileHeaders.
	NewHeader string
	// FileHeaders holds the raw lines before the first hunk, including diff,
	// index, ---, and +++.
	FileHeaders []string
	// Hunks are the patch's hunks, in order.
	Hunks []patchHunk
}

// diffLayout is a view-independent sequence of parsed diff blocks.
// Both unified and split renderers consume the same sequence and line numbers.
type diffLayout struct {
	// Blocks are the blocks, in display order.
	Blocks []diffBlock
	// MaxLineNum is the largest old-file or new-file line number in Blocks.
	MaxLineNum int
}

// diffBlockKind classifies a block in the diff layout.
type diffBlockKind int

const (
	diffBlockFullWidth diffBlockKind = iota // File or hunk header spanning both split panes.
	diffBlockContext                        // Unchanged line displayed on both sides.
	diffBlockChange                         // Consecutive removed lines followed by added lines.
)

// diffBlock contains one logical element in a diffLayout.
//
//   - diffBlockFullWidth: Text is the header line, Type is diffHeader or diffHunk.
//   - diffBlockContext:   Text is the line content, OldLineNum/NewLineNum are set.
//   - diffBlockChange:    Removes and Adds hold the changed lines.
type diffBlock struct {
	// Kind decides which of the fields below the block uses.
	Kind diffBlockKind

	// Text is the text displayed for a full-width or context block.
	Text string
	// Type is the classification used to select the block's colors.
	Type diffLineType

	// OldLineNum is a context block's line number in the old file.
	OldLineNum int
	// NewLineNum is a context block's line number in the new file.
	NewLineNum int

	// Removes are a change block's removed lines, numbered in the old file.
	Removes []diffLine
	// Adds are a change block's added lines, numbered in the new file.
	Adds []diffLine
}

// diffLine contains one changed line and its source-file line number.
type diffLine struct {
	// Content is the line content without its diff prefix.
	Content string
	// LineNum is the line's number in its file.
	LineNum int
}

// patchHunk represents a single hunk from a unified diff.
type patchHunk struct {
	// OldStart is the first old-file line, from the hunk header.
	OldStart int
	// OldLines is the old-file line count; 1 when the header omits it.
	OldLines int
	// NewStart is the first new-file line, from the hunk header.
	NewStart int
	// NewLines is the new-file line count; 1 when the header omits it.
	NewLines int
	// Header is the complete @@ hunk-header line.
	Header string
	// Lines are the hunk's lines, in order.
	Lines []hunkLine
}

// hunkLine is a single line within a hunk.
type hunkLine struct {
	// Content is the line content without its diff prefix.
	Content string
	// Type is one of diffAdded, diffRemoved, or diffContext.
	Type diffLineType
}

// hunkHeaderRe captures old and new ranges from a unified-diff hunk header.
var hunkHeaderRe = regexp.MustCompile(`^@@\s+-(\d+)(?:,(\d+))?\s+\+(\d+)(?:,(\d+))?\s+@@(.*)$`)

// parsePatch parses unified-diff headers and hunks from text.
// Text before the first valid hunk is retained in FileHeaders.
func parsePatch(text string) *structuredPatch {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return nil
	}

	patch := &structuredPatch{}
	i := 0

	// Retain the complete preamble before the first valid hunk header.
	for i < len(lines) && !hunkHeaderRe.MatchString(lines[i]) {
		patch.FileHeaders = append(patch.FileHeaders, lines[i])

		// Extract old and new names while preserving their raw header lines.
		if strings.HasPrefix(lines[i], "---") {
			parts := strings.SplitN(lines[i], " ", 2)
			if len(parts) > 1 {
				patch.OldFileName = parts[1]
			}
		}
		if strings.HasPrefix(lines[i], "+++") {
			parts := strings.SplitN(lines[i], " ", 2)
			if len(parts) > 1 {
				patch.NewFileName = parts[1]
			}
		}
		i++
	}

	// Parse each recognized hunk and skip unrelated lines between hunks.
	for i < len(lines) {
		line := lines[i]
		matches := hunkHeaderRe.FindStringSubmatch(line)
		if matches == nil {
			i++
			continue
		}

		hunk := patchHunk{Header: line}
		hunk.OldStart, _ = strconv.Atoi(matches[1])
		if matches[2] != "" {
			hunk.OldLines, _ = strconv.Atoi(matches[2])
		} else {
			hunk.OldLines = 1
		}
		hunk.NewStart, _ = strconv.Atoi(matches[3])
		if matches[4] != "" {
			hunk.NewLines, _ = strconv.Atoi(matches[4])
		} else {
			hunk.NewLines = 1
		}

		i++

		// Classify prefixed lines until the next hunk header or end of input.
		for i < len(lines) {
			l := lines[i]
			if len(l) == 0 {
				// Treat an unprefixed empty line as empty context.
				hunk.Lines = append(hunk.Lines, hunkLine{Content: "", Type: diffContext})
				i++
				continue
			}

			prefix := l[0]
			content := ""
			if len(l) > 1 {
				content = l[1:]
			}

			switch prefix {
			case '+':
				hunk.Lines = append(hunk.Lines, hunkLine{Content: content, Type: diffAdded})
			case '-':
				hunk.Lines = append(hunk.Lines, hunkLine{Content: content, Type: diffRemoved})
			case ' ':
				hunk.Lines = append(hunk.Lines, hunkLine{Content: content, Type: diffContext})
			case '\\':
				// Ignore the standard "No newline at end of file" marker.
				i++
				continue
			default:
				// Leave the current hunk when the next header starts.
				if hunkHeaderRe.MatchString(l) {
					goto nextHunk
				}
				// Preserve other unprefixed content as context.
				hunk.Lines = append(hunk.Lines, hunkLine{Content: l, Type: diffContext})
			}
			i++
		}
	nextHunk:
		patch.Hunks = append(patch.Hunks, hunk)
	}

	return patch
}

// buildDiffLayout produces a view-independent layout from patch.
// The resulting diffLayout can drive both unified and split renderers:
//
//   - Unified: iterate blocks sequentially; for diffBlockChange emit all
//     Removes then all Adds (standard git unified-diff order).
//   - Split: iterate blocks; for diffBlockChange pair Removes[i] with
//     Adds[i], padding the shorter side with empty rows.
func buildDiffLayout(patch *structuredPatch) *diffLayout {
	if patch == nil {
		return &diffLayout{}
	}

	layout := &diffLayout{}
	maxNum := 0

	trackMax := func(n int) {
		if n > maxNum {
			maxNum = n
		}
	}

	// File headers span the full view width.
	for _, h := range patch.FileHeaders {
		layout.Blocks = append(layout.Blocks, diffBlock{
			Kind: diffBlockFullWidth,
			Text: h,
			Type: diffHeader,
		})
	}

	for _, hunk := range patch.Hunks {
		// Each hunk header also spans the full view width.
		layout.Blocks = append(layout.Blocks, diffBlock{
			Kind: diffBlockFullWidth,
			Text: hunk.Header,
			Type: diffHunk,
		})

		oldNum := hunk.OldStart
		newNum := hunk.NewStart

		i := 0
		for i < len(hunk.Lines) {
			hl := hunk.Lines[i]

			if hl.Type == diffContext {
				trackMax(oldNum)
				trackMax(newNum)
				layout.Blocks = append(layout.Blocks, diffBlock{
					Kind:       diffBlockContext,
					Text:       hl.Content,
					Type:       diffContext,
					OldLineNum: oldNum,
					NewLineNum: newNum,
				})
				oldNum++
				newNum++
				i++
				continue
			}

			// Group adjacent removals and additions for split-view pairing.
			var removes []diffLine
			var adds []diffLine

			for i < len(hunk.Lines) && hunk.Lines[i].Type == diffRemoved {
				trackMax(oldNum)
				removes = append(removes, diffLine{
					Content: hunk.Lines[i].Content,
					LineNum: oldNum,
				})
				oldNum++
				i++
			}
			for i < len(hunk.Lines) && hunk.Lines[i].Type == diffAdded {
				trackMax(newNum)
				adds = append(adds, diffLine{
					Content: hunk.Lines[i].Content,
					LineNum: newNum,
				})
				newNum++
				i++
			}

			layout.Blocks = append(layout.Blocks, diffBlock{
				Kind:    diffBlockChange,
				Removes: removes,
				Adds:    adds,
			})
		}
	}

	layout.MaxLineNum = maxNum
	return layout
}

// logicalDiffLine contains one flattened display line from a parsed diff.
type logicalDiffLine struct {
	// Content is the display text.
	Content string
	// Type is the classification used to select colors.
	Type diffLineType
	// OldLineNum is the line number in the old file, or zero when not
	// applicable.
	OldLineNum int
	// NewLineNum is the line number in the new file, or zero when not
	// applicable.
	NewLineNum int
	// HideLineNum is true for padding rows in split view.
	HideLineNum bool
}

// buildUnifiedLines flattens patch hunks into sequential logical lines.
func buildUnifiedLines(patch *structuredPatch) []logicalDiffLine {
	var result []logicalDiffLine

	for _, hunk := range patch.Hunks {
		// Emit each hunk header before its content.
		result = append(result, logicalDiffLine{
			Content: hunk.Header,
			Type:    diffHunk,
		})

		oldLine := hunk.OldStart
		newLine := hunk.NewStart

		for _, hl := range hunk.Lines {
			dl := logicalDiffLine{
				Content: hl.Content,
				Type:    hl.Type,
			}

			switch hl.Type {
			case diffContext:
				dl.OldLineNum = oldLine
				dl.NewLineNum = newLine
				oldLine++
				newLine++
			case diffAdded:
				dl.NewLineNum = newLine
				newLine++
			case diffRemoved:
				dl.OldLineNum = oldLine
				oldLine++
			}

			result = append(result, dl)
		}
	}

	return result
}

// buildSplitLines flattens patch into aligned old-file and new-file line slices.
func buildSplitLines(patch *structuredPatch) (left, right []logicalDiffLine) {
	for _, hunk := range patch.Hunks {
		// Repeat hunk headers on both sides.
		headerLine := logicalDiffLine{
			Content: hunk.Header,
			Type:    diffHunk,
		}
		left = append(left, headerLine)
		right = append(right, headerLine)

		oldLine := hunk.OldStart
		newLine := hunk.NewStart

		// Process each adjacent removal/addition group as one change block.
		i := 0
		for i < len(hunk.Lines) {
			hl := hunk.Lines[i]

			if hl.Type == diffContext {
				dl := logicalDiffLine{
					Content:    hl.Content,
					Type:       diffContext,
					OldLineNum: oldLine,
					NewLineNum: newLine,
				}
				left = append(left, dl)
				right = append(right, dl)
				oldLine++
				newLine++
				i++
				continue
			}

			// Collect removals followed by additions.
			var removed, added []hunkLine
			for i < len(hunk.Lines) && hunk.Lines[i].Type == diffRemoved {
				removed = append(removed, hunk.Lines[i])
				i++
			}
			for i < len(hunk.Lines) && hunk.Lines[i].Type == diffAdded {
				added = append(added, hunk.Lines[i])
				i++
			}

			// Pair changes by index and pad the shorter side.
			maxLen := max(len(added), len(removed))

			for j := range maxLen {
				if j < len(removed) {
					left = append(left, logicalDiffLine{
						Content:    removed[j].Content,
						Type:       diffRemoved,
						OldLineNum: oldLine,
					})
					oldLine++
				} else {
					left = append(left, logicalDiffLine{
						Type:        diffContext,
						HideLineNum: true,
					})
				}

				if j < len(added) {
					right = append(right, logicalDiffLine{
						Content:    added[j].Content,
						Type:       diffAdded,
						NewLineNum: newLine,
					})
					newLine++
				} else {
					right = append(right, logicalDiffLine{
						Type:        diffContext,
						HideLineNum: true,
					})
				}
			}
		}
	}

	return left, right
}
