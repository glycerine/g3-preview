// Source: src/math/polynomial/polynomial.cpp:3134-3182.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import "testing"

func TestPolynomialSparseWarmActualLexicalAllocationGraph(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	z := m.integer
	serviceMust(t, m.modular.SetZPUint64(97))
	p := polynomialQueryTerms(t, m, []int32{1}, [][]MonomialPower{{{0, 1}}})
	sk, e := NewPolynomialSkeleton(m, p, 0)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, sk.CloseChecked()) })
	s, e := NewPolynomialSparseInterpolator(full, sk)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, s.CloseChecked()) })
	input := NewInt(2)
	q := polynomialQueryTerms(t, m, []int32{2}, [][]MonomialPower{nil})
	ok, e := s.Add(&input, q)
	serviceMust(t, e)
	if !ok {
		t.Fatal("source sample")
	}
	out, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	var callErr error
	call := func() {
		var ok bool
		ok, callErr = s.Interpolate(out)
		if callErr == nil && !ok {
			callErr = ErrPolynomialAssertion
		}
	}
	for i := 0; i < 20; i++ {
		call()
		serviceMust(t, callErr)
	}
	polynomialSparsePhysical(t, s, out, []Polynomial{p, q})
	beforeBytes, beforeStats := z.budget.Used(), m.memory.Stats()
	beforeCount, e := m.memory.AllocationCount()
	serviceMust(t, e)
	allocs := testing.AllocsPerRun(20, call)
	serviceMust(t, callErr)
	if z.budget.Used() != beforeBytes || m.memory.Stats() != beforeStats {
		t.Fatal("actual retained graph changed", beforeBytes, z.budget.Used(), beforeStats, m.memory.Stats())
	}
	afterCount, e := m.memory.AllocationCount()
	serviceMust(t, e)
	if afterCount <= beforeCount {
		t.Fatal("fresh source lexical vector allocations were skipped")
	}
	polynomialSparsePhysical(t, s, out, []Polynomial{p, q})
	t.Log("actual source lexical allocations; Go allocator events", allocs)
}
