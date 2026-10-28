package polynomial

// Source: src/math/polynomial/polynomial.cpp:2504-2542,2665-2744;
// src/util/memory_manager.cpp:274-285.

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// ErrorTransport is an explicit supported Go callback profile, not a native
// callback at malloc failure. The source zero shortcut constructs no SOM row.
func TestPolynomialConstantAllocationTransportCannotRetireExecutingFrame(t *testing.T) {
	budget := arena.NewBudget(1 << 26)
	var full *MultivariatePolynomialManager
	var receipt polynomialArithmeticInvocationReceipt
	active := false
	callbacks := 0
	var rejected error
	runtime, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
		if active {
			callbacks++
			c := full.core
			used, stats := budget.Used(), full.core.memory.Stats()
			closing, stage := full.closing, full.cleanupStage
			invocation := c.constantScope.arithmeticScope.invocation
			rejected = full.CloseChecked()
			if budget.Used() != used || c.memory.Stats() != stats || full.closing != closing || full.cleanupStage != stage || c.constantScope.arithmeticScope.invocation != invocation || !c.constantScope.arithmeticScope.executing {
				t.Error("callback retired executing constant storage/history")
			}
		}
		return f
	}})
	serviceMust(t, e)
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: unumeral.IntegerConfig{Memory: runtime, Cells: arena.Config{PageSize: 8, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}})
	serviceMust(t, e)
	full, e = NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := full.core
	var big unumeral.Int
	words := make([]uint32, 80)
	words[0], words[79] = 1, 1
	serviceMust(t, c.integer.SetLimbs(&big, words))
	pad := int64(99996) - runtime.State().LocalSize - 8
	if pad <= 0 {
		t.Fatal("source sync frontier setup", runtime.State())
	}
	held, e := runtime.Allocate(uint64(pad))
	serviceMust(t, e)
	serviceMust(t, runtime.SetMaxSize(1))
	active = true
	_, cause := c.mulAddWithReceipt(c.zero, Polynomial{}, &big, &receipt)
	active = false
	if !numericPhysicalReservationRefusal(cause) || callbacks != 1 {
		t.Fatal("actual constant SET transport absent", cause, callbacks)
	}
	if rejected != ErrPolynomialPending || !c.matchesArithmeticInvocation(&receipt) {
		t.Fatal("executing constant owning guard absent", rejected, cause)
	}
	polynomialImpPhysicalAuditNamedExtra(t, full, nil, []*unumeral.Int{&big}, 0, []umemory.Block{held})
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, c.cleanupArithmeticInvocation(&receipt, false))
	serviceMust(t, c.cleanupArithmeticInvocation(&receipt, true))
	if receipt.frame.bodyCause != cause {
		t.Fatal("owning acknowledgement changed returned first cause")
	}
	old := receipt
	budget.FailAfter(0)
	_, later := c.mulAddWithReceipt(c.zero, Polynomial{}, &big, &receipt)
	polynomialImpPhysicalAuditNamedExtra(t, full, nil, []*unumeral.Int{&big}, 0, []umemory.Block{held})
	if !numericPhysicalReservationRefusal(later) || receipt.invocation == old.invocation {
		t.Fatal("actual later constant acquisition absent", later)
	}
	current, used := *receipt.frame, budget.Used()
	serviceMust(t, c.cleanupArithmeticInvocation(&old, true))
	if !testNumericScopeEqual(*receipt.frame, current) || budget.Used() != used {
		t.Fatal("stale constant receipt consumed later actual acquisition")
	}
	budget.FailAfter(-1)
	serviceMust(t, c.cleanupArithmeticInvocation(&receipt, false))
	serviceMust(t, c.cleanupArithmeticInvocation(&receipt, true))
	if !numericPhysicalReservationRefusal(cause) {
		t.Fatal("new invocation changed already-returned old cause", cause)
	}
	serviceMust(t, c.integer.Del(&big))
	serviceMust(t, full.CloseChecked())
	serviceMust(t, q.CloseChecked())
	serviceMust(t, runtime.Deallocate(&held))
	serviceMust(t, runtime.Close())
	if budget.Used() != 0 {
		t.Fatal("named constant graph retained", budget.Used())
	}
}
