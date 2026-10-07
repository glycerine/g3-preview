// Source: src/math/polynomial/polynomial.cpp:5817-5895.
package polynomial

import (
	"testing"
)

func TestPolynomialPSCClassicLiteralRegularDefectiveAndZero(t *testing.T) {
	for _, fixture := range []struct {
		p, q   []int32
		pp, qp [][]MonomialPower
		want   []int32
	}{
		{[]int32{1, 1}, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil}, [][]MonomialPower{{{0, 1}}, nil}, []int32{2}},
		{[]int32{1, 1}, []int32{1, 1}, [][]MonomialPower{{{0, 3}}, nil}, [][]MonomialPower{{{0, 2}}, nil}, []int32{2, -1}},
		{[]int32{1, 1}, []int32{1, 1}, [][]MonomialPower{{{0, 4}}, nil}, [][]MonomialPower{{{0, 3}}, nil}, []int32{2, 1}},
		{[]int32{1, 1}, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil}, [][]MonomialPower{{{0, 2}}, nil}, []int32{0}},
	} {
		for _, swap := range []bool{false, true} {
			full := polynomialPSCFullFixture(t)
			m := full.core
			p := polynomialQueryTerms(t, m, fixture.p, fixture.pp)
			q := polynomialQueryTerms(t, m, fixture.q, fixture.qp)
			if swap {
				p, q = q, p
			}
			out, e := NewPolynomialRefVector(m)
			serviceMust(t, e)
			serviceMust(t, full.PSCChainClassic(p, q, 0, out))
			n, e := out.Size()
			serviceMust(t, e)
			if int(n) != len(fixture.want) {
				t.Fatal("literal compact reverse size", n, fixture.want)
			}
			for i, w := range fixture.want {
				v, e := out.Get(uint32(i))
				serviceMust(t, e)
				constant, e := m.IsConstant(v)
				serviceMust(t, e)
				if !constant {
					t.Fatal("spurious variable coefficient", i)
				}
				if w == 0 {
					zero, e := m.IsZero(v)
					serviceMust(t, e)
					if !zero {
						t.Fatal("literal zero fallback")
					}
					continue
				}
				a, e := m.Coefficient(v, 0)
				serviceMust(t, e)
				iv, e := m.integer.Int32(&a)
				serviceMust(t, e)
				if iv != w {
					t.Fatal("independent source elimination", fixture.pp, swap, i, iv, w)
				}
			}
			serviceMust(t, out.Close())
		}
	}
}
