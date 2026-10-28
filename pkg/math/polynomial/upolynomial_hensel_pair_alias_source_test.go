package polynomial

// Source: src/math/polynomial/upolynomial_factorization.cpp:583-674,513-568.

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestUPolynomialHenselSourceBorrowedInputOutputAlias(t *testing.T) {
	for _, debug := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary", true: "conditional"}[debug], func(t *testing.T) {
			q, s := henselDirectFixtureWithConfig(t, func() bool { return true }, debug)
			m := s.manager
			inputs := [5]upolyVectorState{}
			values := [5][]unumeral.Int{{unumeral.NewInt(2)}, {unumeral.NewInt(-1), unumeral.NewInt(1)}, {unumeral.NewInt(-2)}, {unumeral.NewInt(1), unumeral.NewInt(1)}, {unumeral.NewInt(-6), unumeral.NewInt(0), unumeral.NewInt(1)}}
			for i := range inputs {
				serviceMust(t, m.set(&inputs[i], values[i]))
			}
			five := unumeral.NewInt(5)
			body := s.henselPair(&five, &five, &five, m.stateView(&inputs[0]), m.stateView(&inputs[1]), m.stateView(&inputs[2]), m.stateView(&inputs[3]), m.stateView(&inputs[4]), &inputs[1], &inputs[3])
			var roots []*unumeral.Int
			for i := range inputs {
				v := &inputs[i]
				if v.backing != 0 {
					a := m.values(v)
					for j := uint32(0); uint64(j) < uint64(len(a)); j++ {
						roots = append(roots, &a[j])
					}
				}
			}
			primeTrialPhysicalAuditExtra(t, q, s, roots...)
			if body == nil {
				for j, index := range []int{1, 3} {
					for i, want := range [][]unumeral.Int{{unumeral.NewInt(9), unumeral.NewInt(1)}, {unumeral.NewInt(-9), unumeral.NewInt(1)}}[j] {
						got, e := m.read(m.stateView(&inputs[index]), uint32(i))
						serviceMust(t, e)
						if *got != want {
							t.Fatal("independent source alias output", j, i, *got, want)
						}
					}
				}
			}
			h := s.henselPairs
			if h != nil && h.active {
				serviceMust(t, s.cleanupHenselPairInvocation(&h.receipt, false))
			}
			for i := range inputs {
				serviceMust(t, m.reset(&inputs[i]))
				m.freeBacking(&inputs[i])
			}
			if h != nil && h.active {
				serviceMust(t, s.cleanupHenselPairInvocation(&h.receipt, true))
			}
			if body != nil {
				t.Fatal("source-defined borrowed vector alias", body)
			}
		})
	}
}
