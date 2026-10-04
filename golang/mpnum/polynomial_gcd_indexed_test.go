// Source: src/math/polynomial/polynomial.cpp:1863-1876; src/util/mpz.cpp array gcd. Licensed under the MIT license.
package mpnum

import "testing"

func TestPolynomialIndexedArrayGCDEarlyOneLeavesTailUnread(t *testing.T) {
	m := polynomialCellFixture(t)
	v := &m.linearTerms
	serviceMust(t, v.appendCoefficient(m, NewInt(1)))
	serviceMust(t, v.appendCoefficient(m, NewInt(1)))
	serviceMust(t, v.expand(m))
	view := integerArrayView{runtime: m.memory, raw: v.raw, count: 3, offset: 8, stride: 12}
	var g Int
	serviceMust(t, m.modular.GCDManyIndexed(view, &g))
	if g != NewInt(1) {
		t.Fatal(g)
	}
	serviceMust(t, m.modular.Del(&g))
}

// Source count-one array GCD preserves the literal self-set shortcut before
// Abs. The fresh physical digit words are deliberately never read by source.
func TestPolynomialIndexedArrayGCDPreservesSliceSelfIdentity(t *testing.T) {
	m := polynomialCellFixture(t)
	ref, e := m.integer.allocate(6, 2)
	serviceMust(t, e)
	as := [1]Int{{val: -1, cell: ref}}
	m.integer.publishCell(ref)
	m.integer.resizeCell(ref, 2)
	t.Cleanup(func() { serviceMust(t, m.integer.Del(&as[0])) })
	serviceMust(t, m.integer.GCDMany(as[:], &as[0]))
	if as[0].val != 1 || as[0].cell != ref {
		t.Fatal("self set then Abs", as[0])
	}
}
