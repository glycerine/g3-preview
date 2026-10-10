// Source: src/math/polynomial/polynomial.cpp:5094-5114;
// src/util/memory_manager.cpp:274-285.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

// Returning error transport is a supported Go callback. The polynomial borrow
// is the real internal unit, so caller references cannot incidentally protect
// the executing scalar frame against destruction of its whole owner.
func TestPolynomialScalarExactDivActualTransportGuardsWholeOwner(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	var full *MultivariatePolynomialManager
	active := false
	calls := 0
	var rejected error
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
		if active {
			calls++
			closing, stage, used := full.closing, full.cleanupStage, b.Used()
			rejected = full.CloseChecked()
			if closing != full.closing || stage != full.cleanupStage || used != b.Used() {
				t.Error("returning scalar transport mutated whole owner", rejected, closing, stage, full.closing, full.cleanupStage)
			}
		}
		return f
	}})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	q, e := util.NewRationalManager(util.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	full, e = NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := full.core
	var modulus util.Int
	words := make([]uint32, 2048)
	for i := range words {
		words[i] = ^uint32(0)
	}
	words[len(words)-1] = 511
	serviceMust(t, c.integer.SetLimbs(&modulus, words))
	serviceMust(t, c.modular.SetZP(&modulus))
	pad := int64(99996) - rt.State().LocalSize - 8
	if pad <= 0 {
		t.Fatal("source TLS frontier", rt.State())
	}
	held, e := rt.Allocate(uint64(pad))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	active = true
	two := util.NewInt(2)
	result, cause := c.ExactDivScalar(c.unit, &two)
	active = false
	if calls != 1 || !numericPhysicalReservationRefusal(cause) {
		t.Error("actual scalar division transport", calls, cause)
	}
	if rejected != ErrPolynomialPending {
		t.Error("executing scalar whole owner accepted destruction", rejected)
	}
	polynomialImpPhysicalAuditNamedExtra(t, full, nil, []*util.Int{&modulus}, 0, []umemory.Block{held})
	old := c.exactDivision.scalarReceipt
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, c.CleanupExactDivision())
	if old.frame.bodyCause != cause || old.frame.active || (numericIntegerManager_scalarExactAllocationRoots(c.integer, &old)) != nil {
		t.Fatal("own acknowledgment changed scalar body history or acquisition liveness")
	}
	if result != (Polynomial{}) {
		serviceMust(t, c.CleanupValue(&result))
	}
	serviceMust(t, c.integer.Del(&modulus))
	serviceMust(t, full.CloseChecked())
	serviceMust(t, q.CloseChecked())
	serviceMust(t, rt.Deallocate(&held))
	serviceMust(t, rt.Close())
	if b.Used() != 0 {
		t.Fatal("whole owner refund", b.Used())
	}
}
