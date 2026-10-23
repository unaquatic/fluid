package buffer

import (
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/textbuf"
)

// DrawTextSurface draws a TextSurface (or any Surface) into this
// buffer at (x, y). It renders each visible virtual line, respecting the
// view's viewport, selection, and styling.
func (b *Buffer) DrawTextSurface(view textbuf.Surface, x, y int) {
	b.drawSurfaceInternal(view, x, y)
}

// DrawEditSurface draws an EditSurface into this buffer at (x, y).
// EditSurface implements Surface so this delegates to the same code.
func (b *Buffer) DrawEditSurface(view textbuf.Surface, x, y int) {
	b.drawSurfaceInternal(view, x, y)
}

// drawSurfaceInternal is the shared drawing implementation for both
// TextSurface and EditSurface.
//
// It handles three concerns that widgets previously implemented manually:
//
//  1. Per-line background fill: calls LineBackground(sourceLine) and paints
//     a FillRect before the text when the color is non-transparent. This covers
//     active-line highlighting and diff hunk coloring.
//
//  2. Line truncation with ellipsis: when Truncate() is true and a line is
//     wider than the viewport, the last column is reserved for '…'.
//
//  3. Transparent background inheritance: when a chunk's background is
//     transparent (zero value), the renderer inherits the background already
//     present in the buffer cell (painted by FillRect or the widget's own
//     background fill). This lets active-line and diff colors show through
//     without every span needing to know about them.
func (b *Buffer) drawSurfaceInternal(view textbuf.Surface, x, y int) {
	virtualLines := view.VirtualLines()
	if len(virtualLines) == 0 {
		return
	}

	// Determine which virtual lines are visible within the buffer.
	firstVisibleLine := 0
	if y < 0 {
		firstVisibleLine = -y
	}
	if firstVisibleLine >= len(virtualLines) {
		return
	}

	lastPossibleLine := len(virtualLines)
	if y >= 0 {
		remaining := b.height - y
		if remaining < lastPossibleLine {
			lastPossibleLine = remaining
		}
	} else {
		remaining := b.height
		if firstVisibleLine+remaining < lastPossibleLine {
			lastPossibleLine = firstVisibleLine + remaining
		}
	}
	if lastPossibleLine <= firstVisibleLine {
		return
	}

	// Get viewport for horizontal offset and width limit.
	vp := view.Viewport()
	horizontalOffset := 0
	viewportWidth := b.width // default: render to buffer edge
	if vp != nil {
		if vp.X > 0 {
			horizontalOffset = vp.X
		}
		if vp.Width > 0 {
			viewportWidth = vp.Width
		}
	}

	defaultFg := view.DefaultFg()
	defaultBg := view.DefaultBg()
	sel := view.Selection()
	truncate := view.Truncate()

	// Get line info for char offset tracking.
	lineInfo := view.CachedLineInfo()

	currentY := y + firstVisibleLine

	for sliceIdx := firstVisibleLine; sliceIdx < lastPossibleLine; sliceIdx++ {
		if currentY >= b.height {
			break
		}

		vline := virtualLines[sliceIdx]

		// --- Per-line background fill ---
		lineBg := view.LineBackground(vline.SourceLine)
		if !lineBg.IsTransparent() {
			b.FillRect(x, currentY, viewportWidth, 1, lineBg)
		}

		currentX := x
		globalCharPos := vline.CharOffset
		columnInLine := 0

		// Determine if this line needs truncation (wider than viewport).
		needsTruncate := truncate && vline.Width > viewportWidth

		// Get highlights for this virtual line's source (physical) line.
		highlights := view.LineHighlights(vline.SourceLine)

		for _, vchunk := range vline.Chunks {
			if currentX >= b.width {
				break
			}

			chunkFg := vchunk.Fg
			chunkBg := vchunk.Bg
			chunkAttrs := vchunk.Attributes

			// Use defaults if colors are zero
			if chunkFg == (style.Color{}) {
				chunkFg = defaultFg
			}

			// Transparent background inheritance: if the chunk has no
			// background, leave chunkBg as zero so the per-cell logic
			// below can inherit from the buffer (FillRect / widget bg).
			chunkBgTransparent := chunkBg == (style.Color{})
			if !chunkBgTransparent {
				// Chunk has an explicit bg — use it.
			} else if defaultBg != (style.Color{}) {
				chunkBg = defaultBg
			}

			chunkText := vchunk.Text
			for len(chunkText) > 0 {
				cluster, rest, rw := textbuf.NextGraphemeCluster(chunkText)
				chunkText = rest

				// Skip horizontal offset
				if columnInLine < horizontalOffset {
					globalCharPos += rw
					columnInLine += rw
					continue
				}

				// Truncation: reserve last column for ellipsis
				effectiveWidth := viewportWidth
				if needsTruncate {
					effectiveWidth = viewportWidth - 1
				}

				// Past viewport width limit (or wide char would overflow)
				if columnInLine+rw > horizontalOffset+effectiveWidth {
					// Write ellipsis if truncating
					if needsTruncate && currentX < x+viewportWidth {
						b.SetCell(x+viewportWidth-1, currentY, '…', chunkFg, chunkBg, chunkAttrs)
					}
					// Skip rest of this line
					chunkText = ""
					break
				}

				// Past the right edge of the buffer
				if currentX+rw > b.width {
					break
				}

				// Check clipping
				if currentY < 0 || b.isClipped(currentX, currentY) {
					globalCharPos += rw
					currentX += rw
					columnInLine += rw
					continue
				}

				finalFg := chunkFg
				finalBg := chunkBg
				finalAttrs := chunkAttrs

				// Transparent bg inheritance: read existing cell bg
				if chunkBgTransparent && defaultBg == (style.Color{}) {
					existing := b.Cell(currentX, currentY)
					if existing.Bg != (style.Color{}) {
						finalBg = existing.Bg
					}
				}

				// Apply highlight colors using source-line coordinates.
				sourceCol := vline.SourceColOffset + columnInLine
				for _, hl := range highlights {
					if sourceCol >= hl.Start && sourceCol < hl.End {
						if hl.Fg != nil {
							finalFg = *hl.Fg
						}
						if hl.Bg != nil {
							finalBg = *hl.Bg
						}
						finalAttrs |= hl.Attributes
						break
					}
				}

				// Apply selection highlighting (overrides highlights)
				if sel != nil {
					for c := range rw {
						if globalCharPos+c >= sel.Start && globalCharPos+c < sel.End {
							if sel.BgColor != nil {
								finalBg = *sel.BgColor
							} else {
								finalFg, finalBg = defaultBg, chunkFg
								if finalFg.A == 0 {
									finalFg = style.ColorFromInts(0, 0, 0, 255)
								}
							}
							if sel.FgColor != nil {
								finalFg = *sel.FgColor
							}
							break
						}
					}
				}

				// Apply reverse attribute (bit 5)
				if finalAttrs&(1<<5) != 0 {
					finalFg, finalBg = finalBg, finalFg
				}

				b.SetGraphemeCluster(currentX, currentY, cluster, finalFg, finalBg, finalAttrs)

				globalCharPos += rw
				currentX += rw
				columnInLine += rw
			}
		}

		currentY++
	}

	// Suppress unused variable warning
	_ = lineInfo
}
