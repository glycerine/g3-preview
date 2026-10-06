// Source: src/util/ref_vector.h:96-100; src/math/polynomial/polynomial.cpp:5626.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialReferenceVectorPushActualINCBeforeGrowthAndExactAbort(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	out, e := NewPolynomialRefVector(c)
	serviceMust(t, e)
	t.Cleanup(func() { m.originalBudget.FailAfter(-1); serviceMust(t, out.Close()) })
	before, e := c.readWord(p, 0)
	serviceMust(t, e)
	var r polynomialReferenceVectorPushReceipt
	m.originalBudget.FailAfter(0)
	returned := out.pushWithReceipt(p, &r)
	after, e := c.readWord(p, 0)
	serviceMust(t, e)
	if returned == nil || after != before+1 || out.state.incoming != p || !r.matches() || !out.state.pushActive {
		t.Fatal("actual INC erased before source growth failure", returned, before, after)
	}
	subresultantNamedAudit(t, m, out, []Polynomial{p}, nil)
	old := r
	copiedFrame := *r.frame
	forged := r
	forged.frame = &copiedFrame
	serviceMust(t, forged.cleanup())
	copiedOwner := *out
	forged = r
	forged.owner = &copiedOwner
	serviceMust(t, forged.cleanup())
	after, e = c.readWord(p, 0)
	serviceMust(t, e)
	if after != before+1 || out.state.incoming != p {
		t.Fatal("copied push authority consumed original")
	}
	serviceMust(t, r.cleanup())
	after, e = c.readWord(p, 0)
	serviceMust(t, e)
	if after != before || out.state.incoming != (Polynomial{}) {
		t.Fatal("exact incoming owning abort", before, after)
	}
	body := out.state.pushBodyCause
	if body != returned {
		t.Fatal("first push body cause", body, returned)
	}
	m.originalBudget.FailAfter(-1)
	serviceMust(t, out.pushWithReceipt(p, &r))
	serviceMust(t, old.cleanup())
	got, e := out.Get(0)
	serviceMust(t, e)
	if got != p || returned != body || r.invocation == old.invocation {
		t.Fatal("later actual push retargeted old authority")
	}
	subresultantNamedAudit(t, m, out, []Polynomial{p}, nil)
}

// Returning allocation transport reaches the actual Push frame after INC.
func TestPolynomialReferenceVectorPushGrowthTransportRefusesExecutingOwner(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	var out *PolynomialRefVector
	var receipt polynomialReferenceVectorPushReceipt
	calls := 0
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: budget, Transport: func(f memory.Failure) error {
		if out != nil {
			calls++
			s := &out.state
			cause := s.pushCleanupCause
			if !s.pushExecuting || s.incoming == (Polynomial{}) {
				t.Fatal("transport did not reach actual post-INC Push")
			}
			if x := out.Close(); x != ErrPolynomialPending || out.closing {
				t.Fatal("executing actual vector closed", x)
			}
			if x := receipt.cleanup(); x != ErrPolynomialPending || s.pushCleanupCause != cause {
				t.Fatal("executing receipt ownACK mutated", x)
			}
			if x := out.Push(s.incoming); x != ErrPolynomialPending {
				t.Fatal("same actual vector reentry", x)
			}
		}
		return f
	}})
	serviceMust(t, e)
	cfg := serviceConfig(budget).Integers
	cfg.Memory = rt
	cfg.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	rational, e := NewRationalManager(RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: rational.z, Rationals: rational, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := m.core
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	term, e := c.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	p, e := c.MkTerms([]Int{NewInt(1), NewInt(1)}, []Monomial{term, c.monomials.unit})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	out, e = NewPolynomialRefVector(c)
	serviceMust(t, e)
	budget.FailAfter(0)
	returned := out.pushWithReceipt(p, &receipt)
	if returned == nil || calls != 1 {
		t.Fatal("real growth transport", returned, calls)
	}
	subresultantNamedAudit(t, m, out, []Polynomial{p}, nil)
	serviceMust(t, receipt.cleanup())
	budget.FailAfter(-1)
	serviceMust(t, out.Close())
	out = nil
	serviceMust(t, c.DecRef(p))
	serviceMust(t, m.CloseChecked())
	serviceMust(t, rational.CloseChecked())
	serviceMust(t, rt.Close())
	if budget.Used() != 0 {
		t.Fatal("actual owner teardown", budget.Used())
	}
}
