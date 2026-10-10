// Source: src/math/polynomial/polynomial.cpp:5114-5156;
// src/util/scoped_numeral.h; src/util/obj_ref.h.
package polynomial

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

// The public signed-counter recipe is a guarded host/source undefined-domain
// profile, not valid allocation exhaustion. The ExactDiv body and cancellation
// are real; no numeral metadata, pointer or destructor callback is fabricated.
func TestPolynomialExactDivActualNUMStopPreservesOlderReference(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	cfg.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := util.NewRationalManager(util.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: rt, Profile: cfg.Profile})
	serviceMust(t, e)
	serviceMust(t, mm.IncManagerRef())
	visits := uint32(0)
	var m *MultivariatePolynomialCore
	m, e = NewMultivariatePolynomialCore(MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool {
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
		a := util.NewInt(int32(i + 7))
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
	big, negative := util.Int{}, util.Int{}
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
	p, e := m.MkTerms([]util.Int{big, negative}, []Monomial{x2, unit})
	serviceMust(t, e)
	serviceMust(t, m.IncRef(p))
	divisor, e := m.MkTerms([]util.Int{util.NewInt(1), util.NewInt(1)}, []Monomial{x, unit})
	serviceMust(t, e)
	serviceMust(t, m.IncRef(divisor))
	defer func() {
		contextTLSFinish(t, rt, m.CleanupExactDivision)
		contextTLSFinish(t, rt, func() error { return m.CleanupValue(&divisor) })
		contextTLSFinish(t, rt, func() error { return m.CleanupValue(&p) })
		contextTLSFinish(t, rt, func() error { return m.integer.Del(&negative) })
		contextTLSFinish(t, rt, func() error { return m.integer.Del(&big) })
		contextTLSFinish(t, rt, m.CloseChecked)
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
	// Actual source-loop additions with no prelude receipt remain foreign.
	// Owning abort retires only the acquired prelude rows before caller roots;
	// whole Close consequently no longer owns their last reference.
	type foreignRow struct {
		term        Monomial
		coefficient util.Int
		refs        uint32
	}
	var foreign []foreignRow
	refCount := func(term Monomial) (uint32, error) {
		location, e := mm.location(term)
		if e != nil {
			return 0, e
		}
		return mm.allocator.ReadUint32(location.raw, 0)
	}
	count, e := m.som.ms.size(m)
	serviceMust(t, e)
	for i := uint32(0); i < count; i++ {
		if m.som.monomialOwners[i] != receipt.prelude.frame.append.invocation {
			term, e := m.som.ms.monomial(m, i)
			serviceMust(t, e)
			coefficient, e := m.som.as.coefficient(m, i)
			serviceMust(t, e)
			refs, e := refCount(term)
			serviceMust(t, e)
			foreign = append(foreign, foreignRow{term, coefficient, refs})
		}
	}
	contextTLSFinish(t, rt, func() error { return m.cleanupExactDivInvocation(&receipt, true) })
	if m.matchesArithmeticInvocation(&receipt.prelude) {
		t.Fatal("exact completed prelude still owns rows")
	}
	count, e = m.som.ms.size(m)
	serviceMust(t, e)
	if count != uint32(len(foreign)) {
		t.Fatal("owning abort changed foreign row count", count, len(foreign))
	}
	for i, want := range foreign {
		term, e := m.som.ms.monomial(m, uint32(i))
		serviceMust(t, e)
		coefficient, e := m.som.as.coefficient(m, uint32(i))
		serviceMust(t, e)
		refs, e := refCount(term)
		serviceMust(t, e)
		position, e := m.som.positions.Get(term)
		serviceMust(t, e)
		if term != want.term || coefficient != want.coefficient || refs != want.refs || position != uint32(i) {
			t.Fatal("owning abort changed foreign row identity/value/RC/position", i)
		}
	}
	for _, term := range []Monomial{x2, unit} {
		position, e := m.som.positions.Get(term)
		serviceMust(t, e)
		if position != ^uint32(0) {
			t.Fatal("owned prelude position not retired", term, position)
		}
	}
	if !errors.Is(m.exactDivision.sourceCause, arena.ErrBounds) || m.exactDivision.stage != 0 || !errors.Is(m.exactDivision.bodyCause, ErrUPolynomialCanceled) {
		t.Fatal("owning cleanup changed source first cause/frontier")
	}
}

// This real input retains more than one megabyte before the checkpoint. The
// first public host-domain request therefore uses a four-megabyte negative
// lane instead of the smaller fixture's one-megabyte assumption.
func exactDivisionTLSNearMinimum(t *testing.T, r *umemory.Runtime) {
	t.Helper()
	first, e := r.Allocate(8)
	serviceMust(t, e)
	second, e := r.Allocate(8)
	serviceMust(t, e)
	defer func() {
		if first != (umemory.Block{}) {
			contextTLSFinish(t, r, func() error { return r.Deallocate(&first) })
		}
		if second != (umemory.Block{}) {
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
