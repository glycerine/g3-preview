// Source: src/math/polynomial/polynomial.cpp:6737-6830;
// src/test/polynomial.cpp:1017-1033,1102-1117,1825,1847.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import "testing"

// These literal displayed-source cases retain distinct-factor and reconstructed
// polynomial assertions. The original driver returns before tst_mfact.
func TestPolynomialFactorCallerOriginalDormantSimpleAssertions(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	cases := []struct {
		coeff    []int32
		powers   [][]MonomialPower
		distinct uint32
	}{
		{nil, nil, 0},
		{[]int32{3}, [][]MonomialPower{nil}, 0},
		{[]int32{1}, [][]MonomialPower{{{0, 1}}}, 1},
		{[]int32{1, 1}, [][]MonomialPower{{{0, 1}}, {{1, 1}}}, 1},
		{[]int32{1, -1}, [][]MonomialPower{{{0, 1}}, {{1, 1}}}, 1},
	}
	for _, entry := range cases {
		p := polynomialQueryTerms(t, c, entry.coeff, entry.powers)
		fs, e := NewPolynomialFactors(full)
		serviceMust(t, e)
		serviceMust(t, full.Factor(p, fs, DefaultUPolynomialFactorParams()))
		count, e := fs.Distinct()
		serviceMust(t, e)
		if count != entry.distinct {
			t.Fatal("source ENSURE distinct", count, entry.distinct)
		}
		out, e := NewPolynomialRef(c)
		serviceMust(t, e)
		serviceMust(t, fs.Multiply(out))
		same, e := c.Equal(p, out.root.value)
		serviceMust(t, e)
		if !same {
			t.Fatal("source ENSURE reconstruction")
		}
		serviceMust(t, out.Close())
		serviceMust(t, fs.CloseChecked())
	}
}
