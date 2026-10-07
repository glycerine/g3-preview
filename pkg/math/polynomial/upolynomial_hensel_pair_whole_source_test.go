// Source: src/math/polynomial/upolynomial_factorization.cpp:583-674.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestUPolynomialHenselPairEveryReachedReservationAndOwnChild(t *testing.T) {
	for ordinal := int64(0); ordinal < 2048; ordinal++ {
		q, s := mignotteDirectFixture(t)
		m := s.manager
		inputs := [5]upolyVectorState{}
		values := [5][]util.Int{{util.NewInt(2)}, {util.NewInt(-1), util.NewInt(1)}, {util.NewInt(-2)}, {util.NewInt(1), util.NewInt(1)}, {util.NewInt(-6), util.NewInt(0), util.NewInt(1)}}
		for i := range inputs {
			serviceMust(t, m.set(&inputs[i], values[i]))
		}
		outputs := [2]upolyVectorState{}
		five := util.NewInt(5)
		q.Budget().FailAfter(ordinal)
		body := s.henselPair(&five, &five, &five, m.stateView(&inputs[0]), m.stateView(&inputs[1]), m.stateView(&inputs[2]), m.stateView(&inputs[3]), m.stateView(&inputs[4]), &outputs[0], &outputs[1])
		if body == nil {
			for j, wants := range [][]util.Int{{util.NewInt(9), util.NewInt(1)}, {util.NewInt(-9), util.NewInt(1)}} {
				if outputs[j].size != 2 {
					t.Fatal("independent lifted degree", outputs)
				}
				for i, want := range wants {
					got, e := m.read(m.stateView(&outputs[j]), uint32(i))
					serviceMust(t, e)
					if *got != want {
						t.Fatal("independent lifted coefficient", j, i, *got, want)
					}
				}
			}
		}
		var roots []*util.Int
		for _, v := range []*upolyVectorState{&inputs[0], &inputs[1], &inputs[2], &inputs[3], &inputs[4], &outputs[0], &outputs[1]} {
			if v.backing != 0 {
				a := m.values(v)
				for i := uint32(0); i < v.initialized; i++ {
					roots = append(roots, &a[i])
				}
			}
		}
		primeTrialPhysicalAuditExtra(t, q, s, roots...)
		q.Budget().FailAfter(-1)
		h := s.henselPairs
		if h != nil && h.active {
			serviceMust(t, s.cleanupHenselPairInvocation(&h.receipt, false))
		}
		for i := range outputs {
			serviceMust(t, m.reset(&outputs[i]))
			m.freeBacking(&outputs[i])
		}
		for i := range inputs {
			serviceMust(t, m.reset(&inputs[i]))
			m.freeBacking(&inputs[i])
		}
		if h != nil && h.active {
			serviceMust(t, s.cleanupHenselPairInvocation(&h.receipt, true))
		}
		if q.Integers().AddMulInvocation.Active || q.Integers().DivisionRaw.Active || q.Integers().ExtendedRaw.Active || q.Integers().DividesNumeral.Active {
			t.Fatal("real pair child remains after exact own continuation", ordinal, body, q.Integers().AddMulInvocation.Active, q.Integers().DivisionRaw.Active, q.Integers().ExtendedRaw.Active, q.Integers().DividesNumeral.Active)
		}
		if body == nil {
			t.Log("first pair source completion", ordinal)
			return
		}
	}
	t.Fatal("no actual pair completion")
}
