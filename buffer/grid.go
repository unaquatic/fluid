package buffer

import (
	"github.com/unaquatic/fluid/style"
)

// GridDrawOptions configures the DrawGrid method.
type GridDrawOptions struct {
	// Chars defines the border character set to use.
	Chars style.BorderCharacters
	// BorderFg is the foreground color for all border characters.
	BorderFg style.Color
	// BorderBg is the background color for all border characters.
	BorderBg style.Color
	// ColumnOffsets are the X positions of each column boundary (len = columns + 1).
	ColumnOffsets []int
	// RowOffsets are the Y positions of each row boundary (len = rows + 1).
	RowOffsets []int
	// DrawInner draws the internal grid lines between cells.
	DrawInner bool
	// DrawOuter draws the outer border around the whole grid.
	DrawOuter bool
}

// DrawGrid draws a table grid with borders, intersections, and optional
// inner/outer lines. Column and row offsets define where each border line
// sits in buffer coordinates.
//
// Go equivalent of bufferDrawGrid.
func (b *Buffer) DrawGrid(opts GridDrawOptions) {
	columnCount := len(opts.ColumnOffsets) - 1
	rowCount := len(opts.RowOffsets) - 1
	if columnCount <= 0 || rowCount <= 0 {
		return
	}
	if !opts.DrawInner && !opts.DrawOuter {
		return
	}

	hChar := opts.Chars.Horizontal
	vChar := opts.Chars.Vertical

	for rowIdx := 0; rowIdx <= rowCount; rowIdx++ {
		isOuterRow := rowIdx == 0 || rowIdx == rowCount
		shouldDrawHorizontal := (isOuterRow && opts.DrawOuter) || (!isOuterRow && opts.DrawInner)
		borderY := opts.RowOffsets[rowIdx]
		if borderY >= b.height {
			break
		}

		// --- Horizontal border line: intersections + horizontal fills ---
		if shouldDrawHorizontal && borderY >= 0 {
			// Draw intersections
			for colBorderIdx := 0; colBorderIdx <= columnCount; colBorderIdx++ {
				isOuterCol := colBorderIdx == 0 || colBorderIdx == columnCount
				shouldDrawVertical := (isOuterCol && opts.DrawOuter) || (!isOuterCol && opts.DrawInner)
				if !shouldDrawVertical {
					continue
				}

				bx := opts.ColumnOffsets[colBorderIdx]
				if bx >= b.width {
					break
				}
				if bx < 0 {
					continue
				}

				hasUp := rowIdx > 0 && shouldDrawVertical
				hasDown := rowIdx < rowCount && shouldDrawVertical
				hasLeft := colBorderIdx > 0
				hasRight := colBorderIdx < columnCount

				ch := gridIntersection(opts.Chars, hasUp, hasDown, hasLeft, hasRight)
				b.SetCell(bx, borderY, ch, opts.BorderFg, opts.BorderBg, 0)
			}

			// Draw horizontal fills between intersections
			for colIdx := range columnCount {
				hasBoundaryAfter := (colIdx < columnCount-1 && opts.DrawInner) || (colIdx == columnCount-1 && opts.DrawOuter)
				boundaryPad := 0
				if !hasBoundaryAfter {
					boundaryPad = 1
				}

				startX := opts.ColumnOffsets[colIdx] + 1
				endX := opts.ColumnOffsets[colIdx+1] + boundaryPad

				if startX >= b.width {
					break
				}
				if endX <= 0 {
					continue
				}

				cs := max(0, startX)
				ce := min(b.width, endX)
				for cx := cs; cx < ce; cx++ {
					b.SetCell(cx, borderY, hChar, opts.BorderFg, opts.BorderBg, 0)
				}
			}
		}

		if rowIdx >= rowCount {
			break
		}

		// --- Vertical borders for each content line in this row ---
		hasRowBoundaryAfter := (rowIdx < rowCount-1 && opts.DrawInner) || (rowIdx == rowCount-1 && opts.DrawOuter)
		rowBoundaryPad := 0
		if !hasRowBoundaryAfter {
			rowBoundaryPad = 1
		}

		contentStartY := borderY + 1
		contentEndY := opts.RowOffsets[rowIdx+1] + rowBoundaryPad

		for cy := contentStartY; cy < contentEndY && cy < b.height; cy++ {
			if cy < 0 {
				continue
			}

			for colBorderIdx := 0; colBorderIdx <= columnCount; colBorderIdx++ {
				isOuterCol := colBorderIdx == 0 || colBorderIdx == columnCount
				shouldDrawVertical := (isOuterCol && opts.DrawOuter) || (!isOuterCol && opts.DrawInner)
				if !shouldDrawVertical {
					continue
				}

				bx := opts.ColumnOffsets[colBorderIdx]
				if bx >= b.width {
					break
				}
				if bx < 0 {
					continue
				}

				b.SetCell(bx, cy, vChar, opts.BorderFg, opts.BorderBg, 0)
			}
		}
	}
}

// gridIntersection returns the appropriate border character for a junction
// given which directions have connecting lines.
func gridIntersection(chars style.BorderCharacters, hasUp, hasDown, hasLeft, hasRight bool) rune {
	if hasUp && hasDown && hasLeft && hasRight {
		return chars.Cross
	}

	// Corners
	if !hasUp && hasDown && !hasLeft && hasRight {
		return chars.TopLeft
	}
	if !hasUp && hasDown && hasLeft && !hasRight {
		return chars.TopRight
	}
	if hasUp && !hasDown && !hasLeft && hasRight {
		return chars.BottomLeft
	}
	if hasUp && !hasDown && hasLeft && !hasRight {
		return chars.BottomRight
	}

	// T-junctions
	if hasUp && hasDown && !hasLeft && hasRight {
		return chars.LeftT
	}
	if hasUp && hasDown && hasLeft && !hasRight {
		return chars.RightT
	}
	if !hasUp && hasDown && hasLeft && hasRight {
		return chars.TopT
	}
	if hasUp && !hasDown && hasLeft && hasRight {
		return chars.BottomT
	}

	// Lines
	if (hasLeft || hasRight) && !hasUp && !hasDown {
		return chars.Horizontal
	}
	if (hasUp || hasDown) && !hasLeft && !hasRight {
		return chars.Vertical
	}

	return chars.Cross
}
