package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestAlgebraicSelectAllOperandShapesAndAliases(t *testing.T) {
	for _, shape := range []int{0, 1, 2, 3} {
		for _, alias := range []int{0, 1, 2} {
			c, p, _ := algebraicControlledFixture(t)
			a, b := algebraicResultantInputs(t, c, p)
			q := unumeral.NewRational()
			if shape&1 != 0 {
				serviceMust(t, c.config.Rationals.SetInt32(&q, 1))
				serviceMust(t, c.SetRational(&a, &q))
			}
			if shape&2 != 0 {
				serviceMust(t, c.config.Rationals.SetInt32(&q, 2))
				serviceMust(t, c.SetRational(&b, &q))
			}
			var originalA, originalB, result Algebraic
			serviceMust(t, c.Copy(&originalA, &a))
			serviceMust(t, c.Copy(&originalB, &b))
			out := &result
			if alias == 1 {
				out = &a
			}
			if alias == 2 {
				out = &b
			}
			serviceMust(t, c.Select(&a, &b, out))
			lower, e := c.Compare(&originalA, out)
			serviceMust(t, e)
			upper, e := c.Compare(out, &originalB)
			serviceMust(t, e)
			if lower >= 0 || upper >= 0 {
				t.Fatal("selected point outside strict bounds", shape, alias, lower, upper)
			}
			l, e := c.location(out)
			serviceMust(t, e)
			if l != nil && l.kind != 0 {
				t.Fatal("sample is not rational")
			}
			if c.selection.active {
				t.Fatal("selection locals survived")
			}
			serviceMust(t, c.Del(&result))
			serviceMust(t, c.Del(&originalB))
			serviceMust(t, c.Del(&originalA))
			serviceMust(t, c.Del(&b))
			serviceMust(t, c.Del(&a))
			serviceMust(t, c.config.Rationals.Del(&q))
		}
	}
}

func TestAlgebraicSelectSeparationDiscoversRationalAndZero(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	// Initially overlapping intervals. Refining the first input finds exactly 1.
	a := algebraicCompareRoot(t, c, []int32{-1, 0, 1}, 0, 0, 2, 0, false)
	b := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 5, 2, 2, 0, true)
	var out Algebraic
	serviceMust(t, c.Select(&a, &b, &out))
	algebraicExpectBetween(t, c, &out, 1, 2)
	l, e := c.location(&a)
	serviceMust(t, e)
	if l == nil || l.kind != 0 {
		t.Fatal("first input did not become basic")
	}
	serviceMust(t, c.Del(&out))
	serviceMust(t, c.Del(&b))
	serviceMust(t, c.Del(&a))
	// Null cells participate as basic zero, including an aliased destination.
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetInt32(&q, 1))
	serviceMust(t, c.SetRational(&b, &q))
	serviceMust(t, c.Select(&a, &b, &a))
	algebraicExpectBetween(t, c, &a, 0, 1)
	serviceMust(t, c.Del(&a))
	serviceMust(t, c.Del(&b))
	serviceMust(t, c.config.Rationals.Del(&q))
}

func TestAlgebraicSelectOutsideIntegers(t *testing.T) {
	c, p, _ := algebraicControlledFixture(t)
	a, b := algebraicResultantInputs(t, c, p)
	q := unumeral.NewRational()
	var out Algebraic
	defer func() {
		serviceMust(t, c.Del(&out))
		serviceMust(t, c.Del(&b))
		serviceMust(t, c.Del(&a))
		serviceMust(t, c.config.Rationals.Del(&q))
	}()
	for _, tc := range []struct {
		basic        bool
		n            int64
		d            uint64
		lower, upper int32
	}{
		{false, 0, 0, 1, 2}, {true, 3, 2, 0, 3}, {true, -3, 2, -3, 0}, {true, 2, 1, 1, 3}, {true, 0, 1, -1, 1},
	} {
		if tc.basic {
			serviceMust(t, c.config.Rationals.SetFraction(&q, tc.n, tc.d))
			serviceMust(t, c.SetRational(&a, &q))
		}
		for _, upper := range []bool{false, true} {
			want := tc.lower
			if upper {
				want = tc.upper
				serviceMust(t, c.IntGT(&a, &out))
			} else {
				serviceMust(t, c.IntLT(&a, &out))
			}
			serviceMust(t, c.config.Rationals.SetInt32(&q, want))
			cmp, e := c.CompareRational(&out, &q)
			serviceMust(t, e)
			if cmp != 0 {
				t.Fatal("outside integer", tc, upper, cmp)
			}
		}
	}
	serviceMust(t, c.IntGT(&a, &a))
	algebraicExpectBetween(t, c, &a, 0, 2)
}

func TestAlgebraicSelectSourceAssertionsBeforeLocals(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions erased")
	}
	c, _, _ := algebraicControlledFixture(t)
	polynomialAssertionActionsFixture(t, "TT")
	var zero, out Algebraic
	var exception z3exception.DefaultException
	if e := c.Select(&zero, &zero, &out); !errors.As(e, &exception) || c.selection.active {
		t.Fatal("selection precondition after local construction", e)
	}
	if e := c.separate(&zero, &zero); !errors.As(e, &exception) || c.selection.active {
		t.Fatal("separation precondition after local construction", e)
	}
}

func TestAlgebraicSelectExactRefinementTouchingBoundGuard(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	// Erase the source ordering assertions: their Compare calls may refine
	// inputs before separate starts, changing this reached release path.
	if polynomialSourceDebug {
		d := diagnostic.BorrowProcessDebugContext()
		enabled, e := d.AssertionsEnabled()
		serviceMust(t, e)
		serviceMust(t, d.EnableAssertions(false))
		defer func() { serviceMust(t, d.EnableAssertions(enabled)) }()
	}
	a := algebraicCompareRoot(t, c, []int32{-1, 0, 1}, 0, 0, 2, 0, false)
	b := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	var out Algebraic
	body := c.Select(&a, &b, &out)
	if !errors.Is(body, unumeral.ErrBinaryInterval) || out != (Algebraic{}) || c.selection.active {
		t.Fatal("touching source bound guard", body)
	}
	l, e := c.location(&a)
	serviceMust(t, e)
	if l == nil || l.kind != 0 {
		t.Fatal("exact refinement branch not reached")
	}
	serviceMust(t, c.Del(&b))
	serviceMust(t, c.Del(&a))
}
