// Source: src/math/polynomial/polynomial.cpp:2681-2733; src/util/rational.h:54. Licensed under the MIT license.
package mpnum

import "testing"

func TestPolynomialMulAddZeroLeavesRightUnread(t *testing.T) {
	m := polynomialCellFixture(t)
	a := NewInt(0)
	p, e := m.MulAdd(m.zero, Polynomial{}, &a)
	serviceMust(t, e)
	if p != m.zero {
		t.Fatal(p)
	}
}
func TestPolynomialCoefficientGF2SourceMinusOne(t *testing.T) {
	m := polynomialCellFixture(t)
	serviceMust(t, m.SetZPUint64(2))
	x, e := m.MkVariable(0, 1)
	serviceMust(t, e)
	polynomialCaller(t, m, x)
	p, e := m.Sub(x, m.unit)
	serviceMust(t, e)
	polynomialCaller(t, m, p)
	info, e := m.Info(p)
	serviceMust(t, e)
	if info.Size != 2 {
		t.Fatal(info)
	}
	for i := uint32(0); i < 2; i++ {
		a, e := m.Coefficient(p, i)
		serviceMust(t, e)
		if a != NewInt(1) {
			t.Fatal("GF2 source Set(-1) normalizes to one", a)
		}
	}
	twice, e := m.Add(x, x)
	serviceMust(t, e)
	polynomialCaller(t, m, twice)
	info, e = m.Info(twice)
	serviceMust(t, e)
	if info.Size != 0 {
		t.Fatal(info)
	}
}
