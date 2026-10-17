package polynomial

import (
	"errors"
	"strconv"
	"testing"
)

func TestAlgebraicRootConstructionEveryReachedCheckpointAndRetry(t *testing.T) {
	for _, expression := range []bool{false, true} {
		t.Run(strconv.FormatBool(expression), func(t *testing.T) {
			succeeded := false
			for allowance := uint64(0); allowance < 1024 && !succeeded; allowance++ {
				passed := t.Run(strconv.FormatUint(allowance, 10), func(t *testing.T) {
					c, _, sx, sy, q, control := algebraicSexprRootFixture(t)
					pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{0, 0, 2}}, {-2, [3]uint32{}}})
					x := sxPolySym(t, sx, sy, "x")
					two := sxPolyNum(t, sx, q, "2")
					expr := sxPolyApp(t, sx, sy, "-", sxPolyApp(t, sx, sy, "^", x, two), two)
					var out Algebraic
					defer func() { serviceMust(t, c.Del(&out)) }()
					control.limit = control.visits + allowance
					var body error
					if expression {
						body = c.MkSexprRoot(sx, expr, 2, &out)
					} else {
						body = c.MkPolynomialRoot(pm, p, 2, &out)
					}
					succeeded = body == nil
					if body != nil && !errors.Is(body, ErrUPolynomialCanceled) {
						t.Fatal("unexpected cancellation result", body)
					}
					control.limit = 1000000
					serviceMust(t, c.CleanupRetained())
					if c.rootSelection != nil || c.rootConstruction.active || pm.algebraicIsolationBorrowers != 0 {
						t.Fatal("stopped construction retained locals")
					}
					if expression {
						serviceMust(t, c.MkSexprRoot(sx, expr, 2, &out))
					} else {
						serviceMust(t, c.MkPolynomialRoot(pm, p, 2, &out))
					}
					algebraicExpectBetween(t, c, &out, 1, 2)
				})
				if !passed {
					return
				}
			}
			if !succeeded {
				t.Fatal("did not reach successful checkpoint endpoint")
			}
		})
	}
}
