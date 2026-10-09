// Source: src/math/polynomial/polynomial.cpp:2923-2952,2354.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialNewtonVectorSourceFinalizeNativeStorage(t *testing.T) {
	m := polynomialCellFixture(t)
	serviceMust(t, m.SetZPUint64(97))
	v, e := NewPolynomialNewtonVector(m)
	serviceMust(t, e)
	serviceMust(t, v.SetOwner(m))
	n, e := v.At(0)
	serviceMust(t, e)
	x := util.NewInt(3)
	y := util.NewInt(17)
	p, e := m.MkConstantTake(&y)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(p))
	serviceMust(t, n.Add(&x, p))
	serviceMust(t, m.DecRef(p))
	polynomialNewtonPhysicalAudit(t, m, nil, nil, nil, []*PolynomialNewtonVector{v}, nil, nil)
	if len(v.children) != 1 || v.header == 0 {
		t.Fatal("actual source child missing")
	}
	serviceMust(t, v.SourceFinalize())
	polynomialNewtonPhysicalAudit(t, m, nil, nil, nil, []*PolynomialNewtonVector{v}, nil, nil)
	if !n.closed || v.children != nil || !v.sourceFinalized || v.closed || v.header == 0 {
		t.Fatal("source dtor versus owning cold cleanup")
	}

	if _, e = v.At(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("source-destroyed ordinary entry", e)
	}
	serviceMust(t, v.SourceFinalize())
	serviceMust(t, v.Close())
}

func TestPolynomialNewtonVectorSourceFinalizeLeavesFailedChild(t *testing.T) {
	m := polynomialCellFixture(t)
	v, e := NewPolynomialNewtonVector(m)
	serviceMust(t, e)
	serviceMust(t, v.SetOwner(m))
	m.integer.Budget().FailAfter(1) // charged child owner, then actual inverse-vector allocation refusal.
	_, e = v.At(0)
	if !numericPhysicalReservationRefusal(e) || v.pending == nil || !v.pending.constructing {
		t.Fatal("actual failed constructor root", e, v.pending)
	}
	child := v.pending
	polynomialNewtonPhysicalAudit(t, m, nil, nil, nil, []*PolynomialNewtonVector{v}, nil, nil)
	before := m.integer.Budget().Used()
	serviceMust(t, v.SourceFinalize())
	polynomialNewtonPhysicalAudit(t, m, nil, nil, nil, []*PolynomialNewtonVector{v}, nil, nil)
	if v.pending != child || child.closed || child.header == 0 || m.integer.Budget().Used() != before {
		t.Fatal("failed constructor incorrectly received source class destructor")
	}
	// No source class destructor or source counter rollback consumes this
	// returned incomplete child. Only the explicit owning boundary does so.
	m.integer.Budget().FailAfter(-1)
	serviceMust(t, v.Close())
	if !child.closed || v.pending != nil {
		t.Fatal("actual owning child cleanup")
	}
}
