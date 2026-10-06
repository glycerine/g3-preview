// Source: src/math/polynomial/polynomial.cpp:2536-2542.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import "testing"

func TestPolynomialVariableActualInvocationAndCopiedFrameRefusal(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	var receipt polynomialVariableInvocationReceipt
	p, e := m.mkVariableWithReceipt(0, 3, &receipt)
	serviceMust(t, e)
	p = polynomialCaller(t, m, p)
	if !m.matchesVariableInvocation(&receipt) {
		t.Fatal("actual guarded invocation not captured")
	}
	old := receipt
	copied := *old.frame
	wrong := old
	wrong.frame = &copied
	state := m.memory.State()
	serviceMust(t, m.cleanupVariableInvocation(&wrong, true))
	if m.memory.State() != state {
		t.Fatal("copied frame consumed owner roots")
	}
	serviceMust(t, m.cleanupVariableInvocation(&receipt, true))
	serviceMust(t, m.cleanupVariableInvocation(&receipt, true))
	var later polynomialVariableInvocationReceipt
	q, e := m.mkVariableWithReceipt(0, 4, &later)
	serviceMust(t, e)
	q = polynomialCaller(t, m, q)
	state = m.memory.State()
	serviceMust(t, m.cleanupVariableInvocation(&old, true))
	if m.memory.State() != state || later.invocation == old.invocation {
		t.Fatal("stale invocation retargeted")
	}
	serviceMust(t, m.cleanupVariableInvocation(&later, true))
}

func TestPolynomialVariableActualEveryReservationPrefix(t *testing.T) {
	success, failed := false, false
	for prefix := int64(0); prefix < 128; prefix++ {
		full := polynomialPSCFullFixture(t)
		m := full.core
		var receipt polynomialVariableInvocationReceipt
		full.originalBudget.FailAfter(prefix)
		p, returned := m.mkVariableWithReceipt(0, 37, &receipt)
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, nil, 0)
		if returned != nil {
			failed = true
		}
		body := m.variableLocal.bodyCause
		own := m.cleanupVariableInvocation(&receipt, false)
		if own == nil {
			own = m.cleanupVariableInvocation(&receipt, true)
		}
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, nil, 0)
		full.originalBudget.FailAfter(-1)
		serviceMust(t, m.cleanupVariableInvocation(&receipt, true))
		if body != m.variableLocal.bodyCause {
			t.Fatal("own ACK erased body history", prefix)
		}
		if returned == nil {
			p = polynomialCaller(t, m, p)
			degree, e := m.Degree(p, 0)
			serviceMust(t, e)
			if degree != 37 {
				t.Fatal("literal variable exponent", prefix, degree)
			}
			success = true
			t.Log("actual first success", prefix)
			break
		}
		var later polynomialVariableInvocationReceipt
		q, e := m.mkVariableWithReceipt(0, 37, &later)
		serviceMust(t, e)
		q = polynomialCaller(t, m, q)
		state := m.memory.State()
		serviceMust(t, m.cleanupVariableInvocation(&receipt, true))
		if m.memory.State() != state {
			t.Fatal("older receipt consumed retry", prefix)
		}
	}
	if !success || !failed {
		t.Fatal("actual reservation campaign incomplete", success, failed)
	}
}
