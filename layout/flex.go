package layout

import "math"

// flexLine contains the items and dimensions of one flex line.
type flexLine struct {
	items     []*Node
	mainSize  int // The main-axis space consumed by items and gaps.
	crossSize int // The largest outer cross-axis size on the line.
}

// flexItemInfo contains intermediate main-axis sizing for one item.
type flexItemInfo struct {
	basis    int
	minMain  int
	maxMain  int
	frozen   bool
	mainSize int
}

// layoutFlexContainer resolves node and its descendants. parentX and parentY
// give node's absolute border-box position.
func (node *Node) layoutFlexContainer(availableWidth, availableHeight int, parentX, parentY int) {
	// Reuse a clean subtree when its available dimensions are unchanged. If only
	// its origin changed, shift the cached descendant positions by the same delta.
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

	// Clear dirty before measuring descendants. A MeasureFunc can mark the tree
	// dirty again, causing ComputeLayout to repeat.
	node.dirty = false

	dir := node.FlexDirection

	containerWidth := resolveNodeWidth(node, availableWidth)
	containerHeight := resolveNodeHeight(node, availableHeight)

	// Clamp the content box when padding exceeds the resolved dimensions.
	padTop, padRight, padBottom, padLeft := resolvePaddings(node, containerWidth, containerHeight)
	contentWidth := containerWidth - padLeft - padRight
	contentHeight := containerHeight - padTop - padBottom

	if contentWidth < 0 {
		contentWidth = 0
	}
	if contentHeight < 0 {
		contentHeight = 0
	}

	node.Computed.X = parentX
	node.Computed.Y = parentY
	node.Computed.Width = containerWidth
	node.Computed.Height = containerHeight

	contentX := parentX + padLeft
	contentY := parentY + padTop

	// Gone children have no layout; absolute children are resolved after flex flow.
	var flexItems []*Node
	var absoluteItems []*Node
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

	// Reverse source order before line collection for reverse directions.
	if dir.IsReverse() {
		reverseNodes(flexItems)
	}

	mainAvailable := mainDimension(contentWidth, contentHeight, dir)
	crossAvailable := crossDimension(contentWidth, contentHeight, dir)

	mainGap := resolveGap(node, dir)
	crossGap := resolveCrossGap(node, dir)

	lines := collectFlexLines(flexItems, node, dir, mainAvailable, crossAvailable, mainGap)

	// Resolve each line's item sizes before assigning positions.
	isSingleLine := node.FlexWrap == WrapNone || len(lines) <= 1
	for _, line := range lines {
		resolveFlexibleLengths(line, node, dir, mainAvailable, crossAvailable, mainGap)
		resolveLineCrossSize(line, node, dir, crossAvailable, isSingleLine)
	}

	positionMainAxis(lines, node, dir, mainAvailable, mainGap)
	positionCrossAxis(lines, node, dir, crossAvailable, crossGap)

	// Convert positions from content-box coordinates to absolute coordinates,
	// then lay out each child container.
	for _, line := range lines {
		for _, item := range line.items {
			item.Computed.X = contentX + item.Computed.X
			item.Computed.Y = contentY + item.Computed.Y

			if len(item.Children) > 0 {
				item.layoutFlexContainer(item.Computed.Width, item.Computed.Height,
					item.Computed.X, item.Computed.Y)
			} else {
				// The parent fully resolves a leaf node.
				item.dirty = false
			}
		}
	}

	layoutAbsoluteChildren(absoluteItems, node, contentX, contentY, contentWidth, contentHeight)

	// Cache the inputs needed to reuse this subtree.
	node.Cache = Cache{
		AvailWidth:  availableWidth,
		AvailHeight: availableHeight,
		ParentX:     parentX,
		ParentY:     parentY,
		Valid:       true,
	}
}

// collectFlexLines groups items according to the container's wrapping mode.
func collectFlexLines(items []*Node, container *Node, dir FlexDirection, mainAvailable, crossAvailable, mainGap int) []*flexLine {
	if len(items) == 0 {
		return nil
	}

	if container.FlexWrap == WrapNone {
		// A non-wrapping container keeps all items on one line.
		return []*flexLine{{items: items}}
	}

	var lines []*flexLine
	var currentItems []*Node
	var lineMainSize int

	for i, item := range items {
		basis := resolveFlexBasis(item, container, dir, mainAvailable, crossAvailable)
		margins := resolveMainMargins(item, dir, mainAvailable)
		itemMainSize := basis + margins

		// A gap precedes every item except the first on a line.
		gapSize := 0
		if len(currentItems) > 0 {
			gapSize = mainGap
		}

		if len(currentItems) > 0 && lineMainSize+gapSize+itemMainSize > mainAvailable {
			// Move an item that would exceed the main-axis limit to a new line.
			lines = append(lines, &flexLine{items: currentItems, mainSize: lineMainSize})
			currentItems = []*Node{item}
			lineMainSize = itemMainSize
		} else {
			currentItems = append(currentItems, item)
			lineMainSize += gapSize + itemMainSize
		}

		// Append the final open line.
		if i == len(items)-1 {
			lines = append(lines, &flexLine{items: currentItems, mainSize: lineMainSize})
		}
	}

	// WrapReverse changes line order without changing item order within each line.
	if container.FlexWrap == WrapReverse {
		for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
			lines[i], lines[j] = lines[j], lines[i]
		}
	}

	return lines
}

// resolveFlexibleLengths applies flex growth or shrinkage to one line.
func resolveFlexibleLengths(line *flexLine, container *Node, dir FlexDirection, mainAvailable, crossAvailable, mainGap int) {
	items := line.items
	if len(items) == 0 {
		return
	}

	// Free space excludes every item's basis, margins, and inter-item gaps.
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
		// Positive free space is distributed by FlexGrow.
		distributeGrow(items, info, freeSpace)
	} else if freeSpace < 0 && container.Overflow != OverflowScroll {
		// Scroll containers preserve overflowing basis sizes; other containers
		// distribute the deficit by FlexShrink.
		distributeShrink(items, info, -freeSpace)
	} else {
		// With no distributed free space, retain each basis.
		for i := range items {
			info[i].mainSize = info[i].basis
		}
	}

	// Apply min and max constraints after flexible sizing.
	for i, item := range items {
		size := max(clampSize(info[i].mainSize, info[i].minMain, info[i].maxMain), 0)
		setMainComputedSize(item, dir, size)
	}
}

// distributeGrow apportions freeSpace according to positive FlexGrow factors.
// Integer rounding remainder goes to the last eligible item.
func distributeGrow(items []*Node, info []flexItemInfo, freeSpace int) {
	totalGrow := float32(0)
	for _, item := range items {
		totalGrow += float32(item.FlexGrow)
	}

	if totalGrow == 0 {
		// Without a positive growth factor, every item retains its basis.
		for i := range items {
			info[i].mainSize = info[i].basis
		}
		return
	}

	// First distribute proportional integer shares and freeze constrained items.
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

	// Give rounding remainder to the last unconstrained growing item.
	remainder := freeSpace - distributed
	if remainder > 0 && lastGrowIdx >= 0 && !info[lastGrowIdx].frozen {
		info[lastGrowIdx].mainSize += remainder
	}

	// If that item was constrained, redistribute the remainder among unfrozen items.
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
			// Give the second pass's rounding remainder to its last eligible item.
			if leftover := extra - extraDistributed; leftover > 0 && lastUnfrozen >= 0 {
				info[lastUnfrozen].mainSize += leftover
			}
		}
	}
}

// distributeShrink apportions overflow by each item's FlexShrink-weighted basis.
func distributeShrink(items []*Node, info []flexItemInfo, overflow int) {
	// A larger basis or shrink factor receives a larger share of the deficit.
	totalWeightedShrink := float32(0)
	for i, item := range items {
		totalWeightedShrink += float32(item.FlexShrink) * float32(info[i].basis)
	}

	if totalWeightedShrink == 0 {
		// With no positive shrink weight, the line remains overflowing.
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

	// Give integer rounding remainder to the last shrinking item.
	if leftover := overflow - shrunk; leftover > 0 && lastShrinkIdx >= 0 {
		info[lastShrinkIdx].mainSize -= leftover
	}
}

// resolveLineCrossSize resolves the line's outer cross size and each item's
// inner cross size.
func resolveLineCrossSize(line *flexLine, container *Node, dir FlexDirection, crossAvailable int, isSingleLine bool) {
	maxCross := 0

	for _, item := range line.items {
		cross := resolveItemCrossSize(item, container, dir, crossAvailable)
		if cross > maxCross {
			maxCross = cross
		}
	}

	// A single flex line uses at least the container's available cross size.
	if isSingleLine && crossAvailable > maxCross {
		maxCross = crossAvailable
	}

	line.crossSize = maxCross

	// Resolve explicit, stretched, and intrinsically measured item sizes.
	for _, item := range line.items {
		align := resolveAlign(item, container)
		explicitCross, hasExplicit := resolveItemExplicitCrossSize(item, dir, crossAvailable)
		margins := resolveCrossMargins(item, dir, crossAvailable)

		if hasExplicit {
			setCrossComputedSize(item, dir, explicitCross)
		} else if align == AlignStretch {
			// Stretch the margin box to the line's cross size.
			stretchedCross := max(maxCross-margins, 0)
			setCrossComputedSize(item, dir, stretchedCross)
		} else if item.Measure != nil {
			// A non-stretched automatic size comes from intrinsic measurement.
			mainSize := getMainComputedSize(item, dir)
			var result MeasureResult
			if dir.IsRow() {
				result = item.Measure(mainSize, MeasureExactly, crossAvailable, MeasureAtMost)
			} else {
				result = item.Measure(crossAvailable, MeasureAtMost, mainSize, MeasureExactly)
			}
			setCrossComputedSize(item, dir, crossFromResult(result, dir))
		}
		// An unmeasured automatic item that is not stretched remains zero-sized.
	}
}

// positionMainAxis places each line's items according to JustifyContent.
func positionMainAxis(lines []*flexLine, container *Node, dir FlexDirection, mainAvailable, mainGap int) {
	for _, line := range lines {
		items := line.items
		if len(items) == 0 {
			continue
		}

		// Justification distributes space not consumed by items, margins, or gaps.
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

// positionCrossAxis places items within each line and advances between lines.
func positionCrossAxis(lines []*flexLine, container *Node, dir FlexDirection, crossAvailable, crossGap int) {
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

// layoutAbsoluteChildren resolves items against the container's content box.
func layoutAbsoluteChildren(items []*Node, container *Node, contentX, contentY, contentWidth, contentHeight int) {
	for _, item := range items {
		itemWidth := resolveNodeWidth(item, contentWidth)
		itemHeight := resolveNodeHeight(item, contentHeight)

		// Opposing offsets determine an automatic size on that axis.
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

		var x int
		if leftSet {
			x = safeResolve(item.Left, contentWidth)
		} else if rightSet {
			x = contentWidth - safeResolve(item.Right, contentWidth) - itemWidth
		}

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

// mainDimension returns the dimension along dir's main axis.
func mainDimension(width, height int, dir FlexDirection) int {
	if dir.IsRow() {
		return width
	}
	return height
}

// crossDimension returns the dimension along dir's cross axis.
func crossDimension(width, height int, dir FlexDirection) int {
	if dir.IsRow() {
		return height
	}
	return width
}

// resolveGap returns the main-axis gap. A positive axis-specific gap overrides
// the general Gap value.
func resolveGap(node *Node, dir FlexDirection) int {
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

// resolveCrossGap returns the cross-axis gap. A positive axis-specific gap
// overrides the general Gap value.
func resolveCrossGap(node *Node, dir FlexDirection) int {
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

// resolveFlexBasis returns item's initial main-axis size before flexible
// growth or shrinkage.
func resolveFlexBasis(item *Node, container *Node, dir FlexDirection, mainAvailable, crossAvailable int) int {
	if !item.FlexBasis.IsAuto() && !item.FlexBasis.IsUndefined() {
		if resolved, ok := item.FlexBasis.Resolve(mainAvailable); ok {
			return resolved
		}
	}

	// An explicit main-axis dimension is the fallback basis.
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

	// Measure an automatic basis under the available main and cross constraints.
	//
	// A clipping or scrolling container leaves the main axis unconstrained so
	// content can report its natural size beyond the viewport.
	if item.Measure != nil {
		wMode, hMode := measureModes(item, dir, mainAvailable, crossAvailable)
		if container.Overflow == OverflowScroll || container.Overflow == OverflowHidden {
			if dir.IsRow() {
				wMode = MeasureUndefined
			} else {
				hMode = MeasureUndefined
			}
		}
		// Convert main and cross constraints back to physical width and height.
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

	// An automatic container basis is derived from its children's intrinsic
	// main-axis sizes.
	//
	// Intrinsic size can depend on the cross-axis constraint, as with wrapping
	// text. Reuse a clean cached size only when that effective constraint is
	// unchanged.
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

// intrinsicMainSize returns an automatic container's shrink-wrapped size along
// its parent's main axis.
//
// crossAvailable propagates the parent's cross-axis constraint to descendants
// whose measured main-axis size depends on it.
//
// When node and its parent use the same axis, child sizes and gaps are summed.
// When their axes differ, the largest child size determines the result.
func intrinsicMainSize(node *Node, parentDir FlexDirection, mainAvailable, crossAvailable int) int {
	childDir := node.FlexDirection
	sameAxis := childDir.IsRow() == parentDir.IsRow()

	// Main-axis padding contributes directly to the intrinsic size.
	padTop, _, padBottom, padLeft := resolvePaddings(node, mainAvailable, mainAvailable)
	padRight := safeResolve(node.PaddingRight, mainAvailable)
	var mainPad int
	if parentDir.IsRow() {
		mainPad = padLeft + padRight
	} else {
		mainPad = padTop + padBottom
	}

	// Remove node's cross-axis margins and padding before measuring children.
	var crossPad int
	if parentDir.IsRow() {
		crossPad = padTop + padBottom
	} else {
		crossPad = padLeft + padRight
	}
	crossMargins := resolveCrossMargins(node, parentDir, crossAvailable)
	childCross := max(crossAvailable-crossPad-crossMargins, 0)

	// A large main-axis limit lets measured children report their natural size.
	unconstrained := math.MaxInt16

	total := 0
	childCount := 0

	if !sameAxis {
		// With perpendicular axes, a child's size along the parent axis can
		// depend on its resolved size along node's axis, as with wrapping text.
		//
		// Resolve those constraints with a lightweight pass before measuring
		// each child along the parent axis.
		childMainSizes := preResolveMainSizes(node, childDir, childCross, unconstrained)

		idx := 0
		for _, child := range node.Children {
			if child.Position == PositionAbsolute {
				continue
			}
			if child.Visibility == Gone {
				continue
			}

			// The pre-resolved size becomes the child's cross constraint when
			// measuring along the parent axis.
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
		// With matching axes, children stack and contribute their bases directly.
		for _, child := range node.Children {
			if child.Position == PositionAbsolute {
				continue
			}
			if child.Visibility == Gone {
				continue
			}

			childSize := resolveFlexBasis(child, node, childDir, unconstrained, childCross)
			childSize += resolveMainMargins(child, childDir, unconstrained)
			total += childSize
			childCount++
		}

		// Only gaps along the stacking direction contribute to the sum.
		if childCount > 1 {
			total += resolveGap(node, childDir) * (childCount - 1)
		}
	}

	return total + mainPad
}

// preResolveMainSizes returns each in-flow child's tentative main-axis size
// without modifying Computed. intrinsicMainSize uses these values as cross-axis
// constraints when a container and its parent have perpendicular axes.
func preResolveMainSizes(node *Node, dir FlexDirection, mainAvailable, crossAvailable int) []int {
	var items []*Node
	for _, child := range node.Children {
		if child.Position != PositionAbsolute && child.Visibility != Gone {
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

// resolveMainMargins returns the sum of item's main-axis margins.
func resolveMainMargins(item *Node, dir FlexDirection, mainAvailable int) int {
	return resolveMainMarginBefore(item, dir, mainAvailable) + resolveMainMarginAfter(item, dir, mainAvailable)
}

// resolveMainMarginBefore returns item's leading main-axis margin.
func resolveMainMarginBefore(item *Node, dir FlexDirection, mainAvailable int) int {
	var s Scalar
	if dir.IsRow() {
		s = item.MarginLeft
	} else {
		s = item.MarginTop
	}
	return safeResolve(s, mainAvailable)
}

// resolveMainMarginAfter returns item's trailing main-axis margin.
func resolveMainMarginAfter(item *Node, dir FlexDirection, mainAvailable int) int {
	var s Scalar
	if dir.IsRow() {
		s = item.MarginRight
	} else {
		s = item.MarginBottom
	}
	return safeResolve(s, mainAvailable)
}

// resolveChildCrossSize returns item's effective cross-axis constraint for the
// intrinsic-size cache.
func resolveChildCrossSize(item *Node, parentDir FlexDirection, parentCross int) int {
	if explicit, ok := resolveItemExplicitCrossSize(item, parentDir, parentCross); ok {
		return explicit
	}
	return parentCross - resolveCrossMargins(item, parentDir, parentCross)
}

func resolveCrossMargins(item *Node, dir FlexDirection, crossAvailable int) int {
	return resolveCrossMarginBefore(item, dir, crossAvailable) + resolveCrossMarginAfter(item, dir, crossAvailable)
}

// resolveCrossMarginBefore returns item's leading cross-axis margin.
func resolveCrossMarginBefore(item *Node, dir FlexDirection, crossAvailable int) int {
	var s Scalar
	if dir.IsRow() {
		s = item.MarginTop
	} else {
		s = item.MarginLeft
	}
	return safeResolve(s, crossAvailable)
}

// resolveCrossMarginAfter returns item's trailing cross-axis margin.
func resolveCrossMarginAfter(item *Node, dir FlexDirection, crossAvailable int) int {
	var s Scalar
	if dir.IsRow() {
		s = item.MarginBottom
	} else {
		s = item.MarginRight
	}
	return safeResolve(s, crossAvailable)
}

// resolveMainMin returns item's main-axis minimum, or zero when unspecified.
func resolveMainMin(item *Node, dir FlexDirection, mainAvailable int) int {
	var s Scalar
	if dir.IsRow() {
		s = item.MinWidth
	} else {
		s = item.MinHeight
	}
	return safeResolve(s, mainAvailable)
}

// resolveMainMax returns item's main-axis maximum, or maxInt when unspecified.
func resolveMainMax(item *Node, dir FlexDirection, mainAvailable int) int {
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

// resolveItemCrossSize returns item's outer cross-axis size.
func resolveItemCrossSize(item *Node, container *Node, dir FlexDirection, crossAvailable int) int {
	cross, hasExplicit := resolveItemExplicitCrossSize(item, dir, crossAvailable)
	if hasExplicit {
		return cross + resolveCrossMargins(item, dir, crossAvailable)
	}

	// Measure an automatic cross size with an exact main size and bounded cross size.
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

	// Without an explicit or measured size, only margins contribute.
	return resolveCrossMargins(item, dir, crossAvailable)
}

// resolveItemExplicitCrossSize returns a constrained explicit cross-axis size.
// It reports false for automatic or undefined sizes.
func resolveItemExplicitCrossSize(item *Node, dir FlexDirection, crossAvailable int) (int, bool) {
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

	// Apply minimum and maximum constraints on the physical cross axis.
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

// resolveAlign returns item's alignment override or the container default.
func resolveAlign(item *Node, container *Node) Align {
	if item.AlignSelf != AlignAuto {
		return item.AlignSelf
	}
	return container.AlignItems
}

func setMainComputedSize(item *Node, dir FlexDirection, size int) {
	if dir.IsRow() {
		item.Computed.Width = size
	} else {
		item.Computed.Height = size
	}
}

func setCrossComputedSize(item *Node, dir FlexDirection, size int) {
	if dir.IsRow() {
		item.Computed.Height = size
	} else {
		item.Computed.Width = size
	}
}

func getMainComputedSize(item *Node, dir FlexDirection) int {
	if dir.IsRow() {
		return item.Computed.Width
	}
	return item.Computed.Height
}

func getCrossComputedSize(item *Node, dir FlexDirection) int {
	if dir.IsRow() {
		return item.Computed.Height
	}
	return item.Computed.Width
}

func setMainComputedPos(item *Node, dir FlexDirection, pos int) {
	if dir.IsRow() {
		item.Computed.X = pos
	} else {
		item.Computed.Y = pos
	}
}

func setCrossComputedPos(item *Node, dir FlexDirection, pos int) {
	if dir.IsRow() {
		item.Computed.Y = pos
	} else {
		item.Computed.X = pos
	}
}

// resolveNodeDimension resolves one preferred dimension and applies its bounds.
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

func resolveNodeWidth(node *Node, availableWidth int) int {
	return resolveNodeDimension(node.Width, node.MinWidth, node.MaxWidth, availableWidth)
}

func resolveNodeHeight(node *Node, availableHeight int) int {
	return resolveNodeDimension(node.Height, node.MinHeight, node.MaxHeight, availableHeight)
}

// reverseNodes reverses nodes in place.
func reverseNodes(nodes []*Node) {
	for i, j := 0, len(nodes)-1; i < j; i, j = i+1, j-1 {
		nodes[i], nodes[j] = nodes[j], nodes[i]
	}
}

// measureModes returns physical width and height modes for intrinsic
// measurement. Explicit dimensions are exact, automatic dimensions are
// bounded, and undefined dimensions are unconstrained.
func measureModes(item *Node, dir FlexDirection, mainAvailable, crossAvailable int) (widthMode, heightMode MeasureMode) {
	widthMode = scalarToMode(item.Width)
	heightMode = scalarToMode(item.Height)
	if dir.IsRow() {
		// Intrinsic row measurement always bounds an automatic width.
		if widthMode != MeasureExactly {
			widthMode = MeasureAtMost
		}
	} else {
		// Intrinsic column measurement always bounds an automatic height.
		if heightMode != MeasureExactly {
			heightMode = MeasureAtMost
		}
	}
	return
}

// scalarToMode returns the measurement constraint represented by s.
func scalarToMode(s Scalar) MeasureMode {
	if s.IsAuto() {
		return MeasureAtMost
	}
	if s.IsUndefined() {
		return MeasureUndefined
	}
	// Point and percentage values resolve to exact dimensions.
	return MeasureExactly
}

// crossFromResult returns the measured dimension along dir's cross axis.
func crossFromResult(r MeasureResult, dir FlexDirection) int {
	if dir.IsRow() {
		return r.Height
	}
	return r.Width
}
