// Source: src/math/polynomial/polynomial.cpp:5964-6068.
// Returning Runtime allocation-error transport is an explicit Go owner guard
// profile; native destructor reentry is not asserted.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialSqrtActualAllocationTransportRefusesOwnerRetirement(t *testing.T) {
	polynomialSqrtAllocationTransport(t, false)
}

func TestPolynomialSqrtAppendAllocationUsesActualInvocation(t *testing.T) {
	polynomialSqrtAllocationTransport(t, true)
}

func polynomialSqrtAllocationTransport(t *testing.T, appendOnly bool) {
	t.Helper()
	budget := arena.NewBudget(1 << 26)
	var full *MultivariatePolynomialManager
	active := false
	calls := 0
	var rejected error
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
		if active {
			calls++
			c := full.core
			used, stats := budget.Used(), c.memory.Stats()
			closing, stage := full.closing, full.cleanupStage
			inv := full.squareRoot.invocation
			rejected = full.CloseChecked()
			if rejected != ErrPolynomialPending || budget.Used() != used || c.memory.Stats() != stats || full.closing != closing || full.cleanupStage != stage || !full.squareRoot.executing || full.squareRoot.invocation != inv {
				t.Error("callback changed actual executing Sqrt owner", rejected, full.closing, full.cleanupStage)
			}
			if e := full.CleanupSqrt(); e != ErrPolynomialPending || budget.Used() != used || c.memory.Stats() != stats {
				t.Error("callback retired executing square-root resources", e)
			}
		}
		return f
	}})
	serviceMust(t, e)
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: unumeral.IntegerConfig{Memory: rt, Cells: arena.Config{PageSize: 8, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}})
	serviceMust(t, e)
	full, e = NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := full.core
	words := make([]uint32, 80)
	words[79] = 1
	var big unumeral.Int
	serviceMust(t, c.integer.SetLimbs(&big, words))
	p, e := c.MkConstantTakeCopy(&big)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	pad := int64(99996) - rt.State().LocalSize - 8
	if pad <= 0 {
		t.Fatal("source frontier setup", rt.State())
	}
	held, e := rt.Allocate(uint64(pad))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	active = true
	var receipt polynomialSqrtInvocationReceipt
	var body error
	if appendOnly {
		serviceMust(t, full.beginSquareRootWithReceipt(&receipt))
		body = full.squareRootAdd(&c.som, &big, c.monomials.unit)
		if !polynomialNegNumericPending(&full.squareRoot.numeric) {
			t.Fatal("actual append allocation absent from invocation receipt", body)
		}
		if !full.squareRoot.numeric.NumericBound(true) {
			t.Fatal("actual executing square-root numeric policy is unbound")
		}
		body = full.finishSquareRoot(body)
		if !full.squareRoot.numeric.NumericBound(false) || full.squareRoot.numeric.NumericBound(true) {
			t.Fatal("completed numeric policy confused cleanup with execution")
		}
		foreign, e := unumeral.NewIntegerModularManager(c.integer)
		serviceMust(t, e)
		if full.squareRoot.numeric.NumericModularOwner(foreign) {
			t.Fatal("numeric policy accepted a foreign modular receiver")
		}
		serviceMust(t, foreign.Close())

	} else {
		_, body = full.sqrtWithReceipt(p, out, &receipt)
	}
	active = false
	if calls != 1 || !numericPhysicalReservationRefusal(body) {
		t.Fatal("actual square-root allocation transport not reached", calls, body)
	}
	polynomialImpPhysicalAuditNamedExtra(t, full, []Polynomial{p}, []*unumeral.Int{&big}, out.header, []umemory.Block{held})
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, full.cleanupSqrtInvocation(&receipt, false))
	serviceMust(t, full.cleanupSqrtInvocation(&receipt, true))
	if appendOnly && full.squareRoot.numeric.NumericBound(false) {
		t.Fatal("retired numeric policy retained cleanup authority")
	}
	serviceMust(t, rt.Deallocate(&held))
	serviceMust(t, c.integer.Del(&big))
	serviceMust(t, c.DecRef(p))
	serviceMust(t, out.Close())
	serviceMust(t, full.CloseChecked())
	serviceMust(t, q.CloseChecked())
	serviceMust(t, rt.Close())
	if budget.Used() != 0 {
		t.Fatal("square-root transport cleanup retained charge", budget.Used())
	}
}
