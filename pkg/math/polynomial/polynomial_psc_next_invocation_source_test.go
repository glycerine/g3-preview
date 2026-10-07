// Source: src/math/polynomial/polynomial.cpp:5694-5749.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package polynomial

import (
	"testing"
)

func TestPolynomialPSCNextActualChildInvocationEveryReservationPrefix(t *testing.T) {
	success := false
	for prefix := int64(0); prefix < 512; prefix++ {
		full := polynomialPSCFullFixture(t)
		m := full.core
		a := polynomialQueryTerms(t, m, []int32{1, 2, 3}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})
		q := polynomialQueryTerms(t, m, []int32{2, 4}, [][]MonomialPower{{{0, 1}}, nil})
		want := polynomialQueryTerms(t, m, []int32{12}, [][]MonomialPower{nil})
		out, e := NewPolynomialRef(m)
		serviceMust(t, e)
		var receipt polynomialPSCNextInvocationReceipt
		full.originalBudget.FailAfter(prefix)
		returned := m.optimizedSe1WithReceipt(2, 1, a, q, q, m.unit, 0, &out.root, &receipt)
		s := &m.pscNext
		audit := func() {
			ps := []Polynomial{a, q, want, out.root.value, out.root.incoming}
			for i := uint8(0); i < s.constructed; i++ {
				ps = append(ps, s.refs[i].value, s.refs[i].incoming)
			}
			polynomialImpPhysicalAuditNamedRaw(t, full, ps, nil, out.header, nil)
		}
		audit()
		body, source := s.bodyCause, s.sourceCause
		own := m.cleanupPSCNextInvocation(&receipt, false)
		if own == nil {
			own = m.cleanupPSCNextInvocation(&receipt, true)
		}
		audit()
		full.originalBudget.FailAfter(-1)
		serviceMust(t, m.cleanupPSCNextInvocation(&receipt, false))
		serviceMust(t, m.cleanupPSCNextInvocation(&receipt, true))
		if s.bodyCause != body || s.sourceCause != source {
			t.Fatal("own ACK erased immutable history", prefix)
		}
		if returned == nil {
			same, e := m.Equal(out.root.value, want)
			serviceMust(t, e)
			if !same {
				t.Fatal("bounded independent resultant identity", prefix)
			}
			success = true
		}
		serviceMust(t, out.Close())
		if success {
			t.Log("actual first success", prefix)
			break
		}
		_ = own
	}
	if !success {
		t.Fatal("actual prefix horizon")
	}
}

// A Go checkpoint may hold an internal borrowed address. It gains no authority
// to append to or destroy the source-local H owned by the executing invocation.
func TestPolynomialPSCNextBorrowedHCallbackCannotMutateOwnedBuffer(t *testing.T) {
	for _, operation := range []string{"push", "finalize"} {
		t.Run(operation, func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			m := full.core
			a := polynomialQueryTerms(t, m, []int32{1, 2, 3}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})
			q := polynomialQueryTerms(t, m, []int32{2, 4}, [][]MonomialPower{{{0, 1}}, nil})
			out, e := NewPolynomialRef(m)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, m.CleanupPSCNext()); serviceMust(t, out.Close()) })
			checked := false
			m.config.Checkpoint = func() bool {
				if !checked && m.pscNext.executing && m.pscNext.bufferConstructed && m.pscBuffer.count > 0 {
					checked = true
					before := m.pscBuffer
					state := m.memory.State()
					var err error
					if operation == "push" {
						err = m.pscBuffer.push(m, m.unit)
					} else {
						err = m.pscBuffer.finalize(m, true)
					}
					if err != ErrPolynomialPending || m.pscBuffer != before || m.memory.State() != state {
						t.Fatalf("borrowed callback mutated actual H: err=%v count=%d prior=%d", err, m.pscBuffer.count, before.count)
					}
				}
				return true
			}
			var receipt polynomialPSCNextInvocationReceipt
			serviceMust(t, m.optimizedSe1WithReceipt(2, 1, a, q, q, m.unit, 0, &out.root, &receipt))
			if !checked {
				t.Fatal("actual reached H callback absent")
			}
		})
	}
}

func TestPolynomialPSCNextExpiredCompletedChildrenPreserveLaterVariableAndH(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	var receipt polynomialPSCNextInvocationReceipt
	s, e := m.beginPSCNextInvocation(&receipt)
	serviceMust(t, e)
	serviceMust(t, s.refs[0].assign(m, m.unit))
	serviceMust(t, m.pscBuffer.constructWithNextReceipt(m, &receipt))
	s.bufferConstructed = true
	s.bufferInvocation = m.pscBuffer.invocation
	serviceMust(t, m.pscBuffer.pushWithNextReceipt(m, m.unit, &receipt))
	full.originalBudget.FailAfter(0)
	_, body := m.pscNextVariable(s, 0, 37)
	if body == nil {
		t.Fatal("actual stopped variable child absent")
	}
	s.bodyCause = body
	s.executing = false
	m.pscLeave()
	serviceMust(t, m.finishPSCNext(false))
	if !s.active || !s.bufferDestroyed {
		t.Fatal("unjoined actual child or completed H prefix lost")
	}
	full.originalBudget.FailAfter(-1)
	serviceMust(t, m.cleanupVariableInvocation(&s.variable, true))
	var later polynomialVariableInvocationReceipt
	full.originalBudget.FailAfter(0)
	_, laterBody := m.mkVariableWithReceipt(0, 38, &later)
	if laterBody == nil || !later.frame.active {
		t.Fatal("actual later foreign variable continuation absent", laterBody)
	}
	serviceMust(t, m.pscBuffer.construct(m))
	serviceMust(t, m.pscBuffer.push(m, m.unit))
	full.originalBudget.FailAfter(-1)
	t.Cleanup(func() {
		full.originalBudget.FailAfter(-1)
		serviceMust(t, m.cleanupVariableInvocation(&later, true))
		serviceMust(t, m.pscBuffer.finalize(m, true))
		serviceMust(t, m.cleanupPSCNextInvocation(&receipt, true))
	})
	before := m.pscBuffer
	state := m.memory.State()
	rc, e := m.readWord(m.unit, 0)
	serviceMust(t, e)
	serviceMust(t, m.cleanupPSCNextInvocation(&receipt, false))
	serviceMust(t, m.cleanupPSCNextInvocation(&receipt, true))
	rr, e := m.readWord(m.unit, 0)
	serviceMust(t, e)
	if m.pscBuffer != before || m.memory.State() != state || rr != rc || s.bodyCause != body || s.active {
		t.Fatal("expired child tuple read or consumed later authority", s.active, rr, rc)
	}
	serviceMust(t, m.pscBuffer.finalize(m, true))
	serviceMust(t, m.cleanupVariableInvocation(&later, true))
}
