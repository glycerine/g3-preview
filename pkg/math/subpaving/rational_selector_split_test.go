package subpaving

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalSelectorRoundRobinDecisionsAndDefinitions(t *testing.T) {
	c, m, value, x, y := rationalClauseFixture(t)
	z, err := c.DefineMonomial([]RationalPower{{Variable: y, Degree: 2}})
	if err != nil {
		t.Fatal(err)
	}
	root, err := c.Root()
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(value, 4); err != nil {
		t.Fatal(err)
	}
	for _, lower := range []bool{true, false} {
		if err := c.PublishBound(root, x, value, lower, false); err != nil {
			t.Fatal(err)
		}
	}
	variable, found, err := c.SelectBranchVariable(root, true)
	if err != nil || !found || variable != y {
		t.Fatal("root should skip singleton and definition", variable, found, err)
	}
	child, err := c.NewChild(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.DecideBound(child, y, value, false, false); err != nil {
		t.Fatal(err)
	}
	variable, found, err = c.SelectBranchVariable(child, true)
	if err != nil || !found || variable != y {
		t.Fatal("child should wrap past singleton", variable, found, err)
	}
	variable, found, err = c.SelectBranchVariable(child, false)
	if err != nil || !found || variable != z {
		t.Fatal("definition-inclusive selector", variable, found, err)
	}
	undecided, err := c.NewChild(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.SelectBranchVariable(undecided, true); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("child without decision", err)
	}
	if err := c.DecideBound(root, y, value, true, false); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("root accepted a decision", err)
	}
}

func TestRationalMidpointSplitterSourceBounds(t *testing.T) {
	for _, tc := range []struct {
		name         string
		integer      bool
		lower, upper *int32
		left, right  int32
	}{
		{name: "unbounded", left: 0, right: 0},
		{name: "upper-only", upper: rationalInt32Pointer(5), left: 3, right: 3},
		{name: "lower-only", lower: rationalInt32Pointer(5), left: 7, right: 7},
		{name: "finite", lower: rationalInt32Pointer(2), upper: rationalInt32Pointer(6), left: 4, right: 4},
		{name: "integer-half", integer: true, lower: rationalInt32Pointer(1), upper: rationalInt32Pointer(4), left: 2, right: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, m, value, x, _ := rationalClauseFixture(t)
			c.variables[x] = tc.integer
			root, err := c.Root()
			if err != nil {
				t.Fatal(err)
			}
			for _, b := range []struct {
				value *int32
				lower bool
			}{{tc.lower, true}, {tc.upper, false}} {
				if b.value == nil {
					continue
				}
				if err := m.SetInt32(value, *b.value); err != nil {
					t.Fatal(err)
				}
				if err := c.PublishBound(root, x, value, b.lower, false); err != nil {
					t.Fatal(err)
				}
			}
			left, right, err := c.SplitMidpoint(root, x, true, 2)
			if err != nil {
				t.Fatal(err)
			}
			if left == nil || right == nil || left.parent != root || right.parent != root || left.upper[x] == nil || right.lower[x] == nil || !left.upper[x].axiom || !right.lower[x].axiom || left.upper[x].open == tc.integer || right.lower[x].open || left.upper[x].reasonDefinition != math.MaxUint32 || right.lower[x].reasonDefinition != math.MaxUint32 {
				t.Fatal("child decisions, openness or justification")
			}
			for _, check := range []struct {
				bound *RationalBound
				want  int32
			}{{left.upper[x], tc.left}, {right.lower[x], tc.right}} {
				if err := m.SetInt32(value, check.want); err != nil {
					t.Fatal(err)
				}
				cmp, err := m.Compare(&check.bound.value, value)
				if err != nil || cmp != 0 {
					t.Fatal("split midpoint", cmp, err)
				}
			}
			if len(c.queue) != 0 || root.children != 2 || c.leafHead != right || c.leafTail != root {
				t.Fatal("split child source order")
			}
		})
	}
}

func rationalInt32Pointer(v int32) *int32 { return &v }

func TestRationalMidpointSplitterSingletonRefusalLeavesChildren(t *testing.T) {
	c, m, value, x, _ := rationalClauseFixture(t)
	root, err := c.Root()
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(value, 3); err != nil {
		t.Fatal(err)
	}
	for _, lower := range []bool{true, false} {
		if err := c.PublishBound(root, x, value, lower, false); err != nil {
			t.Fatal(err)
		}
	}
	left, right, err := c.SplitMidpoint(root, x, true, 1)
	if !errors.Is(err, ErrSubpavingFailure) || left == nil || right == nil || root.children != 2 || left.trail != root.trail || right.trail != root.trail {
		t.Fatal("source creates children before midpoint refusal", err)
	}
	if _, _, err := c.SplitMidpoint(root, x, true, math.MaxInt32); !errors.Is(err, arena.ErrInvalidSlot) || root.children != 2 {
		t.Fatal("invalid delta created children", err)
	}
}
