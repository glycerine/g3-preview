package polynomial

import (
	"errors"
	"strconv"
	"testing"
)

func algebraicFilterFixture(t *testing.T) (*AlgebraicRootCore, *MultivariatePolynomialManager, Polynomial, *AlgebraicVector, *algebraicSignTestValues, *algebraicTestControl, UPolynomial) {
	t.Helper()
	c, up, control := algebraicControlledFixture(t)
	c.minMagnitude = 0
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {-1, [3]uint32{0, 1, 0}}})
	a, b := algebraicResultantInputs(t, c, up)
	serviceMust(t, c.Del(&b))
	values := &algebraicSignTestValues{}
	values.values[1] = a
	candidates, e := c.NewVector()
	serviceMust(t, e)
	// All four roots of (x^2-2)*(x^2-3) are candidate roots for
	// x-sqrt(2), but only its positive sqrt(2) root survives filtering.
	poly := algebraicPolynomial(t, up, 6, 0, -5, 0, 1)
	view, e := up.wholeView(poly)
	serviceMust(t, e)
	serviceMust(t, c.IsolateRoots(view, candidates))
	if candidates.Len() != 4 {
		t.Fatal("candidate roots", candidates.Len())
	}
	t.Cleanup(func() {
		serviceMust(t, c.CleanupRetained())
		serviceMust(t, candidates.Close())
		serviceMust(t, c.Del(&values.values[1]))
	})
	return c, pm, p, candidates, values, control, poly
}

func TestAlgebraicFilterRootsConjugatesAndStableOrder(t *testing.T) {
	c, pm, p, candidates, values, _, _ := algebraicFilterFixture(t)
	serviceMust(t, c.filterRoots(pm, p, values, 0, candidates))
	if candidates.Len() != 1 {
		t.Fatal("spurious conjugates retained", candidates.Len())
	}
	root, e := candidates.Get(0)
	serviceMust(t, e)
	cmp, e := c.Compare(&root, &values.values[1])
	serviceMust(t, e)
	if cmp != 0 {
		t.Fatal("wrong retained root", cmp)
	}
	// x^2-2 retains both conjugates, in their original increasing order.
	serviceMust(t, candidates.Reset())
	view, e := c.config.Polynomials.wholeView(algebraicPolynomial(t, c.config.Polynomials, 6, 0, -5, 0, 1))
	serviceMust(t, e)
	serviceMust(t, c.IsolateRoots(view, candidates))
	p2 := polynomialQueryTerms(t, pm.core, []int32{1, -2}, [][]MonomialPower{{{0, 2}}, nil})
	serviceMust(t, c.filterRoots(pm, p2, values, 0, candidates))
	if candidates.Len() != 2 {
		t.Fatal("quadratic roots", candidates.Len())
	}
	negative, e := candidates.Get(0)
	serviceMust(t, e)
	positive, e := candidates.Get(1)
	serviceMust(t, e)
	algebraicExpectBetween(t, c, &negative, -2, -1)
	algebraicExpectBetween(t, c, &positive, 1, 2)
}

func TestAlgebraicFilterRootsConstantsAndOverrideShortCircuit(t *testing.T) {
	c, pm, _, candidates, values, _, _ := algebraicFilterFixture(t)
	marker := errors.New("unused assignment must not be read")
	values.failure = marker
	serviceMust(t, c.filterRoots(pm, pm.core.zero, values, 0, candidates))
	if candidates.Len() != 4 || len(values.reads) != 0 {
		t.Fatal("zero polynomial changed candidates or touched mapping")
	}
	extended := algebraicExtendedAssignment{base: values, x: 0, value: &candidates.values[0]}
	has, e := extended.Contains(0)
	serviceMust(t, e)
	if !has {
		t.Fatal("override not present")
	}
	actual, e := extended.Algebraic(0)
	serviceMust(t, e)
	if actual != &candidates.values[0] || len(values.reads) != 0 {
		t.Fatal("override copied or delegated")
	}
	if _, e = extended.Algebraic(1); !errors.Is(e, marker) {
		t.Fatal("unmatched lookup did not delegate", e)
	}
	serviceMust(t, c.filterRoots(pm, pm.core.unit, values, 0, candidates))
	if candidates.Len() != 0 {
		t.Fatal("constant polynomial retained roots")
	}
}

func TestAlgebraicFilterRootsEveryCheckpointCancellationPrefix(t *testing.T) {
	success := false
	for allowance := uint64(0); allowance < 512 && !success; allowance++ {
		t.Run(strconv.FormatUint(allowance, 10), func(t *testing.T) {
			c, pm, p, candidates, values, control, poly := algebraicFilterFixture(t)
			control.limit = control.visits + allowance
			body := c.filterRoots(pm, p, values, 0, candidates)
			control.limit = 1000000
			if body == nil {
				success = true
			} else {
				if !errors.Is(body, ErrUPolynomialCanceled) {
					t.Fatal(body)
				}
				if candidates.Len() != 4 {
					t.Fatal("unreached shrink executed after cancellation", candidates.Len())
				}
				serviceMust(t, c.CleanupRetained())
				// Compaction mutates the caller's prefix before an interruption.
				// Rebuild that input before retrying; source does not roll it back.
				serviceMust(t, candidates.Reset())
				view, e := c.config.Polynomials.wholeView(poly)
				serviceMust(t, e)
				serviceMust(t, c.IsolateRoots(view, candidates))
				serviceMust(t, c.filterRoots(pm, p, values, 0, candidates))
			}
			if candidates.Len() != 1 {
				t.Fatal("filter result after retry", candidates.Len())
			}
		})
	}
	if !success {
		t.Fatal("filter never reached success")
	}
}
