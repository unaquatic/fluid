package buffer

import (
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/textbuf"
)

// DrawTextSurface draws view at (x, y), respecting its viewport, selection,
// highlights, and styles.
func (b *Buffer) DrawTextSurface(view textbuf.Surface, x, y int) {
	b.drawSurfaceInternal(view, x, y)
}

// DrawEditSurface draws an edit surface at (x, y). It accepts the Surface
// interface so text and edit surfaces follow the same rendering path.
func (b *Buffer) DrawEditSurface(view textbuf.Surface, x, y int) {
	b.drawSurfaceInternal(view, x, y)
}

// drawSurfaceInternal renders viewport-adjusted virtual lines with line
// backgrounds, highlights, selection, and optional truncation. A transparent
// chunk inherits the destination cell's background.
func (b *Buffer) drawSurfaceInternal(view textbuf.Surface, x, y int) {
	virtualLines := view.VirtualLines()
	if len(virtualLines) == 0 {
		return
	}

	// Exclude virtual lines above and below the destination buffer.
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

	// The viewport controls horizontal scrolling and the maximum line width.
	vp := view.Viewport()
	horizontalOffset := 0
	viewportWidth := b.width // Without a viewport, draw to the buffer edge.
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

	// VirtualLine currently supplies the offsets used below, so this metadata is unused.
	lineInfo := view.CachedLineInfo()

	currentY := y + firstVisibleLine

	for sliceIdx := firstVisibleLine; sliceIdx < lastPossibleLine; sliceIdx++ {
		if currentY >= b.height {
			break
		}

		vline := virtualLines[sliceIdx]

		// Paint the line background before its chunks so transparent chunks inherit it.
		lineBg := view.LineBackground(vline.SourceLine)
		if !lineBg.IsTransparent() {
			b.FillRect(x, currentY, viewportWidth, 1, lineBg)
		}

		currentX := x
		globalCharPos := vline.CharOffset
		columnInLine := 0

		// A truncated line reserves its final viewport column for an ellipsis.
		needsTruncate := truncate && vline.Width > viewportWidth

		// Wrapped virtual lines share highlights with their source line.
		highlights := view.LineHighlights(vline.SourceLine)

		for _, vchunk := range vline.Chunks {
			if currentX >= b.width {
				break
			}

			chunkFg := vchunk.Fg
			chunkBg := vchunk.Bg
			chunkAttrs := vchunk.Attributes

			// A zero foreground selects the surface default.
			if chunkFg == (style.Color{}) {
				chunkFg = defaultFg
			}

			// A zero chunk background uses the surface default when available;
			// otherwise, the per-cell logic below inherits the destination.
			chunkBgTransparent := chunkBg == (style.Color{})
			if !chunkBgTransparent {
				// Preserve the chunk's explicit background.
			} else if defaultBg != (style.Color{}) {
				chunkBg = defaultBg
			}

			chunkText := vchunk.Text
			for len(chunkText) > 0 {
				cluster, rest, rw := textbuf.NextGraphemeCluster(chunkText)
				chunkText = rest

				// Discard complete clusters before the horizontal viewport origin.
				if columnInLine < horizontalOffset {
					globalCharPos += rw
					columnInLine += rw
					continue
				}

				// Exclude the column reserved for a truncation ellipsis.
				effectiveWidth := viewportWidth
				if needsTruncate {
					effectiveWidth = viewportWidth - 1
				}

				// Stop before a cluster that would cross the viewport boundary.
				if columnInLine+rw > horizontalOffset+effectiveWidth {
					// Replace the last visible column with the ellipsis.
					if needsTruncate && currentX < x+viewportWidth {
						b.SetCell(x+viewportWidth-1, currentY, '…', chunkFg, chunkBg, chunkAttrs)
					}
					// No later chunk on this line can be visible.
					chunkText = ""
					break
				}

				// Stop before a cluster that would cross the buffer boundary.
				if currentX+rw > b.width {
					break
				}

				// Clipped clusters still consume source and destination columns.
				if currentY < 0 || b.isClipped(currentX, currentY) {
					globalCharPos += rw
					currentX += rw
					columnInLine += rw
					continue
				}

				finalFg := chunkFg
				finalBg := chunkBg
				finalAttrs := chunkAttrs

				// Inherit a background already painted by the line or widget.
				if chunkBgTransparent && defaultBg == (style.Color{}) {
					existing := b.Cell(currentX, currentY)
					if existing.Bg != (style.Color{}) {
						finalBg = existing.Bg
					}
				}

				// Highlights use coordinates from the unwrapped source line.
				sourceCol := vline.SourceColOffset + columnInLine
				for _, hl := range highlights {
					if sourceCol >= hl.Start && sourceCol < hl.End {
						if !hl.Fg.IsTransparent() {
							finalFg = hl.Fg
						}
						if !hl.Bg.IsTransparent() {
							finalBg = hl.Bg
						}
						finalAttrs |= hl.Attributes
						break
					}
				}

				// Selection colors take precedence over line highlights.
				if sel != nil {
					for c := range rw {
						if globalCharPos+c >= sel.Start && globalCharPos+c < sel.End {
							if !sel.BgColor.IsTransparent() {
								finalBg = sel.BgColor
							} else {
								finalFg, finalBg = defaultBg, chunkFg
								if finalFg.A == 0 {
									finalFg = style.ColorFromInts(0, 0, 0, 255)
								}
							}
							if !sel.FgColor.IsTransparent() {
								finalFg = sel.FgColor
							}
							break
						}
					}
				}

				// The reverse-video attribute swaps the final colors.
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

	_ = lineInfo
}
