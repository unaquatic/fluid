package widget

import (
	"regexp"
	"strconv"
	"strings"
)

// StructuredPatch contains the headers and hunks parsed from a unified diff.
type StructuredPatch struct {
	OldFileName string
	NewFileName string
	OldHeader   string
	NewHeader   string
	FileHeaders []string // Raw lines before the first hunk, including diff, index, ---, and +++.
	Hunks       []DiffHunk
}

// DiffLayout is a view-independent sequence of parsed diff blocks.
// Both unified and split renderers consume the same sequence and line numbers.
type DiffLayout struct {
	Blocks     []DiffBlock
	MaxLineNum int // Largest old-file or new-file line number in Blocks.
}

// DiffBlockKind classifies a block in the diff layout.
type DiffBlockKind int

const (
	DiffBlockFullWidth DiffBlockKind = iota // File or hunk header spanning both split panes.
	DiffBlockContext                        // Unchanged line displayed on both sides.
	DiffBlockChange                         // Consecutive removed lines followed by added lines.
)

// DiffBlock contains one logical element in a DiffLayout.
//
//   - DiffBlockFullWidth: Text is the header line, Type is diffHeader or diffHunk.
//   - DiffBlockContext:   Text is the line content, OldLineNum/NewLineNum are set.
//   - DiffBlockChange:    Removes and Adds hold the changed lines.
type DiffBlock struct {
	Kind DiffBlockKind

	// Text and Type describe full-width and context blocks.
	Text string       // Text displayed for the block.
	Type diffLineType // Classification used to select colors.

	// Context blocks carry a line number for each file.
	OldLineNum int // Line number in the old file.
	NewLineNum int // Line number in the new file.

	// Change blocks separate old-file removals from new-file additions.
	Removes []DiffLine
	Adds    []DiffLine
}

// DiffLine contains one changed line and its source-file line number.
type DiffLine struct {
	Content string
	LineNum int
}

// DiffHunk represents a single hunk from a unified diff.
type DiffHunk struct {
	OldStart int
	OldLines int
	NewStart int
	NewLines int
	Header   string // Complete @@ hunk-header line.
	Lines    []HunkLine
}

// HunkLine is a single line within a hunk.
type HunkLine struct {
	Content string       // Line content without its diff prefix.
	Type    diffLineType // One of diffAdded, diffRemoved, or diffContext.
}

// hunkHeaderRe captures old and new ranges from a unified-diff hunk header.
var hunkHeaderRe = regexp.MustCompile(`^@@\s+-(\d+)(?:,(\d+))?\s+\+(\d+)(?:,(\d+))?\s+@@(.*)$`)

// ParsePatch parses unified-diff headers and hunks from text.
// Text before the first valid hunk is retained in FileHeaders.
func ParsePatch(text string) *StructuredPatch {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return nil
	}

	patch := &StructuredPatch{}
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

		hunk := DiffHunk{Header: line}
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
				hunk.Lines = append(hunk.Lines, HunkLine{Content: "", Type: diffContext})
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
				hunk.Lines = append(hunk.Lines, HunkLine{Content: content, Type: diffAdded})
			case '-':
				hunk.Lines = append(hunk.Lines, HunkLine{Content: content, Type: diffRemoved})
			case ' ':
				hunk.Lines = append(hunk.Lines, HunkLine{Content: content, Type: diffContext})
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
				hunk.Lines = append(hunk.Lines, HunkLine{Content: l, Type: diffContext})
			}
			i++
		}
	nextHunk:
		patch.Hunks = append(patch.Hunks, hunk)
	}

	return patch
}

// BuildDiffLayout produces a view-independent layout from patch.
// The resulting DiffLayout can drive both unified and split renderers:
//
//   - Unified: iterate blocks sequentially; for DiffBlockChange emit all
//     Removes then all Adds (standard git unified-diff order).
//   - Split: iterate blocks; for DiffBlockChange pair Removes[i] with
//     Adds[i], padding the shorter side with empty rows.
func BuildDiffLayout(patch *StructuredPatch) *DiffLayout {
	if patch == nil {
		return &DiffLayout{}
	}

	layout := &DiffLayout{}
	maxNum := 0

	trackMax := func(n int) {
		if n > maxNum {
			maxNum = n
		}
	}

	// File headers span the full view width.
	for _, h := range patch.FileHeaders {
		layout.Blocks = append(layout.Blocks, DiffBlock{
			Kind: DiffBlockFullWidth,
			Text: h,
			Type: diffHeader,
		})
	}

	for _, hunk := range patch.Hunks {
		// Each hunk header also spans the full view width.
		layout.Blocks = append(layout.Blocks, DiffBlock{
			Kind: DiffBlockFullWidth,
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
				layout.Blocks = append(layout.Blocks, DiffBlock{
					Kind:       DiffBlockContext,
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
			var removes []DiffLine
			var adds []DiffLine

			for i < len(hunk.Lines) && hunk.Lines[i].Type == diffRemoved {
				trackMax(oldNum)
				removes = append(removes, DiffLine{
					Content: hunk.Lines[i].Content,
					LineNum: oldNum,
				})
				oldNum++
				i++
			}
			for i < len(hunk.Lines) && hunk.Lines[i].Type == diffAdded {
				trackMax(newNum)
				adds = append(adds, DiffLine{
					Content: hunk.Lines[i].Content,
					LineNum: newNum,
				})
				newNum++
				i++
			}

			layout.Blocks = append(layout.Blocks, DiffBlock{
				Kind:    DiffBlockChange,
				Removes: removes,
				Adds:    adds,
			})
		}
	}

	layout.MaxLineNum = maxNum
	return layout
}

// LogicalDiffLine contains one flattened display line from a parsed diff.
type LogicalDiffLine struct {
	Content     string       // Display text.
	Type        diffLineType // Classification used to select colors.
	OldLineNum  int          // Zero when not applicable.
	NewLineNum  int          // Zero when not applicable.
	HideLineNum bool         // True for padding rows in split view.
}

// BuildUnifiedLines flattens patch hunks into sequential logical lines.
func BuildUnifiedLines(patch *StructuredPatch) []LogicalDiffLine {
	var result []LogicalDiffLine

	for _, hunk := range patch.Hunks {
		// Emit each hunk header before its content.
		result = append(result, LogicalDiffLine{
			Content: hunk.Header,
			Type:    diffHunk,
		})

		oldLine := hunk.OldStart
		newLine := hunk.NewStart

		for _, hl := range hunk.Lines {
			dl := LogicalDiffLine{
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

// BuildSplitLines flattens patch into aligned old-file and new-file line slices.
func BuildSplitLines(patch *StructuredPatch) (left, right []LogicalDiffLine) {
	for _, hunk := range patch.Hunks {
		// Repeat hunk headers on both sides.
		headerLine := LogicalDiffLine{
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
				dl := LogicalDiffLine{
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
			var removed, added []HunkLine
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
					left = append(left, LogicalDiffLine{
						Content:    removed[j].Content,
						Type:       diffRemoved,
						OldLineNum: oldLine,
					})
					oldLine++
				} else {
					left = append(left, LogicalDiffLine{
						Type:        diffContext,
						HideLineNum: true,
					})
				}

				if j < len(added) {
					right = append(right, LogicalDiffLine{
						Content:    added[j].Content,
						Type:       diffAdded,
						NewLineNum: newLine,
					})
					newLine++
				} else {
					right = append(right, LogicalDiffLine{
						Type:        diffContext,
						HideLineNum: true,
					})
				}
			}
		}
	}

	return left, right
}
