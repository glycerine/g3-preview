// Source: src/util/vector.h:59-101; src/util/mpz.h:97-99. Licensed under the MIT license.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialNumeralVectorMovesOnlyLogicalFields(t *testing.T) {
	m := polynomialCellFixture(t)
	v := &m.linearTerms
	serviceMust(t, v.appendCoefficient(m, util.NewInt(17)))
	serviceMust(t, v.appendCoefficient(m, util.NewInt(-8)))
	serviceMust(t, v.publishSize(m, 1))
	serviceMust(t, v.expand(m))
	a, e := v.coefficient(m, 0)
	serviceMust(t, e)
	if a != util.NewInt(17) {
		t.Fatal(a)
	}
	_, e = v.coefficient(m, 1)
	if !errors.Is(e, util.ErrRuntimeUninitialized) {
		t.Fatal("mpz is nontrivial: source move constructs logical size only", e)
	}
}

// The source unsigned vector extent accepts this wrapped arithmetic: old
// bytes12, newbytes20. Two real initialized source records fit the old backing;
// the second source move reaches the first actual endangered destination word.
// This is source malformed-header/overflow-domain guard evidence, not OOM.
func TestPolynomialNumeralVectorWrappedGrowthRetainsReachedMovePrefix(t *testing.T) {
	m := polynomialCellFixture(t)
	v := &m.linearTerms
	for _, n := range []int64{1 << 40, (1 << 40) + 3} {
		var a util.Int
		serviceMust(t, m.integer.SetInt64(&a, n))
		serviceMust(t, v.appendCoefficient(m, a))
	}
	old := v.raw
	first, e := v.coefficient(m, 0)
	serviceMust(t, e)
	second, e := v.coefficient(m, 1)
	serviceMust(t, e)
	serviceMust(t, m.memory.WriteUint32(v.raw, 0, 0x2aaaaaab))
	e = v.expand(m)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual destination word guard", e)
	}
	if !v.interrupted || v.old != old || v.raw == old || v.moved != 1 || v.initialized != 1 {
		t.Fatal("real publication/move prefix", v)
	}
	got, e := v.coefficient(m, 0)
	serviceMust(t, e)
	if got != first {
		t.Fatal("first new owning field", got, first)
	}
	prior := polynomialVector{raw: v.old, width: 12}
	a, e := prior.coefficient(m, 0)
	serviceMust(t, e)
	if (a).Cell != 0 || a.NativeScalar() != first.NativeScalar() {
		t.Fatal("actual mpz move clears old ref only", a)
	}
	a, e = prior.coefficient(m, 1)
	serviceMust(t, e)
	if a != second {
		t.Fatal("unreached old source remains owning", a, second)
	}
	if _, e = v.coefficient(m, 1); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("did not guard actual second destination", e)
	}
	polynomialPhysicalAudit(t, m)
	// The test never repairs the header or retries the undefined source move.
	// Owning cleanup drains the two disjoint actual initialized field roots.
	serviceMust(t, v.close(m))
	polynomialPhysicalAudit(t, m)
}
