package widget

import (
	"regexp"
	"strconv"
	"strings"
)

// StructuredPatch represents a parsed unified diff for a single file.
type StructuredPatch struct {
	OldFileName string
	NewFileName string
	OldHeader   string
	NewHeader   string
	FileHeaders []string // raw header lines before the first hunk (diff, index, ---, +++)
	Hunks       []DiffHunk
}

// ---------------------------------------------------------------------------
// DiffLayout — view-independent layout for unified & split rendering
// ---------------------------------------------------------------------------

// DiffLayout is the view-independent layout produced after parsing a diff.
// It organises the diff into typed blocks that can be rendered in both
// unified and split modes without redundant line-number tracking or
// change-block pairing. The block sequence mirrors git diff output.
type DiffLayout struct {
	Blocks     []DiffBlock
	MaxLineNum int // largest source line number across all blocks
}

// DiffBlockKind classifies a block in the diff layout.
type DiffBlockKind int

const (
	DiffBlockFullWidth DiffBlockKind = iota // File header or hunk header (spans full width)
	DiffBlockContext                        // Unchanged context line (same on both sides)
	DiffBlockChange                         // Change block: consecutive removes then adds
)

// DiffBlock represents one logical element in the diff layout.
//
//   - DiffBlockFullWidth: Text is the header line, Type is diffHeader or diffHunk.
//   - DiffBlockContext:   Text is the line content, OldLineNum/NewLineNum are set.
//   - DiffBlockChange:    Removes and Adds hold the changed lines.
type DiffBlock struct {
	Kind DiffBlockKind

	// Shared by FullWidth and Context:
	Text string       // display text
	Type diffLineType // line classification

	// Context only:
	OldLineNum int // line number in old file
	NewLineNum int // line number in new file

	// Change only:
	Removes []DiffLine // removed lines (old file side)
	Adds    []DiffLine // added lines (new file side)
}

// DiffLine represents a single line within a change block.
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
	Header   string // the @@ line
	Lines    []HunkLine
}

// HunkLine is a single line within a hunk.
type HunkLine struct {
	Content string       // line content without the prefix character
	Type    diffLineType // diffAdded, diffRemoved, or diffContext
}

// hunkHeaderRe matches @@ -old,oldlen +new,newlen @@ ...
var hunkHeaderRe = regexp.MustCompile(`^@@\s+-(\d+)(?:,(\d+))?\s+\+(\d+)(?:,(\d+))?\s+@@(.*)$`)

// ParsePatch parses a unified diff string into a StructuredPatch.
// Returns nil if the input doesn't look like a unified diff.
func ParsePatch(text string) *StructuredPatch {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return nil
	}

	patch := &StructuredPatch{}
	i := 0

	// Collect all lines before the first hunk header as file headers.
	// This captures diff, index, ---, +++ and any other preamble lines.
	for i < len(lines) && !hunkHeaderRe.MatchString(lines[i]) {
		patch.FileHeaders = append(patch.FileHeaders, lines[i])

		// Also extract --- / +++ file names.
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

	// Parse hunks
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

		// Parse hunk lines
		for i < len(lines) {
			l := lines[i]
			if len(l) == 0 {
				// Empty line in diff means context line with no content
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
				// "\ No newline at end of file" — skip
				i++
				continue
			default:
				// Hit next hunk header or non-diff content
				if hunkHeaderRe.MatchString(l) {
					goto nextHunk
				}
				// Treat as context
				hunk.Lines = append(hunk.Lines, HunkLine{Content: l, Type: diffContext})
			}
			i++
		}
	nextHunk:
		patch.Hunks = append(patch.Hunks, hunk)
	}

	return patch
}

// ---------------------------------------------------------------------------
// BuildDiffLayout
// ---------------------------------------------------------------------------

// BuildDiffLayout produces a view-independent layout from a StructuredPatch.
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

	// File headers as full-width blocks.
	for _, h := range patch.FileHeaders {
		layout.Blocks = append(layout.Blocks, DiffBlock{
			Kind: DiffBlockFullWidth,
			Text: h,
			Type: diffHeader,
		})
	}

	for _, hunk := range patch.Hunks {
		// Hunk header.
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

			// Collect consecutive removed then added lines as a change block.
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

// ---------------------------------------------------------------------------
// Legacy helpers (kept for API compatibility)
// ---------------------------------------------------------------------------

// LogicalDiffLine represents a fully-resolved line for display in the diff widget.
type LogicalDiffLine struct {
	Content     string       // display text
	Type        diffLineType // diffAdded, diffRemoved, diffContext, diffHeader, diffHunk
	OldLineNum  int          // 0 = not applicable
	NewLineNum  int          // 0 = not applicable
	HideLineNum bool         // true for padding lines in split view
}

// BuildUnifiedLines produces a flat list of logical lines from a structured patch.
func BuildUnifiedLines(patch *StructuredPatch) []LogicalDiffLine {
	var result []LogicalDiffLine

	for _, hunk := range patch.Hunks {
		// Add hunk header
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

// BuildSplitLines produces left and right line lists for a side-by-side diff view.
func BuildSplitLines(patch *StructuredPatch) (left, right []LogicalDiffLine) {
	for _, hunk := range patch.Hunks {
		// Add hunk header to both sides
		headerLine := LogicalDiffLine{
			Content: hunk.Header,
			Type:    diffHunk,
		}
		left = append(left, headerLine)
		right = append(right, headerLine)

		oldLine := hunk.OldStart
		newLine := hunk.NewStart

		// Collect change blocks (consecutive - and + lines)
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

			// Collect consecutive removed and added lines as a change block
			var removed, added []HunkLine
			for i < len(hunk.Lines) && hunk.Lines[i].Type == diffRemoved {
				removed = append(removed, hunk.Lines[i])
				i++
			}
			for i < len(hunk.Lines) && hunk.Lines[i].Type == diffAdded {
				added = append(added, hunk.Lines[i])
				i++
			}

			// Pair up removed (left) and added (right), padding the shorter side
			maxLen := max(len(added), len(removed))

			for j := 0; j < maxLen; j++ {
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
