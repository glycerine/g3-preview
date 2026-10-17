package polynomial

import (
	"errors"
	"strconv"
	"testing"
)

func TestAlgebraicUnaryResultantIndependentIdentities(t *testing.T) {
	c, p, _ := algebraicControlledFixture(t)
	a, b := algebraicResultantInputs(t, c, p)
	out, e := p.NewPolynomial()
	serviceMust(t, e)
	defer func() { serviceMust(t, p.DelPolynomial(&out)); serviceMust(t, c.Del(&b)); serviceMust(t, c.Del(&a)) }()
	for _, tc := range []struct {
		k            uint32
		root         bool
		coefficients []int32
	}{
		{2, true, []int32{-2, 0, 0, 0, 1}},
		{3, true, []int32{-2, 0, 0, 0, 0, 0, 1}},
		{2, false, []int32{4, -4, 1}},
		{3, false, []int32{-8, 0, 1}},
		{4, false, []int32{16, -8, 1}},
	} {
		serviceMust(t, c.unaryResultPolynomial(&a, tc.k, out, tc.root))
		checkAlgebraicResultantPolynomial(t, p, out, tc.coefficients...)
		if c.resultant.active {
			t.Fatal("references survived successful functor")
		}
	}
	// Odd degree and a negative root distinguish the subtraction orientation
	// and show the repeated factor when all three conjugates cube to -3.
	negative := algebraicCompareRoot(t, c, []int32{3, 0, 0, 1}, -2, 0, -1, 0, true)
	serviceMust(t, c.unaryResultPolynomial(&negative, 2, out, true))
	checkAlgebraicResultantPolynomial(t, p, out, -3, 0, 0, 0, 0, 0, -1)
	serviceMust(t, c.unaryResultPolynomial(&negative, 2, out, false))
	checkAlgebraicResultantPolynomial(t, p, out, -9, 0, 0, 1)
	serviceMust(t, c.unaryResultPolynomial(&negative, 3, out, false))
	checkAlgebraicResultantPolynomial(t, p, out, -27, -27, -9, -1)
	serviceMust(t, c.Del(&negative))
}

func TestAlgebraicUnaryResultantEveryCheckpointRetry(t *testing.T) {
	for _, root := range []bool{true, false} {
		success := false
		for allowance := uint64(0); allowance < 256 && !success; allowance++ {
			t.Run(strconv.FormatBool(root)+"/"+strconv.FormatUint(allowance, 10), func(t *testing.T) {
				c, p, control := algebraicControlledFixture(t)
				a, b := algebraicResultantInputs(t, c, p)
				out, e := p.NewPolynomial()
				serviceMust(t, e)
				defer func() { serviceMust(t, p.DelPolynomial(&out)); serviceMust(t, c.Del(&b)); serviceMust(t, c.Del(&a)) }()
				serviceMust(t, p.SetInts(out, nil))
				control.limit = control.visits + allowance
				body := c.unaryResultPolynomial(&a, 3, out, root)
				control.limit = 1000000
				if body == nil {
					success = true
				} else {
					if !errors.Is(body, ErrUPolynomialCanceled) {
						t.Fatal(body)
					}
					if !errors.Is(c.unaryResultPolynomial(&a, 3, out, root), ErrAlgebraicPending) {
						t.Fatal("pending functor overwritten")
					}
					serviceMust(t, c.CleanupRetained())
					serviceMust(t, c.unaryResultPolynomial(&a, 3, out, root))
				}
				if root {
					checkAlgebraicResultantPolynomial(t, p, out, -2, 0, 0, 0, 0, 0, 1)
				} else {
					checkAlgebraicResultantPolynomial(t, p, out, -8, 0, 1)
				}
			})
		}
		if !success {
			t.Fatal("unary resultant never succeeded")
		}
	}
}
