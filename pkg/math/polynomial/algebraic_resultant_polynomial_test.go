package polynomial

import (
	"errors"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func algebraicResultantInputs(t *testing.T, c *AlgebraicRootCore, p *UPolynomialManager) (Algebraic, Algebraic) {
	t.Helper()
	var roots [2]Algebraic
	for i, n := range []int32{2, 3} {
		poly := algebraicPolynomial(t, p, -n, 0, 1)
		v, e := p.wholeView(poly)
		serviceMust(t, e)
		var lower, upper unumeral.BinaryRational
		serviceMust(t, c.config.Binary.SetInt32(&lower, 1))
		serviceMust(t, c.config.Binary.SetInt32(&upper, 2))
		serviceMust(t, c.SetRoot(&roots[i], v, &lower, &upper, true))
		serviceMust(t, c.config.Binary.Del(&upper))
		serviceMust(t, c.config.Binary.Del(&lower))
	}
	return roots[0], roots[1]
}

func checkAlgebraicResultantPolynomial(t *testing.T, p *UPolynomialManager, out UPolynomial, coefficients ...int32) {
	t.Helper()
	v, e := p.wholeView(out)
	serviceMust(t, e)
	if v.count != uint32(len(coefficients)) {
		t.Fatal("resultant degree", v.count, coefficients)
	}
	for i, expected := range coefficients {
		a, e := p.read(v, uint32(i))
		serviceMust(t, e)
		if a.NativeScalar() != expected {
			t.Fatal("resultant coefficient", i, a.NativeScalar(), expected)
		}
	}
}

func TestAlgebraicResultantSumDifferenceAndProductPolynomials(t *testing.T) {
	c, p := algebraicFixture(t)
	a, b := algebraicResultantInputs(t, c, p)
	out, e := p.NewPolynomial()
	serviceMust(t, e)
	defer func() { serviceMust(t, p.DelPolynomial(&out)); serviceMust(t, c.Del(&b)); serviceMust(t, c.Del(&a)) }()
	// The sum and difference of sqrt(2),sqrt(3) satisfy (x^2-5)^2-24.
	serviceMust(t, c.addResultPolynomial(&a, &b, out, false))
	checkAlgebraicResultantPolynomial(t, p, out, 1, 0, -10, 0, 1)
	owner := c.multivariate
	serviceMust(t, c.addResultPolynomial(&a, &b, out, true))
	checkAlgebraicResultantPolynomial(t, p, out, 1, 0, -10, 0, 1)
	// Both conjugate pairs yield each product: the resultant is (x^2-6)^2.
	serviceMust(t, c.mulResultPolynomial(&a, &b, out))
	checkAlgebraicResultantPolynomial(t, p, out, 36, 0, -12, 0, 1)
	serviceMust(t, c.addResultPolynomial(&a, &a, out, false))
	checkAlgebraicResultantPolynomial(t, p, out, 0, 0, -8, 0, 1)
	// An odd-degree second polynomial distinguishes x-y from x+y; the even
	// quadratic pair above intentionally has the same resultant for both.
	cubic := algebraicPolynomial(t, p, -3, 0, 0, 1)
	cubicView, e := p.wholeView(cubic)
	serviceMust(t, e)
	var lower, upper unumeral.BinaryRational
	serviceMust(t, c.config.Binary.SetInt32(&lower, 1))
	serviceMust(t, c.config.Binary.SetInt32(&upper, 2))
	serviceMust(t, c.SetRoot(&b, cubicView, &lower, &upper, true))
	serviceMust(t, c.config.Binary.Del(&upper))
	serviceMust(t, c.config.Binary.Del(&lower))
	serviceMust(t, c.addResultPolynomial(&a, &b, out, false))
	checkAlgebraicResultantPolynomial(t, p, out, 1, -36, 12, -6, -6, 0, 1)
	serviceMust(t, c.addResultPolynomial(&a, &b, out, true))
	checkAlgebraicResultantPolynomial(t, p, out, 1, 36, 12, 6, -6, 0, 1)
	serviceMust(t, c.mulResultPolynomial(&a, &b, out))
	checkAlgebraicResultantPolynomial(t, p, out, -72, 0, 0, 0, 0, 0, 1)
	if c.multivariate != owner || c.resultantVariableCount != 2 || c.resultant.active {
		t.Fatal("source owner/variables were replaced or references remained active")
	}
	if owner.factorNumeralVector.outputManager != p || len(p.bindings) != 0 {
		t.Fatal("conversion did not use the actual algebraic dense owner")
	}
	for i := range c.resultant.refs {
		if c.resultant.refs[i].root.value != (Polynomial{}) {
			t.Fatal("source local reference survived its scope", i)
		}
	}
}

func TestAlgebraicResultantPolynomialCancellationCleanupAndRetry(t *testing.T) {
	c, p, control := algebraicControlledFixture(t)
	a, b := algebraicResultantInputs(t, c, p)
	out, e := p.NewPolynomial()
	serviceMust(t, e)
	defer func() { serviceMust(t, p.DelPolynomial(&out)); serviceMust(t, c.Del(&b)); serviceMust(t, c.Del(&a)) }()
	serviceMust(t, c.mulResultPolynomial(&a, &b, out))
	control.limit = control.visits
	body := c.addResultPolynomial(&a, &b, out, false)
	control.limit = 1000000
	if !errors.Is(body, ErrUPolynomialCanceled) {
		t.Fatal("resultant cancellation", body)
	}
	checkAlgebraicResultantPolynomial(t, p, out, 36, 0, -12, 0, 1)
	if !errors.Is(c.addResultPolynomial(&a, &b, out, false), ErrAlgebraicPending) {
		t.Fatal("failed scope was overwritten before owning cleanup")
	}
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, c.addResultPolynomial(&a, &b, out, false))
	checkAlgebraicResultantPolynomial(t, p, out, 1, 0, -10, 0, 1)
}
