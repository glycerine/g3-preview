// Source: src/math/polynomial/polynomial.cpp:5114-5156;
// src/util/scoped_numeral.h; src/util/obj_ref.h.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"testing"
)

// The public signed-counter recipe is a guarded host/source undefined-domain
// profile, not valid allocation exhaustion. The ExactDiv body and cancellation
// are real; no numeral metadata, pointer or destructor callback is fabricated.
func TestPolynomialExactDivActualNUMStopPreservesOlderReference(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	cfg.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := NewRationalManager(RationalConfig{Integers: cfg})
	serviceMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: rt, Profile: cfg.Profile})
	serviceMust(t, e)
	serviceMust(t, mm.IncManagerRef())
	visits := uint32(0)
	var m *MultivariatePolynomialCore
	m, e = NewMultivariatePolynomialCore(MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Monomials: mm, Checkpoint: func() bool {
		visits++
		if visits == 2 {
			exactDivisionTLSNearMinimum(t, rt)
			return false
		}
		return visits < 3
	}, Cooperation: func() error { return nil }})
	serviceMust(t, e)
	_, e = mm.MkVar()
	serviceMust(t, e)
	// Ordinary source object creation/INC/DEC reserves the real polynomial
	// ID recycle backing before the guarded-domain destructor-only boundary.
	var warmed [6]Polynomial
	for i := range warmed {
		a := NewInt(int32(i + 7))
		warmed[i], e = m.MkConstantTake(&a)
		serviceMust(t, e)
		serviceMust(t, m.IncRef(warmed[i]))
	}
	for _, p := range warmed {
		serviceMust(t, m.DecRef(p))
	}

	warm, e := mm.MkVariable(0, 9)
	serviceMust(t, e)
	serviceMust(t, mm.IncRef(warm))
	serviceMust(t, mm.DecRef(warm))
	big, negative := Int{}, Int{}
	words := make([]uint32, 30001)
	words[0] = 1
	words[len(words)-1] = 1
	serviceMust(t, m.integer.SetLimbs(&big, words))
	serviceMust(t, m.integer.Copy(&negative, &big))
	serviceMust(t, m.integer.Neg(&negative))
	x2, e := mm.MkVariable(0, 2)
	serviceMust(t, e)
	x, e := mm.MkVariable(0, 1)
	serviceMust(t, e)
	unit, e := mm.Unit()
	serviceMust(t, e)
	p, e := m.MkTerms([]Int{big, negative}, []Monomial{x2, unit})
	serviceMust(t, e)
	serviceMust(t, m.IncRef(p))
	divisor, e := m.MkTerms([]Int{NewInt(1), NewInt(1)}, []Monomial{x, unit})
	serviceMust(t, e)
	serviceMust(t, m.IncRef(divisor))
	defer func() {
		contextTLSFinish(t, rt, m.CleanupExactDivision)
		contextTLSFinish(t, rt, func() error { return m.CleanupValue(&divisor) })
		contextTLSFinish(t, rt, func() error { return m.CleanupValue(&p) })
		contextTLSFinish(t, rt, func() error { return m.integer.Del(&negative) })
		contextTLSFinish(t, rt, func() error { return m.integer.Del(&big) })
		seenConsumedDrop := false
		contextTLSFinish(t, rt, func() error {
			e := m.CloseChecked()
			if errors.Is(e, arena.ErrBounds) && m.closeStage == 2 && m.som.cleanupDropConsumed && m.som.cleanupDropLast {
				// Exact current slot and actual DEC outcome prove the released
				// root, before the next owning attempt; no global-count inference.
				x, read := m.som.ms.monomial(m, m.som.cleanupIndex)
				serviceMust(t, read)
				l, lookup := mm.location(x)
				serviceMust(t, lookup)
				if l.raw != (memory.ObjectRef{}) || l.stage != 2 || !l.live {
					t.Fatal("actual consumed-last leaf", l)
				}
				seenConsumedDrop = true
			}
			return e
		})
		if !seenConsumedDrop {
			t.Fatal("real SOM consumed-last cleanup boundary not reached")
		}
		dropCause := mm.DecManagerRef()
		if dropCause != nil {
			if !errors.Is(dropCause, arena.ErrBounds) || mm.managerRefs != 0 {
				t.Fatal("actual consumed manager DEC", dropCause, mm.managerRefs)
			}
			contextTLSFinish(t, rt, mm.CloseChecked)
		}
		contextTLSFinish(t, rt, q.CloseChecked)
		serviceMust(t, rt.Close())
		if b.Used() != 0 {
			t.Fatal("actual whole named graph", b.Used())
		}
	}()
	var receipt polynomialExactDivInvocationReceipt
	_, first := m.exactDivWithReceipt(p, divisor, &receipt)
	if !errors.Is(first, ErrUPolynomialCanceled) || !errors.Is(first, arena.ErrBounds) || visits != 2 || m.exactDivision.stage != 0 {
		t.Fatal("actual canceled body then NUM source stop", first, visits, m.exactDivision.stage)
	}
	polynomialExactDivisionPhysical(t, m, []Polynomial{p, divisor}, &big, &negative)
	if !errors.Is(m.exactDivInvocationSourceCause(&receipt), arena.ErrBounds) {
		t.Fatal("matched callee cause unavailable")
	}
	retained := receipt
	beforeRejected := b.Used()
	if _, reject := m.exactDivWithReceipt(p, divisor, &retained); !errors.Is(reject, ErrPolynomialPending) || retained != receipt || b.Used() != beforeRejected {
		t.Fatal("rejected entry acquired/mutated invocation", reject)
	}
	older := m.exactDivision.monomial.value
	state := rt.State()
	again := m.exactDivisionCleanup(false)
	if !errors.Is(again, arena.ErrBounds) || rt.State() != state || m.exactDivision.monomial.value != older || m.exactDivision.stage != 0 {
		t.Fatal("ordinary source retry acknowledged consumed NUM or dropped older reference", again, m.exactDivision.stage)
	}
	contextTLSFinish(t, rt, func() error { return m.cleanupExactDivInvocation(&receipt, false) })
	contextTLSFinish(t, rt, func() error { return m.cleanupExactDivInvocation(&receipt, true) })
	if !errors.Is(m.exactDivision.sourceCause, arena.ErrBounds) || m.exactDivision.stage != 0 || !errors.Is(m.exactDivision.bodyCause, ErrUPolynomialCanceled) {
		t.Fatal("owning cleanup changed source first cause/frontier")
	}
}

// This real input retains more than one megabyte before the checkpoint. The
// first public host-domain request therefore uses a four-megabyte negative
// lane instead of the smaller fixture's one-megabyte assumption.
func exactDivisionTLSNearMinimum(t *testing.T, r *memory.Runtime) {
	t.Helper()
	first, e := r.Allocate(8)
	serviceMust(t, e)
	second, e := r.Allocate(8)
	serviceMust(t, e)
	defer func() {
		if first != (memory.Block{}) {
			contextTLSFinish(t, r, func() error { return r.Deallocate(&first) })
		}
		if second != (memory.Block{}) {
			contextTLSFinish(t, r, func() error { return r.Deallocate(&second) })
		}
	}()
	delta := int64(-4000000)
	request := uint64(16) + uint64(delta) - 8
	if e = r.Reallocate(&first, request); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	serviceMust(t, r.Deallocate(&first))
	state := r.State()
	if state.LocalSize != 0 || state.GlobalSize >= 0 {
		t.Fatal("bounded public initial lane", state)
	}
	target := int64(math.MinInt64) + 1000 - state.GlobalSize + 16
	request = uint64(target) + 16 - 8
	if e = r.Reallocate(&second, request); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	serviceMust(t, r.Deallocate(&second))
	if state = r.State(); state.GlobalSize != math.MinInt64+1000 || state.LocalSize != 0 {
		t.Fatal("public near-minimum lane", state)
	}
}
