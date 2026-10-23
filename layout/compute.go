package layout

import "math"

// ComputeLayout resolves the dirty subtree rooted at n within the available
// width and height.
//
// A clean root returns without traversing the tree. During layout, each node
// clears its own dirty flag before measuring descendants. If a measurement
// marks the tree dirty again, ComputeLayout repeats until the root remains
// clean.
//
// Callers must perform tree mutations and layout on the same goroutine.
func (n *LayoutNode) ComputeLayout(availableWidth, availableHeight int) {
	for n.dirty {
		n.layoutFlexContainer(availableWidth, availableHeight, 0, 0)
	}
}

// safeResolve resolves s and returns zero when s is automatic or undefined.
func safeResolve(s Scalar, parentDim int) int {
	v, ok := s.Resolve(parentDim)
	if !ok {
		return 0
	}
	return v
}

// resolveMargins resolves node's margins against the parent dimensions.
func resolveMargins(node *LayoutNode, parentW, parentH int) (top, right, bottom, left int) {
	top = safeResolve(node.MarginTop, parentH)
	right = safeResolve(node.MarginRight, parentW)
	bottom = safeResolve(node.MarginBottom, parentH)
	left = safeResolve(node.MarginLeft, parentW)
	return
}

// resolvePaddings resolves node's paddings against the parent dimensions.
func resolvePaddings(node *LayoutNode, parentW, parentH int) (top, right, bottom, left int) {
	top = safeResolve(node.PaddingTop, parentH)
	right = safeResolve(node.PaddingRight, parentW)
	bottom = safeResolve(node.PaddingBottom, parentH)
	left = safeResolve(node.PaddingLeft, parentW)
	return
}

// clampSize restricts size to the inclusive range [min, max].
func clampSize(size, min, max int) int {
	if size < min {
		size = min
	}
	if size > max {
		size = max
	}
	return size
}

// maxInt represents an absent maximum while leaving room for layout arithmetic.
const maxInt = math.MaxInt32
