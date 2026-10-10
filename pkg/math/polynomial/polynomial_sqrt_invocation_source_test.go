// Source: src/math/polynomial/polynomial.cpp:5954-6068; src/util/obj_ref.h;
// src/util/scoped_numeral.h.
package polynomial

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialSqrtInvocationGuardAndActualAcquisition(t *testing.T) {
	m := polynomialPrimitiveFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{1, 2, 1}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupSqrt()); serviceMust(t, out.Close()) })
	var receipt polynomialSqrtInvocationReceipt
	ok, e := m.sqrtWithReceipt(c.zero, out, &receipt)
	serviceMust(t, e)
	if !ok || receipt.invocation != 0 {
		t.Fatal("zero route constructs no lexical scope", receipt)
	}
	c.integer.Budget().FailAfter(0)
	_, e = m.sqrtWithReceipt(p, out, &receipt)
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, out.root.value, out.root.incoming}, nil, out.header)
	c.integer.Budget().FailAfter(-1)
	if !numericPhysicalReservationRefusal(e) || !m.matchesSqrtInvocation(&receipt) {
		t.Fatal("actual guarded begin receipt", e, receipt)
	}
	first := receipt
	serviceMust(t, m.cleanupSqrtInvocation(&receipt, false))
	serviceMust(t, m.cleanupSqrtInvocation(&receipt, true))
	ok, e = m.sqrtWithReceipt(p, out, &receipt)
	serviceMust(t, e)
	if !ok || receipt.invocation == first.invocation {
		t.Fatal("actual unique subsequent invocation", receipt, first)
	}
	polynomialQueryExpect(t, c, out.root.value, map[[3]uint32]int64{{1, 0, 0}: 1, {}: 1})
	used := c.integer.Budget().Used()
	serviceMust(t, m.cleanupSqrtInvocation(&first, true))
	if used != c.integer.Budget().Used() {
		t.Fatal("stale receipt changed later owned backing")
	}
	before := receipt
	copied := *out
	_, e = m.sqrtWithReceipt(p, &copied, &receipt)
	if e != ErrPolynomialOwner || receipt != before {
		t.Fatal("rejected owner leaves receipt unchanged", e, receipt, before)
	}
}

func TestPolynomialSqrtReceiptPreservesOlderActualFalseQuotientCohort(t *testing.T) {
	m := polynomialPrimitiveFixture(t)
	c := m.core
	old := polynomialQueryTerms(t, c, []int32{1, 3, 1}, [][]MonomialPower{{{0, 4}}, {{0, 3}}, nil})
	square := polynomialQueryTerms(t, c, []int32{1, 2, 1}, [][]MonomialPower{{{1, 2}}, {{1, 1}}, nil})
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupSqrt()); serviceMust(t, out.Close()) })
	var previous, current polynomialSqrtInvocationReceipt
	ok, e := m.sqrtWithReceipt(old, out, &previous)
	serviceMust(t, e)
	if ok {
		t.Fatal("literal three is not divisible by two")
	}
	var root Monomial
	var slot int
	for i, a := range m.squareRoot.orphans {
		if a != (Monomial{}) {
			if root != (Monomial{}) {
				t.Fatal("source has exactly one abandoned created quotient")
			}
			root = a
			slot = i
		}
	}
	if root == (Monomial{}) || m.squareRoot.orphanInvocations[slot] != previous.invocation {
		t.Fatal("actual returned-created old root authority")
	}
	polynomialSqrtPhysicalAudit(t, m, []Polynomial{old, square}, nil, out.header)
	ok, e = m.sqrtWithReceipt(square, out, &current)
	serviceMust(t, e)
	if !ok || current.invocation == previous.invocation {
		t.Fatal("actual later source invocation")
	}
	serviceMust(t, m.cleanupSqrtInvocation(&current, false))
	serviceMust(t, m.cleanupSqrtInvocation(&current, true))
	if m.squareRoot.orphans[slot] != root || m.squareRoot.orphanInvocations[slot] != previous.invocation {
		t.Fatal("later callee abort crossed actual created cohort boundary")
	}
	if _, e = c.monomials.Info(root); e != nil {
		t.Fatal("earlier actual monomial lost", e)
	}
	polynomialQueryExpect(t, c, out.root.value, map[[3]uint32]int64{{0, 1, 0}: 1, {}: 1})
	polynomialSqrtPhysicalAudit(t, m, []Polynomial{old, square, out.root.value}, nil, out.header)
	serviceMust(t, m.cleanupSqrtInvocation(&previous, true))
	if m.squareRoot.orphans[slot] != (Monomial{}) || m.squareRoot.orphanInvocations[slot] != 0 {
		t.Fatal("matched old root owning consumption")
	}
}

// This genuine public TLS release is an owner-primitive signed-host guarded
// domain. It does not claim ordinary mathematical free-failure reachability.
func TestPolynomialSqrtReceiptCompletedNUMBeforeUnjoinedAbort(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	cfg.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { panic("destructor helper checkpoint unreachable") }, Cooperation: func() error { panic("destructor helper cooperation unreachable") }}})
	serviceMust(t, e)
	var term Monomial
	defer func() {
		contextTLSFinish(t, rt, m.CleanupSqrt)
		contextTLSFinish(t, rt, func() error { return m.core.monomials.CleanupValue(&term) })
		contextTLSFinish(t, rt, m.CloseChecked)
		contextTLSFinish(t, rt, q.CloseChecked)
		contextTLSFinish(t, rt, rt.Close)
		if b.Used() != 0 {
			t.Fatal("whole named graph", b.Used())
		}
	}()
	// Ordinary source create/INC/DEC warms the real ID freelists before the
	// bounded overflow recipe, isolating the intended NUM destructor frontier.
	var warm [4]Polynomial
	for i := range warm {
		v := unumeral.NewInt(int32(7 + i))
		warm[i], e = m.core.MkConstantTake(&v)
		serviceMust(t, e)
		serviceMust(t, m.core.IncRef(warm[i]))
	}
	for i := range warm {
		serviceMust(t, m.core.DecRef(warm[i]))
	}
	x, e := m.core.monomials.MkVar()
	serviceMust(t, e)
	w, e := m.core.monomials.MkVariable(x, 3)
	serviceMust(t, e)
	serviceMust(t, m.core.monomials.IncRef(w))
	serviceMust(t, m.core.monomials.DecRef(w))
	term, e = m.core.monomials.MkVariable(x, 2)
	serviceMust(t, e)
	var receipt polynomialSqrtInvocationReceipt
	serviceMust(t, m.beginSquareRootWithReceipt(&receipt))
	s := &m.squareRoot
	// The same actual returned-created monomial helper publishes one abort root
	// before any INC; this is not a reconstructed registry or refcount owner.
	orphan, ok, e := m.squareRootMonomial(0, term, Monomial{})
	serviceMust(t, e)
	if !ok || orphan == (Monomial{}) {
		t.Fatal("actual helper created root")
	}
	s.constructNumeral(1)
	s.constructNumeral(2)
	s.constructNumeral(3)
	s.referenceCompleted = true
	serviceMust(t, q.Integers().SetInt32(&s.values[0], 13))
	serviceMust(t, q.Integers().Copy(&s.values[1], &q.Integers().Two64))
	serviceMust(t, q.Integers().Copy(&s.values[2], &q.Integers().Two64))
	serviceMust(t, q.Integers().Copy(&s.values[3], &q.Integers().Two64))
	serviceMust(t, q.Integers().EnsureOwnedCapacity(&s.values[3], 40000))
	older := [3]unumeral.Int{s.values[0], s.values[1], s.values[2]}
	contextTLSNearMinimum(t, rt)
	first := m.finishSquareRoot(nil)
	if !errors.Is(first, arena.ErrBounds) || !s.stopped || s.stage != 0 || s.sourceCause == nil || [3]unumeral.Int{s.values[0], s.values[1], s.values[2]} != older {
		t.Fatal("actual source younger NUM stop preserves older fields", first, s.stage)
	}
	polynomialSqrtPhysicalAudit(t, m, nil, nil, 0, term)
	saved := s.sourceCause
	state := rt.State()
	again := m.finishSquareRoot(nil)
	if !errors.Is(again, arena.ErrBounds) || s.sourceCause != saved || rt.State() != state {
		t.Fatal("literal stopped retry must not own-ack", again)
	}
	contextTLSFinish(t, rt, func() error { return m.cleanupSqrtInvocation(&receipt, false) })
	if s.sourceCause != saved || s.stage != 0 || !s.stopped {
		t.Fatal("completed own phase erased source frontier")
	}
	if len(s.orphans) != 1 || s.orphans[0] != orphan || s.orphanInvocations[0] != receipt.invocation {
		t.Fatal("completed NUM phase aborted unjoined root")
	}
	if _, e = m.core.monomials.Info(orphan); e != nil {
		t.Fatal("actual unjoined root lost", e)
	}
	polynomialSqrtPhysicalAudit(t, m, nil, nil, 0, term)
	contextTLSFinish(t, rt, func() error { return m.cleanupSqrtInvocation(&receipt, true) })
	if s.orphans[0] != (Monomial{}) || s.orphanInvocations[0] != 0 || s.sourceCause != saved || s.stage != 0 {
		t.Fatal("exact owning abort/history")
	}
	polynomialSqrtPhysicalAudit(t, m, nil, nil, 0, term)
}

// Real TLS allocation publishes a raw token before its limit refusal. These
// six RAW locals have no source destructor until the end label is entered.
func TestPolynomialSqrtInvocationCompletedPhasePreservesUnenteredPerfectSquare(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	cfg.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { panic("hi Copy refuses before checkpoint") }, Cooperation: func() error { panic("hi Copy refuses before cooperation") }}})
	serviceMust(t, e)
	c := m.core
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	term, e := c.monomials.MkVariable(0, 2)
	serviceMust(t, e)
	digits := make([]uint32, 40000)
	digits[39999] = 1
	a := unumeral.Int{}
	serviceMust(t, q.Integers().SetLimbs(&a, digits))
	p, e := c.MkTerms([]unumeral.Int{a}, []Monomial{term})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	serviceMust(t, q.Integers().Del(&a))
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	var receipt polynomialSqrtInvocationReceipt
	defer func() {
		serviceMust(t, rt.SetMaxSize(0))
		contextTLSFinish(t, rt, func() error { return m.cleanupSqrtInvocation(&receipt, true) })
		contextTLSFinish(t, rt, out.Close)
		contextTLSFinish(t, rt, func() error { return c.CleanupValue(&p) })
		contextTLSFinish(t, rt, m.CloseChecked)
		contextTLSFinish(t, rt, q.CloseChecked)
		contextTLSFinish(t, rt, rt.Close)
		if b.Used() != 0 {
			t.Fatal("named whole graph", b.Used())
		}
	}()
	serviceMust(t, rt.SetMaxSize(1))
	_, cause := m.sqrtWithReceipt(p, out, &receipt)
	child := q.Integers().PerfectSquare
	if !numericPhysicalReservationRefusal(cause) || !child.Active || child.EndEntered || child.Raw[0] == (umemory.Block{}) || m.squareRoot.perfectSquare.Invocation != child.Generation {
		t.Fatal("actual unentered raw child", cause, child)
	}
	polynomialSqrtPhysicalAudit(t, m, []Polynomial{p, out.root.value, out.root.incoming}, nil, out.header)
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, m.cleanupSqrtInvocation(&receipt, false))
	if q.Integers().PerfectSquare != child || m.squareRoot.perfectSquare.Invocation != child.Generation {
		t.Fatal("completed phase consumed body-exception RAW child")
	}
	polynomialSqrtPhysicalAudit(t, m, []Polynomial{p, out.root.value, out.root.incoming}, nil, out.header)
	serviceMust(t, m.cleanupSqrtInvocation(&receipt, true))
	if q.Integers().PerfectSquare.Active {
		t.Fatal("matched unjoined abort retained child")
	}
	polynomialSqrtPhysicalAudit(t, m, []Polynomial{p, out.root.value, out.root.incoming}, nil, out.header)
}

func TestPolynomialSqrtInvocationPreservesLaterForeignPerfectSquare(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	cfg.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { panic("hi Copy refuses before checkpoint") }, Cooperation: func() error { panic("hi Copy refuses before cooperation") }}})
	serviceMust(t, e)
	c := m.core
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	term, e := c.monomials.MkVariable(0, 2)
	serviceMust(t, e)
	digits := make([]uint32, 40000)
	digits[39999] = 1
	a := unumeral.Int{}
	serviceMust(t, q.Integers().SetLimbs(&a, digits))
	p, e := c.MkTerms([]unumeral.Int{a}, []Monomial{term})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	serviceMust(t, q.Integers().Del(&a))
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	var receipt polynomialSqrtInvocationReceipt
	defer func() {
		serviceMust(t, rt.SetMaxSize(0))
		contextTLSFinish(t, rt, func() error { return m.cleanupSqrtInvocation(&receipt, true) })
		contextTLSFinish(t, rt, out.Close)
		contextTLSFinish(t, rt, func() error { return c.CleanupValue(&p) })
		contextTLSFinish(t, rt, m.CloseChecked)
		contextTLSFinish(t, rt, q.CloseChecked)
		contextTLSFinish(t, rt, rt.Close)
		if b.Used() != 0 {
			t.Fatal("named whole graph", b.Used())
		}
	}()
	serviceMust(t, rt.SetMaxSize(1))
	_, cause := m.sqrtWithReceipt(p, out, &receipt)
	child := q.Integers().PerfectSquare
	if !numericPhysicalReservationRefusal(cause) || !child.Active || child.EndEntered || child.Raw[0] == (umemory.Block{}) || m.squareRoot.perfectSquare.Invocation != child.Generation {
		t.Fatal("actual unentered raw child", cause, child)
	}
	polynomialSqrtPhysicalAudit(t, m, []Polynomial{p, out.root.value, out.root.incoming}, nil, out.header)
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, m.cleanupSqrtInvocation(&receipt, false))
	if q.Integers().PerfectSquare != child || m.squareRoot.perfectSquare.Invocation != child.Generation {
		t.Fatal("completed phase consumed body-exception RAW child")
	}
	polynomialSqrtPhysicalAudit(t, m, []Polynomial{p, out.root.value, out.root.incoming}, nil, out.header)
	// Caller explicitly aborts the old Integer invocation, then starts its own
	// actual replacement at the same embedded scope address.
	serviceMust(t, q.Integers().CleanupPerfectSquare())
	foreignInput, foreignOutput := unumeral.Int{}, unumeral.Int{}
	serviceMust(t, q.Integers().SetLimbs(&foreignInput, digits))
	defer func() {
		contextTLSFinish(t, rt, func() error { return q.Integers().Del(&foreignInput) })
		contextTLSFinish(t, rt, func() error { return q.Integers().Del(&foreignOutput) })
	}()
	serviceMust(t, rt.SetMaxSize(1))
	_, foreignCause := q.Integers().IsPerfectSquare(&foreignInput, &foreignOutput)
	foreign := q.Integers().PerfectSquare
	if !numericPhysicalReservationRefusal(foreignCause) || !foreign.Active || foreign.Generation == child.Generation || foreign.Raw[0] == (umemory.Block{}) {
		t.Fatal("actual replacement invocation", foreignCause, foreign)
	}
	polynomialSqrtPhysicalAudit(t, m, []Polynomial{p, out.root.value, out.root.incoming}, []*unumeral.Int{&foreignInput, &foreignOutput}, out.header)
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, m.cleanupSqrtInvocation(&receipt, true))
	if q.Integers().PerfectSquare != foreign {
		t.Fatal("old matched Sqrt receipt consumed foreign child")
	}
	polynomialSqrtPhysicalAudit(t, m, []Polynomial{p, out.root.value, out.root.incoming}, []*unumeral.Int{&foreignInput, &foreignOutput}, out.header)
	serviceMust(t, q.Integers().CleanupPerfectSquare())
}

func TestPolynomialSqrtReceiptOlderCohortTransfersAtActualLaterINC(t *testing.T) {
	m := polynomialPrimitiveFixture(t)
	c := m.core
	old := polynomialQueryTerms(t, c, []int32{1, 3, 1}, [][]MonomialPower{{{0, 4}}, {{0, 3}}, nil})
	square := polynomialQueryTerms(t, c, []int32{1}, [][]MonomialPower{{{0, 2}}})
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupSqrt()); serviceMust(t, out.Close()) })
	var previous, current polynomialSqrtInvocationReceipt
	ok, e := m.sqrtWithReceipt(old, out, &previous)
	serviceMust(t, e)
	if ok {
		t.Fatal("literal three is not divisible by two")
	}
	var root Monomial
	var slot int
	for i, a := range m.squareRoot.orphans {
		if a != (Monomial{}) {
			if root != (Monomial{}) {
				t.Fatal("source has exactly one abandoned created quotient")
			}
			root = a
			slot = i
		}
	}
	if root == (Monomial{}) || m.squareRoot.orphanInvocations[slot] != previous.invocation {
		t.Fatal("actual returned-created old root authority")
	}
	polynomialSqrtPhysicalAudit(t, m, []Polynomial{old, square}, nil, out.header)
	ok, e = m.sqrtWithReceipt(square, out, &current)
	serviceMust(t, e)
	if !ok || current.invocation == previous.invocation {
		t.Fatal("actual later source invocation")
	}
	serviceMust(t, m.cleanupSqrtInvocation(&current, false))
	serviceMust(t, m.cleanupSqrtInvocation(&current, true))
	if m.squareRoot.orphans[slot] != (Monomial{}) || m.squareRoot.orphanInvocations[slot] != 0 {
		t.Fatal("actual later successful INC must transfer the old created root")
	}
	if _, e = c.monomials.Info(root); e != nil {
		t.Fatal("caller result actual monomial lost", e)
	}
	polynomialQueryExpect(t, c, out.root.value, map[[3]uint32]int64{{1, 0, 0}: 1})
	polynomialSqrtPhysicalAudit(t, m, []Polynomial{old, square, out.root.value}, nil, out.header)
	serviceMust(t, m.cleanupSqrtInvocation(&previous, true))
	polynomialQueryExpect(t, c, out.root.value, map[[3]uint32]int64{{1, 0, 0}: 1})
	if m.squareRoot.orphans[slot] != (Monomial{}) || m.squareRoot.orphanInvocations[slot] != 0 {
		t.Fatal("stale old authority reappeared after successful INC")
	}
}

func TestPolynomialSqrtInvocationBigNumeralEveryReservation(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 1024; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}})
			c := m.core
			a := unumeral.Int{}
			serviceMust(t, c.integer.SetLimbs(&a, []uint32{3, 256}))
			unit, e := c.monomials.Unit()
			serviceMust(t, e)
			term, e := c.monomialCore(p, 0)
			serviceMust(t, e)
			base, e := c.MkTerms([]unumeral.Int{a, unumeral.NewInt(3)}, []Monomial{term, unit})
			serviceMust(t, e)
			serviceMust(t, c.integer.Del(&a))
			serviceMust(t, c.IncRef(base))
			defer func() { serviceMust(t, c.DecRef(base)) }()
			square, e := c.Mul(base, base)
			serviceMust(t, e)
			serviceMust(t, c.IncRef(square))
			defer func() { serviceMust(t, c.DecRef(square)) }()
			r, e := NewPolynomialRef(c)
			serviceMust(t, e)
			var receipt polynomialSqrtInvocationReceipt
			defer func() {
				c.integer.Budget().FailAfter(-1)
				serviceMust(t, m.cleanupSqrtInvocation(&receipt, false))
				serviceMust(t, m.cleanupSqrtInvocation(&receipt, true))
				serviceMust(t, r.Close())
			}()
			c.integer.Budget().FailAfter(ordinal)
			ok, cause := m.sqrtWithReceipt(square, r, &receipt)
			polynomialSqrtPhysicalAudit(t, m, []Polynomial{p, base, square, r.root.value, r.root.incoming}, nil, r.header)
			if cause == nil {
				if !ok {
					t.Fatal("actual big-coefficient square rejected")
				}
				success = true
			} else if !numericPhysicalReservationRefusal(cause) {
				t.Fatal("actual big numeric refusal", cause)
			}
			c.integer.Budget().FailAfter(-1)
			serviceMust(t, m.cleanupSqrtInvocation(&receipt, false))
			serviceMust(t, m.cleanupSqrtInvocation(&receipt, true))
			if c.integer.PendingAllocationObject() != (umemory.ObjectRef{}) || c.pending.raw != (umemory.ObjectRef{}) || c.pending.value != (Polynomial{}) {
				t.Fatal("actual operation-owned pending roots not consumed")
			}
			if cause == nil {
				product, e := c.Mul(r.root.value, r.root.value)
				serviceMust(t, e)
				serviceMust(t, c.IncRef(product))
				equal, e := c.Equal(product, square)
				serviceMust(t, e)
				serviceMust(t, c.DecRef(product))
				if !equal {
					t.Fatal("big exact root squared identity")
				}
			}
		})
		if success {
			t.Log("big actual success horizon", ordinal)
			break
		}
	}
	if !success {
		t.Fatal("bounded big source operation never completed")
	}
}
