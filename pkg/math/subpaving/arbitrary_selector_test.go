package subpaving

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestArbitrarySelectorsBreadthFirstAndRoundRobin(t *testing.T) {
	f := newArbitraryClauseFixture(t)
	c := f.context
	definition, err := c.DefinePolynomial(&f.five, []numeral.ArbitraryFloat{f.four}, []uint32{f.y})
	if err != nil {
		t.Fatal(err)
	}
	root, err := c.Root()
	if err != nil {
		t.Fatal(err)
	}
	if err := c.PublishBound(root, f.x, &f.four, true, false, false); err != nil {
		t.Fatal(err)
	}
	if err := c.PublishBound(root, f.x, &f.four, false, false, false); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateQueued(root); err != nil {
		t.Fatal(err)
	}
	if variable, found, err := c.SelectBranchVariable(root, true); err != nil || !found || variable != f.y {
		t.Fatal("root round-robin skips singleton input and definition", variable, found, err)
	}
	if oldest, err := c.SelectLeafBFS(); err != nil || oldest != root {
		t.Fatal("root must be initial breadth-first leaf", err)
	}
	if popped, err := c.PopLeafBFS(); err != nil || popped != root {
		t.Fatal("root is removed before child creation", err)
	}
	left, err := c.NewChildNode(root)
	if err != nil {
		t.Fatal(err)
	}
	right, err := c.NewChildNode(root)
	if err != nil {
		t.Fatal(err)
	}
	var three numeral.ArbitraryFloat
	if err := f.numbers.SetInt32(&three, 3); err != nil {
		t.Fatal(err)
	}
	if err := c.DecideBound(left, f.y, &three, false, false); err != nil {
		t.Fatal(err)
	}
	if err := f.numbers.Del(&three); err != nil {
		t.Fatal(err)
	}
	if c.leafHead != right || c.leafTail != left || root.queuedLeaf || !left.queuedLeaf || !right.queuedLeaf {
		t.Fatal("push-front children leave first child at breadth-first back")
	}
	if popped, err := c.PopLeafBFS(); err != nil || popped != left || c.leafTail != right {
		t.Fatal("breadth-first pop must take the first-created child", err)
	}
	if variable, found, err := c.SelectBranchVariable(left, true); err != nil || !found || variable != f.y {
		t.Fatal("child round-robin wraps after latest axiom decision", variable, found, err)
	}
	if variable, found, err := c.SelectBranchVariable(left, false); err != nil || !found || variable != definition {
		t.Fatal("including definitions selects earlier unbounded output", variable, found, err)
	}
	undecided, err := c.NewChildNode(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.SelectBranchVariable(undecided, true); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("undecided child lacks a source axiom split", err)
	}
}
