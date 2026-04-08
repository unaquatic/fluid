package layout

import "math"

// flexLine represents a single line of flex items (used when wrapping).
type flexLine struct {
	items     []*LayoutNode
	mainSize  int // total main-axis size consumed by items + gaps
	crossSize int // max cross-axis size among items on this line
}

// flexItemInfo holds intermediate layout data for a single flex item.
type flexItemInfo struct {
	basis    int
	minMain  int
	maxMain  int
	frozen   bool
	mainSize int
}

// layoutFlexContainer computes layout for a flex container and its children.
// parentX and parentY are the absolute position of the container's content box.
func (node *LayoutNode) layoutFlexContainer(availableWidth, availableHeight int, parentX, parentY int) {
	// Yoga-style cache check: if the node is clean and the available
	// dimensions match the last layout, skip the entire computation.
	// Only update absolute position if the parent moved.
	if !node.dirty && node.Cache.Valid &&
		node.Cache.AvailWidth == availableWidth &&
		node.Cache.AvailHeight == availableHeight {
		node.Computed.X = parentX
		node.Computed.Y = parentY
		if dx, dy := parentX-node.Cache.ParentX, parentY-node.Cache.ParentY; dx != 0 || dy != 0 {
			node.ShiftDescendantPositions(dx, dy)
			node.Cache.ParentX = parentX
			node.Cache.ParentY = parentY
		}
		return
	}

	// Clear dirty at start so concurrent MarkDirty during layout is
	// not overwritten — it will re-dirty the node and the ComputeLayout
	// loop will re-run.
	node.dirty = false

	dir := node.FlexDirection

	// Resolve container dimensions.
	containerWidth := resolveNodeWidth(node, availableWidth)
	containerHeight := resolveNodeHeight(node, availableHeight)

	// Apply padding
	padTop, padRight, padBottom, padLeft := resolvePaddings(node, containerWidth, containerHeight)
	contentWidth := containerWidth - padLeft - padRight
	contentHeight := containerHeight - padTop - padBottom

	if contentWidth < 0 {
		contentWidth = 0
	}
	if contentHeight < 0 {
		contentHeight = 0
	}

	// Set container computed layout
	node.Computed.X = parentX
	node.Computed.Y = parentY
	node.Computed.Width = containerWidth
	node.Computed.Height = containerHeight

	// Content box origin
	contentX := parentX + padLeft
	contentY := parentY + padTop

	// Separate children into flex items and absolute items
	var flexItems []*LayoutNode
	var absoluteItems []*LayoutNode
	for _, child := range node.Children {
		if child.Visibility == Gone {
			child.Computed = ComputedLayout{}
			continue
		}
		if child.Position == PositionAbsolute {
			absoluteItems = append(absoluteItems, child)
		} else {
			flexItems = append(flexItems, child)
		}
	}

	// Reverse item order for reverse directions
	if dir.IsReverse() {
		reverseNodes(flexItems)
	}

	// Determine axis helpers
	mainAvailable := mainDimension(contentWidth, contentHeight, dir)
	crossAvailable := crossDimension(contentWidth, contentHeight, dir)

	// Resolve gap
	mainGap := resolveGap(node, dir)
	crossGap := resolveCrossGap(node, dir)

	// Collect items into flex lines
	lines := collectFlexLines(flexItems, node, dir, mainAvailable, crossAvailable, mainGap)

	// Resolve flexible lengths and cross sizes for each line
	isSingleLine := node.FlexWrap == WrapNone || len(lines) <= 1
	for _, line := range lines {
		resolveFlexibleLengths(line, node, dir, mainAvailable, crossAvailable, mainGap)
		resolveLineCrossSize(line, node, dir, crossAvailable, isSingleLine)
	}

	// Position items
	positionMainAxis(lines, node, dir, mainAvailable, mainGap)
	positionCrossAxis(lines, node, dir, crossAvailable, crossGap)

	// Write absolute positions and recurse into children.
	// The cache check at the top of layoutFlexContainer handles subtree skip.
	for _, line := range lines {
		for _, item := range line.items {
			item.Computed.X = contentX + item.Computed.X
			item.Computed.Y = contentY + item.Computed.Y

			if len(item.Children) > 0 {
				item.layoutFlexContainer(item.Computed.Width, item.Computed.Height,
					item.Computed.X, item.Computed.Y)
			} else {
				// Leaf node: parent computed everything, clear dirty.
				item.dirty = false
			}
		}
	}

	// Handle absolute-positioned children
	layoutAbsoluteChildren(absoluteItems, node, contentX, contentY, contentWidth, contentHeight)

	// Save cache for this node
	node.Cache = LayoutCache{
		AvailWidth:  availableWidth,
		AvailHeight: availableHeight,
		ParentX:     parentX,
		ParentY:     parentY,
		Valid:       true,
	}
}

// collectFlexLines groups flex items into lines based on wrapping behavior.
func collectFlexLines(items []*LayoutNode, container *LayoutNode, dir FlexDirection, mainAvailable, crossAvailable, mainGap int) []*flexLine {
	if len(items) == 0 {
		return nil
	}

	if container.FlexWrap == WrapNone {
		// Single line: all items on one line
		return []*flexLine{{items: items}}
	}

	var lines []*flexLine
	var currentItems []*LayoutNode
	var lineMainSize int

	for i, item := range items {
		basis := resolveFlexBasis(item, container, dir, mainAvailable, crossAvailable)
		margins := resolveMainMargins(item, dir, mainAvailable)
		itemMainSize := basis + margins

		// Add gap if not the first item on this line
		gapSize := 0
		if len(currentItems) > 0 {
			gapSize = mainGap
		}

		if len(currentItems) > 0 && lineMainSize+gapSize+itemMainSize > mainAvailable {
			// Start a new line
			lines = append(lines, &flexLine{items: currentItems, mainSize: lineMainSize})
			currentItems = []*LayoutNode{item}
			lineMainSize = itemMainSize
		} else {
			currentItems = append(currentItems, item)
			lineMainSize += gapSize + itemMainSize
		}

		// Last item: close the line
		if i == len(items)-1 {
			lines = append(lines, &flexLine{items: currentItems, mainSize: lineMainSize})
		}
	}

	// Reverse lines for WrapReverse
	if container.FlexWrap == WrapReverse {
		for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
			lines[i], lines[j] = lines[j], lines[i]
		}
	}

	return lines
}

// resolveFlexibleLengths distributes free space among items using grow/shrink.
func resolveFlexibleLengths(line *flexLine, container *LayoutNode, dir FlexDirection, mainAvailable, crossAvailable, mainGap int) {
	items := line.items
	if len(items) == 0 {
		return
	}

	// Calculate total basis, margins, and gaps
	totalBasis := 0
	totalMargins := 0
	totalGaps := mainGap * (len(items) - 1)

	info := make([]flexItemInfo, len(items))

	for i, item := range items {
		basis := resolveFlexBasis(item, container, dir, mainAvailable, crossAvailable)
		margins := resolveMainMargins(item, dir, mainAvailable)
		minMain := resolveMainMin(item, dir, mainAvailable)
		maxMain := resolveMainMax(item, dir, mainAvailable)

		info[i] = flexItemInfo{
			basis:   basis,
			minMain: minMain,
			maxMain: maxMain,
		}
		totalBasis += basis
		totalMargins += margins
	}

	freeSpace := mainAvailable - totalBasis - totalMargins - totalGaps

	if freeSpace > 0 {
		// Grow phase
		distributeGrow(items, info, freeSpace)
	} else if freeSpace < 0 && container.Overflow != OverflowScroll {
		// Shrink phase — skipped for scroll containers where content
		// overflows the viewport and scrolling handles the excess.
		distributeShrink(items, info, -freeSpace)
	} else {
		// No free space; items stay at basis
		for i := range items {
			info[i].mainSize = info[i].basis
		}
	}

	// Apply resolved main sizes to computed layout.
	for i, item := range items {
		size := max(clampSize(info[i].mainSize, info[i].minMain, info[i].maxMain), 0)
		setMainComputedSize(item, dir, size)
	}
}

// distributeGrow distributes positive free space to items with FlexGrow > 0.
// Uses float intermediates for proportional distribution, then rounds to int
// with remainder assigned to the last growing item.
func distributeGrow(items []*LayoutNode, info []flexItemInfo, freeSpace int) {
	totalGrow := float32(0)
	for _, item := range items {
		totalGrow += float32(item.FlexGrow)
	}

	if totalGrow == 0 {
		// No grow: items stay at basis
		for i := range items {
			info[i].mainSize = info[i].basis
		}
		return
	}

	// First pass: distribute proportionally, tracking remainder
	distributed := 0
	lastGrowIdx := -1
	for i, item := range items {
		if item.FlexGrow > 0 {
			share := int(float32(freeSpace) * (float32(item.FlexGrow) / totalGrow))
			proposed := info[i].basis + share
			clamped := clampSize(proposed, info[i].minMain, info[i].maxMain)
			if clamped != proposed {
				info[i].mainSize = clamped
				info[i].frozen = true
				distributed += clamped - info[i].basis
			} else {
				info[i].mainSize = proposed
				distributed += share
			}
			lastGrowIdx = i
		} else {
			info[i].mainSize = info[i].basis
		}
	}

	// Assign remainder to last growing item
	remainder := freeSpace - distributed
	if remainder > 0 && lastGrowIdx >= 0 && !info[lastGrowIdx].frozen {
		info[lastGrowIdx].mainSize += remainder
	}

	// Second pass: re-distribute if items were clamped
	if remainder > 0 && (lastGrowIdx < 0 || info[lastGrowIdx].frozen) {
		unfrozenGrow := float32(0)
		for i, item := range items {
			if !info[i].frozen && item.FlexGrow > 0 {
				unfrozenGrow += float32(item.FlexGrow)
			}
		}
		if unfrozenGrow > 0 {
			extra := remainder
			extraDistributed := 0
			lastUnfrozen := -1
			for i, item := range items {
				if !info[i].frozen && item.FlexGrow > 0 {
					share := int(float32(extra) * (float32(item.FlexGrow) / unfrozenGrow))
					info[i].mainSize += share
					extraDistributed += share
					lastUnfrozen = i
				}
			}
			// Give leftover to last unfrozen
			if leftover := extra - extraDistributed; leftover > 0 && lastUnfrozen >= 0 {
				info[lastUnfrozen].mainSize += leftover
			}
		}
	}
}

// distributeShrink distributes negative free space to items with FlexShrink > 0.
func distributeShrink(items []*LayoutNode, info []flexItemInfo, overflow int) {
	// Weighted shrink factor: FlexShrink * FlexBasis
	totalWeightedShrink := float32(0)
	for i, item := range items {
		totalWeightedShrink += float32(item.FlexShrink) * float32(info[i].basis)
	}

	if totalWeightedShrink == 0 {
		// No shrink: items stay at basis (will overflow)
		for i := range items {
			info[i].mainSize = info[i].basis
		}
		return
	}

	shrunk := 0
	lastShrinkIdx := -1
	for i, item := range items {
		weight := float32(item.FlexShrink) * float32(info[i].basis)
		shrinkAmount := int(float32(overflow) * (weight / totalWeightedShrink))
		info[i].mainSize = info[i].basis - shrinkAmount
		shrunk += shrinkAmount
		if item.FlexShrink > 0 {
			lastShrinkIdx = i
		}
	}

	// Assign remainder to last shrinking item
	if leftover := overflow - shrunk; leftover > 0 && lastShrinkIdx >= 0 {
		info[lastShrinkIdx].mainSize -= leftover
	}
}

// resolveLineCrossSize determines the cross-axis size for each item in a line
// and the overall line cross size.
func resolveLineCrossSize(line *flexLine, container *LayoutNode, dir FlexDirection, crossAvailable int, isSingleLine bool) {
	maxCross := 0

	for _, item := range line.items {
		cross := resolveItemCrossSize(item, container, dir, crossAvailable)
		if cross > maxCross {
			maxCross = cross
		}
	}

	// For single-line containers, the line cross size equals the container's
	// inner cross size (CSS flexbox spec).
	if isSingleLine && crossAvailable > maxCross {
		maxCross = crossAvailable
	}

	line.crossSize = maxCross

	// Set computed cross-size for all items
	for _, item := range line.items {
		align := resolveAlign(item, container)
		explicitCross, hasExplicit := resolveItemExplicitCrossSize(item, dir, crossAvailable)
		margins := resolveCrossMargins(item, dir, crossAvailable)

		if hasExplicit {
			// Item has an explicit cross-axis size
			setCrossComputedSize(item, dir, explicitCross)
		} else if align == AlignStretch {
			// No explicit cross size + stretch: fill to line cross size
			stretchedCross := max(maxCross-margins, 0)
			setCrossComputedSize(item, dir, stretchedCross)
		} else if item.Measure != nil {
			// No explicit cross size + non-stretch alignment: use MeasureFunc
			mainSize := getMainComputedSize(item, dir)
			var result MeasureResult
			if dir.IsRow() {
				result = item.Measure(mainSize, MeasureExactly, crossAvailable, MeasureAtMost)
			} else {
				result = item.Measure(crossAvailable, MeasureAtMost, mainSize, MeasureExactly)
			}
			setCrossComputedSize(item, dir, crossFromResult(result, dir))
		}
		// AlignStart/Center/End with no explicit size and no MeasureFunc: cross-size stays 0
	}
}

// positionMainAxis positions items along the main axis using JustifyContent.
func positionMainAxis(lines []*flexLine, container *LayoutNode, dir FlexDirection, mainAvailable, mainGap int) {
	for _, line := range lines {
		items := line.items
		if len(items) == 0 {
			continue
		}

		// Calculate total used space
		totalUsed := 0
		for _, item := range items {
			totalUsed += getMainComputedSize(item, dir) + resolveMainMargins(item, dir, mainAvailable)
		}
		totalGaps := mainGap * (len(items) - 1)
		freeSpace := max(mainAvailable-totalUsed-totalGaps, 0)

		var offset int
		var spaceBetween int

		switch container.JustifyContent {
		case JustifyStart:
			offset = 0
			spaceBetween = mainGap
		case JustifyEnd:
			offset = freeSpace
			spaceBetween = mainGap
		case JustifyCenter:
			offset = freeSpace / 2
			spaceBetween = mainGap
		case JustifySpaceBetween:
			offset = 0
			if len(items) > 1 {
				spaceBetween = (freeSpace + totalGaps) / (len(items) - 1)
			} else {
				spaceBetween = 0
			}
		case JustifySpaceAround:
			itemSpace := (freeSpace + totalGaps) / len(items)
			offset = itemSpace / 2
			spaceBetween = itemSpace
		case JustifySpaceEvenly:
			itemSpace := (freeSpace + totalGaps) / (len(items) + 1)
			offset = itemSpace
			spaceBetween = itemSpace
		}

		// Position each item.
		pos := offset
		for _, item := range items {
			marginBefore := resolveMainMarginBefore(item, dir, mainAvailable)
			marginAfter := resolveMainMarginAfter(item, dir, mainAvailable)
			pos += marginBefore
			setMainComputedPos(item, dir, pos)
			pos += getMainComputedSize(item, dir) + marginAfter + spaceBetween
		}
	}
}

// positionCrossAxis positions items and lines along the cross axis.
func positionCrossAxis(lines []*flexLine, container *LayoutNode, dir FlexDirection, crossAvailable, crossGap int) {
	crossOffset := 0

	for _, line := range lines {
		for _, item := range line.items {
			align := resolveAlign(item, container)
			itemCross := getCrossComputedSize(item, dir)
			margins := resolveCrossMargins(item, dir, crossAvailable)
			marginBefore := resolveCrossMarginBefore(item, dir, crossAvailable)

			var itemOffset int
			switch align {
			case AlignStart:
				itemOffset = marginBefore
			case AlignEnd:
				itemOffset = line.crossSize - itemCross - (margins - marginBefore)
			case AlignCenter:
				itemOffset = (line.crossSize-itemCross-margins)/2 + marginBefore
			case AlignStretch:
				itemOffset = marginBefore
			default:
				itemOffset = marginBefore
			}

			setCrossComputedPos(item, dir, crossOffset+itemOffset)
			setCrossComputedSize(item, dir, itemCross)
		}
		crossOffset += line.crossSize + crossGap
	}
}

// layoutAbsoluteChildren positions absolute children relative to the container.
func layoutAbsoluteChildren(items []*LayoutNode, container *LayoutNode, contentX, contentY, contentWidth, contentHeight int) {
	for _, item := range items {
		// Resolve dimensions
		itemWidth := resolveNodeWidth(item, contentWidth)
		itemHeight := resolveNodeHeight(item, contentHeight)

		// If both left and right are set, derive width
		leftSet := !item.Left.IsUndefined()
		rightSet := !item.Right.IsUndefined()
		topSet := !item.Top.IsUndefined()
		bottomSet := !item.Bottom.IsUndefined()

		if leftSet && rightSet && item.Width.IsAuto() {
			l := safeResolve(item.Left, contentWidth)
			r := safeResolve(item.Right, contentWidth)
			itemWidth = max(contentWidth-l-r, 0)
		}
		if topSet && bottomSet && item.Height.IsAuto() {
			t := safeResolve(item.Top, contentHeight)
			b := safeResolve(item.Bottom, contentHeight)
			itemHeight = max(contentHeight-t-b, 0)
		}

		// Position horizontally
		var x int
		if leftSet {
			x = safeResolve(item.Left, contentWidth)
		} else if rightSet {
			x = contentWidth - safeResolve(item.Right, contentWidth) - itemWidth
		}

		// Position vertically
		var y int
		if topSet {
			y = safeResolve(item.Top, contentHeight)
		} else if bottomSet {
			y = contentHeight - safeResolve(item.Bottom, contentHeight) - itemHeight
		}

		item.Computed.X = contentX + x
		item.Computed.Y = contentY + y
		item.Computed.Width = itemWidth
		item.Computed.Height = itemHeight

		if len(item.Children) > 0 {
			item.layoutFlexContainer(itemWidth, itemHeight, item.Computed.X, item.Computed.Y)
		} else {
			item.dirty = false
		}
	}
}

// --- Axis Helpers ---

// mainDimension returns the main-axis size from width/height based on direction.
func mainDimension(width, height int, dir FlexDirection) int {
	if dir.IsRow() {
		return width
	}
	return height
}

// crossDimension returns the cross-axis size from width/height based on direction.
func crossDimension(width, height int, dir FlexDirection) int {
	if dir.IsRow() {
		return height
	}
	return width
}

// resolveGap returns the main-axis gap.
func resolveGap(node *LayoutNode, dir FlexDirection) int {
	if dir.IsRow() {
		if node.ColumnGap > 0 {
			return node.ColumnGap
		}
		return node.Gap
	}
	if node.RowGap > 0 {
		return node.RowGap
	}
	return node.Gap
}

// resolveCrossGap returns the cross-axis gap.
func resolveCrossGap(node *LayoutNode, dir FlexDirection) int {
	if dir.IsRow() {
		if node.RowGap > 0 {
			return node.RowGap
		}
		return node.Gap
	}
	if node.ColumnGap > 0 {
		return node.ColumnGap
	}
	return node.Gap
}

// resolveFlexBasis returns the resolved flex basis for an item.
func resolveFlexBasis(item *LayoutNode, container *LayoutNode, dir FlexDirection, mainAvailable, crossAvailable int) int {
	if !item.FlexBasis.IsAuto() && !item.FlexBasis.IsUndefined() {
		if resolved, ok := item.FlexBasis.Resolve(mainAvailable); ok {
			return resolved
		}
	}

	// Fall back to main-axis size
	var sizeScalar Scalar
	if dir.IsRow() {
		sizeScalar = item.Width
	} else {
		sizeScalar = item.Height
	}

	if !sizeScalar.IsAuto() && !sizeScalar.IsUndefined() {
		if resolved, ok := sizeScalar.Resolve(mainAvailable); ok {
			return resolved
		}
	}

	// For nodes with a MeasureFunc, use intrinsic measurement.
	// Main axis is Auto (otherwise we'd have returned above), so use AtMost.
	// Cross axis mode depends on whether the item has an explicit cross size.
	//
	// Exception: if the container has overflow:scroll or overflow:hidden, the
	// main-axis constraint is Undefined so children can report their full
	// natural size (the container handles clipping/scrolling).
	if item.Measure != nil {
		wMode, hMode := measureModes(item, dir, mainAvailable, crossAvailable)
		if container.Overflow == OverflowScroll || container.Overflow == OverflowHidden {
			if dir.IsRow() {
				wMode = MeasureUndefined
			} else {
				hMode = MeasureUndefined
			}
		}
		// Swap main/cross into width/height based on direction so the
		// MeasureFunc receives the correct axis values.
		var result MeasureResult
		if dir.IsRow() {
			result = item.Measure(mainAvailable, wMode, crossAvailable, hMode)
		} else {
			result = item.Measure(crossAvailable, wMode, mainAvailable, hMode)
		}
		if dir.IsRow() {
			return result.Width
		}
		return result.Height
	}

	// For container nodes (those with children) that have Auto main-axis size,
	// compute the intrinsic content size by summing children's flex bases.
	//
	// Cache: the intrinsic main-axis size depends on the cross-axis dimension
	// (word-wrapping text produces a different height for a different width).
	// Compare the child's effective cross-axis size against the layout cache.
	// When only the main axis changes (e.g. slash menu open/close), the
	// cross dimension matches and the cached size is reused. When the cross
	// axis changes (terminal resize), the cache misses and the subtree is
	// re-measured.
	if len(item.Children) > 0 {
		if !item.dirty && item.Cache.Valid {
			childCross := resolveChildCrossSize(item, dir, crossAvailable)
			var cachedCross int
			if dir.IsRow() {
				cachedCross = item.Cache.AvailHeight
			} else {
				cachedCross = item.Cache.AvailWidth
			}
			if cachedCross == childCross {
				if dir.IsRow() {
					return item.Computed.Width
				}
				return item.Computed.Height
			}
		}
		return intrinsicMainSize(item, dir, mainAvailable, crossAvailable)
	}

	return 0
}

// intrinsicMainSize computes the intrinsic size of a container node along the
// parent's main axis. This is used when a container has Auto sizing on the
// main axis so it shrink-wraps its content instead of collapsing to zero.
//
// crossAvailable is the cross-axis space available from the parent. It is
// propagated to MeasureFunc nodes so that width-dependent heights (e.g.
// word-wrapping textareas) are measured against the actual available width
// rather than an unconstrained sentinel.
//
// The logic depends on how the node arranges its own children:
//   - Same axis as parent (e.g. both column): sum children heights + gaps
//   - Perpendicular (e.g. parent=column, node=row): max child height
func intrinsicMainSize(node *LayoutNode, parentDir FlexDirection, mainAvailable, crossAvailable int) int {
	childDir := node.FlexDirection
	sameAxis := childDir.IsRow() == parentDir.IsRow()

	// Padding along the parent's main axis.
	padTop, _, padBottom, padLeft := resolvePaddings(node, mainAvailable, mainAvailable)
	padRight := safeResolve(node.PaddingRight, mainAvailable)
	var mainPad int
	if parentDir.IsRow() {
		mainPad = padLeft + padRight
	} else {
		mainPad = padTop + padBottom
	}

	// Cross-axis padding and margins of the node — subtract from
	// crossAvailable so children get the content-box dimension.
	// Margins reduce the space the node occupies within its parent,
	// and padding further reduces the content area for children.
	var crossPad int
	if parentDir.IsRow() {
		crossPad = padTop + padBottom
	} else {
		crossPad = padLeft + padRight
	}
	crossMargins := resolveCrossMargins(node, parentDir, crossAvailable)
	childCross := max(crossAvailable-crossPad-crossMargins, 0)

	// Use a large unconstrained value for the MAIN axis so that
	// MeasureFunc nodes report their full natural size instead of being
	// clamped to the parent's viewport.
	unconstrained := math.MaxInt16

	// Sum or max children's sizes depending on axis alignment.
	total := 0
	childCount := 0

	if !sameAxis {
		// Perpendicular case (e.g. Row inside Column parent).
		// We need each child's size along parentDir (height), which may
		// depend on its resolved size along the node's own axis (width).
		// For example, a wrapping textarea needs its actual width to
		// compute how many lines it produces.
		//
		// Step 1: resolve children's main-axis sizes (widths for a Row)
		// using a lightweight flex pass along the node's own direction.
		childMainSizes := preResolveMainSizes(node, childDir, childCross, unconstrained)

		idx := 0
		for _, child := range node.Children {
			if child.Position == PositionAbsolute {
				continue
			}

			// Use the child's resolved main-axis size as its cross
			// constraint when measuring along parentDir.
			childW := childCross
			if idx < len(childMainSizes) {
				childW = childMainSizes[idx]
			}
			idx++

			childSize := resolveFlexBasis(child, node, parentDir, unconstrained, childW)
			childSize += resolveMainMargins(child, parentDir, unconstrained)

			if childSize > total {
				total = childSize
			}
			childCount++
		}
	} else {
		// Same-axis case (e.g. both Column): children stack along the
		// same axis — resolveFlexBasis with childDir gives the correct
		// dimension (height for Column).
		for _, child := range node.Children {
			if child.Position == PositionAbsolute {
				continue
			}

			childSize := resolveFlexBasis(child, node, childDir, unconstrained, childCross)
			childSize += resolveMainMargins(child, childDir, unconstrained)
			total += childSize
			childCount++
		}

		// Gaps only apply along the stacking direction.
		if childCount > 1 {
			total += resolveGap(node, childDir) * (childCount - 1)
		}
	}

	return total + mainPad
}

// preResolveMainSizes runs a lightweight flex resolution along the given
// direction to determine each child's main-axis size. This is used by
// intrinsicMainSize when a perpendicular container (e.g. Row inside Column)
// needs to know children's widths before measuring their heights.
// It does NOT write to Computed fields.
func preResolveMainSizes(node *LayoutNode, dir FlexDirection, mainAvailable, crossAvailable int) []int {
	var items []*LayoutNode
	for _, child := range node.Children {
		if child.Position != PositionAbsolute {
			items = append(items, child)
		}
	}
	if len(items) == 0 {
		return nil
	}

	mainGap := resolveGap(node, dir)
	totalGaps := mainGap * (len(items) - 1)

	info := make([]flexItemInfo, len(items))
	totalBasis := 0
	totalMargins := 0

	for i, item := range items {
		basis := resolveFlexBasis(item, node, dir, mainAvailable, crossAvailable)
		margins := resolveMainMargins(item, dir, mainAvailable)
		minMain := resolveMainMin(item, dir, mainAvailable)
		maxMain := resolveMainMax(item, dir, mainAvailable)

		info[i] = flexItemInfo{
			basis:   basis,
			minMain: minMain,
			maxMain: maxMain,
		}
		totalBasis += basis
		totalMargins += margins
	}

	freeSpace := mainAvailable - totalBasis - totalMargins - totalGaps
	if freeSpace > 0 {
		distributeGrow(items, info, freeSpace)
	} else if freeSpace < 0 {
		distributeShrink(items, info, -freeSpace)
	} else {
		for i := range items {
			info[i].mainSize = info[i].basis
		}
	}

	sizes := make([]int, len(items))
	for i := range items {
		size := max(clampSize(info[i].mainSize, info[i].minMain, info[i].maxMain), 0)
		sizes[i] = size
	}
	return sizes
}

// resolveMainMargins returns total main-axis margins.
func resolveMainMargins(item *LayoutNode, dir FlexDirection, mainAvailable int) int {
	return resolveMainMarginBefore(item, dir, mainAvailable) + resolveMainMarginAfter(item, dir, mainAvailable)
}

// resolveMainMarginBefore returns the margin before the item on the main axis.
func resolveMainMarginBefore(item *LayoutNode, dir FlexDirection, mainAvailable int) int {
	var s Scalar
	if dir.IsRow() {
		s = item.MarginLeft
	} else {
		s = item.MarginTop
	}
	return safeResolve(s, mainAvailable)
}

// resolveMainMarginAfter returns the margin after the item on the main axis.
func resolveMainMarginAfter(item *LayoutNode, dir FlexDirection, mainAvailable int) int {
	var s Scalar
	if dir.IsRow() {
		s = item.MarginRight
	} else {
		s = item.MarginBottom
	}
	return safeResolve(s, mainAvailable)
}

// resolveChildCrossSize computes the cross-axis size a child will receive
// from its parent's flex layout. Used as the cache key for the intrinsic
// size shortcut in resolveFlexBasis.
func resolveChildCrossSize(item *LayoutNode, parentDir FlexDirection, parentCross int) int {
	if explicit, ok := resolveItemExplicitCrossSize(item, parentDir, parentCross); ok {
		return explicit
	}
	return parentCross - resolveCrossMargins(item, parentDir, parentCross)
}

func resolveCrossMargins(item *LayoutNode, dir FlexDirection, crossAvailable int) int {
	return resolveCrossMarginBefore(item, dir, crossAvailable) + resolveCrossMarginAfter(item, dir, crossAvailable)
}

// resolveCrossMarginBefore returns the margin before the item on the cross axis.
func resolveCrossMarginBefore(item *LayoutNode, dir FlexDirection, crossAvailable int) int {
	var s Scalar
	if dir.IsRow() {
		s = item.MarginTop
	} else {
		s = item.MarginLeft
	}
	return safeResolve(s, crossAvailable)
}

// resolveCrossMarginAfter returns the margin after the item on the cross axis.
func resolveCrossMarginAfter(item *LayoutNode, dir FlexDirection, crossAvailable int) int {
	var s Scalar
	if dir.IsRow() {
		s = item.MarginBottom
	} else {
		s = item.MarginRight
	}
	return safeResolve(s, crossAvailable)
}

// resolveMainMin returns the main-axis minimum size, or 0 if undefined.
func resolveMainMin(item *LayoutNode, dir FlexDirection, mainAvailable int) int {
	var s Scalar
	if dir.IsRow() {
		s = item.MinWidth
	} else {
		s = item.MinHeight
	}
	return safeResolve(s, mainAvailable)
}

// resolveMainMax returns the main-axis maximum size, or maxInt if undefined.
func resolveMainMax(item *LayoutNode, dir FlexDirection, mainAvailable int) int {
	var s Scalar
	if dir.IsRow() {
		s = item.MaxWidth
	} else {
		s = item.MaxHeight
	}
	v, ok := s.Resolve(mainAvailable)
	if !ok {
		return maxInt
	}
	return v
}

// resolveItemCrossSize resolves the cross-axis size for an item.
func resolveItemCrossSize(item *LayoutNode, container *LayoutNode, dir FlexDirection, crossAvailable int) int {
	cross, hasExplicit := resolveItemExplicitCrossSize(item, dir, crossAvailable)
	if hasExplicit {
		return cross + resolveCrossMargins(item, dir, crossAvailable)
	}

	// For nodes with a MeasureFunc, use intrinsic cross-axis size.
	// Main axis is already resolved (Exactly), cross axis is AtMost.
	if item.Measure != nil {
		mainSize := getMainComputedSize(item, dir)
		var result MeasureResult
		if dir.IsRow() {
			result = item.Measure(mainSize, MeasureExactly, crossAvailable, MeasureAtMost)
		} else {
			result = item.Measure(crossAvailable, MeasureAtMost, mainSize, MeasureExactly)
		}
		measuredCross := crossFromResult(result, dir)
		return measuredCross + resolveCrossMargins(item, dir, crossAvailable)
	}

	// No explicit cross size: use 0 (will be stretched if AlignStretch)
	return resolveCrossMargins(item, dir, crossAvailable)
}

// resolveItemExplicitCrossSize resolves an explicit cross-axis size, or (0, false) if auto.
func resolveItemExplicitCrossSize(item *LayoutNode, dir FlexDirection, crossAvailable int) (int, bool) {
	var s Scalar
	if dir.IsRow() {
		s = item.Height
	} else {
		s = item.Width
	}
	if s.IsAuto() || s.IsUndefined() {
		return 0, false
	}
	v, ok := s.Resolve(crossAvailable)
	if !ok {
		return 0, false
	}

	// Apply cross min/max
	var minS, maxS Scalar
	if dir.IsRow() {
		minS = item.MinHeight
		maxS = item.MaxHeight
	} else {
		minS = item.MinWidth
		maxS = item.MaxWidth
	}
	minV := safeResolve(minS, crossAvailable)
	maxV, maxOk := maxS.Resolve(crossAvailable)
	if !maxOk {
		maxV = maxInt
	}
	return clampSize(v, minV, maxV), true
}

// resolveAlign resolves the effective alignment for an item.
func resolveAlign(item *LayoutNode, container *LayoutNode) Align {
	if item.AlignSelf != AlignAuto {
		return item.AlignSelf
	}
	return container.AlignItems
}

// --- Computed Layout Setters/Getters by Axis ---

func setMainComputedSize(item *LayoutNode, dir FlexDirection, size int) {
	if dir.IsRow() {
		item.Computed.Width = size
	} else {
		item.Computed.Height = size
	}
}

func setCrossComputedSize(item *LayoutNode, dir FlexDirection, size int) {
	if dir.IsRow() {
		item.Computed.Height = size
	} else {
		item.Computed.Width = size
	}
}

func getMainComputedSize(item *LayoutNode, dir FlexDirection) int {
	if dir.IsRow() {
		return item.Computed.Width
	}
	return item.Computed.Height
}

func getCrossComputedSize(item *LayoutNode, dir FlexDirection) int {
	if dir.IsRow() {
		return item.Computed.Height
	}
	return item.Computed.Width
}

func setMainComputedPos(item *LayoutNode, dir FlexDirection, pos int) {
	if dir.IsRow() {
		item.Computed.X = pos
	} else {
		item.Computed.Y = pos
	}
}

func setCrossComputedPos(item *LayoutNode, dir FlexDirection, pos int) {
	if dir.IsRow() {
		item.Computed.Y = pos
	} else {
		item.Computed.X = pos
	}
}

// resolveNodeDimension resolves a node's size along one axis, applying min/max.
func resolveNodeDimension(size, minSize, maxSize Scalar, available int) int {
	v, ok := size.Resolve(available)
	if !ok {
		v = available
	}
	minV := safeResolve(minSize, available)
	maxV, maxOk := maxSize.Resolve(available)
	if !maxOk {
		maxV = maxInt
	}
	return clampSize(v, minV, maxV)
}

func resolveNodeWidth(node *LayoutNode, availableWidth int) int {
	return resolveNodeDimension(node.Width, node.MinWidth, node.MaxWidth, availableWidth)
}

func resolveNodeHeight(node *LayoutNode, availableHeight int) int {
	return resolveNodeDimension(node.Height, node.MinHeight, node.MaxHeight, availableHeight)
}

// reverseNodes reverses a slice of LayoutNode pointers in place.
func reverseNodes(nodes []*LayoutNode) {
	for i, j := 0, len(nodes)-1; i < j; i, j = i+1, j-1 {
		nodes[i], nodes[j] = nodes[j], nodes[i]
	}
}

// --- MeasureFunc helpers ---

// measureModes determines the MeasureMode for width and height based on the
// item's sizing scalars and the parent's flex direction.
// - Explicit size (Points/Percent) -> MeasureExactly
// - Auto on the main axis (why we're measuring) -> MeasureAtMost
// - Auto on the cross axis -> MeasureAtMost (bounded by available)
// - Undefined -> MeasureUndefined
func measureModes(item *LayoutNode, dir FlexDirection, mainAvailable, crossAvailable int) (widthMode, heightMode MeasureMode) {
	widthMode = scalarToMode(item.Width)
	heightMode = scalarToMode(item.Height)
	if dir.IsRow() {
		// Main axis = width. We're here because width is Auto -> AtMost.
		if widthMode != MeasureExactly {
			widthMode = MeasureAtMost
		}
	} else {
		// Main axis = height. We're here because height is Auto -> AtMost.
		if heightMode != MeasureExactly {
			heightMode = MeasureAtMost
		}
	}
	return
}

// scalarToMode converts a Scalar to the appropriate MeasureMode.
func scalarToMode(s Scalar) MeasureMode {
	if s.IsAuto() {
		return MeasureAtMost
	}
	if s.IsUndefined() {
		return MeasureUndefined
	}
	// Points or Percent -> resolves to a concrete value
	return MeasureExactly
}

// crossFromResult extracts the cross-axis value from a MeasureResult.
func crossFromResult(r MeasureResult, dir FlexDirection) int {
	if dir.IsRow() {
		return r.Height
	}
	return r.Width
}
