package polynomial

// Source: src/util/scoped_ptr_vector.h; src/math/polynomial/polynomial.cpp:4070-4071.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
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

func polynomialSkeletonVectorPhysicalAudit(t *testing.T, m *MultivariatePolynomialCore, v *PolynomialSkeletonVector, callerSkeletons []*PolynomialSkeleton, callers ...Polynomial) {
	t.Helper()
	var roots []Monomial
	extra := v.header
	// Published child contexts are owned; callerSkeletons are
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
		extra += s.header
		roots = append(roots, polynomialSkeletonRoots(t, s)...)
	}
	polynomialNewtonPhysicalWithExtra(t, m, nil, nil, nil, nil, callers, nil, nil, extra)
	polynomialOperationMonomialAudit(t, m, callers, roots)
}

func TestPolynomialSkeletonVectorNativeReplacementPreservesChildOrder(t *testing.T) {
	m := polynomialCellFixture(t)
	p, _ := polynomialSkeletonInput(t, m)
	a, e := NewPolynomialSkeleton(m, m.unit, 0)
	serviceMust(t, e)
	b, e := NewPolynomialSkeleton(m, p, 1)
	serviceMust(t, e)
	v, e := NewPolynomialSkeletonVector(m)
	serviceMust(t, e)
	defer func() {
		m.integer.Budget().FailAfter(-1)
		serviceMust(t, v.CloseChecked())
		serviceMust(t, m.DecRef(p))
	}()
	serviceMust(t, v.Push(a))
	m.integer.Budget().FailAfter(0)
	serviceMust(t, v.Set(0, b))
	if !a.sourceFinalized || a.closed || b.sourceStarted || v.data[0] != b || len(v.children) != 2 || v.children[0] != a || v.children[1] != b {
		t.Fatal("native replacement lost source destruction order or owning history")
	}
	polynomialSkeletonVectorPhysicalAudit(t, m, v, nil, p)
	m.integer.Budget().FailAfter(-1)
	serviceMust(t, v.CloseChecked())
	if !a.closed || !b.closed {
		t.Fatal("owning completion missed replaced child")
	}
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
// skeleton. Its last DEC recycles an ID through native slice storage before
// the replacement pointer is published.
func TestPolynomialSkeletonVectorSetNativeChildRecycle(t *testing.T) {
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
	serviceMust(t, v.Set(0, b))
	m.integer.Budget().FailAfter(-1)
	if !a.sourceFinalized || b.sourceStarted || v.data[0] != b {
		t.Fatal("native replacement did not retire old child", a.sourceFinalized, v.data[0])
	}
	polynomialSkeletonVectorPhysicalAudit(t, m, v, nil, p)
	serviceMust(t, v.CloseChecked())
	serviceMust(t, m.DecRef(p))
	polynomialPhysicalAudit(t, m)
}
