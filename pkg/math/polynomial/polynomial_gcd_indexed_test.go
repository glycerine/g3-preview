// Source: src/math/polynomial/polynomial.cpp:1863-1876; src/util/mpz.cpp array gcd.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialIndexedArrayGCDEarlyOneLeavesTailUnread(t *testing.T) {
	m := polynomialCellFixture(t)
	v := &m.linearTerms
	serviceMust(t, v.appendCoefficient(m, util.NewInt(1)))
	serviceMust(t, v.appendCoefficient(m, util.NewInt(1)))
	serviceMust(t, v.expand(m))
	view := util.IntegerArrayView{Runtime: m.memory, Raw: v.raw, Count: 3, Offset: 8, Stride: 12}
	var g util.Int
	serviceMust(t, m.modular.GCDManyIndexed(view, &g))
	if g != util.NewInt(1) {
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
	as := [1]util.Int{{Val: -1, Cell: ref}}
	m.integer.PublishCell(ref)
	m.integer.ResizeCell(ref, 2)
	t.Cleanup(func() { serviceMust(t, m.integer.Del(&as[0])) })
	serviceMust(t, m.integer.GCDMany(as[:], &as[0]))
	if as[0].NativeScalar() != 1 || (as[0]).Cell != ref {
		t.Fatal("self set then Abs", as[0])
	}
}
