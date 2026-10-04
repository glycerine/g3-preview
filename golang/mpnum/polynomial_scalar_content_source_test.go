// Source: src/math/polynomial/polynomial.cpp:3343-3359; src/math/polynomial/polynomial.h:394-398.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import "testing"

func TestPolynomialScalarIntegerContentSignedFirstAndEarlyOne(t *testing.T) {
	for route := uint32(0); route < 5; route++ {
		full := polynomialPSCFullFixture(t)
		m := full.core
		p := m.zero
		want := int32(0)
		switch route {
		case 1:
			p = polynomialQueryTerms(t, m, []int32{-7}, [][]MonomialPower{nil})
			want = -7
		case 2:
			p = polynomialQueryTerms(t, m, []int32{-7}, [][]MonomialPower{{{0, 2}}})
			want = -7
		case 3:
			p = polynomialQueryTerms(t, m, []int32{-6, 10, -14}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})
			want = 2
		case 4:
			p = polynomialQueryTerms(t, m, []int32{1, 12, 18}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})
			want = 1
		}
		out := NewInt(99)
		m.config.Checkpoint = func() bool { t.Fatal("scalar content has no source checkpoint"); return false }
		serviceMust(t, full.IntegerScalarContent(p, &out))
		if out != NewInt(want) {
			t.Fatal("source initial SET preserves lone signed coefficient", route, out, want)
		}
		serviceMust(t, m.modular.Del(&out))
	}
}
