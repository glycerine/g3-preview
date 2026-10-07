// Source: src/math/polynomial/polynomial.cpp:5154-5194;
// src/util/mpz.cpp:2054-2069; src/util/scoped_numeral.h:31-33.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// This is the actual signed-host TLS owner-primitive guard profile. The
// public Runtime operations stage the guard; ordinary mathematical OOM is
// not claimed. The source checkpoint arms it after the real R/NUM/ref prefix.
func TestPolynomialDividesYoungerScopedNUMStopsOlderSourceSuffix(t *testing.T) {
	budget := arena.NewBudget(1 << 26)
	rt, e := util.NewRuntime(util.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	cfg := serviceConfig(budget).Integers
	cfg.Memory = rt
	rationals, e := util.NewRationalManager(util.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: rationals.Integers(), Rationals: rationals, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := full.core
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	var a, d util.Int
	words := make([]uint32, 64)
	words[63] = 2
	serviceMust(t, c.integer.SetLimbs(&a, words))
	words[0], words[63] = 7, 3
	serviceMust(t, c.integer.SetLimbs(&d, words))
	divisor, e := c.MkConstantTakeCopy(&a)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(divisor))
	term, e := c.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	p, e := c.MkTerms([]util.Int{d}, []Monomial{term})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	answer, e := c.Divides(divisor, p)
	serviceMust(t, e)
	if answer {
		t.Fatal("independent nonzero BIG remainder")
	}
	serviceMust(t, c.integer.EnsureTmp(0, 64))
	serviceMust(t, c.integer.EnsureTmp(1, 64))
	contextTLSNearMinimum(t, rt)
	lane, e := rt.Allocate(8)
	serviceMust(t, e)
	calls := 0
	c.config.Checkpoint = func() bool {
		calls++
		if calls == 1 {
			if !c.dividesLocal.active || !c.dividesLocal.executing || c.dividesLocal.constructed != 2 {
				t.Fatal("guard staged before actual source declarations")
			}
			delta := int64(-99900) - rt.State().LocalSize
			x := rt.Reallocate(&lane, uint64(16)+uint64(delta)-8)
			if !errors.Is(x, arena.ErrBounds) {
				t.Fatal("actual signed allocation guard", x)
			}
		}
		return true
	}
	var receipt polynomialDividesInvocationReceipt
	answer, returned := c.dividesWithReceipt(divisor, p, &receipt)
	s := receipt.frame
	if answer || !errors.Is(returned, arena.ErrBounds) || s == nil || s.stage != 0 || s.sourceCause == nil || !s.active || s.monomial.value == (Monomial{}) {
		t.Fatal("actual younger source DT must stop before older NUM/ref", answer, returned, s)
	}
	captured, source := returned.Error(), s.sourceCause
	polynomialImpPhysicalAuditNamedExtra(t, full, []Polynomial{p, divisor}, []*util.Int{&a, &d}, 0, []util.Block{lane})
	own := c.cleanupDividesInvocation(&receipt, false)
	if own != nil && !errors.Is(own, arena.ErrBounds) {
		t.Fatal("completed younger then older own suffix", own)
	}
	if c.dividesInvocationSourceCause(&receipt) != source || c.finishDividesSource() != source {
		t.Fatal("own acknowledgment erased first source stop")
	}
	recovery, e := rt.Allocate(2000000)
	serviceMust(t, e)
	serviceMust(t, c.cleanupDividesInvocation(&receipt, true))
	if s.active || c.dividesInvocationSourceCause(&receipt) != source || returned.Error() != captured {
		t.Fatal("true own phase altered source history")
	}
	polynomialImpPhysicalAuditNamedExtra(t, full, []Polynomial{p, divisor}, []*util.Int{&a, &d}, 0, []util.Block{lane, recovery})
	c.config.Checkpoint = func() bool { return true }
	answer, e = c.Divides(c.zero, c.zero)
	serviceMust(t, e)
	if !answer || returned.Error() != captured {
		t.Fatal("early route changed returned cause")
	}
	answer, e = c.Divides(divisor, p)
	serviceMust(t, e)
	if answer || returned.Error() != captured || !errors.Is(returned, arena.ErrBounds) {
		t.Fatal("new invocation changed old returned cause")
	}
	used := budget.Used()
	serviceMust(t, c.cleanupDividesInvocation(&receipt, true))
	if budget.Used() != used {
		t.Fatal("stale invocation consumed new persistent rows")
	}
	serviceMust(t, c.DecRef(p))
	serviceMust(t, c.DecRef(divisor))
	serviceMust(t, c.integer.Del(&a))
	serviceMust(t, c.integer.Del(&d))
	serviceMust(t, rt.Deallocate(&lane))
	serviceMust(t, full.CloseChecked())
	serviceMust(t, rationals.CloseChecked())
	e = rt.Deallocate(&recovery)
	if e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (util.Block{})) {
		t.Fatal("recovery consumed outcome", e)
	}
	contextTLSFinish(t, rt, rt.Close)
	if budget.Used() != 0 {
		t.Fatal("complete named owner graph", budget.Used())
	}
}
