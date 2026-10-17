// Source: src/math/polynomial/linear_eq_solver.h:35; src/math/polynomial/polynomial.cpp:3142.
package polynomial

import (
	"errors"
	"testing"
)

func TestLinearConcreteLocalConstructorGuardAndNoHeaderReservation(t *testing.T) {
	for _, field := range []bool{false, true} {
		prime := uint64(0)
		if field {
			prime = 97
		}
		m, z, budget := modularManager(t, prime)
		if !field && polynomialSourceDebug {
			polynomialThrowAssertionFixture(t)
		}
		before := budget.Used()
		budget.FailAfter(0)
		var local ModularLinearSolver
		e := local.constructDefault(m)
		if !field && polynomialSourceDebug {
			if !errors.Is(e, ErrPolynomialAssertion) || local.self != nil || local.manager != nil {
				t.Fatal("failed source ctor not completed", e)
			}
			budget.FailAfter(-1)
			continue
		}
		serviceMust(t, e)
		if local.self != &local || local.header != 0 || budget.Used() != before {
			t.Fatal("actual embedded local fields/header", local.header, budget.Used(), before)
		}
		integerPhysicalAudit(t, z, &m.P, &m.Lower, &m.Upper)
		if field {
			heap, e := NewModularLinearSolver(m)
			if heap != nil || !numericPhysicalReservationRefusal(e) {
				t.Fatal("public header reserve ordering", heap, e)
			}
		}
		budget.FailAfter(-1)
		serviceMust(t, local.SourceFinalize())
		serviceMust(t, local.Close())
		if budget.Used() != before {
			t.Fatal("embedded local destructor refund", budget.Used(), before)
		}
	}
}
