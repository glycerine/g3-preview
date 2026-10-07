// Source: src/math/polynomial/polynomial.cpp:2558-2633.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialRationalArraySourceResetAssertionAndRetainedTail(t *testing.T) {
	m := polynomialCellFixture(t)
	u, e := m.monomials.Unit()
	serviceMust(t, e)
	a := util.NewRational()
	serviceMust(t, m.config.Rationals.SetInt64(&a, 4))
	p, e := m.MkRationalTerms([]util.Rational{a}, []Monomial{u})
	if polynomialSourceDebug {
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
	if polynomialSourceDebug {
		want = 1
	}
	if n != want {
		t.Fatal("actual source reset prefix", n, want)
	}
	v, e := m.rationalTerms.coefficient(m, 0)
	serviceMust(t, e)
	if v != util.NewInt(4) {
		t.Fatal("actual initialized source tail retained", v)
	}
	polynomialPhysicalAudit(t, m)
}
func TestPolynomialRationalUnivariateConsumesActualConversionFields(t *testing.T) {
	m := polynomialCellFixture(t)
	as := [3]util.Rational{util.NewRational(), util.NewRational(), util.NewRational()}
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
		if a != (util.Int{}) {
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
	if a != util.NewInt(-2) {
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
	as := [3]util.Int{}
	serviceMust(t, m.integer.SetInt64(&as[0], 1<<40))
	as[2] = util.NewInt(-4)
	c := util.NewInt(5)
	p, e := m.MkLinearTake(as[:], []uint32{2, 1, 0}, &c)
	serviceMust(t, e)
	polynomialCaller(t, m, p)
	if as != ([3]util.Int{}) || c != (util.Int{}) {
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
	if (coef).Cell == (tail).Cell {
		t.Fatal("source SOM copy aliased retained original")
	}
	polynomialPhysicalAudit(t, m, &coef)
}
func TestPolynomialRationalLinearSourceConsumesCopiedFieldsOnly(t *testing.T) {
	m := polynomialCellFixture(t)
	as := [2]util.Rational{util.NewRational(), util.NewRational()}
	c := util.NewRational()
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
	for _, a := range []*util.Rational{&as[0], &as[1], &c} {
		if !util.IsIntegerRational(a) {
			t.Fatal("const input changed", a)
		}
	}
	for i := uint32(0); i < 2; i++ {
		a, e := m.rationalTerms.coefficient(m, i)
		serviceMust(t, e)
		if a != (util.Int{}) {
			t.Fatal("actual converted array swaps", i, a)
		}
	}
	if m.linearConstant != (util.Int{}) || m.linearConstantActive {
		t.Fatal("source tmp_c zero at lexical end", m.linearConstant, m.linearConstantActive)
	}
	coef, e := m.Coefficient(p, 0)
	serviceMust(t, e)
	polynomialPhysicalAudit(t, m, &(*as[0].BorrowNumerator()), &(*as[1].BorrowNumerator()), &(*c.BorrowNumerator()), &coef)
}
