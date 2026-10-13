package subpaving

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestHardwareSelectorsRoundRobinAndBreadthFirstSourceOrder(t *testing.T) {
	m, err := numeral.NewHardwareFloatNumeralManager(numeral.NewHardwareFloatManager(), 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewHardwareContext(m, &control.Reslimit{Limit: 1000}, 0, 2, 8192)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
	}()
	variables := make([]uint32, 3)
	for i := range variables {
		variables[i], err = c.NewVariable(false)
		if err != nil {
			t.Fatal(err)
		}
	}
	var zero, one, three numeral.HardwareFloat
	for _, entry := range []struct {
		out   *numeral.HardwareFloat
		value int32
	}{{&zero, 0}, {&one, 1}, {&three, 3}} {
		if err := m.SetInt32(entry.out, entry.value); err != nil {
			t.Fatal(err)
		}
	}
	definition, err := c.DefinePolynomial(&zero, []HardwarePolynomialTerm{{Variable: variables[1], Coefficient: &one}})
	if err != nil {
		t.Fatal(err)
	}
	root, err := c.CreateRootNode()
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range []struct {
		variable uint32
		value    *numeral.HardwareFloat
	}{{variables[0], &one}, {variables[2], &three}} {
		if err := c.PropagateBound(root, b.variable, b.value, true, false); err != nil {
			t.Fatal(err)
		}
		if err := c.PropagateBound(root, b.variable, b.value, false, false); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.PropagateQueued(root); err != nil {
		t.Fatal(err)
	}
	x, found, err := c.SelectBranchVariable(root, true)
	if err != nil || !found || x != variables[1] {
		t.Fatal("root round-robin must skip singleton inputs and definitions", x, found, err)
	}
	oldest, err := c.SelectLeafBFS()
	if err != nil || oldest != root {
		t.Fatal("root must be initial BFS leaf", err)
	}
	if popped, err := c.PopLeafBFS(); err != nil || popped != root {
		t.Fatal("source removes selected root before splitting", err)
	}
	left, right, err := c.SplitMidpoint(root, variables[1], true, 1)
	if err != nil {
		t.Fatal(err)
	}
	if c.leafHead != right || c.leafTail != left || root.queuedLeaf || !left.queuedLeaf || !right.queuedLeaf {
		t.Fatal("push-front children leave older left child at BFS back")
	}
	if oldest, err := c.PopLeafBFS(); err != nil || oldest != left {
		t.Fatal("BFS must choose first-created left child", err)
	}
	x, found, err = c.SelectBranchVariable(left, true)
	if err != nil || !found || x != variables[1] {
		t.Fatal("child scan wraps after latest axiom split", x, found, err)
	}
	x, found, err = c.SelectBranchVariable(left, false)
	if err != nil || !found || x != definition {
		t.Fatal("including definitions selects earlier unbounded output", x, found, err)
	}
	if err := c.PropagateBound(left, variables[0], &one, true, false); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateQueued(left); err != nil {
		t.Fatal(err)
	}
	x, found, err = c.SelectBranchVariable(left, true)
	if err != nil || !found || x != variables[1] {
		t.Fatal("later non-axiom bound must not reset child round-robin start", x, found, err)
	}
	if err := left.Close(); err != nil {
		t.Fatal(err)
	}
	if c.leafTail != right {
		t.Fatal("closing selected child must leave right sibling queued")
	}
	if err := right.Close(); err != nil {
		t.Fatal(err)
	}
	if c.leafHead != nil || c.leafTail != nil {
		t.Fatal("closing final child removes its BFS leaf")
	}
}

func TestHardwareSelectorSingletonAndUndecidedChild(t *testing.T) {
	m, err := numeral.NewHardwareFloatNumeralManager(numeral.NewHardwareFloatManager(), 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewHardwareContext(m, &control.Reslimit{Limit: 100}, 0, 1, 8192)
	if err != nil {
		t.Fatal(err)
	}
	x, err := c.NewVariable(false)
	if err != nil {
		t.Fatal(err)
	}
	root, err := c.CreateRootNode()
	if err != nil {
		t.Fatal(err)
	}
	var seven numeral.HardwareFloat
	if err := m.SetInt32(&seven, 7); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateBound(root, x, &seven, true, false); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateBound(root, x, &seven, false, false); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateQueued(root); err != nil {
		t.Fatal(err)
	}
	if _, found, err := c.SelectBranchVariable(root, true); err != nil || found {
		t.Fatal("singleton root should have no branch variable", found, err)
	}
	child, err := c.NewChildNode(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.SelectBranchVariable(child, true); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("undecided child has no source axiom split", err)
	}
	if err := child.Close(); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}
