package subpaving

import (
	"bytes"
	"errors"
	"testing"
)

func TestFastSearchBreadthFirstCancellationAndOwnedChildren(t *testing.T) {
	_, c, _, root := fastMonomialFixture(t, []bool{false})
	c.maxDepth = 1
	c.limit.Limit, c.limit.Steps = 1, 1
	if err := c.Search(); !errors.Is(err, ErrFastContextCanceled) || len(c.leafQueue) != 1 || c.leafQueue[0] != root || c.searchNode != nil {
		t.Fatal("pre-selection checkpoint", err)
	}
	c.limit.Limit, c.limit.Steps = 1, 0
	if err := c.Search(); !errors.Is(err, ErrFastContextCanceled) || c.numSplits != 1 || len(c.leafQueue) != 2 || root.children != 2 || len(c.searchOwned) != 2 {
		t.Fatal("child queue retained on cancellation", err)
	}
	if c.leafQueue[0] != c.searchOwned[0] || c.leafQueue[1] != c.searchOwned[1] {
		t.Fatal("breadth-first child order")
	}
	c.limit.Limit, c.limit.Steps = 2, 1
	if err := c.Search(); !errors.Is(err, ErrFastContextCanceled) || c.searchNode != c.searchOwned[0] || c.searchPhase != 1 || len(c.queue) != 1 || len(c.leafQueue) != 1 {
		t.Fatal("selected child and pending propagation retained", err)
	}
	c.limit.Limit, c.limit.Steps = 100, 0
	if err := c.Search(); err != nil || c.numSplits != 1 || len(c.leafQueue) != 0 || c.searchNode != nil {
		t.Fatal("resumed child search", err)
	}
	var out bytes.Buffer
	if err := c.DisplayBounds(&out); err != nil || out.String() != "x0 < 0\n=========\n0 <= x0 \n" {
		t.Fatal("searched leaves", out.String(), err)
	}
	if err := c.Close(); err != nil || len(c.nodes) != 0 || len(c.searchOwned) != 0 {
		t.Fatal("owned child cleanup", err)
	}
}

func TestFastSearchMaxNodesStopsBeforeNextLeaf(t *testing.T) {
	_, c, _, root := fastMonomialFixture(t, []bool{false})
	c.maxDepth, c.maxNodes = 2, 1
	if err := c.Search(); err != nil || c.numSplits != 1 || len(c.leafQueue) != 2 || root.children != 2 {
		t.Fatal("source node-count stop", err)
	}
}

func TestFastSearchRetainsRefusedSplitChildrenForOwnerClose(t *testing.T) {
	m, c, v, root := fastMonomialFixture(t, []bool{false})
	if err := m.SetInt32(v, 1); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateBound(root, 0, v, true, false); err != nil {
		t.Fatal(err)
	}
	if err := m.Next(v); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateBound(root, 0, v, false, false); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateQueued(root); err != nil {
		t.Fatal(err)
	}
	if err := c.Search(); !errors.Is(err, ErrFastSplitNoInterior) || !errors.Is(err, ErrSubpavingFailure) || len(c.searchOwned) != 2 || root.children != 2 {
		t.Fatal("source split refusal ownership", err)
	}
	if err := c.Close(); err != nil {
		t.Fatal("refused search cleanup", err)
	}
}
