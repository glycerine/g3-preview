// Source: src/math/polynomial/polynomial.cpp:4507-4528,6070-6097,7228-7229,7356-7357;
// src/test/polynomial.cpp:46,126-141.
package polynomial

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// The original driver displays derivative/rename results without an equality
// assertion. These are independent bounded mathematical/action oracles.
func TestPolynomialDerivativeIndependentFiniteValue(t *testing.T) {
	m, p, _ := polynomialLexFixture(t)
	d, e := m.Derivative(p, 0)
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(d))
	defer func() { serviceMust(t, m.core.DecRef(d)) }()
	h, e := m.core.Info(d)
	serviceMust(t, e)
	if h.Size != 3 {
		t.Fatal("independent derivative term count", h)
	}
	sum := int64(0)
	for i := uint32(0); i < h.Size; i++ {
		a, e := m.core.coefficientCore(d, i)
		serviceMust(t, e)
		term, e := m.core.monomialCore(d, i)
		serviceMust(t, e)
		info, e := m.core.monomials.Info(term)
		serviceMust(t, e)
		v := int64(a.NativeScalar())
		for j := uint32(0); j < info.Size; j++ {
			power, e := m.core.monomials.ReadPower(term, j)
			serviceMust(t, e)
			for k := uint32(0); k < power.Degree; k++ {
				v *= []int64{2, 3}[power.Var]
			}
		}
		sum += v
	}
	if sum != 208 {
		t.Fatal("derivative finite value at x2,y3", sum)
	}
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p, d})
}

func TestPolynomialRenameAllDirectoryCellsAndFirstMaximum(t *testing.T) {
	m, p, ms := polynomialLexFixture(t)
	other, e := m.core.MkTerms([]unumeral.Int{unumeral.NewInt(29), unumeral.NewInt(31)}, []Monomial{ms[1], ms[2]})
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(other))
	defer func() { serviceMust(t, m.core.DecRef(other)) }()
	serviceMust(t, m.LexSort(p))
	serviceMust(t, m.LexSort(other))
	serviceMust(t, m.Rename([]uint32{1, 0}))
	for _, v := range []Polynomial{p, other} {
		h, e := m.core.Info(v)
		serviceMust(t, e)
		if h.LexSorted {
			t.Fatal("source rename must clear multi-term lex bit", h)
		}
		term, e := m.core.monomialCore(v, 0)
		serviceMust(t, e)
		power, e := m.core.monomials.ReadPower(term, 0)
		serviceMust(t, e)
		if power.Var != 1 || power.Degree != 2 {
			t.Fatal("first maximal variable and degree after globalrename", power)
		}
		a, e := m.core.coefficientCore(v, 0)
		serviceMust(t, e)
		want := int32(13)
		if v == other {
			want = 29
		}
		if a != unumeral.NewInt(want) {
			t.Fatal("renamed monomial lost paired coefficient", a, want)
		}
	}
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p, other})
}

func polynomialDerivativePhysicalAudit(t *testing.T, m *MultivariatePolynomialManager, ps []Polynomial, borrowed ...Monomial) {
	t.Helper()
	polynomialImpPhysicalAuditValues(t, m, ps)
	c := m.core
	mm := c.monomials
	named := map[Monomial]bool{}
	add := func(a Monomial) {
		if a == (Monomial{}) || named[a] {
			return
		}
		l, e := mm.location(a)
		serviceMust(t, e)
		if l.raw != (umemory.ObjectRef{}) {
			_, e = mm.allocator.ObjectExtent(l.raw)
			serviceMust(t, e)
		}
		if l.published {
			named[a] = true
		}
	}
	add(mm.unit)
	add(m.derivative.generated)
	for _, a := range m.derivative.orphans {
		add(a)
	}
	for _, a := range borrowed {
		add(a)
	}
	if c.generatedActive {
		add(c.generatedMonomial)
	}
	if mm.creation.slot != 0 {
		l := &mm.locations[mm.creation.slot-1]
		if l.live {
			add(Monomial{owner: mm.origin, slot: mm.creation.slot, generation: l.generation})
		}
	}
	for _, b := range [4]*polynomialSOM{&c.som, &c.som2, &c.cheap, &c.cheap2} {
		n, e := b.ms.size(c)
		serviceMust(t, e)
		for i := b.transferred; i < n; i++ {
			a, e := b.ms.monomial(c, i)
			serviceMust(t, e)
			add(a)
		}
		if b.hasPendingReference {
			add(b.pendingMonomial)
		}
	}
	values := append([]Polynomial{c.zero, c.unit, c.pending.value}, ps...)
	for _, p := range values {
		if p == (Polynomial{}) {
			continue
		}
		l, e := c.polynomialLocation(p)
		serviceMust(t, e)
		for i := l.deleteIndex; i < l.initializedMonomials; i++ {
			a, e := c.monomialCore(p, i)
			serviceMust(t, e)
			add(a)
		}
	}
	if uint32(len(named)) != mm.publishedObjects {
		t.Fatal("unnamed actual published monomial", len(named), mm.publishedObjects)
	}
}

func polynomialDerivativeEvenFixture(t *testing.T, existing bool) (*MultivariatePolynomialManager, Polynomial, *Monomial) {
	t.Helper()
	q, _ := rationalManager(t, false)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { panic("derivative does not checkpoint") }, Cooperation: func() error { panic("derivative does not cooperate") }}})
	serviceMust(t, e)
	_, e = m.core.monomials.MkVar()
	serviceMust(t, e)
	ms := make([]Monomial, 3)
	for i := range ms {
		ms[i], e = m.core.monomials.MkVariable(0, uint32(2+2*i))
		serviceMust(t, e)
	}
	p, e := m.core.MkTerms([]unumeral.Int{unumeral.NewInt(1), unumeral.NewInt(1), unumeral.NewInt(1)}, ms)
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(p))
	borrowed := Monomial{}
	if existing {
		borrowed, e = m.core.monomials.MkVariable(0, 1)
		serviceMust(t, e)
	}
	serviceMust(t, m.core.modular.SetZPUint64(2))
	t.Cleanup(func() {
		serviceMust(t, m.CleanupDerivative())
		if borrowed != (Monomial{}) {
			serviceMust(t, m.core.monomials.CleanupValue(&borrowed))
		}
		serviceMust(t, m.core.DecRef(p))
		serviceMust(t, m.CloseChecked())
	})
	return m, p, &borrowed
}

func TestPolynomialDerivativeModularZerosCreatedOnlyCohort(t *testing.T) {
	m, p, _ := polynomialDerivativeEvenFixture(t, false)
	d, e := m.Derivative(p, 0)
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(d))
	defer func() { serviceMust(t, m.core.DecRef(d)) }()
	h, e := m.core.Info(d)
	serviceMust(t, e)
	if h.Size != 0 || m.derivative.active || len(m.derivative.orphans) != 3 {
		t.Fatal("actual successful source zero derivatives", d, m.derivative)
	}
	polynomialDerivativePhysicalAudit(t, m, []Polynomial{p, d})
	if e = m.CloseChecked(); e != ErrPolynomialPending {
		t.Fatal("owning close must retain actual abandoned creations", e)
	}
	// Source cleanup does not DEC these zero coefficient creations.
	before := m.core.monomials.publishedObjects
	serviceMust(t, m.finishDerivativeNumerals())
	if m.core.monomials.publishedObjects != before {
		t.Fatal("source numeral dtors swept orphan cohort")
	}
	serviceMust(t, m.CleanupDerivative())
	if m.core.monomials.publishedObjects != before-3 {
		t.Fatal("explicit created-only owning drain")
	}
	polynomialDerivativePhysicalAudit(t, m, []Polynomial{p, d})
}

func TestPolynomialDerivativeBorrowedZeroAndLaterActualINC(t *testing.T) {
	m, p, borrowed := polynomialDerivativeEvenFixture(t, true)
	d, e := m.Derivative(p, 0)
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(d))
	h, e := m.core.Info(d)
	serviceMust(t, e)
	if h.Size != 0 || len(m.derivative.orphans) != 2 {
		t.Fatal("preexisting zero-ref result acquired false ownership", m.derivative.orphans)
	}
	polynomialDerivativePhysicalAudit(t, m, []Polynomial{p, d}, *borrowed)
	serviceMust(t, m.core.DecRef(d))
	serviceMust(t, m.core.modular.SetZ())
	d, e = m.Derivative(p, 0)
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(d))
	for _, a := range m.derivative.orphans {
		if a != (Monomial{}) {
			t.Fatal("successful INC did not transfer exact cohort token", a)
		}
	}
	polynomialDerivativePhysicalAudit(t, m, []Polynomial{p, d}, *borrowed)
	serviceMust(t, m.core.DecRef(d))
	// d acquired the borrowed result legitimately, and its DEC deletes it.
	// The fixture caller's zero-RC snapshot was never an owning RC token.
	*borrowed = Monomial{}
	// Mark the fixture's original borrowed snapshot as consumed via its exact
	// actual source lifetime, so it is not passed to owning cleanup twice.
}

func TestPolynomialDerivativeEveryReservationNamedPrefix(t *testing.T) {
	for _, modular := range []bool{false, true} {
		t.Run(fmt.Sprint(modular), func(t *testing.T) {
			success := false
			failures := 0
			for ordinal := int64(0); ordinal < 150; ordinal++ {
				t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
					var m *MultivariatePolynomialManager
					var p Polynomial
					if modular {
						m, p, _ = polynomialDerivativeEvenFixture(t, false)
					} else {
						m, p, _ = polynomialLexFixture(t)
					}
					b := m.core.integer.Budget()
					b.FailAfter(ordinal)
					d, e := m.Derivative(p, 0)
					if e != nil {
						failures++
						if !numericPhysicalReservationRefusal(e) {
							t.Fatal("actual transport", e)
						}
					} else {
						success = true
						serviceMust(t, m.core.IncRef(d))
					}
					values := []Polynomial{p}
					if e == nil {
						values = append(values, d)
					}
					polynomialDerivativePhysicalAudit(t, m, values)
					if m.derivative.executing {
						t.Fatal("source execution continued into lexical unwind")
					}
					b.FailAfter(-1)
					serviceMust(t, m.CleanupDerivative())
					if m.core.pending.value != (Polynomial{}) || m.core.pending.raw != (umemory.ObjectRef{}) || m.core.integer.PendingAllocationObject() != (umemory.ObjectRef{}) {
						t.Fatal("derivative cleanup left its actual final factory/numeric pending root", m.core.pending, m.core.integer.PendingCell)
					}
					if e == nil {
						serviceMust(t, m.core.DecRef(d))
					}
				})
				if success {
					break
				}
			}
			if !success || failures == 0 {
				t.Fatal("all reached reservations not exercised", failures, success)
			}
		})
	}
}

func TestPolynomialNativeRenameKeepsActualAllCells(t *testing.T) {
	m, p := polynomialRenameFixture(t)
	b := m.core.integer.Budget()
	b.FailAfter(0)
	t.Cleanup(func() { b.FailAfter(-1) })
	before, used := m.core.memory.Stats(), b.Used()
	serviceMust(t, m.Rename([]uint32{1, 0}))
	polynomialQueryExpect(t, m.core, *p, map[[3]uint32]int64{
		{2, 1, 0}: 11, {0, 2, 0}: 13, {3, 0, 0}: 15, {1, 1, 0}: 17, {}: 19,
	})
	if m.core.memory.Stats() != before || b.Used() != used {
		t.Fatal("native rename changed Runtime accounting")
	}
	polynomialDerivativePhysicalAudit(t, m, []Polynomial{*p})
}

func polynomialRenameFixture(t *testing.T) (*MultivariatePolynomialManager, *Polynomial) {
	t.Helper()
	q, b := rationalManager(t, false)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { panic("rename does not checkpoint") }, Cooperation: func() error { panic("rename does not cooperate") }}})
	serviceMust(t, e)
	for i := 0; i < 2; i++ {
		_, e = m.core.monomials.MkVar()
		serviceMust(t, e)
	}
	powers := [][]MonomialPower{{{0, 1}, {1, 2}}, {{0, 2}}, {{1, 3}}, {{0, 1}, {1, 1}}, nil}
	ms := make([]Monomial, len(powers))
	as := make([]unumeral.Int, len(powers))
	for i := range powers {
		ms[i], e = m.core.monomials.MkMonomial(powers[i])
		serviceMust(t, e)
		as[i] = unumeral.NewInt(int32(11 + 2*i))
	}
	p, e := m.core.MkTerms(as, ms)
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(p))
	t.Cleanup(func() {
		if p != (Polynomial{}) {
			serviceMust(t, m.core.DecRef(p))
		}
		if !m.closed {
			serviceMust(t, m.CloseChecked())
		}
		if b.Used() == 0 {
			t.Fatal("borrowed actual owner disappeared")
		}
	})
	return m, &p
}

func TestMonomialUnitNativeRecycleDuringFinalization(t *testing.T) {
	m := monomialFixture(t)
	b := m.memory.Budget()
	unit := m.unit
	b.FailAfter(0)
	serviceMust(t, m.SourceFinalize())
	b.FailAfter(-1)
	if _, e := m.location(unit); e == nil {
		t.Fatal("unit remained live after finalization")
	}
	monomialPhysicalAudit(t, m)
	serviceMust(t, m.CloseChecked())
}

func TestPolynomialDerivativeNumericPendingIdentity(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{PageSize: 8, SlotLimit: 8, Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { panic("derivative has no checkpoint") }, Cooperation: func() error { panic("derivative has no cooperation") }}})
	serviceMust(t, e)
	_, e = m.core.monomials.MkVar()
	serviceMust(t, e)
	mono, e := m.core.monomials.MkVariable(0, ^uint32(0))
	serviceMust(t, e)
	p, e := m.core.MkTerms([]unumeral.Int{unumeral.NewInt(1)}, []Monomial{mono})
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(p))
	_, cause := m.Derivative(p, 0)
	if !errors.Is(cause, arena.ErrSlotLimit) || q.Integers().PendingCell.Object == (umemory.ObjectRef{}) || m.derivative.numeric.Cell != q.Integers().PendingCell {
		t.Fatal("actual n setter descriptor-refusal root", cause, m.derivative.numeric.Cell.Object, q.Integers().PendingCell)
	}
	polynomialDerivativePhysicalAudit(t, m, []Polynomial{p})
	// Explicit old callee cleanup leaves the derivative's original receipt stale.
	oldBacking := m.derivative.numeric.Cell
	old := oldBacking.Object
	serviceMust(t, q.Integers().CleanupPendingAllocation())
	if q.Integers().PendingCell.Object != (umemory.ObjectRef{}) || m.derivative.numeric.Cell != oldBacking {
		t.Fatal("old physical root or stale receipt changed unexpectedly")
	}
	// A later caller reuses the allocator but owns a distinct object generation.
	out := unumeral.NewInt(11)
	cause = q.Integers().Copy(&out, &q.Integers().Two64)
	if !errors.Is(cause, arena.ErrSlotLimit) || q.Integers().PendingCell.Object == (umemory.ObjectRef{}) || q.Integers().PendingCell.Object == old {
		t.Fatal("later pending identity", cause, q.Integers().PendingCell)
	}
	pending := q.Integers().PendingCell
	polynomialDerivativePhysicalAudit(t, m, []Polynomial{p})
	serviceMust(t, m.CleanupDerivative())
	if q.Integers().PendingCell != pending {
		t.Fatal("derivative cleanup swept later numeric caller")
	}
	serviceMust(t, q.Integers().CleanupPendingAllocation())
	serviceMust(t, q.Integers().Del(&out))
	serviceMust(t, m.core.DecRef(p))
	serviceMust(t, m.CloseChecked())
	serviceMust(t, q.CloseChecked())
	if b.Used() != 0 {
		t.Fatal("actual final owner refund", b.Used())
	}
}

// The public oversized realloc setup reaches a guarded signed-host counter
// domain. This is postphysical TLS synchronization evidence, not OOM on free.
func TestPolynomialDerivativeStoppedNumeralDestructorKeepsSourceCause(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: unumeral.IntegerConfig{Memory: r, Cells: arena.Config{PageSize: 8, Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { panic("derivative no checkpoint") }, Cooperation: func() error { panic("derivative no cooperation") }}})
	serviceMust(t, e)
	_, e = m.core.monomials.MkVar()
	serviceMust(t, e)
	term, e := m.core.monomials.MkVariable(0, 2)
	serviceMust(t, e)
	words := make([]uint32, 40000)
	words[39999] = 1
	a := unumeral.NewInt(0)
	serviceMust(t, q.Integers().SetLimbs(&a, words))
	p, e := m.core.MkTerms([]unumeral.Int{a}, []Monomial{term})
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(p))
	serviceMust(t, q.Integers().Del(&a))
	// Real source result scratch is prepared before the selected publication call.
	serviceMust(t, q.Integers().EnsureTmp(0, 40001))
	contextTLSNearMinimum(t, r)
	b.FailAfter(0)
	_, cause := m.Derivative(p, 0)
	if cause != m.derivative.bodyCause || !numericPhysicalReservationRefusal(m.derivative.bodyCause) || !errors.Is(m.derivative.sourceCause, arena.ErrBounds) || !m.derivative.stopped {
		t.Fatal("actual body publication then stopped a destructor", cause, m.derivative.bodyCause, m.derivative.sourceCause)
	}
	polynomialDerivativePhysicalAudit(t, m, []Polynomial{p})
	before := r.State()
	source := m.derivative.sourceCause
	if m.finishDerivativeNumerals() != source || r.State() != before {
		t.Fatal("source replayed completed physical release")
	}
	b.FailAfter(-1)
	serviceMust(t, m.CleanupDerivative())
	if m.derivative.sourceCause != source {
		t.Fatal("owning abort erased source history")
	}
	contextTLSFinish(t, r, func() error { return m.core.CleanupValue(&p) })
	contextTLSFinish(t, r, m.CloseChecked)
	contextTLSFinish(t, r, q.CloseChecked)
	contextTLSFinish(t, r, r.Close)
	if b.Used() != 0 {
		t.Fatal("actual final whole graph", b.Used())
	}
}

func TestPolynomialDerivativePostphysicalAllocatorRootOwnedByOperation(t *testing.T) {
	for _, mode := range []string{"own", "foreign", "postfree"} {
		t.Run(mode, func(t *testing.T) {
			b := arena.NewBudget(1 << 26)
			r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
			serviceMust(t, e)
			q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: unumeral.IntegerConfig{Memory: r, Cells: arena.Config{PageSize: 8, Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}})
			serviceMust(t, e)
			m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { panic("derivative no checkpoint") }, Cooperation: func() error { panic("derivative no cooperation") }}})
			serviceMust(t, e)
			_, e = m.core.monomials.MkVar()
			serviceMust(t, e)
			term, e := m.core.monomials.MkVariable(0, 2)
			serviceMust(t, e)
			digits := make([]uint32, 40000)
			digits[39999] = 1
			a := unumeral.NewInt(0)
			serviceMust(t, q.Integers().SetLimbs(&a, digits))
			p, e := m.core.MkTerms([]unumeral.Int{a}, []Monomial{term})
			serviceMust(t, e)
			serviceMust(t, m.core.IncRef(p))
			serviceMust(t, q.Integers().Del(&a))
			serviceMust(t, r.SetMaxSize(1))
			_, cause := m.Derivative(p, 0)
			if !numericPhysicalReservationRefusal(cause) || q.Integers().Allocator.Stats().PendingBlocks != 1 || q.Integers().PendingCell.Object != (umemory.ObjectRef{}) {
				t.Fatal("actual postphysical raw allocation retained", cause, q.Integers().Allocator.Stats(), q.Integers().PendingCell)
			}
			polynomialDerivativePhysicalAudit(t, m, []Polynomial{p})
			roots, e := q.Integers().Allocator.PendingRoots()
			serviceMust(t, e)
			if roots != m.derivative.numeric.Allocator || roots[0] == (umemory.Block{}) {
				t.Fatal("exact executing raw receipt", roots, m.derivative.numeric.Allocator)
			}
			rawSize, e := r.Size(roots[0])
			serviceMust(t, e)
			if rawSize != 240016 {
				t.Fatal("actual large source allocation", rawSize)
			}

			if mode == "foreign" {
				// Explicit old callee abort precedes this separate later setter invocation.
				serviceMust(t, q.Integers().Allocator.CleanupPending())
				later := unumeral.NewInt(3)
				laterCause := q.Integers().SetLimbs(&later, digits)
				foreign, e := q.Integers().Allocator.PendingRoots()
				serviceMust(t, e)
				if !numericPhysicalReservationRefusal(laterCause) || foreign == roots || foreign[0] == (umemory.Block{}) {
					t.Fatal("actual foreign raw replacement not reached", laterCause, roots, foreign)
				}
				polynomialDerivativePhysicalAudit(t, m, []Polynomial{p})
				state := r.State()
				serviceMust(t, m.CleanupDerivative())
				after, e := q.Integers().Allocator.PendingRoots()
				serviceMust(t, e)
				if after != foreign || r.State() != state {
					t.Fatal("derivative swept foreign raw replacement", foreign, after)
				}
				serviceMust(t, r.SetMaxSize(0))
				serviceMust(t, q.Integers().Allocator.CleanupPending()) // this later caller's explicit abort
				serviceMust(t, q.Integers().Del(&later))
			} else {
				serviceMust(t, r.SetMaxSize(0))
				if mode == "postfree" {
					contextTLSNearMinimum(t, r)
					cleanup := m.CleanupDerivative()
					if !errors.Is(cleanup, arena.ErrBounds) || m.derivative.numeric.Allocator != ([3]umemory.Block{}) || m.derivative.bodyCause != cause {
						t.Fatal("actual physical consumption not recorded with original cause", cleanup, m.derivative)
					}
					polynomialDerivativePhysicalAudit(t, m, []Polynomial{p})
					state := r.State()
					serviceMust(t, m.CleanupDerivative())
					if r.State() != state || m.derivative.bodyCause != cause {
						t.Fatal("owning retry replayed consumed allocator free")
					}
				} else {
					serviceMust(t, m.CleanupDerivative())
				}
				if q.Integers().Allocator.Stats().PendingBlocks != 0 {
					t.Fatal("derivative cleanup lost its actual allocator raw callee root")
				}
			}

			if mode == "postfree" {
				contextTLSFinish(t, r, func() error { return m.core.CleanupValue(&p) })
				contextTLSFinish(t, r, m.CloseChecked)
				contextTLSFinish(t, r, q.CloseChecked)
				contextTLSFinish(t, r, r.Close)
			} else {
				serviceMust(t, m.core.DecRef(p))
				serviceMust(t, m.CloseChecked())
				serviceMust(t, q.CloseChecked())
				serviceMust(t, r.Close())
			}
			if b.Used() != 0 {
				t.Fatal("actual final refund", b.Used())
			}

		})
	}
}

func TestPolynomialDerivativeTermTraceAllSourceBranches(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{3, 5, 7}, [][]MonomialPower{{{0, 2}, {1, 1}}, {{1, 1}}, nil})
	t.Cleanup(func() { c.integer.Budget().FailAfter(-1); serviceMust(t, m.CleanupDerivative()) })
	var output strings.Builder
	w := &derivativeTermTraceWriter{t: t, m: m, output: &output}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("polynomial")
	c.monomials.config.Trace = tags
	defer tags.Finalize()
	d, e := m.Derivative(p, 0)
	serviceMust(t, e)
	d = polynomialCaller(t, c, d)
	polynomialQueryExpect(t, c, d, map[[3]uint32]int64{{1, 1, 0}: 6})
	if tags.ShouldTrace("polynomial") {
		for _, text := range []string{"x0^2 x1 degree_of x0: 2\n", "x1 degree_of x0: 0\n", "1 degree_of x0: 0\n"} {
			if !strings.Contains(output.String(), text) {
				t.Fatal("literal monomial degree", text, output.String())
			}
		}
		if strings.Count(output.String(), ":4518 ") != 3 || w.calls == 0 || w.records != 3 || w.gate {
			t.Fatal("term records", output.String(), w.calls)
		}
	} else if output.Len() != 0 || w.calls != 0 {
		t.Fatal("erased derivative diagnostic executed")
	}
}

type derivativeTermTraceWriter struct {
	t       *testing.T
	m       *MultivariatePolynomialManager
	output  *strings.Builder
	calls   int
	gate    bool
	records int
}

func (w *derivativeTermTraceWriter) Write(p []byte) (int, error) {
	n, e := w.output.Write(p)
	// Test only this source record, not older polynomial creation/deletion tags.
	if w.m.core.quadraticTrace != nil && w.m.core.quadraticTrace.tags.OutputCallbackActive() {
		if !w.gate {
			w.gate = true
			w.records++
			w.m.core.integer.Budget().FailAfter(0)
		}
		w.calls++
		if w.m.derivative.constructed != 0 {
			w.t.Fatal("term record crossed numeral construction", w.m.derivative.constructed)
		}
		for _, action := range []func() error{w.m.CloseChecked, w.m.SourceFinalize, w.m.CleanupDerivative} {
			if e := action(); !errors.Is(e, ErrPolynomialPending) {
				w.t.Fatal("executing term diagnostic owner", e)
			}
		}
		if e := w.m.core.integer.CloseChecked(); e == nil {
			w.t.Fatal("term output released numeral manager")
		}
	}
	if w.gate && strings.Contains(string(p), "------------------------------------------------") {
		w.gate = false
		w.m.core.integer.Budget().FailAfter(-1)
	}
	return n, e
}
