// Source: src/math/polynomial/polynomial.cpp:2558-2633. Licensed under the MIT license.
package mpnum

import "testing"

func TestPolynomialRationalArraySourceResetAssertionAndRetainedTail(t *testing.T) {
	m := polynomialCellFixture(t)
	u, e := m.monomials.Unit()
	serviceMust(t, e)
	a := NewRational()
	serviceMust(t, m.config.Rationals.SetInt64(&a, 4))
	p, e := m.MkRationalTerms([]Rational{a}, []Monomial{u})
	if modularSourceDebug {
		if e != ErrPolynomialAssertion || p != (Polynomial{}) || m.overloadResult == (Polynomial{}) {
			t.Fatal("source nonzero rational-array tail assertion before return", p, e)
		}
	} else {
		serviceMust(t, e)
		polynomialCaller(t, m, p)
	}
	n, e := m.rationalTerms.size(m)
	serviceMust(t, e)
	want := uint32(0)
	if modularSourceDebug {
		want = 1
	}
	if n != want {
		t.Fatal("actual source reset prefix", n, want)
	}
	v, e := m.rationalTerms.coefficient(m, 0)
	serviceMust(t, e)
	if v != NewInt(4) {
		t.Fatal("actual initialized source tail retained", v)
	}
	polynomialPhysicalAudit(t, m)
}
func TestPolynomialRationalUnivariateConsumesActualConversionFields(t *testing.T) {
	m := polynomialCellFixture(t)
	as := [3]Rational{NewRational(), NewRational(), NewRational()}
	for i, v := range []int64{3, 0, -2} {
		serviceMust(t, m.config.Rationals.SetInt64(&as[i], v))
	}
	p, e := m.MkRationalUnivariate(0, 2, as[:])
	serviceMust(t, e)
	polynomialCaller(t, m, p)
	n, e := m.rationalTerms.size(m)
	serviceMust(t, e)
	if n != 0 {
		t.Fatal(n)
	}
	for i := uint32(0); i < 3; i++ {
		a, e := m.rationalTerms.coefficient(m, i)
		serviceMust(t, e)
		if a != (Int{}) {
			t.Fatal("source add_reset/zeroDel", i, a)
		}
	}
	info, e := m.Info(p)
	serviceMust(t, e)
	if info.Size != 2 {
		t.Fatal(info)
	}
	a, e := m.Coefficient(p, 0)
	serviceMust(t, e)
	if a != NewInt(-2) {
		t.Fatal(a)
	}
	mono, e := m.TermMonomial(p, 0)
	serviceMust(t, e)
	d, e := m.monomials.DegreeOf(mono, 0)
	serviceMust(t, e)
	if d != 2 {
		t.Fatal(d)
	}
}
func TestPolynomialLinearSourceSwapCopiesAndRetainedBigTail(t *testing.T) {
	m := polynomialCellFixture(t)
	as := [3]Int{}
	serviceMust(t, m.integer.SetInt64(&as[0], 1<<40))
	as[2] = NewInt(-4)
	c := NewInt(5)
	p, e := m.MkLinearTake(as[:], []uint32{2, 1, 0}, &c)
	serviceMust(t, e)
	polynomialCaller(t, m, p)
	if as != ([3]Int{}) || c != (Int{}) {
		t.Fatal("actual source input swaps", as, c)
	}
	size, e := m.linearTerms.size(m)
	serviceMust(t, e)
	if size != 0 || m.linearTerms.initialized != 3 {
		t.Fatal(size, m.linearTerms.initialized)
	}
	tail, e := m.linearTerms.coefficient(m, 0)
	serviceMust(t, e)
	got, e := m.integer.Int64(&tail)
	serviceMust(t, e)
	if got != 1<<40 {
		t.Fatal(got)
	}
	coef, e := m.Coefficient(p, 0)
	serviceMust(t, e)
	if coef.cell == tail.cell {
		t.Fatal("source SOM copy aliased retained original")
	}
	polynomialPhysicalAudit(t, m, &coef)
}
func TestPolynomialRationalLinearSourceConsumesCopiedFieldsOnly(t *testing.T) {
	m := polynomialCellFixture(t)
	as := [2]Rational{NewRational(), NewRational()}
	c := NewRational()
	serviceMust(t, m.config.Rationals.SetInt64(&as[0], 1<<40))
	serviceMust(t, m.config.Rationals.SetInt64(&as[1], 0))
	serviceMust(t, m.config.Rationals.SetInt64(&c, -3))
	t.Cleanup(func() {
		for i := range as {
			serviceMust(t, m.config.Rationals.Del(&as[i]))
		}
		serviceMust(t, m.config.Rationals.Del(&c))
	})
	p, e := m.MkRationalLinear(as[:], []uint32{1, 2}, &c)
	serviceMust(t, e)
	polynomialCaller(t, m, p)
	for _, a := range []*Rational{&as[0], &as[1], &c} {
		if !isIntegerRational(a) {
			t.Fatal("const input changed", a)
		}
	}
	for i := uint32(0); i < 2; i++ {
		a, e := m.rationalTerms.coefficient(m, i)
		serviceMust(t, e)
		if a != (Int{}) {
			t.Fatal("actual converted array swaps", i, a)
		}
	}
	if m.linearConstant != (Int{}) || m.linearConstantActive {
		t.Fatal("source tmp_c zero at lexical end", m.linearConstant, m.linearConstantActive)
	}
	coef, e := m.Coefficient(p, 0)
	serviceMust(t, e)
	polynomialPhysicalAudit(t, m, &as[0].num, &as[1].num, &c.num, &coef)
}
