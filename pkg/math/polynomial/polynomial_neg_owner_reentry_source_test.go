// Source: src/math/polynomial/polynomial.cpp:5940-5951.
// Returning Runtime allocation-error transport is an explicit Go owner guard
// profile; native destructor reentry is not asserted.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialNegActualCopyTransportRefusesOwnerRetirement(t *testing.T) {
	budget := arena.NewBudget(1 << 26)
	var full *MultivariatePolynomialManager
	active := false
	calls := 0
	var rejected error
	rt, e := util.NewRuntime(util.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64, Transport: func(f util.Failure) error {
		if active {
			calls++
			c := full.core
			used, stats := budget.Used(), c.memory.Stats()
			closing, stage := full.closing, full.cleanupStage
			inv := c.invocation
			rejected = full.CloseChecked()
			if rejected != ErrPolynomialPending || budget.Used() != used || c.memory.Stats() != stats || full.closing != closing || full.cleanupStage != stage || !c.negativeExecuting || c.invocation != inv {
				t.Error("callback changed actual executing Neg owner", rejected, full.closing, full.cleanupStage)
			}
		}
		return f
	}})
	serviceMust(t, e)
	q, e := util.NewRationalManager(util.RationalConfig{Integers: util.IntegerConfig{Memory: rt, Cells: arena.Config{PageSize: 8, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}}})
	serviceMust(t, e)
	full, e = NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := full.core
	words := make([]uint32, 80)
	words[0], words[79] = 1, 1
	var big util.Int
	serviceMust(t, c.integer.SetLimbs(&big, words))
	term, e := c.monomials.MkMonomial([]MonomialPower{{0, 1}})
	serviceMust(t, e)
	p, e := c.MkTerms([]util.Int{big}, []Monomial{term})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	pad := int64(99996) - rt.State().LocalSize - 8
	if pad <= 0 {
		t.Fatal("source frontier setup", rt.State())
	}
	held, e := rt.Allocate(uint64(pad))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	active = true
	var receipt polynomialNegInvocationReceipt
	_, body := c.negWithReceipt(p, &receipt)
	active = false
	if calls != 1 || !numericPhysicalReservationRefusal(body) {
		t.Fatal("actual Copy transport not reached", calls, body)
	}
	polynomialImpPhysicalAuditNamedExtra(t, full, []Polynomial{p}, []*util.Int{&big}, 0, []util.Block{held})
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, c.cleanupNegInvocation(&receipt, false))
	serviceMust(t, c.cleanupNegInvocation(&receipt, true))
	serviceMust(t, rt.Deallocate(&held))
	serviceMust(t, c.integer.Del(&big))
	serviceMust(t, c.DecRef(p))
	serviceMust(t, full.CloseChecked())
	serviceMust(t, q.CloseChecked())
}
