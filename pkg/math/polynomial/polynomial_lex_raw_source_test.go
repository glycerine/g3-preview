// Source: src/math/polynomial/polynomial.cpp:1315-1366,1440-1448;
// src/util/permutation.h:56-91.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// The concrete native mapping is borrowed by the two column permutations.
func TestPolynomialLexNativeMappingCyclesPreserveTermPairs(t *testing.T) {
	m, p, _ := polynomialLexFixture(t)
	oldA := make([]util.Int, 5)
	oldM := make([]Monomial, 5)
	for i := uint32(0); i < 5; i++ {
		var e error
		oldA[i], e = m.core.coefficientCore(p, i)
		serviceMust(t, e)
		oldM[i], e = m.core.monomialCore(p, i)
		serviceMust(t, e)
	}
	mapping := []uint32{1, 0, 2, 3, 4}
	m.lexPermutation = append([]uint32(nil), mapping...)
	state, used := m.core.memory.State(), m.core.integer.Budget().Used()
	m.core.integer.Budget().FailAfter(0)
	defer m.core.integer.Budget().FailAfter(-1)
	serviceMust(t, m.lexApplyPermutation(p, 0, 5, true))
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
	serviceMust(t, m.lexApplyPermutation(p, 0, 5, false))
	if m.core.memory.State() != state || m.core.integer.Budget().Used() != used {
		t.Fatal("actual source pointer cycles allocate nothing")
	}
	for i, k := range mapping {
		a, e := m.core.coefficientCore(p, uint32(i))
		serviceMust(t, e)
		mono, e := m.core.monomialCore(p, uint32(i))
		serviceMust(t, e)
		if a != oldA[k] || mono != oldM[k] {
			t.Fatal("independent pair permutation", i, a, mono)
		}
	}
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
	m.core.integer.Budget().FailAfter(-1)
}

func TestPolynomialLexCapturedPayloadViewsKeepStableColumnAndExactObject(t *testing.T) {
	m, p, _ := polynomialLexFixture(t)
	a, e := m.core.captureCoefficientPointer(p, 1)
	serviceMust(t, e)
	mono, e := m.core.captureMonomialPointer(p, 1)
	serviceMust(t, e)
	oldA, e := m.core.coefficientCore(p, 1)
	serviceMust(t, e)
	oldM, e := m.core.monomialCore(p, 1)
	serviceMust(t, e)
	// Typed safety-domain header mutation: source m_as/m_ms remain fixed fields.
	// Restore only this deliberate fixture before ordinary polynomial destruction.
	serviceMust(t, m.core.writeWord(p, 8, 0))
	defer func() { serviceMust(t, m.core.writeWord(p, 8, 5)) }()
	gotA, e := a.read(0)
	serviceMust(t, e)
	gotM, e := mono.read(0)
	serviceMust(t, e)
	if gotA != oldA || gotM != oldM {
		t.Fatal("captured physical columns", gotA, gotM)
	}
	serviceMust(t, m.core.writeWord(p, 8, 5))
	if _, e = a.read(4); !errors.Is(e, ErrPolynomialPartial) {
		t.Fatal("actual constructed prefix", e)
	}
	// Exact borrowed object token is authority; a second real object cannot be
	// substituted while keeping the first polynomial locator.
	other, e := m.core.MkTerms([]util.Int{util.NewInt(2)}, []Monomial{oldM})
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(other))
	l, e := m.core.polynomialLocation(other)
	serviceMust(t, e)
	foreign := a
	foreign.raw = l.raw
	if _, e = foreign.read(0); !errors.Is(e, ErrPolynomialOwner) {
		t.Fatal("foreign captured ObjectRef", e)
	}
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p, other})
	serviceMust(t, m.core.DecRef(other))
}
func TestPolynomialLexRawCycleSignedSkippedLoopLeavesPayloadProviderUnread(t *testing.T) {
	m, p, _ := polynomialLexFixture(t)
	// The signed core does no accesses; unsigned restoring loop subsequently
	// reaches its empty mapping guard. No polynomial SIZE/index read is required.
	e := m.lexApplyPermutation(p, 0, 0x80000000, false)
	serviceMust(t, e)
	e = m.lexApplyPermutation(p, 0, 0x80000000, true)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("restoring mapping reached guard", e)
	}
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
}

func TestPolynomialLexRawCycleMarkPrecedesReachedConstructedPayloadGuard(t *testing.T) {
	m, p, _ := polynomialLexFixture(t)
	old, e := m.core.coefficientCore(p, 4)
	serviceMust(t, e)
	m.lexPermutation = []uint32{1, 0}
	state, used := m.core.memory.State(), m.core.integer.Budget().Used()
	m.core.integer.Budget().FailAfter(0)
	defer m.core.integer.Budget().FailAfter(-1)
	// Invalid source data subrange: mark is completed before the second NUM
	// dereference reaches the typed constructed-prefix guard. No body repair.
	e = m.lexApplyPermutation(p, 4, 2, true)
	if !errors.Is(e, ErrPolynomialPartial) {
		t.Fatal("reached NUM initialized prefix", e)
	}
	x := m.lexPermutation[0]
	if x != ^uint32(1) {
		t.Fatal("mark before guarded payload read", x)
	}
	x = m.lexPermutation[1]
	if x != 0 {
		t.Fatal("later marker untouched", x)
	}
	now, e := m.core.coefficientCore(p, 4)
	serviceMust(t, e)
	if now != old {
		t.Fatal("swap reads both operands before stores", now)
	}
	if m.core.memory.State() != state || m.core.integer.Budget().Used() != used {
		t.Fatal("guard kernel allocated")
	}
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
}

func TestPolynomialLexCapturedPayloadRejectsRealDeletedReusedSlot(t *testing.T) {
	m, p, ms := polynomialLexFixture(t)
	old, e := m.core.MkTerms([]util.Int{util.NewInt(31)}, []Monomial{ms[0]})
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(old))
	a, e := m.core.captureCoefficientPointer(old, 0)
	serviceMust(t, e)
	mono, e := m.core.captureMonomialPointer(old, 0)
	serviceMust(t, e)
	serviceMust(t, m.core.DecRef(old))
	fresh, e := m.core.MkTerms([]util.Int{util.NewInt(47)}, []Monomial{ms[1]})
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(fresh))
	if fresh.slot != old.slot || fresh.generation == old.generation {
		t.Fatal("actual slot reuse fixture", old, fresh)
	}
	if _, e = a.read(0); !errors.Is(e, ErrPolynomialOwner) {
		t.Fatal("stale captured NUM pointer", e)
	}
	if _, e = mono.read(0); !errors.Is(e, ErrPolynomialOwner) {
		t.Fatal("stale captured monomial pointer", e)
	}
	got, e := m.core.coefficientCore(fresh, 0)
	serviceMust(t, e)
	if got != util.NewInt(47) {
		t.Fatal("replacement untouched", got)
	}
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p, fresh})
	serviceMust(t, m.core.DecRef(fresh))
}
