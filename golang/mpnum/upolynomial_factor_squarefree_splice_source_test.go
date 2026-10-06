// Source: src/math/polynomial/upolynomial_factorization.cpp:1030-1034,1020-1299.
// Licensed under the MIT license. Source authors: Leonardo de Moura; Dejan (t-dejanj).
package mpnum

import (
	"fmt"
	"reflect"
	"testing"
)

func TestUPolynomialFactorSquareFreeRejectsSplicedDiagnosticAcquisition(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source DEBUG diagnostic erased")
	}
	reached := false
	for ordinal := int64(0); ordinal < 64; ordinal++ {
		found := false
		okay := t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			z, s := preparationPhysicalFixture(t)
			m := s.manager
			out, e := m.NewFactors()
			serviceMust(t, e)
			var f, big upolyVectorState
			var value Int
			serviceMust(t, m.set(&f, []Int{NewInt(-6), NewInt(1), NewInt(1)}))
			serviceMust(t, z.z.SetString(&value, "1208925819614629174706176"))
			serviceMust(t, m.set(&big, []Int{value, NewInt(1), NewInt(1)}))
			serviceMust(t, z.z.Del(&value))
			var old, next upolyFactorSquareFreeInvocationReceipt
			defer func() {
				z.z.budget.FailAfter(-1)
				serviceMust(t, s.cleanupFactorSquareFreeInvocation(&next, true))
				serviceMust(t, s.CleanupRetained())
				serviceMust(t, out.Close())
				serviceMust(t, m.reset(&f))
				m.freeBacking(&f)
				serviceMust(t, m.reset(&big))
				m.freeBacking(&big)
			}()
			_, e = s.factorSquareFreeWithReceipt(m.stateView(&f), out, 1, DefaultUPolynomialFactorParams(), &old)
			serviceMust(t, e)
			z.z.budget.FailAfter(ordinal)
			_, body := s.factorSquareFreeWithReceipt(m.stateView(&big), out, 1, DefaultUPolynomialFactorParams(), &next)
			big0, e := m.read(m.stateView(&big), 0)
			serviceMust(t, e)
			primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{out}, big0)
			if body == nil || next.frame != nil || !s.diagnostic.active {
				return
			}
			found = true
			before := s.diagnostic
			oldFrame := *old.frame
			used := z.z.budget.Used()
			forged := old
			forged.diagnostic = next.diagnostic
			forged.history = next.history
			if x := s.cleanupFactorSquareFreeInvocation(&forged, true); x != ErrUPolynomialOwner {
				t.Fatal("old retired scope accepted later genuine diagnostic acquisition", x)
			}
			if !reflect.DeepEqual(before, s.diagnostic) || !reflect.DeepEqual(oldFrame, *old.frame) || used != z.z.budget.Used() {
				t.Fatal("spliced old receipt changed later roots/history")
			}
			if s.factorSquareFreeInvocationSourceCause(&forged) != nil {
				t.Fatal("spliced diagnostic changed old source history query")
			}
			primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{out}, big0)
		})
		if !okay {
			return
		}
		if found {
			reached = true
			break
		}
	}
	if !reached {
		t.Fatal("no later actual diagnostic retained numeric acquisition")
	}
}
