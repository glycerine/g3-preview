package polynomial

import (
	"errors"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestAlgebraicIsolatePolynomialUnivariateAndReceivingOwner(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{7, [3]uint32{0, 0, 2}}, {-14, [3]uint32{}}})
	roots, e := c.NewVector()
	serviceMust(t, e)
	defer func() { serviceMust(t, roots.Close()) }()
	serviceMust(t, c.IsolatePolynomialRoots(pm, p, roots))
	if roots.Len() != 2 {
		t.Fatal("quadratic root count", roots.Len())
	}
	negative, e := roots.Get(0)
	serviceMust(t, e)
	positive, e := roots.Get(1)
	serviceMust(t, e)
	algebraicExpectBetween(t, c, &negative, -2, -1)
	algebraicExpectBetween(t, c, &positive, 1, 2)
	if pm.factorNumeralVector.outputManager != c.config.Polynomials {
		t.Fatal("dense conversion used an intermediate owner")
	}
	checkAlgebraicResultantPolynomial(t, c.config.Polynomials, c.isolateTmp1, -14, 0, 7)
	if c.isolationConversion.active {
		t.Fatal("successful conversion stayed pending")
	}
}

func TestAlgebraicIsolatePolynomialZeroPreservesOutput(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	pm, _, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{}}})
	roots, e := c.NewVector()
	serviceMust(t, e)
	defer func() { serviceMust(t, roots.Close()) }()
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetInt32(&q, 42))
	var a Algebraic
	serviceMust(t, c.SetRational(&a, &q))
	serviceMust(t, roots.pushTake(&a))
	serviceMust(t, c.IsolatePolynomialRoots(pm, pm.core.zero, roots))
	if roots.Len() != 1 || c.isolateTmp1.slot != 0 {
		t.Fatal("zero reached conversion or changed output")
	}
	got, e := roots.Get(0)
	serviceMust(t, e)
	cmp, e := c.CompareRational(&got, &q)
	serviceMust(t, e)
	if cmp != 0 {
		t.Fatal("zero changed prior root")
	}
	serviceMust(t, c.IsolatePolynomialRoots(pm, pm.core.unit, roots))
	if roots.Len() != 1 {
		t.Fatal("nonzero constant added roots")
	}
	serviceMust(t, c.config.Rationals.Del(&q))
}

func TestAlgebraicIsolatePolynomialUnivariateAssertionBeforeWorkspace(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions erased")
	}
	c, _, _ := algebraicControlledFixture(t)
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{1, 1, 0}}})
	roots, e := c.NewVector()
	serviceMust(t, e)
	defer func() { serviceMust(t, roots.Close()) }()
	polynomialAssertionActionsFixture(t, "T")
	e = c.IsolatePolynomialRoots(pm, p, roots)
	var exception z3exception.DefaultException
	if !errors.As(e, &exception) || c.isolateTmp1.slot != 0 || c.isolationConversion.active {
		t.Fatal("univariate assertion after workspace construction", e)
	}
}
