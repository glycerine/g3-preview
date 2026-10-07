// Source: src/math/polynomial/polynomial.cpp:2958-3068; src/util/scoped_ptr_vector.h. Licensed under the MIT license.
package polynomial

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func polynomialSkeletonInput(t *testing.T, m *MultivariatePolynomialCore) (Polynomial, []Monomial) {
	t.Helper()
	powers := [][]MonomialPower{{{0, 3}, {2, 1}}, {{0, 1}, {2, 1}}, {{1, 2}}, {{0, 2}, {1, 2}}, {{0, 4}}}
	ms := make([]Monomial, len(powers))
	as := make([]util.Int, len(powers))
	for i, p := range powers {
		var e error
		ms[i], e = m.monomials.MkMonomial(p)
		serviceMust(t, e)
		as[i] = util.NewInt(int32(5 + 2*i))
	}
	p, e := m.MkTerms(as, ms)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(p))
	return p, ms
}

func TestPolynomialSkeletonSourceGroupsAndLifetime(t *testing.T) {
	m := polynomialCellFixture(t)
	p, original := polynomialSkeletonInput(t, m)
	s, e := NewPolynomialSkeleton(m, p, 0)
	serviceMust(t, e)
	n, e := s.NumEntries()
	serviceMust(t, e)
	max, e := s.MaxNumPowers()
	serviceMust(t, e)
	if n != 3 || max != 2 {
		t.Fatal("independent finite group model", n, max)
	}
	wantPowers := [][]uint32{{4}, {0, 2}, {1, 3}}
	wantOriginals := [][]Monomial{{original[4]}, {original[2], original[3]}, {original[1], original[0]}}
	for i := uint32(0); i < n; i++ {
		entry, e := s.Entry(i)
		serviceMust(t, e)
		if entry.NumPowers != uint32(len(wantPowers[i])) {
			t.Fatal("group cardinality", i, entry)
		}
		index, e := s.EntryIndex(entry.Monomial)
		serviceMust(t, e)
		if index != i {
			t.Fatal("source pointer identity lookup", index, i)
		}
		for j, want := range wantPowers[i] {
			got, e := s.Power(entry, uint32(j))
			serviceMust(t, e)
			if got != want {
				t.Fatal(i, j, got, want)
			}
			orig, e := s.Original(entry, uint32(j))
			serviceMust(t, e)
			if orig != wantOriginals[i][j] {
				t.Fatal("actual original ownership", i, j, orig)
			}
		}
	}
	serviceMust(t, m.DecRef(p))
	serviceMust(t, s.SourceFinalize())
	if _, e = s.NumEntries(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("source-destroyed ordinary entry", e)
	}
	serviceMust(t, s.CloseChecked())
	polynomialOperationMonomialAudit(t, m, nil, nil)
	polynomialPhysicalAudit(t, m)
}

// A real fresh physical array has an initialized monomial pointer while its
// unrelated first/count words remain uninitialized. This is bounded reached-
// field evidence for the source get_entry_idx/destructor, not a successful
// native entry-constructor claim or a fabricated allocation refusal.
func TestPolynomialSkeletonSourcePointerOnlyRead(t *testing.T) {
	m := polynomialCellFixture(t)
	zero := m.zero
	s, e := NewPolynomialSkeleton(m, zero, 0)
	serviceMust(t, e)
	raw, e := m.memory.Allocate(32)
	serviceMust(t, e)
	serviceMust(t, m.memory.WriteUint32(raw, 0, 1))
	serviceMust(t, m.memory.WriteUint32(raw, 4, 1))
	unit, e := m.monomials.Unit()
	serviceMust(t, e)
	serviceMust(t, m.monomials.IncRef(unit))
	serviceMust(t, skeletonWriteMonomial(m.memory, raw, 8, unit))
	s.entries.raw = raw
	s.entries.initialized = 1
	s.entryOwned = 1
	index, e := s.EntryIndex(unit)
	if e != nil || index != 0 {
		t.Fatal("source identity lookup read unrelated uninitialized words", index, e)
	}
	serviceMust(t, s.SourceFinalize())
	serviceMust(t, s.CloseChecked())
	polynomialPhysicalAudit(t, m)
}

func polynomialSkeletonRoots(t *testing.T, s *PolynomialSkeleton) []Monomial {
	t.Helper()
	if s == nil {
		return nil
	}
	var roots []Monomial
	for i := s.entryCursor; i < s.entryOwned; i++ {
		if i < s.entryTransferred {
			continue
		}
		a, e := s.entryMonomialCore(i)
		serviceMust(t, e)
		roots = append(roots, a)
	}
	for i := s.originalCursor; i < s.originalOwned; i++ {
		if i < s.originalTransferred {
			continue
		}
		a, e := s.originals.monomial(s.owner, i)
		serviceMust(t, e)
		roots = append(roots, a)
	}
	roots = append(roots, s.retiredEntries[s.retiredEntryCursor:s.retiredEntryCount]...)
	roots = append(roots, s.retiredOriginals[s.retiredOriginalCursor:s.retiredOriginalCount]...)
	roots = append(roots, s.incomingEntry, s.incomingOriginal)
	if s.generatedActive {
		roots = append(roots, s.generated)
	}
	return roots
}
func polynomialSkeletonPhysicalAudit(t *testing.T, m *MultivariatePolynomialCore, s *PolynomialSkeleton, callers ...Polynomial) {
	t.Helper()
	var raw []util.Block
	extra := uint64(0)
	if s != nil {
		extra = s.header + uint64(cap(s.retiredEntries)+cap(s.retiredOriginals))*uint64(reflect.TypeOf(Monomial{}).Size())
		for _, v := range []*polynomialVector{&s.entries, &s.originals, &s.powers} {
			for _, r := range []util.Block{v.raw, v.pending, v.old} {
				if r != (util.Block{}) {
					raw = append(raw, r)
				}
			}
		}
		for _, r := range []util.Block{s.ms.raw, s.ms.pending} {
			if r != (util.Block{}) {
				raw = append(raw, r)
			}
		}
	}
	polynomialNewtonPhysicalWithExtra(t, m, nil, nil, nil, nil, callers, nil, raw, extra)
	polynomialOperationMonomialAudit(t, m, callers, polynomialSkeletonRoots(t, s))
}
func TestPolynomialSkeletonConstructorEveryReservation(t *testing.T) {
	success := false
	failedRetained := false
	for ordinal := int64(0); ordinal < 100 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := polynomialCellFixture(t)
			p, orig := polynomialSkeletonInput(t, m)
			baseline := make([]uint32, len(orig))
			for i, a := range orig {
				h, e := m.monomials.headerCore(a)
				serviceMust(t, e)
				baseline[i] = h.RefCount
			}
			m.integer.Budget().FailAfter(ordinal)
			s, e := NewPolynomialSkeleton(m, p, 0)
			if e == nil {
				success = true
			} else {
				var actual *PolynomialSkeleton
				if s != nil {
					if !errors.As(e, &actual) || actual != s || s.originalCause == nil || s.completed || s.sourceStarted || s.unwindStage != 4 || s.cleanupCause != nil {
						t.Fatal("failed constructor member-only unwind", e, s)
					}
					failedRetained = true
				}
				cause := e
				if actual != nil {
					cause = actual.originalCause
				}
				if !numericPhysicalReservationRefusal(cause) {
					t.Fatal("actual metadata/runtime refusal", cause)
				}
			}
			polynomialSkeletonPhysicalAudit(t, m, s, p)
			// None of the source failed-class member destructors DEC these INC roots.
			if e != nil {
				owned := polynomialSkeletonRoots(t, s)
				for i, a := range orig {
					want := baseline[i]
					for _, r := range owned {
						if r == a {
							want++
						}
					}
					h, x := m.monomials.headerCore(a)
					serviceMust(t, x)
					if h.RefCount != want {
						t.Fatal("source unwind ran class DEC or lost INC", i, h.RefCount, want)
					}
				}
			}
			m.integer.Budget().FailAfter(-1)
			if s != nil {
				serviceMust(t, s.CloseChecked())
			}
			serviceMust(t, m.DecRef(p))
			polynomialPhysicalAudit(t, m)
			polynomialOperationMonomialAudit(t, m, nil, nil)
		})
	}
	if !success || !failedRetained {
		t.Fatal("bounded real sweep incomplete", success, failedRetained)
	}
}

func TestPolynomialSkeletonSourceDisplay(t *testing.T) {
	m := polynomialCellFixture(t)
	p, _ := polynomialSkeletonInput(t, m)
	s, e := NewPolynomialSkeleton(m, p, 0)
	serviceMust(t, e)
	var sink monomialTestSink
	serviceMust(t, s.DisplayInto(&sink))
	if got := sink.text.String(); got != "(x0^4)*1 (x0^0 x0^2)*x1^2 (x0^1 x0^3)*x2" {
		t.Fatal("independent source spelling", got)
	}
	serviceMust(t, s.CloseChecked())
	serviceMust(t, m.DecRef(p))
}

func TestPolynomialSkeletonSourceDisplayReachedPointer(t *testing.T) {
	m := polynomialCellFixture(t)
	s, e := NewPolynomialSkeleton(m, m.zero, 0)
	serviceMust(t, e)
	raw, e := m.memory.Allocate(32)
	serviceMust(t, e)
	serviceMust(t, m.memory.WriteUint32(raw, 0, 1))
	serviceMust(t, m.memory.WriteUint32(raw, 4, 1))
	serviceMust(t, m.memory.WriteUint32(raw, 24, 0))
	serviceMust(t, m.memory.WriteUint32(raw, 28, 0))
	s.entries.raw = raw
	var sink monomialTestSink
	e = s.DisplayInto(&sink)
	if !errors.Is(e, util.ErrRuntimeUninitialized) || sink.text.String() != "()*" {
		t.Fatal("source reference formation eagerly loaded pointer", e, sink.text.String())
	}
	polynomialSkeletonPhysicalAudit(t, m, s)
	serviceMust(t, s.CloseChecked())
}

func TestPolynomialSkeletonSourceInline128Overflow(t *testing.T) {
	m := polynomialCellFixture(t)
	as := make([]util.Int, 129)
	ms := make([]Monomial, 129)
	for i := range as {
		as[i] = util.NewInt(1)
		var e error
		ms[i], e = m.monomials.MkVariable(0, uint32(i+1))
		serviceMust(t, e)
	}
	p, e := m.MkTerms(as, ms)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(p))
	before, e := m.memory.AllocationCount()
	serviceMust(t, e)
	s, e := NewPolynomialSkeleton(m, p, 0)
	serviceMust(t, e)
	after, e := m.memory.AllocationCount()
	serviceMust(t, e)
	if after <= before || s.ms.capacity != 256 || s.ms.size != 129 || s.ms.raw != (util.Block{}) || s.ms.pending != (util.Block{}) {
		t.Fatal("actual local inline128 overflow and source destruction", s.ms.capacity, s.ms.size, before, after)
	}
	n, e := s.NumEntries()
	serviceMust(t, e)
	max, e := s.MaxNumPowers()
	serviceMust(t, e)
	if n != 1 || max != 129 {
		t.Fatal("independent pure-power model", n, max)
	}
	entry, e := s.Entry(0)
	serviceMust(t, e)
	for i := uint32(0); i < 129; i++ {
		k, e := s.Power(entry, i)
		serviceMust(t, e)
		if k != i+1 {
			t.Fatal("source k ordering", i, k)
		}
	}
	polynomialSkeletonPhysicalAudit(t, m, s, p)
	serviceMust(t, s.SourceFinalize())
	serviceMust(t, s.CloseChecked())
	serviceMust(t, m.DecRef(p))
	polynomialPhysicalAudit(t, m)
	polynomialOperationMonomialAudit(t, m, nil, nil)
}

// Source vector operator[] checks its own logical size in DEBUG, after the
// entry's NumPowers assertion. Release can read actual retained initialized
// payload. These public Runtime header writes probe reached scalar domains.
func TestPolynomialSkeletonSourceUnderlyingVectorGuards(t *testing.T) {
	m := polynomialCellFixture(t)
	p, original := polynomialSkeletonInput(t, m)
	s, e := NewPolynomialSkeleton(m, p, 0)
	serviceMust(t, e)
	a, e := s.Entry(0)
	serviceMust(t, e)
	n, e := s.powers.size(m)
	serviceMust(t, e)
	serviceMust(t, m.memory.WriteUint32(s.powers.raw, 4, 0))
	k, e := s.Power(a, 0)
	if polynomialSourceDebug {
		if !errors.Is(e, ErrPolynomialAssertion) {
			t.Fatal("reached power vector DEBUG index", k, e)
		}
	} else if e != nil || k != 4 {
		t.Fatal("release physical initialized power", k, e)
	}
	polynomialSkeletonPhysicalAudit(t, m, s, p)
	serviceMust(t, m.memory.WriteUint32(s.powers.raw, 4, n))
	n, e = s.originals.size(m)
	serviceMust(t, e)
	serviceMust(t, m.memory.WriteUint32(s.originals.raw, 4, 0))
	got, e := s.Original(a, 0)
	if polynomialSourceDebug {
		if !errors.Is(e, ErrPolynomialAssertion) {
			t.Fatal("reached original vector DEBUG index", got, e)
		}
	} else if e != nil || got != original[4] {
		t.Fatal("release actual initialized pointer", got, e)
	}
	polynomialSkeletonPhysicalAudit(t, m, s, p)
	serviceMust(t, m.memory.WriteUint32(s.originals.raw, 4, n))
	serviceMust(t, s.CloseChecked())
	serviceMust(t, m.DecRef(p))
	polynomialPhysicalAudit(t, m)
}
func polynomialSkeletonOverflowInput(t *testing.T, m *MultivariatePolynomialCore) (Polynomial, []Monomial) {
	t.Helper()
	as := make([]util.Int, 129)
	ms := make([]Monomial, 129)
	for i := range as {
		as[i] = util.NewInt(1)
		var e error
		ms[i], e = m.monomials.MkVariable(0, uint32(i+1))
		serviceMust(t, e)
	}
	p, e := m.MkTerms(as, ms)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(p))
	return p, ms
}
func TestPolynomialSkeletonOverflowConstructorEveryReservation(t *testing.T) {
	success := false
	failedRetained := false
	for ordinal := int64(0); ordinal < 100 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := polynomialCellFixture(t)
			p, orig := polynomialSkeletonOverflowInput(t, m)
			baseline := make([]uint32, len(orig))
			for i, a := range orig {
				h, e := m.monomials.headerCore(a)
				serviceMust(t, e)
				baseline[i] = h.RefCount
			}
			m.integer.Budget().FailAfter(ordinal)
			s, e := NewPolynomialSkeleton(m, p, 0)
			if e == nil {
				success = true
			} else {
				var actual *PolynomialSkeleton
				if s != nil {
					if !errors.As(e, &actual) || actual != s || s.originalCause == nil || s.completed || s.sourceStarted || s.unwindStage != 4 || s.cleanupCause != nil {
						t.Fatal("failed constructor member-only unwind", e, s)
					}
					failedRetained = true
				}
				cause := e
				if actual != nil {
					cause = actual.originalCause
				}
				if !numericPhysicalReservationRefusal(cause) {
					t.Fatal("actual metadata/runtime refusal", cause)
				}
			}
			polynomialSkeletonPhysicalAudit(t, m, s, p)
			// None of the source failed-class member destructors DEC these INC roots.
			if e != nil {
				owned := polynomialSkeletonRoots(t, s)
				for i, a := range orig {
					want := baseline[i]
					for _, r := range owned {
						if r == a {
							want++
						}
					}
					h, x := m.monomials.headerCore(a)
					serviceMust(t, x)
					if h.RefCount != want {
						t.Fatal("source unwind ran class DEC or lost INC", i, h.RefCount, want)
					}
				}
			}
			m.integer.Budget().FailAfter(-1)
			if s != nil {
				serviceMust(t, s.CloseChecked())
			}
			serviceMust(t, m.DecRef(p))
			polynomialPhysicalAudit(t, m)
			polynomialOperationMonomialAudit(t, m, nil, nil)
		})
	}
	if !success || !failedRetained {
		t.Fatal("bounded real sweep incomplete", success, failedRetained)
	}
}
