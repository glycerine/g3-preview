// Source: src/math/polynomial/polynomial.cpp:5654-5901; src/util/obj_ref.h.
package polynomial

import (
	"testing"
)

func TestPolynomialPSCComposedEveryReservationPrefix(t *testing.T) {
	failures := 0
	for ordinal := int64(0); ordinal < 512; ordinal++ {
		m := polynomialCellFixture(t)
		p := polynomialQueryTerms(t, m, []int32{1, 1, 1, 2}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{0, 1}, {1, 1}}, {{1, 1}}, nil})
		q := polynomialQueryTerms(t, m, []int32{1, 3}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
		want := polynomialQueryTerms(t, m, []int32{6, -1}, [][]MonomialPower{{{0, 1}}, {{0, 2}}})
		S, e := NewPolynomialRefVector(m)
		serviceMust(t, e)
		m.integer.Budget().FailAfter(ordinal)
		e = m.PSCChain(p, q, 1, S)
		// Immediate source/public/local census precedes any fault lift or abort.
		polynomialPSCPhysicalVectors(t, m, nil, []*PolynomialRefVector{S}, []Polynomial{p, q, want}, nil)
		n, z := S.Size()
		serviceMust(t, z)
		if n > 1 {
			t.Fatal("source first fixture has only one principal coefficient", n)
		}
		if n == 1 {
			value, z := S.Get(0)
			serviceMust(t, z)
			polynomialQueryExpect(t, m, value, map[[3]uint32]int64{{1, 0, 0}: 6, {2, 0, 0}: -1})
		}
		if S.state.incoming != (Polynomial{}) {
			polynomialQueryExpect(t, m, S.state.incoming, map[[3]uint32]int64{{1, 0, 0}: 6, {2, 0, 0}: -1})
		}
		if e != nil && !numericPhysicalReservationRefusal(e) {
			t.Fatal("source prefix refusal category", ordinal, e)
		}
		m.integer.Budget().FailAfter(-1)
		serviceMust(t, m.CleanupPSC())
		polynomialPSCPhysicalVectors(t, m, nil, []*PolynomialRefVector{S}, []Polynomial{p, q, want}, nil)
		if e == nil {
			got, z := S.Get(0)
			serviceMust(t, z)
			equal, z := m.Equal(got, want)
			serviceMust(t, z)
			if !equal {
				neg, z := m.Neg(want)
				serviceMust(t, z)
				serviceMust(t, m.IncRef(neg))
				equal, z = m.Equal(got, neg)
				serviceMust(t, z)
				serviceMust(t, m.DecRef(neg))
			}
			if !equal {
				t.Fatal("independent completed principal coefficient", ordinal)
			}
			serviceMust(t, S.Close())
			if failures == 0 {
				t.Fatal("no real reservation refused")
			}
			t.Log("refusals before success", ordinal)
			return
		}
		failures++
		serviceMust(t, S.Close())
	}
	t.Fatal("actual source route did not complete within finite campaign")
}

func TestPolynomialPSCOptimizedAndHeapEveryReservationPrefix(t *testing.T) {
	for _, route := range []uint32{0, 1} {
		t.Run(string(rune('A'+route)), func(t *testing.T) {
			failures := 0
			for ordinal := int64(0); ordinal < 1024; ordinal++ {
				m := polynomialCellFixture(t)
				var p, q, want Polynomial
				if route == 0 {
					p = polynomialQueryTerms(t, m, []int32{1, 2, 3, 4}, [][]MonomialPower{{{0, 4}}, {{0, 2}}, {{0, 1}}, nil})
					q = polynomialQueryTerms(t, m, []int32{4, 4, 3}, [][]MonomialPower{{{0, 3}}, {{0, 1}}, nil})
					want = polynomialQueryTerms(t, m, []int32{17109}, [][]MonomialPower{nil})
				} else {
					p = polynomialQueryTerms(t, m, []int32{1, 3}, [][]MonomialPower{{{0, 18}}, nil})
					q = polynomialQueryTerms(t, m, []int32{1, 2}, [][]MonomialPower{{{0, 17}}, nil})
					want = polynomialQueryTerms(t, m, []int32{-2, 3}, [][]MonomialPower{{{0, 1}}, nil})
				}
				S, e := NewPolynomialRefVector(m)
				serviceMust(t, e)
				r, e := NewPolynomialRef(m)
				serviceMust(t, e)
				var visits, cooperation uint32
				m.config.Checkpoint = func() bool { visits++; return true }
				m.config.Cooperation = func() error { cooperation++; return nil }
				m.integer.Budget().FailAfter(ordinal)
				if route == 0 {
					e = m.PSCChain(p, q, 0, S)
				} else {
					e = m.optimizedSe1(18, 17, p, q, q, m.unit, 0, &r.root)
				}
				polynomialPSCPhysicalVectors(t, m, []*PolynomialRef{r}, []*PolynomialRefVector{S}, []Polynomial{p, q, want}, nil)
				if route == 0 {
					n, z := S.Size()
					serviceMust(t, z)
					if n > 3 {
						t.Fatal("source principal degrees2/1/0", n)
					}
					values := [3]int64{16, 132, 17109}
					if e == nil {
						values = [3]int64{17109, 132, 16}
					}
					for i := uint32(0); i < n; i++ {
						value, z := S.Get(i)
						serviceMust(t, z)
						polynomialQueryExpect(t, m, value, map[[3]uint32]int64{{}: values[i]})
					}
					if S.state.incoming != (Polynomial{}) {
						if n >= 3 {
							t.Fatal("extra unjoined source push")
						}
						polynomialQueryExpect(t, m, S.state.incoming, map[[3]uint32]int64{{}: values[n]})
					}
				} else if r.root.value != (Polynomial{}) {
					polynomialQueryExpect(t, m, r.root.value, map[[3]uint32]int64{{}: 3, {1, 0, 0}: -2})
				}
				if e != nil && !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual refused source action", ordinal, e)
				}
				if visits != cooperation {
					t.Fatal("source checkpoint before cooperation", visits, cooperation)
				}
				m.integer.Budget().FailAfter(-1)
				serviceMust(t, m.CleanupPSC())
				polynomialPSCPhysicalVectors(t, m, []*PolynomialRef{r}, []*PolynomialRefVector{S}, []Polynomial{p, q, want}, nil)
				if e == nil {
					got := r.root.value
					if route == 0 {
						got, e = S.Get(0)
						serviceMust(t, e)
					}
					equal, z := m.Equal(got, want)
					serviceMust(t, z)
					if !equal {
						neg, z := m.Neg(want)
						serviceMust(t, z)
						serviceMust(t, m.IncRef(neg))
						equal, z = m.Equal(got, neg)
						serviceMust(t, z)
						serviceMust(t, m.DecRef(neg))
					}
					if !equal || visits == 0 {
						t.Fatal("independent completed identity/actual checkpoint", ordinal, visits)
					}
					serviceMust(t, r.Close())
					serviceMust(t, S.Close())
					if failures == 0 {
						t.Fatal("missing genuine reservation prefixes")
					}
					t.Log("refusals then success", ordinal, "checkpoint visits", visits)
					return
				}
				failures++
				serviceMust(t, r.Close())
				serviceMust(t, S.Close())
			}
			t.Fatal("source campaign never completed")
		})
	}
}
