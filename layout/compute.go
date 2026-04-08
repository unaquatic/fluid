package layout

import "math"

// ComputeLayout runs the flexbox algorithm starting from this node.
// availableWidth and availableHeight define the viewport (typically terminal size).
//
// When the tree is clean (no MarkDirty since the last layout), the entire
// flexbox computation is skipped — O(1) instead of O(N). This is critical
// for large trees (thousands of nodes) where layout can take 100-200ms.
//
// Each node clears its own dirty flag after being processed (no recursive
// clearDirty). The loop handles concurrent tree modification: if MarkDirty
// runs during layout and re-dirties the root, we re-run.
func (n *LayoutNode) ComputeLayout(availableWidth, availableHeight int) {
	for n.dirty {
		n.layoutFlexContainer(availableWidth, availableHeight, 0, 0)
	}
}

// --- Helper Functions ---

// safeResolve resolves a scalar, returning 0 for Auto/Undefined.
func safeResolve(s Scalar, parentDim int) int {
	v, ok := s.Resolve(parentDim)
	if !ok {
		return 0
	}
	return v
}

// resolveMargins resolves all four margin scalars for a node.
func resolveMargins(node *LayoutNode, parentW, parentH int) (top, right, bottom, left int) {
	top = safeResolve(node.MarginTop, parentH)
	right = safeResolve(node.MarginRight, parentW)
	bottom = safeResolve(node.MarginBottom, parentH)
	left = safeResolve(node.MarginLeft, parentW)
	return
}

// resolvePaddings resolves all four padding scalars for a node.
func resolvePaddings(node *LayoutNode, parentW, parentH int) (top, right, bottom, left int) {
	top = safeResolve(node.PaddingTop, parentH)
	right = safeResolve(node.PaddingRight, parentW)
	bottom = safeResolve(node.PaddingBottom, parentH)
	left = safeResolve(node.PaddingLeft, parentW)
	return
}

// clampSize applies min/max constraints to a size value.
func clampSize(size, min, max int) int {
	if size < min {
		size = min
	}
	if size > max {
		size = max
	}
	return size
}

// maxInt is a sentinel for "no max constraint". We use a large value
// instead of math.MaxInt to avoid overflow during arithmetic.
const maxInt = math.MaxInt32
