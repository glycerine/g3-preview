// Source: src/util/scoped_ptr_vector.h; src/math/polynomial/polynomial.cpp:4070-4071.
package polynomial

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialSkeletonVectorSourceOperations(t *testing.T) {
	m := polynomialCellFixture(t)
	p, _ := polynomialSkeletonInput(t, m)
	a, e := NewPolynomialSkeleton(m, p, 0)
	serviceMust(t, e)
	b, e := NewPolynomialSkeleton(m, p, 1)
	serviceMust(t, e)
	v, e := NewPolynomialSkeletonVector(m)
	serviceMust(t, e)
	serviceMust(t, v.Push(a))
	serviceMust(t, v.Push(b))
	serviceMust(t, v.Set(0, a))
	if a.sourceStarted {
		t.Fatal("source same-pointer set must not delete")
	}
	serviceMust(t, v.Set(0, nil))
	if !a.sourceFinalized || a.closed {
		t.Fatal("source child destructor distinct cold owning cleanup")
	}
	got, e := v.Get(1)
	serviceMust(t, e)
	if got != b {
		t.Fatal("actual pointer slot", got)
	}
	serviceMust(t, v.Resize(4))
	got, e = v.Get(3)
	serviceMust(t, e)
	if got != nil {
		t.Fatal("source null growth")
	}
	serviceMust(t, v.Pop())
	serviceMust(t, v.Resize(1))
	if !b.sourceFinalized {
		t.Fatal("shrinking source child destruction")
	}
	serviceMust(t, v.SourceFinalize())
	if _, e = v.Size(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("destroyed vector entry", e)
	}
	serviceMust(t, v.CloseChecked())
	serviceMust(t, m.DecRef(p))
	polynomialPhysicalAudit(t, m)
}

func TestPolynomialSkeletonVectorPushFailureRetainsCaller(t *testing.T) {
	m := polynomialCellFixture(t)
	p, _ := polynomialSkeletonInput(t, m)
	a, e := NewPolynomialSkeleton(m, p, 0)
	serviceMust(t, e)
	v, e := NewPolynomialSkeletonVector(m)
	serviceMust(t, e)
	// One real cold preparation succeeds; Runtime backing reservation then
	// refuses. Source push has not published the caller pointer.
	m.integer.Budget().FailAfter(1)
	e = v.Push(a)
	if !numericPhysicalReservationRefusal(e) {
		t.Fatal("actual pointer backing refusal", e)
	}
	if a.sourceStarted || a.closed {
		t.Fatal("failed push must preserve caller owner")
	}
	polynomialSkeletonVectorPhysicalAudit(t, m, v, []*PolynomialSkeleton{a}, p)
	m.integer.Budget().FailAfter(-1)
	serviceMust(t, v.CloseChecked())
	if a.closed {
		t.Fatal("unpublished pointer is still caller owned")
	}
	serviceMust(t, a.CloseChecked())
	serviceMust(t, m.DecRef(p))
	polynomialPhysicalAudit(t, m)
}

func polynomialSkeletonVectorPhysicalAudit(t *testing.T, m *MultivariatePolynomialCore, v *PolynomialSkeletonVector, callerSkeletons []*PolynomialSkeleton, callers ...Polynomial) {
	t.Helper()
	var raw []util.Block
	var roots []Monomial
	extra := v.header + uint64(cap(v.children))*uint64(reflect.TypeOf((*PolynomialSkeleton)(nil)).Size())
	for _, b := range []util.Block{v.data.raw, v.data.pending, v.data.old} {
		if b != (util.Block{}) {
			raw = append(raw, b)
		}
	}
	// Published scalar IDs bind actual owner pointers; callerSkeletons are
	// explicitly named independent caller roots, never inferred from liveness.
	contexts := append([]*PolynomialSkeleton{}, v.children...)
	for _, s := range callerSkeletons {
		found := false
		for _, a := range contexts {
			if a == s {
				found = true
			}
		}
		if !found {
			contexts = append(contexts, s)
		}
	}
	for _, s := range contexts {
		if s == nil {
			continue
		}
		extra += s.header + uint64(cap(s.retiredEntries)+cap(s.retiredOriginals))*uint64(reflect.TypeOf(Monomial{}).Size())
		for _, a := range []*polynomialVector{&s.entries, &s.originals, &s.powers} {
			for _, b := range []util.Block{a.raw, a.pending, a.old} {
				if b != (util.Block{}) {
					raw = append(raw, b)
				}
			}
		}
		for _, b := range []util.Block{s.ms.raw, s.ms.pending} {
			if b != (util.Block{}) {
				raw = append(raw, b)
			}
		}
		roots = append(roots, polynomialSkeletonRoots(t, s)...)
	}
	polynomialNewtonPhysicalWithExtra(t, m, nil, nil, nil, nil, callers, nil, raw, extra)
	polynomialOperationMonomialAudit(t, m, callers, roots)
}

func TestPolynomialSkeletonVectorPushEveryReservation(t *testing.T) {
	success := false
	failures := 0
	for ordinal := int64(0); ordinal < 100 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := polynomialCellFixture(t)
			p, _ := polynomialSkeletonInput(t, m)
			a, e := NewPolynomialSkeleton(m, p, 0)
			serviceMust(t, e)
			v, e := NewPolynomialSkeletonVector(m)
			serviceMust(t, e)
			m.integer.Budget().FailAfter(ordinal)
			e = v.Push(a)
			if e == nil {
				success = true
				if len(v.children) != 1 || v.children[0] != a {
					t.Fatal("actual published pointer")
				}
			} else {
				failures++
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual reservation refusal", e)
				}
				if len(v.children) != 0 || a.sourceStarted || a.closed {
					t.Fatal("unpublished caller pointer consumed")
				}
			}
			polynomialSkeletonVectorPhysicalAudit(t, m, v, []*PolynomialSkeleton{a}, p)
			m.integer.Budget().FailAfter(-1)
			serviceMust(t, v.CloseChecked())
			if !success {
				if a.closed {
					t.Fatal("failed push caller lifetime")
				}
				serviceMust(t, a.CloseChecked())
			}
			serviceMust(t, m.DecRef(p))
			polynomialPhysicalAudit(t, m)
		})
	}
	if !success || failures == 0 {
		t.Fatal("bounded every prefix did not reach success", failures)
	}
}

func TestPolynomialSkeletonVectorSetDeleteBeforeColdReplacementRefusal(t *testing.T) {
	m := polynomialCellFixture(t)
	p, _ := polynomialSkeletonInput(t, m)
	a, e := NewPolynomialSkeleton(m, m.unit, 0)
	serviceMust(t, e)
	b, e := NewPolynomialSkeleton(m, p, 1)
	serviceMust(t, e)
	v, e := NewPolynomialSkeletonVector(m)
	serviceMust(t, e)
	serviceMust(t, v.Push(a))
	if len(v.children) != cap(v.children) {
		t.Fatal("fixture must reach actual cold directory reserve")
	}
	blocks := m.memory.Stats().LiveBlocks
	m.integer.Budget().FailAfter(0)
	e = v.Set(0, b)
	if !errors.Is(e, arena.ErrInjectedFailure) || !a.sourceFinalized || a.closed || b.sourceStarted || len(v.children) != 1 || v.children[0] != a || m.memory.Stats().LiveBlocks+3 != blocks {
		t.Fatal("source old delete before Go replacement metadata", e, blocks, m.memory.Stats().LiveBlocks)
	}
	polynomialSkeletonVectorPhysicalAudit(t, m, v, []*PolynomialSkeleton{b}, p)
	cause := v.sourceCause
	used := m.integer.Budget().Used()
	if e = v.SourceFinalize(); e != cause || m.integer.Budget().Used() != used {
		t.Fatal("stopped source history repaired", e, cause)
	}
	m.integer.Budget().FailAfter(-1)
	serviceMust(t, v.CloseChecked())
	serviceMust(t, b.CloseChecked())
	serviceMust(t, m.DecRef(p))
	polynomialPhysicalAudit(t, m)
}
func TestPolynomialSkeletonVectorOwnerGuards(t *testing.T) {
	m := polynomialCellFixture(t)
	v, e := NewPolynomialSkeletonVector(m)
	serviceMust(t, e)
	copy := *v
	if _, e = copy.Size(); !errors.Is(e, ErrPolynomialOwner) {
		t.Fatal("copied owner", e)
	}
	serviceMust(t, v.SourceFinalize())
	if e = v.Push(nil); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("source destroyed ordinary mutation", e)
	}
	serviceMust(t, v.CloseChecked())
	serviceMust(t, v.CloseChecked())
	if _, e = v.Get(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("owning closed ordinary read", e)
	}
	polynomialPhysicalAudit(t, m)
}

// Unlike the unit-only replacement test, x2 is newly owned by the old
// skeleton. Its last DEC reaches the real cold ID recycle allocation before
// any pointer replacement metadata. Keep this reached source stop distinct.
func TestPolynomialSkeletonVectorSetStoppedChildDestructor(t *testing.T) {
	m := polynomialCellFixture(t)
	p, _ := polynomialSkeletonInput(t, m)
	a, e := NewPolynomialSkeleton(m, p, 0)
	serviceMust(t, e)
	b, e := NewPolynomialSkeleton(m, p, 1)
	serviceMust(t, e)
	v, e := NewPolynomialSkeletonVector(m)
	serviceMust(t, e)
	serviceMust(t, v.Push(a))
	m.integer.Budget().FailAfter(0)
	e = v.Set(0, b)
	if !numericPhysicalReservationRefusal(e) || a.sourceFinalized || !a.sourceStarted || a.sourceStage != 0 || a.entryCursor != 2 || !a.entryDropping || b.sourceStarted || v.sourceCause != e {
		t.Fatal("actual old child DEC/ref-recycle stop", e, a.sourceStage, a.entryCursor, a.entryDropping)
	}
	polynomialSkeletonVectorPhysicalAudit(t, m, v, []*PolynomialSkeleton{b}, p)
	used := m.integer.Budget().Used()
	cause := e
	if e = v.SourceFinalize(); e != cause || m.integer.Budget().Used() != used {
		t.Fatal("source stop resumed owning DEC", e, cause)
	}
	m.integer.Budget().FailAfter(-1)
	serviceMust(t, v.CloseChecked())
	if a.sourceFinalized || a.sourceCause != cause || a.sourceStage != 0 {
		t.Fatal("owning abort changed literal source history")
	}
	serviceMust(t, b.CloseChecked())
	serviceMust(t, m.DecRef(p))
	polynomialPhysicalAudit(t, m)
}
