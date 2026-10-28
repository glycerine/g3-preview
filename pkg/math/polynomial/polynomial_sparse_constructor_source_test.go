package polynomial

// Source: src/math/polynomial/polynomial.cpp:3077-3090; src/util/vector.h.

import (
	"fmt"
	"testing"
)

func TestPolynomialSparseConstructorHeaderReservationPrefixes(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 48 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			m := full.core
			p := polynomialQueryTerms(t, m, []int32{1, 1, 1, 1}, [][]MonomialPower{{{0, 3}}, {{0, 2}}, {{0, 1}}, nil})
			sk, e := NewPolynomialSkeleton(m, p, 0)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, sk.CloseChecked()) })
			beforeBytes, beforeStats := m.integer.Budget().Used(), m.memory.Stats()
			m.integer.Budget().FailAfter(ordinal)
			s, e := NewPolynomialSparseInterpolator(full, sk)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual class header reservation", e)
			}
			if s != nil {
				polynomialSparsePhysical(t, s, nil, []Polynomial{p})
				if e != nil && s.constructed {
					t.Fatal("failed class construction falsely completed")
				}
			} else if m.integer.Budget().Used() != beforeBytes || m.memory.Stats() != beforeStats {
				t.Fatal("cold-header refusal changed preexisting graph")
			}
			m.integer.Budget().FailAfter(-1)
			if s != nil {
				serviceMust(t, s.CloseChecked())
				polynomialSparsePhysical(t, s, nil, []Polynomial{p})
			}
			if e == nil {
				success = true
				t.Log("constructor header-prefix endpoint", ordinal)
			}
		})
	}
	if !success {
		t.Fatal("constructor never completed")
	}
}
