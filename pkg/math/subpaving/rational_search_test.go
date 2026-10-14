package subpaving

import (
	"errors"
	"math"
	"testing"
)

func TestRationalSearchBreadthFirstChildrenAndDefinitionPropagation(t *testing.T) {
	c, _, _, x, _ := rationalClauseFixture(t)
	z, err := c.DefineMonomial([]RationalPower{{Variable: x, Degree: 2}})
	if err != nil {
		t.Fatal(err)
	}
	c.maxDepth = 1
	if err := c.Search(); err != nil {
		t.Fatal(err)
	}
	if c.root == nil || len(c.nodes) != 3 || c.numSplits != 1 || c.leafHead != nil || c.leafTail != nil || c.searchNode != nil || c.queueHead != 0 || len(c.queue) != 0 {
		t.Fatal("breadth-first search did not finish both depth-one children")
	}
	left, right := c.nodes[1], c.nodes[2]
	if left.parent != c.root || right.parent != c.root || left.depth != 1 || right.depth != 1 || left.upper[x] == nil || right.lower[x] == nil || left.lower[z] == nil || right.lower[z] == nil || left.lower[z].reasonDefinition != z || right.lower[z].reasonDefinition != z || left.conflictVar != math.MaxUint32 || right.conflictVar != math.MaxUint32 {
		t.Fatal("selected child decisions did not propagate to the watched monomial")
	}
	steps, visits := c.limit.Steps, c.numVisited
	if err := c.Search(); err != nil || c.limit.Steps != steps || c.numVisited != visits {
		t.Fatal("completed search replayed", err)
	}
}

func TestRationalSearchResumesSelectedChildAfterQueueCancellation(t *testing.T) {
	c, _, _, x, _ := rationalClauseFixture(t)
	c.maxDepth = 1
	if err := c.InitializeRoot(); err != nil {
		t.Fatal(err)
	}
	root, err := c.PopLeafBFS()
	if err != nil || root != c.root {
		t.Fatal("root selection", err)
	}
	left, right, err := c.SplitMidpoint(root, x, true, 1)
	if err != nil {
		t.Fatal(err)
	}
	c.limit.Limit = 1
	if err := c.Search(); !errors.Is(err, ErrRationalContextCanceled) || c.searchNode != left || c.searchPhase != 1 || c.queueHead != 0 || len(c.queue) != 1 || c.queue[0] != left.trail || c.queue[0].timestamp != c.timestamp {
		t.Fatal("canceled child propagation did not retain selected decision", err)
	}
	c.limit.Limit = 0
	if err := c.Search(); err != nil {
		t.Fatal(err)
	}
	if c.searchNode != nil || c.leafHead != nil || c.leafTail != nil || c.queueHead != 0 || len(c.queue) != 0 || left.depth != 1 || right.depth != 1 || c.numBounds != 2 {
		t.Fatal("resumed search replayed a decision or failed to drain children")
	}
}

func TestRationalAddRecentBoundsOnlyNewestDirection(t *testing.T) {
	c, m, value, x, _ := rationalClauseFixture(t)
	root, err := c.Root()
	if err != nil {
		t.Fatal(err)
	}
	child, err := c.NewChild(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []int32{1, 2} {
		if err := m.SetInt32(value, n); err != nil {
			t.Fatal(err)
		}
		if err := c.DecideBound(child, x, value, true, false); err != nil {
			t.Fatal(err)
		}
	}
	old := child.trail.previous
	if err := c.AddRecentBounds(child); err != nil {
		t.Fatal(err)
	}
	if len(c.queue) != 1 || c.queue[0] != child.trail || c.queue[0] == old || old.timestamp == c.timestamp && old != child.trail {
		t.Fatal("older overridden bound was queued")
	}
}
