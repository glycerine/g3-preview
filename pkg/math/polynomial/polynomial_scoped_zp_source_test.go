// Source: src/math/polynomial/polynomial.h:1040-1047;
// src/math/polynomial/polynomial.cpp:3769-3865.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestPolynomialScopedZPActualModeAndPrimeRestoration(t *testing.T) {
	for _, oldField := range []bool{false, true} {
		full := polynomialPSCFullFixture(t)
		c := full.core
		if oldField {
			serviceMust(t, c.modular.SetZPUint64(7))
		}
		var scope polynomialScopedSetZP
		next := util.NewInt(11)
		serviceMust(t, scope.construct(full, &next))
		field, e := c.modular.Modular()
		serviceMust(t, e)
		if !field || (*c.modular.BorrowModulus()) != next {
			t.Fatal("actual reached manager binding", field, (*c.modular.BorrowModulus()))
		}
		serviceMust(t, scope.sourceEnd())
		field, e = c.modular.Modular()
		serviceMust(t, e)
		if field != oldField {
			t.Fatal("source completed body restores old mode", field, oldField)
		}
		if oldField && (*c.modular.BorrowModulus()) != util.NewInt(7) {
			t.Fatal("source completed body restores old prime", (*c.modular.BorrowModulus()))
		}
		// The next loop iteration constructs a fresh source local in this
		// reusable embedded carrier, without an owning-abort action.
		next = util.NewInt(13)
		serviceMust(t, scope.construct(full, &next))
		serviceMust(t, scope.sourceEnd())
		serviceMust(t, scope.cleanup())
		polynomialImpPhysicalAudit(t, full)
	}
}

func TestPolynomialScopedZPFailedBodyDoesNotRestoreMode(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	var scope polynomialScopedSetZP
	c.integer.Budget().FailAfter(0)
	e := scope.construct(full, &c.integer.Two64)
	if !numericPhysicalReservationRefusal(e) || scope.completed {
		t.Fatal("actual failed class constructor", e, scope.completed)
	}
	// set_zp publishes m_z=false BEFORE its actual numeral SET may throw.
	// A failed class body has no completed scoped_set_zp destructor.
	field, x := c.modular.Modular()
	serviceMust(t, x)
	if !field {
		t.Fatal("failed class constructor silently restored source mode")
	}
	polynomialImpPhysicalAuditNamed(t, full, nil, []*util.Int{&scope.p}, 0)
	c.integer.Budget().FailAfter(-1)
	serviceMust(t, scope.cleanup())
	field, x = c.modular.Modular()
	serviceMust(t, x)
	if !field {
		t.Fatal("explicit owning abort invented source mode rollback")
	}
	// The failed numeral remains a guarded source prefix. The actual child
	// owning Close acknowledges it; ordinary reinitialization is not a repair.
	polynomialImpPhysicalAudit(t, full)
}

func TestPolynomialScopedZPFailedMemberCopyRetainsPhysicalPending(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	q, e := util.NewRationalManager(util.RationalConfig{Integers: util.IntegerConfig{Cells: arena.Config{PageSize: 8, SlotLimit: 11, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}})
	serviceMust(t, e)
	t.Cleanup(func() {
		q.Close()
		if budget.Used() != 0 {
			t.Errorf("actual remaining bytes %d", budget.Used())
		}
	})
	full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := full.core
	t.Cleanup(func() { serviceMust(t, full.CloseChecked()) })
	serviceMust(t, c.modular.SetZP(&q.Integers().Two64))
	old := (*c.modular.BorrowModulus())
	var scope polynomialScopedSetZP
	next := util.NewInt(7)
	e = scope.construct(full, &next)
	if !errors.Is(e, arena.ErrSlotLimit) || q.Integers().PendingCell.Object == (umemory.ObjectRef{}) || scope.completed {
		t.Fatal("actual old-prime member Copy descriptor refusal", e, q.Integers().PendingCell, scope.completed)
	}
	if (*c.modular.BorrowModulus()) != old {
		t.Fatal("unreached new prime SET")
	}
	polynomialImpPhysicalAuditNamed(t, full, nil, []*util.Int{&scope.p}, 0)
	serviceMust(t, scope.cleanup())
	if q.Integers().PendingCell.Object != (umemory.ObjectRef{}) {
		t.Fatal("scoped class failed member root not drained by explicit owning abort")
	}
	if (*c.modular.BorrowModulus()) != old {
		t.Fatal("owning abort changed untouched source manager prime")
	}
	polynomialImpPhysicalAudit(t, full)
}

// Guarded helper domain, not a claim that an ordinary prime restoration grows
// a saved numeral. The real Copy replacement prefix supplies the source guard.
func TestPolynomialScopedZPCompletedMemberGuardIsSticky(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	z := full.core.integer
	var scope polynomialScopedSetZP
	next := util.NewInt(7)
	serviceMust(t, scope.construct(full, &next))
	var large util.Int
	limbs := make([]uint32, 8192)
	limbs[0], limbs[len(limbs)-1] = 17, 1
	serviceMust(t, z.SetLimbs(&large, limbs))
	t.Cleanup(func() { z.Budget().FailAfter(-1); serviceMust(t, scope.cleanup()); serviceMust(t, z.Del(&large)) })
	serviceMust(t, z.Copy(&scope.p, &z.Two64))
	z.Budget().FailAfter(0)
	body := z.Copy(&scope.p, &large)
	if !numericPhysicalReservationRefusal(body) || (scope.p).Cell != z.FailedRef {
		t.Fatal("actual dangling replacement", body, scope.p)
	}
	saved := scope.p
	polynomialImpPhysicalAuditNamed(t, full, nil, []*util.Int{&large, &scope.p}, 0)
	first := scope.sourceEnd()
	if !errors.Is(first, arena.ErrInvalidSlot) || scope.p != saved || scope.stage != 1 || scope.memberEnded {
		t.Fatal("reached source member destruction", first, scope)
	}
	field, e := full.core.modular.Modular()
	serviceMust(t, e)
	if field {
		t.Fatal("completed class body did not precede stopped member")
	}
	if second := scope.sourceEnd(); second != first || scope.p != saved {
		t.Fatal("source destructor replay", first, second)
	}
	z.Budget().FailAfter(-1)
	serviceMust(t, scope.cleanup())
	polynomialImpPhysicalAuditNamed(t, full, nil, []*util.Int{&large}, 0)
}
