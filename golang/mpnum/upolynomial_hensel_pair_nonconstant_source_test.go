// Source: src/math/polynomial/upolynomial_factorization.cpp:583-674.
// Licensed under the MIT license. Source author: Leonardo de Moura.
// Source author: Dejan (t-dejanj), 2011-11-15.
package mpnum

import (
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestUPolynomialHenselPairNonconstantDivisionEveryReachedPrefix(t *testing.T) {
	for ordinal := int64(0); ordinal < 4096; ordinal++ {
		var service *UPolynomialFactorizationService
		sawBIGAm := false
		q, s := henselDirectFixtureWithCheckpoint(t, func() bool {
			if service != nil && service.henselPairs != nil {
				h := service.henselPairs
				if h.amConstructed && h.am.cell != 0 {
					sawBIGAm = true
				}
			}
			return true
		})
		service = s
		m := s.manager
		var prime, n, negativeN, constant, want, negativeWant, leading, quarter, temporary Int
		serviceMust(t, q.z.SetLimbs(&prime, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
		one, two := NewInt(1), NewInt(2)
		serviceMust(t, q.z.Sub(&prime, &one, &n))
		serviceMust(t, q.z.Div(&n, &two, &n))
		serviceMust(t, q.z.Copy(&negativeN, &n))
		serviceMust(t, q.z.Neg(&negativeN))
		serviceMust(t, q.z.Mul(&n, &n, &constant))
		serviceMust(t, q.z.Add(&constant, &prime, &constant))
		serviceMust(t, q.z.Neg(&constant))
		serviceMust(t, q.z.Add(&prime, &one, &leading))
		serviceMust(t, q.z.SetLimbs(&quarter, []uint32{0, 0, 0, 0x20000000}))
		serviceMust(t, q.z.Sub(&one, &quarter, &temporary))
		serviceMust(t, q.z.Mul(&prime, &temporary, &want))
		serviceMust(t, q.z.Sub(&want, &n, &want))
		serviceMust(t, q.z.Add(&quarter, &one, &temporary))
		serviceMust(t, q.z.Mul(&prime, &temporary, &negativeWant))
		serviceMust(t, q.z.Sub(&n, &negativeWant, &negativeWant))
		args := []*Int{&prime, &n, &negativeN, &constant, &want, &negativeWant, &leading, &quarter, &temporary}
		inputs := [5]upolyVectorState{}
		vals := [5][]Int{{NewInt(1)}, {negativeN, NewInt(1)}, {NewInt(-1)}, {n, NewInt(1)}, {constant, NewInt(0), leading}}
		for i := range inputs {
			serviceMust(t, m.set(&inputs[i], vals[i]))
		}
		outputs := [2]upolyVectorState{}
		q.budget.FailAfter(ordinal)
		body := s.henselPair(&prime, &prime, &prime, m.stateView(&inputs[0]), m.stateView(&inputs[1]), m.stateView(&inputs[2]), m.stateView(&inputs[3]), m.stateView(&inputs[4]), &outputs[0], &outputs[1])
		roots := append([]*Int{}, args...)
		for _, v := range []*upolyVectorState{&inputs[0], &inputs[1], &inputs[2], &inputs[3], &inputs[4], &outputs[0], &outputs[1]} {
			if v.backing != 0 {
				a := m.values(v)
				for i := uint32(0); i < v.initialized; i++ {
					roots = append(roots, &a[i])
				}
			}
		}
		primeTrialPhysicalAuditExtra(t, q, s, roots...)
		if body == nil {
			if !sawBIGAm {
				t.Fatal("actual nonconstant source division never reached BIG scoped am")
			}
			if outputs[0].size != 2 || outputs[1].size != 2 {
				t.Fatal("independent BIG output degrees", outputs)
			}
			for j := 0; j < 2; j++ {
				a, e := m.read(m.stateView(&outputs[j]), 0)
				serviceMust(t, e)
				expected := &want
				if j == 1 {
					expected = &negativeWant
				}
				ok, e := q.z.Equal(a, expected)
				serviceMust(t, e)
				if !ok {
					t.Fatal("independent BIG lifted constant", j, *a, want)
				}
				lead, e := m.read(m.stateView(&outputs[j]), 1)
				serviceMust(t, e)
				expectedLead := &one
				if j == 1 {
					expectedLead = &leading
				}
				ok, e = q.z.Equal(lead, expectedLead)
				serviceMust(t, e)
				if !ok {
					t.Fatal("independent nonconstant lifted leading coefficient", j, *lead, *expectedLead)
				}
			}
		}
		q.budget.FailAfter(-1)
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
		pending, e := q.z.allocator.PendingRoots()
		serviceMust(t, e)
		if q.z.pendingCell.object != (memory.ObjectRef{}) || pending != ([3]memory.Block{}) || q.z.addMulInvocation.active || q.z.divisionRaw.active || q.z.extendedRaw.active || q.z.dividesNumeral.active {
			t.Fatal("actual pair child or numeric leaf remains after own ACK", ordinal, body, q.z.pendingCell, pending, q.z.addMulInvocation.active, q.z.divisionRaw.active, q.z.extendedRaw.active, q.z.dividesNumeral.active)
		}
		for _, a := range args {
			serviceMust(t, q.z.Del(a))
		}
		if body == nil {
			t.Log("first nonconstant pair source completion", ordinal)
			return
		}
	}
	t.Fatal("no actual BIG pair completion")
}
