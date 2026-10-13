package subpaving

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func fastInt32Pointer(v int32) *int32 { return &v }

func TestFastSelectorRoundRobinAndDefinitionFilter(t *testing.T) {
	m, c, v, root := fastMonomialFixture(t, []bool{false, false})
	definition, err := c.DefineMonomial([]FastPower{{Variable: 1, Degree: 2}})
	if err != nil {
		t.Fatal(err)
	}
	for _, lower := range []bool{true, false} {
		if err := m.SetInt32(v, 4); err != nil {
			t.Fatal(err)
		}
		if err := c.PropagateBound(root, 0, v, lower, false); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.PropagateQueued(root); err != nil {
		t.Fatal(err)
	}
	if x, found, err := c.SelectBranchVariable(root, true); err != nil || !found || x != 1 {
		t.Fatal("root selector", x, found, err)
	}
	child, err := c.NewChild(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := child.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := m.SetInt32(v, 5); err != nil {
		t.Fatal(err)
	}
	if err := c.DecideBound(child, 1, v, false, false); err != nil {
		t.Fatal(err)
	}
	if x, found, err := c.SelectBranchVariable(child, true); err != nil || !found || x != 1 {
		t.Fatal("child wrap", x, found, err)
	}
	if x, found, err := c.SelectBranchVariable(child, false); err != nil || !found || x != definition {
		t.Fatal("definition included", x, found, err)
	}
	undecided, err := c.NewChild(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := undecided.Close(); err != nil {
			t.Error(err)
		}
	})
	if _, _, err := c.SelectBranchVariable(undecided, true); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("missing child axiom", err)
	}
}

func TestFastMidpointSplitterSourceBranchesAndRefusal(t *testing.T) {
	for _, tc := range []struct {
		name                string
		integer             bool
		lower, upper        *int32
		wantLeft, wantRight int32
	}{
		{name: "unbounded", wantLeft: 0, wantRight: 0},
		{name: "upper only", upper: fastInt32Pointer(5), wantLeft: 3, wantRight: 3},
		{name: "lower only", lower: fastInt32Pointer(5), wantLeft: 7, wantRight: 7},
		{name: "finite", lower: fastInt32Pointer(2), upper: fastInt32Pointer(6), wantLeft: 4, wantRight: 4},
		{name: "integer finite", integer: true, lower: fastInt32Pointer(1), upper: fastInt32Pointer(4), wantLeft: 2, wantRight: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, c, v, root := fastMonomialFixture(t, []bool{tc.integer})
			for _, bound := range []struct {
				value *int32
				lower bool
			}{{tc.lower, true}, {tc.upper, false}} {
				if bound.value == nil {
					continue
				}
				if err := m.SetInt32(v, *bound.value); err != nil {
					t.Fatal(err)
				}
				if err := c.PropagateBound(root, 0, v, bound.lower, false); err != nil {
					t.Fatal(err)
				}
			}
			if err := c.PropagateQueued(root); err != nil {
				t.Fatal(err)
			}
			left, right, err := c.SplitMidpoint(root, 0, true, 2)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := right.Close(); err != nil {
					t.Error(err)
				}
				if err := left.Close(); err != nil {
					t.Error(err)
				}
			})
			for _, check := range []struct {
				got  *FastBound
				want int32
			}{{left.upper[0], tc.wantLeft}, {right.lower[0], tc.wantRight}} {
				if check.got == nil {
					t.Fatal("missing decision")
				}
				if err := m.SetInt32(v, check.want); err != nil {
					t.Fatal(err)
				}
				equal, err := m.Equal(check.got.value, v)
				if err != nil || !equal {
					t.Fatal("midpoint", check.want, equal, err)
				}
			}
			if left.upper[0].open == tc.integer || right.lower[0].open || left.upper[0].reason.kind != fastReasonAxiom || right.lower[0].reason.kind != fastReasonAxiom || root.children != 2 || len(c.queue) != 0 {
				t.Fatal("source child decisions")
			}
		})
	}
	t.Run("adjacent values retain children", func(t *testing.T) {
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
		left, right, err := c.SplitMidpoint(root, 0, true, 1)
		if !errors.Is(err, ErrFastSplitNoInterior) || left == nil || right == nil || root.children != 2 || len(c.nodes) != 3 {
			t.Fatal("interior refusal after children", left, right, err)
		}
		t.Cleanup(func() {
			if err := right.Close(); err != nil {
				t.Error(err)
			}
			if err := left.Close(); err != nil {
				t.Error(err)
			}
		})
	})
}
