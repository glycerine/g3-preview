// Source: src/math/polynomial/polynomial.cpp:1863-1876; src/util/mpz.cpp array gcd.
package polynomial

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialIndexedArrayGCDEarlyOneLeavesTailUnread(t *testing.T) {
	m := polynomialCellFixture(t)
	v := &m.linearTerms
	serviceMust(t, v.appendCoefficient(m, unumeral.NewInt(1)))
	serviceMust(t, v.appendCoefficient(m, unumeral.NewInt(1)))
	view := unumeral.IntegerArrayView{Values: v.values, Count: 3}
	var g unumeral.Int
	serviceMust(t, m.modular.GCDManyIndexed(view, &g))
	if g != unumeral.NewInt(1) {
		t.Fatal(g)
	}
	serviceMust(t, m.modular.Del(&g))
}

// Source count-one array GCD preserves the literal self-set shortcut before
// Abs. The fresh physical digit words are deliberately never read by source.
func TestPolynomialIndexedArrayGCDPreservesSliceSelfIdentity(t *testing.T) {
	m := polynomialCellFixture(t)
	ref, e := m.integer.Allocate(6, 2)
	serviceMust(t, e)
	as := [1]unumeral.Int{{Val: -1, Cell: ref}}
	m.integer.PublishCell(ref)
	m.integer.ResizeCell(ref, 2)
	t.Cleanup(func() { serviceMust(t, m.integer.Del(&as[0])) })
	serviceMust(t, m.integer.GCDMany(as[:], &as[0]))
	if as[0].NativeScalar() != 1 || (as[0]).Cell != ref {
		t.Fatal("self set then Abs", as[0])
	}
}
