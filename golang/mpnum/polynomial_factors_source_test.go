// Source: src/math/polynomial/polynomial.cpp:1613-1683,6577-6590;
// src/math/polynomial/polynomial.h:120-190.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"testing"
)

// These are literal class/action tests and independent finite models; the
// registered polynomial driver returns before the dormant factor test.
func TestPolynomialFactorsSourceControlAndOwnership(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {1, [3]uint32{}}})
	f, e := NewPolynomialFactors(m)
	serviceMust(t, e)
	defer func() { serviceMust(t, f.CloseChecked()) }()
	c, e := f.Constant()
	serviceMust(t, e)
	if c.val != 1 || c.cell != 0 {
		t.Fatal("source ctor sets one", *c)
	}
	serviceMust(t, f.Push(p, 2))
	serviceMust(t, f.Push(p, 3))
	n, e := f.Distinct()
	serviceMust(t, e)
	total, e := f.Total()
	serviceMust(t, e)
	if n != 2 || total != 5 {
		t.Fatal("literal published counts", n, total)
	}
	r, e := f.At(0)
	serviceMust(t, e)
	serviceMust(t, f.Reset())
	got, e := r.Get()
	serviceMust(t, e)
	if got != p {
		t.Fatal("operator[] returned a genuine owning reference", got, p)
	}
	serviceMust(t, r.Close())
	n, e = f.Distinct()
	serviceMust(t, e)
	total, e = f.Total()
	serviceMust(t, e)
	if n != 0 || total != 0 {
		t.Fatal("reset clears both vectors and total", n, total)
	}
	c, e = f.Constant()
	serviceMust(t, e)
	if c.val != 1 || c.cell != 0 {
		t.Fatal("reset resets constant", *c)
	}
}
func TestPolynomialFactorsSourceDegreeQuirkAndWrap(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, nil)
	f, e := NewPolynomialFactors(m)
	serviceMust(t, e)
	defer func() { serviceMust(t, f.CloseChecked()) }()
	serviceMust(t, f.Push(p, ^uint32(0)))
	serviceMust(t, f.Push(p, 2))
	n, e := f.Total()
	serviceMust(t, e)
	if n != 1 {
		t.Fatal("source unsigned wrap", n)
	}
	e = f.SetDegree(0, 4)
	if modularSourceDebug {
		if !errors.Is(e, ErrPolynomialAssertion) {
			t.Fatal("literal index>0 assertion", e)
		}
	} else {
		serviceMust(t, e)
	}
	serviceMust(t, f.SetDegree(1, 0))
	d, e := f.Degree(1)
	serviceMust(t, e)
	if d != 0 {
		t.Fatal("source setter accepts zero degree", d)
	}
}
func TestPolynomialFactorsMultiplyFiniteAndAlias(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {1, [3]uint32{}}})
	f, e := NewPolynomialFactors(m)
	serviceMust(t, e)
	defer func() { serviceMust(t, f.CloseChecked()) }()
	serviceMust(t, f.Push(p, 2))
	serviceMust(t, f.Push(p, 1))
	c := NewInt(-2)
	serviceMust(t, f.SetConstant(&c))
	r, e := NewPolynomialRef(m.core)
	serviceMust(t, e)
	defer func() { serviceMust(t, r.Close()) }()
	serviceMust(t, r.Assign(p))
	serviceMust(t, f.Multiply(r))
	result, e := r.Get()
	serviceMust(t, e)
	// Independent expansion -2(x+1)^3, evaluated at the frozen scalar x=2.
	v := NewInt(2)
	out := Int{}
	defer func() { serviceMust(t, m.core.integer.Del(&out)) }()
	serviceMust(t, m.UnivariateEvaluate(result, 0, &v, &out))
	if out.cell != 0 || out.val != -54 {
		t.Fatal("finite independent product", out)
	}
	serviceMust(t, f.Reset())
	serviceMust(t, f.SetConstant(&c))
	serviceMust(t, f.Multiply(r))
	result, e = r.Get()
	serviceMust(t, e)
	serviceMust(t, m.UnivariateEvaluate(result, 0, &v, &out))
	if out.cell != 0 || out.val != -2 {
		t.Fatal("empty product is source constant", out)
	}
}
func TestPolynomialFactorsLiteralDisplayAndConstantHelpers(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {1, [3]uint32{}}})
	f, e := NewPolynomialFactors(m)
	serviceMust(t, e)
	defer func() { serviceMust(t, f.CloseChecked()) }()
	serviceMust(t, f.Push(p, 3))
	c := NewInt(-2)
	serviceMust(t, f.SetConstant(&c))
	sink := &polynomialDisplayTestSink{}
	serviceMust(t, f.DisplayInto(sink))
	if sink.text.String() != "-2 * (x0 + 1)^3" {
		t.Fatal("source output sequence", sink.text.String())
	}
	c = NewInt(4)
	serviceMust(t, m.AccumulateFactorConstant(f, &c))
	serviceMust(t, m.FlipFactorSign(f))
	a, e := f.Constant()
	serviceMust(t, e)
	if a.val != 8 || a.cell != 0 {
		t.Fatal("literal scoped helper engines", *a)
	}
}
func TestPolynomialFactorsDisplayReachesSizeAfterCallback(t *testing.T) {
	m, _, _ := polynomialEvalFixture(t, nil)
	f, e := NewPolynomialFactors(m)
	serviceMust(t, e)
	sink := &polynomialDisplayTestSink{}
	called := false
	sink.literalHook = func(s string) error {
		if !called {
			called = true
			return f.CloseChecked()
		}
		return nil
	}
	e = f.DisplayInto(sink)
	if e == nil || !called || !f.closed {
		t.Fatal("source reaches class SIZE after successful constant insertion destroyed its owner")
	}
	serviceMust(t, m.CleanupDisplay())
}

func TestPolynomialFactorsModularAndZeroDegreeSource(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {1, [3]uint32{}}})
	serviceMust(t, m.core.modular.SetZPUint64(7))
	f, e := NewPolynomialFactors(m)
	serviceMust(t, e)
	defer func() { serviceMust(t, f.CloseChecked()) }()
	serviceMust(t, f.Push(p, 2))
	serviceMust(t, f.Push(p, 1))
	// Source SetDegree permits zero but Multiply tests only d>1. Thus this
	// zero exponent is intentionally still one copy, not mathematical pow(0).
	serviceMust(t, f.SetDegree(1, 0))
	c := NewInt(5)
	// The source modular precondition requires actual balanced ring input.
	serviceMust(t, m.core.modular.PNormalize(&c))
	serviceMust(t, f.SetConstant(&c))
	serviceMust(t, m.AccumulateFactorConstant(f, &c)) // 5*5 has balanced representative -3 mod 7
	serviceMust(t, m.FlipFactorSign(f))               // negation of balanced -3 is 3
	r, e := NewPolynomialRef(m.core)
	serviceMust(t, e)
	defer func() { serviceMust(t, r.Close()) }()
	serviceMust(t, f.Multiply(r))
	result, e := r.Get()
	serviceMust(t, e)
	x := NewInt(2)
	out := Int{}
	serviceMust(t, m.UnivariateEvaluate(result, 0, &x, &out))
	if out.cell != 0 || out.val != -3 {
		t.Fatal("literal balanced GF7 3*(2+1)^3", out)
	}
	serviceMust(t, m.core.integer.Del(&out))
}
