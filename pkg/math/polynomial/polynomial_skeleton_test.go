// Source: src/math/polynomial/polynomial.cpp:2958-3068; src/util/scoped_ptr_vector.h.
package polynomial

import (
	"errors"
	"fmt"
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

func polynomialSkeletonRoots(t *testing.T, s *PolynomialSkeleton) []Monomial {
	t.Helper()
	if s == nil {
		return nil
	}
	var roots []Monomial
	for i := s.entryCursor; uint64(i) < uint64(len(s.entries)); i++ {
		roots = append(roots, s.entries[i].Monomial)
	}
	for i := s.originalCursor; uint64(i) < uint64(len(s.originals)); i++ {
		roots = append(roots, s.originals[i])
	}
	if s.generatedActive {
		roots = append(roots, s.generated)
	}
	return roots
}
func polynomialSkeletonPhysicalAudit(t *testing.T, m *MultivariatePolynomialCore, s *PolynomialSkeleton, callers ...Polynomial) {
	t.Helper()
	var extra uint64
	if s != nil {
		extra = s.header
	}
	polynomialNewtonPhysicalWithExtra(t, m, nil, nil, nil, nil, callers, nil, nil, extra)
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
					if !errors.As(e, &actual) || actual != s || s.originalCause == nil || s.completed || s.sourceStarted || s.CleanupCause() != nil {
						t.Fatal("failed constructor retained actual references", e, s)
					}
					failedRetained = true
				}
				cause := e
				if actual != nil {
					cause = actual.originalCause
				}
				if !numericPhysicalReservationRefusal(cause) {
					t.Fatal("actual header/monomial refusal", cause)
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

func TestPolynomialSkeletonDisplayReachedOwnerLifetime(t *testing.T) {
	m := polynomialCellFixture(t)
	s, e := NewPolynomialSkeleton(m, m.unit, 0)
	serviceMust(t, e)
	sink := &polynomialDisplayTestSink{literalHook: func(v string) error {
		if v == ")*" {
			return s.SourceFinalize()
		}
		return nil
	}}
	e = s.DisplayInto(sink)
	if !errors.Is(e, arena.ErrClosed) || sink.text.String() != "(x0^0)*" {
		t.Fatal("reached child owner after output", e, sink.text.String())
	}
	polynomialSkeletonPhysicalAudit(t, m, s)
	serviceMust(t, s.CloseChecked())
	polynomialPhysicalAudit(t, m)
}

func TestPolynomialSkeletonNativePurePowerGrowth(t *testing.T) {
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
	used := m.integer.Budget().Used()
	m.integer.Budget().FailAfter(1) // The actual skeleton owner header is the only reservation.
	s, e := NewPolynomialSkeleton(m, p, 0)
	m.integer.Budget().FailAfter(-1)
	serviceMust(t, e)
	after, e := m.memory.AllocationCount()
	serviceMust(t, e)
	if after != before || len(s.powers) != 129 || len(s.originals) != 129 || m.integer.Budget().Used() != used+s.header {
		t.Fatal("native pure-power storage", before, after, len(s.powers), len(s.originals))
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

// Native payload bounds retain the source DEBUG assertion; release returns
// the Go bounds error rather than exposing unused vector capacity.
func TestPolynomialSkeletonNativePayloadBounds(t *testing.T) {
	m := polynomialCellFixture(t)
	p, originals := polynomialSkeletonInput(t, m)
	s, e := NewPolynomialSkeleton(m, p, 0)
	serviceMust(t, e)
	a, e := s.Entry(0)
	serviceMust(t, e)
	a.FirstPower = uint32(len(s.powers))
	_, e = s.Power(a, 0)
	want := arena.ErrBounds
	if polynomialSourceDebug {
		want = ErrPolynomialAssertion
	}
	if !errors.Is(e, want) {
		t.Fatal("native power bounds", e)
	}
	_, e = s.Original(a, 0)
	if !errors.Is(e, want) {
		t.Fatal("native original bounds", e)
	}
	_, e = s.Entry(uint32(len(s.entries)))
	if !errors.Is(e, want) {
		t.Fatal("native entry bounds", e)
	}

	// The source unsigned index addition wraps before the native bounds check.
	a.FirstPower = ^uint32(0)
	a.NumPowers = 2
	k, e := s.Power(a, 1)
	serviceMust(t, e)
	orig, e := s.Original(a, 1)
	serviceMust(t, e)
	if k != 4 || orig != originals[4] {
		t.Fatal("wrapped source index", k, orig)
	}
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
					if !errors.As(e, &actual) || actual != s || s.originalCause == nil || s.completed || s.sourceStarted || s.CleanupCause() != nil {
						t.Fatal("failed constructor retained actual references", e, s)
					}

				}
				cause := e
				if actual != nil {
					cause = actual.originalCause
				}
				if !numericPhysicalReservationRefusal(cause) {
					t.Fatal("actual header/monomial refusal", cause)
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
	if !success {
		t.Fatal("bounded real sweep incomplete", success)
	}
}
