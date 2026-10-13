package subpaving

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestArbitraryMidpointSplitterSourceBranches(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		integer                 bool
		lower, upper            *int32
		wantLeft, wantRight     float64
		wantLeftOpen, rightOpen bool
	}{
		{name: "upper only", upper: int32Pointer(5), wantLeft: 3, wantRight: 3, wantLeftOpen: true},
		{name: "lower only", lower: int32Pointer(5), wantLeft: 7, wantRight: 7, wantLeftOpen: true},
		{name: "finite", lower: int32Pointer(2), upper: int32Pointer(6), wantLeft: 4, wantRight: 4, wantLeftOpen: true},
		{name: "integer finite", integer: true, lower: int32Pointer(1), upper: int32Pointer(4), wantLeft: 2, wantRight: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newArbitraryClauseFixtureWithPolicy(t, 0, tc.integer)
			c := f.context
			variable := f.x
			if tc.integer {
				variable = f.y
			}
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
				var v numeral.ArbitraryFloat
				if err := f.numbers.SetInt32(&v, *b.value); err != nil {
					t.Fatal(err)
				}
				if err := c.PublishBound(root, variable, &v, b.lower, false, true); err != nil {
					t.Fatal(err)
				}
				if err := f.numbers.Del(&v); err != nil {
					t.Fatal(err)
				}
			}
			if err := c.PropagateQueued(root); err != nil {
				t.Fatal(err)
			}
			left, right, err := c.SplitMidpoint(root, variable, true, 2)
			if err != nil {
				t.Fatal(err)
			}
			core, err := f.numbers.BorrowCore()
			if err != nil {
				t.Fatal(err)
			}
			lo, err := core.ToDouble(&left.upper[variable].value)
			if err != nil {
				t.Fatal(err)
			}
			hi, err := core.ToDouble(&right.lower[variable].value)
			if err != nil {
				t.Fatal(err)
			}
			if lo != tc.wantLeft || hi != tc.wantRight || left.upper[variable].open != tc.wantLeftOpen ||
				right.lower[variable].open != tc.rightOpen || !left.upper[variable].axiom || !right.lower[variable].axiom ||
				left.parent != root || right.parent != root || len(c.queue) != 0 {
				t.Fatal("source midpoint decisions", lo, hi)
			}
		})
	}
	t.Run("unbounded arithmetic refusal after children", func(t *testing.T) {
		f := newArbitraryClauseFixture(t)
		root, err := f.context.Root()
		if err != nil {
			t.Fatal(err)
		}
		left, right, err := f.context.SplitMidpoint(root, f.x, true, 1)
		if !errors.Is(err, numeral.ErrArbitraryFloatNonRegular) || left == nil || right == nil || root.children != 2 ||
			left.trail != root.trail || right.trail != root.trail {
			t.Fatal("source creates children before f2n refuses formatted zero midpoint", err)
		}
	})
	t.Run("adjacent formatted values", func(t *testing.T) {
		f := newArbitraryClauseFixture(t)
		c := f.context
		root, err := c.Root()
		if err != nil {
			t.Fatal(err)
		}
		for _, b := range []struct {
			value float64
			lower bool
		}{{1, true}, {math.Nextafter(1, 2), false}} {
			var v numeral.ArbitraryFloat
			if err := f.numbers.SetDouble(&v, b.value); err != nil {
				t.Fatal(err)
			}
			if err := c.PublishBound(root, f.x, &v, b.lower, false, true); err != nil {
				t.Fatal(err)
			}
			if err := f.numbers.Del(&v); err != nil {
				t.Fatal(err)
			}
		}
		if err := c.PropagateQueued(root); err != nil {
			t.Fatal(err)
		}
		left, right, err := c.SplitMidpoint(root, f.x, true, 1)
		if !errors.Is(err, ErrArbitrarySplitNoInterior) || !errors.Is(err, ErrSubpavingFailure) || left == nil || right == nil || root.children != 2 ||
			left.trail != root.trail || right.trail != root.trail {
			t.Fatal("source creates children before refusing non-interior midpoint", err)
		}
	})
}

func int32Pointer(value int32) *int32 { return &value }
