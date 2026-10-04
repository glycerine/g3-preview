// Source: src/math/polynomial/polynomial.cpp:6237-6425; src/math/polynomial/polynomial.h:942-966.
// Licensed under the MIT license.
// Author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialSubstitutionIntegerIndependentIdentity(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{3, 5, 7, 11}, [][]MonomialPower{{{0, 2}, {1, 1}}, {{1, 2}}, {{2, 1}}, nil})
	xs, _ := variableHelperArray(t, m.memory, []uint32{0, 2})
	values := []Int{{val: 2}, {val: -3}}
	result, e := m.SubstituteIntegers(p, 2, xs, values)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(result))
	t.Cleanup(func() { serviceMust(t, m.DecRef(result)) })
	polynomialQueryExpect(t, m, result, map[[3]uint32]int64{{0, 1, 0}: 12, {0, 2, 0}: 5, {0, 0, 0}: -10})
}
func TestPolynomialSubstitutionRationalDenominatorClearing(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1, 1, 1}, [][]MonomialPower{{{0, 2}, {2, 1}}, {{1, 1}}, {{0, 1}, {2, 1}}})
	xs, _ := variableHelperArray(t, m.memory, []uint32{0, 1})
	values := []Rational{{num: Int{val: 3}, den: Int{val: 2}}, {num: Int{val: 3}, den: Int{val: 1}}}
	result, e := m.SubstituteRationals(p, 2, xs, values)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(result))
	t.Cleanup(func() { serviceMust(t, m.DecRef(result)) })
	// D=2^2, yielding 15*x2+12, then actual source R.mk(true)
	// divides the common content3.
	polynomialQueryExpect(t, m, result, map[[3]uint32]int64{{0, 0, 1}: 5, {0, 0, 0}: 4})
}
func TestPolynomialSubstitutionPolynomialFractionIndependentIdentity(t *testing.T) {
	m := polynomialCellFixture(t)
	r := polynomialQueryTerms(t, m, []int32{1, 3, 5}, [][]MonomialPower{{{0, 2}}, {{0, 1}, {1, 1}}, nil})
	p := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{1, 1}}, nil})
	q := polynomialQueryTerms(t, m, []int32{2}, [][]MonomialPower{nil})
	out, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	serviceMust(t, m.SubstitutePolynomialFraction(r, 0, p, q, out))
	result, e := out.Get()
	serviceMust(t, e)
	// (y+1)^2+6y(y+1)+20 =7y²+8y+21.
	polynomialQueryExpect(t, m, result, map[[3]uint32]int64{{0, 2, 0}: 7, {0, 1, 0}: 8, {0, 0, 0}: 21})
}

type substitutionReadControl struct {
	containsCalls, valueCalls uint32
	value                     Rational
	match                     bool
}

func (c *substitutionReadControl) Contains(x uint32) (bool, error) {
	c.containsCalls++
	return c.match && x == 0, nil
}
func (c *substitutionReadControl) Rational(x uint32) (*Rational, error) {
	c.valueCalls++
	if !c.match || x != 0 {
		panic("unreached source rational value")
	}
	return &c.value, nil
}
func TestPolynomialSubstitutionAbsentAndReachedRawFields(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, {{1, 1}}})
	control := substitutionReadControl{}
	stats, count := m.memory.Stats(), m.memory.State()
	oldAS, oldMS := m.som.as.raw, m.som.ms.raw
	a, e := m.SubstituteRationalMap(p, &control)
	serviceMust(t, e)
	if a != p || control.containsCalls != 2 || control.valueCalls != 0 || m.som.as.raw != oldAS || m.som.ms.raw != oldMS || m.substitution.found.raw != (memory.Block{}) {
		t.Fatal("source absent branch before numeric/SOM/found vars", a, control)
	}
	// The helper's variable arrays really allocate, but no scalar roots or SOM
	// storage were created on the early-return path.
	if m.memory.Stats().LiveBlocks <= stats.LiveBlocks || m.memory.State().GlobalCount <= count.GlobalCount {
		t.Fatal("actual scoped maximum vectors")
	}
	control = substitutionReadControl{match: true, value: Rational{num: NewInt(1), den: NewInt(-2)}}
	a, e = m.SubstituteRationalMap(p, &control)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(a))
	polynomialQueryExpect(t, m, a, map[[3]uint32]int64{{0, 0, 0}: 1, {0, 1, 0}: -2})
	serviceMust(t, m.DecRef(a))
	// DEN0 is source-defined denominator clearing: 1+0*y, not an ordinary
	// Rational reader. The actual zero coefficient is removed by source SOM.
	control.value = Rational{num: NewInt(1), den: Int{}}
	a, e = m.SubstituteRationalMap(p, &control)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(a))
	polynomialQueryExpect(t, m, a, map[[3]uint32]int64{{0, 0, 0}: 1})
	serviceMust(t, m.DecRef(a))
	control.value = Rational{}
	a, e = m.SubstituteRationalMap(p, &control)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(a))
	polynomialQueryExpect(t, m, a, map[[3]uint32]int64{})
	serviceMust(t, m.DecRef(a))
	polynomialSubstitutionPhysical(t, m, nil, []Polynomial{p}, nil)
}
func TestPolynomialSubstitutionIntegerBoundedGridAndScalarForward(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{3, -5, 7, 11}, [][]MonomialPower{{{0, 3}, {1, 1}}, {{0, 1}, {2, 2}}, {{2, 1}}, nil})
	xs, xsRaw := variableHelperArray(t, m.memory, []uint32{0, 2})
	for x := int32(-3); x <= 3; x++ {
		for z := int32(-2); z <= 2; z++ {
			v := []Int{NewInt(x), NewInt(z)}
			a, e := m.SubstituteIntegers(p, 2, xs, v)
			serviceMust(t, e)
			serviceMust(t, m.IncRef(a))
			want := map[[3]uint32]int64{}
			c := int64(3 * x * x * x)
			if c != 0 {
				want[[3]uint32{0, 1, 0}] = c
			}
			c = int64(-5*x*z*z + 7*z + 11)
			if c != 0 {
				want[[3]uint32{}] = c
			}
			polynomialQueryExpect(t, m, a, want)
			serviceMust(t, m.DecRef(a))
		}
	}
	v := NewInt(-2)
	a, e := m.SubstituteInteger(p, 0, &v)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(a))
	polynomialQueryExpect(t, m, a, map[[3]uint32]int64{{0, 1, 0}: -24, {0, 0, 2}: 10, {0, 0, 1}: 7, {0, 0, 0}: 11})
	serviceMust(t, m.DecRef(a))
	polynomialSubstitutionPhysical(t, m, nil, []Polynomial{p}, []memory.Block{*xsRaw})
}
func TestPolynomialSubstitutionFractionZeroAndAliasProfiles(t *testing.T) {
	m := polynomialCellFixture(t)
	r := polynomialQueryTerms(t, m, []int32{1, 3, 5}, [][]MonomialPower{{{0, 2}}, {{0, 1}, {1, 1}}, nil})
	p := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{1, 1}}, nil})
	q := polynomialQueryTerms(t, m, []int32{2}, [][]MonomialPower{nil})
	out, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	serviceMust(t, out.Assign(r))
	serviceMust(t, m.SubstitutePolynomialFraction(r, 2, Polynomial{}, Polynomial{}, out))
	got, e := out.Get()
	serviceMust(t, e)
	if got != r || m.substitution.scope.active {
		t.Fatal("degree0 ignores p/q and returns original", got)
	}
	serviceMust(t, m.SubstitutePolynomialFraction(r, 0, p, q, out))
	got, e = out.Get()
	serviceMust(t, e)
	polynomialQueryExpect(t, m, got, map[[3]uint32]int64{{0, 2, 0}: 7, {0, 1, 0}: 8, {0, 0, 0}: 21})
	// Equal p/q and reused output exercise actual reference replacement and
	// temporary product construction, not a denominator shortcut.
	serviceMust(t, m.SubstitutePolynomialFraction(r, 0, q, q, out))
	got, e = out.Get()
	serviceMust(t, e)
	polynomialQueryExpect(t, m, got, map[[3]uint32]int64{{0, 1, 0}: 12, {0, 0, 0}: 24})
	serviceMust(t, m.SubstitutePolynomialFraction(r, 0, m.zero, q, out))
	got, e = out.Get()
	serviceMust(t, e)
	polynomialQueryExpect(t, m, got, map[[3]uint32]int64{{0, 0, 0}: 20})
	serviceMust(t, m.SubstitutePolynomialFraction(r, 0, q, m.zero, out))
	got, e = out.Get()
	serviceMust(t, e)
	polynomialQueryExpect(t, m, got, map[[3]uint32]int64{{0, 0, 0}: 4})
	polynomialSubstitutionPhysical(t, m, []*PolynomialRef{out}, []Polynomial{r, p, q}, nil)
}
func TestPolynomialSubstitutionWarmAbsentBranchActualStorage(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{3, 5}, [][]MonomialPower{{{0, 2}}, {{1, 3}}})
	xs, _ := variableHelperArray(t, m.memory, []uint32{2})
	values := []Rational{{num: NewInt(2), den: NewInt(3)}}
	a, e := m.SubstituteRationals(p, 1, xs, values)
	serviceMust(t, e)
	if a != p {
		t.Fatal(a)
	}
	before, count, used, state := m.memory.Stats(), uint64(0), m.integer.budget.Used(), m.memory.State()
	count, e = m.memory.AllocationCount()
	serviceMust(t, e)
	m.integer.budget.FailAfter(0)
	allocs := testing.AllocsPerRun(25, func() {
		a, e := m.SubstituteRationals(p, 1, xs, values)
		if e != nil || a != p {
			panic("source absent warm branch")
		}
	})
	polynomialSubstitutionPhysical(t, m, nil, []Polynomial{p}, []memory.Block{xs.raw}, &values[0].num, &values[0].den)
	after, e := m.memory.AllocationCount()
	serviceMust(t, e)
	if allocs != 0 || before != m.memory.Stats() || count != after || state != m.memory.State() || used != m.integer.budget.Used() {
		t.Fatal("source allocation-free absent path", allocs)
	}
	m.integer.budget.FailAfter(-1)
}
func TestPolynomialSubstitutionColdOwnersAndSourceInputLifetime(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{3}, [][]MonomialPower{{{0, 2}}})
	xs, b := variableHelperArray(t, m.memory, []uint32{0})
	stale := xs
	serviceMust(t, m.memory.Reallocate(b, 16))
	_, e := m.SubstituteIntegers(p, 1, stale, []Int{NewInt(2)})
	if !errors.Is(e, memory.ErrRuntimeOwner) {
		t.Fatal("captured real source pointer accessed during scoped ctor", e)
	}
	if m.substitution.scope.active {
		t.Fatal("failed scope ctor has no reset or completed numeric locals")
	}
	polynomialSubstitutionPhysical(t, m, nil, []Polynomial{p}, []memory.Block{*b})
	shallow := *m
	if _, e = shallow.SubstituteIntegers(p, 0, PolynomialVariablesView{}, nil); !errors.Is(e, ErrPolynomialOwner) {
		t.Fatal(e)
	}
}

func TestPolynomialSubstitutionZeroTermsRepeatAndLaterSourceINC(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 1}, {1, 1}}, {{0, 2}, {2, 1}}})
	zero := Int{}
	for i := 0; i < 3; i++ {
		a, e := m.SubstituteInteger(p, 0, &zero)
		serviceMust(t, e)
		serviceMust(t, m.IncRef(a))
		polynomialQueryExpect(t, m, a, map[[3]uint32]int64{})
		serviceMust(t, m.DecRef(a))
	}
	// Source integer substitution creates each remaining monomial even when
	// the coefficient is zero. A later actual INC must join the existing token.
	y, e := m.monomials.MkMonomial([]MonomialPower{{1, 1}})
	serviceMust(t, e)
	l, e := m.monomials.location(y)
	serviceMust(t, e)
	if l.unreferencedCohort != &m.substitution || m.substitution.orphans[l.unreferencedSlot] != y {
		t.Fatal("actual zero-add cohort authority")
	}
	index := l.unreferencedSlot
	before := m.memory.State().GlobalCount
	budgetBefore := m.integer.budget.Used()
	serviceMust(t, m.monomials.IncRef(y))
	if m.substitution.orphans[index] != (Monomial{}) || l.unreferencedCohort != nil || m.integer.budget.Used() != budgetBefore {
		t.Fatal("exact allocation-free INC authority transfer")
	}
	serviceMust(t, m.CleanupUnreferencedSubstitutionMonomials())
	if m.memory.State().GlobalCount != before {
		t.Fatal("source INC unexpectedly allocates")
	}
	power, e := m.monomials.ReadPower(y, 0)
	serviceMust(t, e)
	if power != (MonomialPower{1, 1}) {
		t.Fatal(power)
	}
	serviceMust(t, m.monomials.DecRef(y))
}

func TestPolynomialSubstitutionEveryReservationNamedPrefixes(t *testing.T) {
	for route := 0; route < 3; route++ {
		success := false
		refusals := 0
		for fault := int64(0); fault < 128 && !success; fault++ {
			m := polynomialCellFixture(t)
			p := polynomialQueryTerms(t, m, []int32{3, 5, 7}, [][]MonomialPower{{{0, 3}, {1, 1}}, {{0, 2}}, nil})
			xs, xraw := variableHelperArray(t, m.memory, []uint32{0})
			var input Int
			serviceMust(t, m.integer.SetLimbs(&input, []uint32{17, 0, 1}))
			value := Rational{num: input, den: NewInt(2)}
			q := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{1, 1}}, nil})
			out, e := NewPolynomialRef(m)
			serviceMust(t, e)
			m.integer.budget.FailAfter(fault)
			result := Polynomial{}
			if route == 0 {
				result, e = m.SubstituteIntegers(p, 1, xs, []Int{input})
			}
			if route == 1 {
				result, e = m.SubstituteRationals(p, 1, xs, []Rational{value})
			}
			if route == 2 {
				e = m.SubstitutePolynomialFraction(p, 0, q, q, out)
			}
			polys := []Polynomial{p, q, result}
			polynomialSubstitutionPhysical(t, m, []*PolynomialRef{out}, polys, []memory.Block{*xraw}, &input)
			if e != nil {
				refusals++
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal(route, fault, e)
				}
			} else {
				success = true
			}
			m.integer.budget.FailAfter(-1)
			// Explicit owning cleanup follows the immediate named census. No source
			// arithmetic retry or counter/numeral repair is used.
			serviceMust(t, m.CleanupSubstitution())
			if m.powerLocal.active || m.pending.raw != (memory.ObjectRef{}) || m.pending.value != (Polynomial{}) || m.generatedActive || m.generatedCalleePending {
				t.Fatal("standalone substitution cleanup did not drain started callees", route, fault, m.powerLocal, m.pending, m.generatedActive, m.generatedCalleePending, m.substitution.generatedReceipt, m.substitution.generatedCalleeReceipt)
			}
			if result != (Polynomial{}) {
				serviceMust(t, m.CleanupValue(&result))
			}
			serviceMust(t, out.Close())
			serviceMust(t, m.integer.Del(&input))
		}
		if !success || refusals == 0 {
			t.Fatal("missing genuine reservation sweep", route, refusals)
		}
		t.Log("source route", route, "refusals", refusals, "first success", refusals)
	}
}

func TestPolynomialSubstitutionCleanupLeavesForeignFactoryContinuation(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1}, [][]MonomialPower{{{0, 1}, {1, 1}}})
	var foreign monomialTemp
	serviceMust(t, m.monomials.tempConstruct(&foreign))
	serviceMust(t, m.monomials.tempSetSize(&foreign, 3))
	for i := uint32(0); i < 3; i++ {
		serviceMust(t, m.monomials.tempSetPower(&foreign, i, MonomialPower{i, 19 + i}))
	}
	m.integer.budget.FailAfter(0)
	outcome := monomialFactoryOutcome{}
	_, e := m.monomials.internBorrowedTempCreated(&foreign, &outcome)
	if !numericPhysicalReservationRefusal(e) {
		t.Fatal("real foreign factory refusal", e)
	}
	polynomialSubstitutionPhysical(t, m, nil, []Polynomial{p}, []memory.Block{foreign.raw})
	m.integer.budget.FailAfter(-1)
	oldCreation := m.monomials.creation
	one := NewInt(1)
	_, e = m.SubstituteInteger(p, 0, &one)
	if !errors.Is(e, ErrMonomialPending) {
		t.Fatal(e)
	}
	serviceMust(t, m.CleanupSubstitution())
	if m.monomials.borrowedTemp != &foreign || m.monomials.creation != oldCreation || m.generatedCalleePending {
		t.Fatal("foreign continuation acquired by substitution")
	}
	polynomialSubstitutionPhysical(t, m, nil, []Polynomial{p}, []memory.Block{foreign.raw})
	serviceMust(t, m.monomials.CleanupCreation())
	serviceMust(t, m.monomials.tempCleanup(&foreign))
}

// The source borrowed deletion callback executes a genuine failing allocation
// at a completed product destructor. No deallocation refusal is fabricated.
func TestPolynomialSubstitutionCompletedLocalStopsOlderSourceUnwind(t *testing.T) {
	m := polynomialCellFixture(t)
	r := polynomialQueryTerms(t, m, []int32{3}, [][]MonomialPower{{{0, 2}, {1, 1}}})
	p := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{1, 1}}, nil})
	q := polynomialQueryTerms(t, m, []int32{2}, [][]MonomialPower{nil})
	out, e := NewPolynomialRef(m)
	serviceMust(t, e)
	h, e := NewPolynomialDeletionHandler(m.integer.budget, func(h *PolynomialDeletionHandler, m *MultivariatePolynomialCore, p Polynomial) error {
		s := &m.substitution.scope
		if !s.productConstructed || p != s.product.value {
			return nil
		}
		h.State++
		m.integer.budget.FailAfter(0)
		raw, e := m.memory.Allocate(1)
		if raw != (memory.Block{}) {
			t.Fatal("callback unexpectedly owns completed allocation", raw)
		}
		return e
	})
	serviceMust(t, e)
	serviceMust(t, m.AddDeletionHandler(h))
	e = m.SubstitutePolynomialFraction(r, 0, p, q, out)
	if !numericPhysicalReservationRefusal(e) || h.State != 1 {
		t.Fatal("actual completed source destructor", e, h.State)
	}
	s := &m.substitution.scope
	if !s.active || s.cleanupCause == nil || s.stage != 0 || s.product.value == (Polynomial{}) || s.p1.value == (Polynomial{}) || s.q1.value == (Polynomial{}) {
		t.Fatal("older source locals must remain reached and owned", *s)
	}
	polynomialSubstitutionPhysicalWithHandlers(t, m, []*PolynomialRef{out}, []Polynomial{r, p, q}, nil, []*PolynomialDeletionHandler{h})
	// A second public invocation may not replay stopped source destructors.
	if e = m.SubstitutePolynomialFraction(r, 0, p, q, out); !errors.Is(e, ErrPolynomialPending) || h.State != 1 {
		t.Fatal(e, h.State)
	}
	m.integer.budget.FailAfter(-1)
	serviceMust(t, m.CleanupSubstitution())
	if m.substitution.scope.active || h.State != 1 {
		t.Fatal("owning abort replayed source callback", h.State)
	}
	serviceMust(t, m.CleanupSubstitution())
	polynomialSubstitutionPhysicalWithHandlers(t, m, []*PolynomialRef{out}, []Polynomial{r, p, q}, nil, []*PolynomialDeletionHandler{h})
	serviceMust(t, m.RemoveDeletionHandler(h))
	serviceMust(t, h.CloseChecked())
	serviceMust(t, out.Close())
}

func TestPolynomialSubstitutionZeroTermWarmPhysicalCohort(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 1}, {1, 1}}, {{0, 2}, {2, 1}}})
	zero := Int{}
	run := func() {
		a, e := m.SubstituteInteger(p, 0, &zero)
		serviceMust(t, e)
		serviceMust(t, m.IncRef(a))
		serviceMust(t, m.DecRef(a))
	}
	for i := 0; i < 5; i++ {
		run()
	}
	before := numericWarmProfileSnapshot(t, m.integer, m.integer.budget)
	if !modularSourceDebug {
		m.integer.budget.FailAfter(0)
	}
	allocs := testing.AllocsPerRun(20, run)
	m.integer.budget.FailAfter(-1)
	numericWarmProfileAudit(t, m.integer, m.integer.budget, before, allocs, modularSourceDebug)
	polynomialSubstitutionPhysical(t, m, nil, []Polynomial{p}, nil)
}

func TestPolynomialSubstitutionCohortReservationPrecedesFactory(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 1}, {1, 1}}, {{0, 2}, {2, 1}}})
	zero := Int{}
	a, e := m.SubstituteInteger(p, 0, &zero)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(a))
	serviceMust(t, m.DecRef(a))
	d := &m.substitution
	if len(d.orphans) != 2 || cap(d.orphans) != 2 {
		t.Fatal("two actual zero-add roots", len(d.orphans), cap(d.orphans))
	}
	old := [2]Monomial{d.orphans[0], d.orphans[1]}
	count, e := m.memory.AllocationCount()
	serviceMust(t, e)
	stats, bytes := m.memory.Stats(), m.integer.budget.Used()
	m.integer.budget.FailAfter(0)
	a, e = m.SubstituteInteger(p, 0, &zero)
	if !numericPhysicalReservationRefusal(e) || a != (Polynomial{}) {
		t.Fatal("cold cohort growth refusal", a, e)
	}
	after, err := m.memory.AllocationCount()
	serviceMust(t, err)
	if count != after || stats != m.memory.Stats() || bytes != m.integer.budget.Used() || cap(d.orphans) != 2 || d.orphans[0] != old[0] || d.orphans[1] != old[1] || m.generatedActive || m.monomials.borrowedTemp != nil {
		t.Fatal("factory started before owning metadata capacity existed")
	}
	polynomialSubstitutionPhysical(t, m, nil, []Polynomial{p}, nil)
	m.integer.budget.FailAfter(-1)
	serviceMust(t, m.CleanupSubstitution())
}

func TestPolynomialSubstitutionReceiptRejectsReplacementFactory(t *testing.T) {
	m := polynomialCellFixture(t)
	r := polynomialQueryTerms(t, m, []int32{3, 5, 7}, [][]MonomialPower{{{0, 3}, {1, 1}}, {{0, 2}}, nil})
	p := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{1, 1}}, nil})
	out, e := NewPolynomialRef(m)
	serviceMust(t, e)
	m.integer.budget.FailAfter(0)
	e = m.SubstitutePolynomialFraction(r, 0, p, p, out)
	if !numericPhysicalReservationRefusal(e) || m.substitution.generatedCalleeReceipt == 0 {
		t.Fatal("real started subordinate factory", e, m.monomials.creation)
	}
	polynomialSubstitutionPhysical(t, m, []*PolynomialRef{out}, []Polynomial{r, p}, nil)
	m.integer.budget.FailAfter(-1)
	serviceMust(t, m.monomials.CleanupCreation())
	m.integer.budget.FailAfter(0)
	_, e = m.monomials.MkMonomial([]MonomialPower{{0, 11}, {1, 13}, {2, 17}})
	if !numericPhysicalReservationRefusal(e) || m.monomials.creation.stage == 0 {
		t.Fatal("new actual foreign factory", e, m.monomials.creation)
	}
	before := m.monomials.creation
	polynomialSubstitutionPhysical(t, m, []*PolynomialRef{out}, []Polynomial{r, p}, nil)
	m.integer.budget.FailAfter(-1)
	serviceMust(t, m.CleanupSubstitution())
	if m.monomials.creation != before {
		t.Fatal("stale receipt swept replacement factory", before, m.monomials.creation)
	}
	serviceMust(t, m.monomials.CleanupCreation())
	serviceMust(t, out.Close())
}

func TestPolynomialSubstitutionBorrowedReceiptRejectsReplacement(t *testing.T) {
	reached := false
	for fault := int64(0); fault < 64 && !reached; fault++ {
		m := polynomialCellFixture(t)
		p := polynomialQueryTerms(t, m, []int32{1}, [][]MonomialPower{{{0, 1}, {1, 19}, {2, 23}}})
		one := NewInt(1)
		m.integer.budget.FailAfter(fault)
		out, e := m.SubstituteInteger(p, 0, &one)
		polynomialSubstitutionPhysical(t, m, nil, []Polynomial{p, out}, nil)
		reached = m.substitution.borrowedReceipt != 0
		m.integer.budget.FailAfter(-1)
		if !reached {
			serviceMust(t, m.CleanupSubstitution())
			if out != (Polynomial{}) {
				serviceMust(t, m.CleanupValue(&out))
			}
			continue
		}
		if !numericPhysicalReservationRefusal(e) {
			t.Fatal("actual stopped borrowed factory", e)
		}
		serviceMust(t, m.monomials.CleanupCreation())
		if m.substitution.borrowedReceipt != 0 {
			t.Fatal("actual completion did not relinquish old receipt")
		}
		serviceMust(t, m.monomials.tempSetSize(&m.tmp1, 3))
		for i := uint32(0); i < 3; i++ {
			serviceMust(t, m.monomials.tempSetPower(&m.tmp1, i, MonomialPower{i, 31 + i}))
		}
		m.integer.budget.FailAfter(0)
		_, e = m.monomials.internBorrowedTempCreated(&m.tmp1, nil)
		if !numericPhysicalReservationRefusal(e) {
			t.Fatal("new same-address borrowed factory", e)
		}
		before, generation := m.monomials.creation, m.monomials.borrowedGeneration
		polynomialSubstitutionPhysical(t, m, nil, []Polynomial{p}, nil)
		m.integer.budget.FailAfter(-1)
		serviceMust(t, m.CleanupSubstitution())
		if m.monomials.creation != before || m.monomials.borrowedGeneration != generation || m.monomials.borrowedTemp != &m.tmp1 {
			t.Fatal("old receipt acquired replacement at same address")
		}
		serviceMust(t, m.monomials.CleanupCreation())
		t.Log("first actual borrowed continuation fault", fault)
	}
	if !reached {
		t.Fatal("no actual borrowed continuation reached")
	}
}
