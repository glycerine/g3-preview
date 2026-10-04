// Source: src/math/polynomial/polynomial.cpp:2923-2952,2354. Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialNewtonVectorSourceFinalizePrivateBacking(t *testing.T) {
	m := polynomialCellFixture(t)
	serviceMust(t, m.SetZPUint64(97))
	v, e := NewPolynomialNewtonVector(m)
	serviceMust(t, e)
	serviceMust(t, v.SetOwner(m))
	n, e := v.At(0)
	serviceMust(t, e)
	x := NewInt(3)
	y := NewInt(17)
	p, e := m.MkConstantTake(&y)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(p))
	serviceMust(t, n.Add(&x, p))
	serviceMust(t, m.DecRef(p))
	polynomialNewtonPhysicalAudit(t, m, nil, nil, nil, []*PolynomialNewtonVector{v}, nil, nil)
	raw := v.data.raw
	if raw == (memory.Block{}) || v.header == 0 {
		t.Fatal("actual source children/backing")
	}
	serviceMust(t, v.SourceFinalize())
	polynomialNewtonPhysicalAudit(t, m, nil, nil, nil, []*PolynomialNewtonVector{v}, nil, nil)
	if !n.closed || v.data.raw != (memory.Block{}) || !v.sourceFinalized || v.closed || v.header == 0 {
		t.Fatal("source dtor versus owning cold cleanup")
	}
	if _, e = m.memory.Read(raw, 0); !errors.Is(e, memory.ErrRuntimeOwner) {
		t.Fatal("actual pointer-vector deallocation", e)
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
	m.integer.budget.FailAfter(1) // charged child owner, then actual inverse-vector allocation refusal.
	_, e = v.At(0)
	if !numericPhysicalReservationRefusal(e) || v.pending == nil || !v.pending.constructing {
		t.Fatal("actual failed constructor root", e, v.pending)
	}
	child := v.pending
	polynomialNewtonPhysicalAudit(t, m, nil, nil, nil, []*PolynomialNewtonVector{v}, nil, nil)
	before := m.integer.budget.Used()
	serviceMust(t, v.SourceFinalize())
	polynomialNewtonPhysicalAudit(t, m, nil, nil, nil, []*PolynomialNewtonVector{v}, nil, nil)
	if v.pending != child || child.closed || child.header == 0 || m.integer.budget.Used() != before {
		t.Fatal("failed constructor incorrectly received source class destructor")
	}
	// No source class destructor or source counter rollback consumes this
	// returned incomplete child. Only the explicit owning boundary does so.
	m.integer.budget.FailAfter(-1)
	serviceMust(t, v.Close())
	if !child.closed || v.pending != nil {
		t.Fatal("actual owning child cleanup")
	}
}

// Public At then Flush leaves an actual large retained pointer-vector member.
// The public TLS realloc recipe reaches signed-host/source undefined arithmetic;
// this is postphysical guard evidence, never an allocation-limit free refusal.
func TestPolynomialNewtonVectorSourceFinalizeStickyPostPhysicalSuffix(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	profile := memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := NewRationalManager(RationalConfig{Integers: IntegerConfig{Memory: r, Profile: profile, Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
	serviceMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: r, Profile: profile})
	serviceMust(t, e)
	serviceMust(t, mm.IncManagerRef())
	m, e := NewMultivariatePolynomialCore(MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Monomials: mm, Checkpoint: func() bool { panic("unreached") }, Cooperation: func() error { panic("unreached") }})
	serviceMust(t, e)
	v, e := NewPolynomialNewtonVector(m)
	serviceMust(t, e)
	serviceMust(t, v.SetOwner(m))
	_, e = v.At(13000)
	serviceMust(t, e)
	serviceMust(t, v.Flush())
	cap, e := v.data.capacity(m)
	serviceMust(t, e)
	if uint64(cap)*8+8 < 100000 {
		t.Fatal("real pointer vector insufficient for TLS sync", cap)
	}
	polynomialNewtonPhysicalAudit(t, m, nil, nil, nil, []*PolynomialNewtonVector{v}, nil, nil)
	// Establish real ordinary ID freelist capacity before entering the guarded
	// signed-overflow domain. This isolates the reached vector-member free;
	// unprimed ID allocation-before-free refusal is separate evidence.
	var constants [6]Polynomial
	for i := range constants {
		a := NewInt(int32(i + 2))
		constants[i], e = m.MkConstantTake(&a)
		serviceMust(t, e)
		serviceMust(t, m.IncRef(constants[i]))
	}
	for _, p := range constants {
		serviceMust(t, m.DecRef(p))
	}
	idSize, e := m.ids.Size()
	serviceMust(t, e)
	idCap, e := m.ids.Capacity()
	serviceMust(t, e)
	if idSize != 6 || idCap < idSize+2 {
		t.Fatal("actual polynomial freelist room", idSize, idCap)
	}
	_, e = mm.MkVar()
	serviceMust(t, e)
	mon, e := mm.MkVariable(0, 1)
	serviceMust(t, e)
	serviceMust(t, mm.IncRef(mon))
	serviceMust(t, mm.DecRef(mon))
	monoSize, e := mm.ids.Size()
	serviceMust(t, e)
	monoCap, e := mm.ids.Capacity()
	serviceMust(t, e)
	if monoSize != 1 || monoCap < 2 {
		t.Fatal("actual monomial freelist room", monoSize, monoCap)
	}
	polynomialNewtonPhysicalAudit(t, m, nil, nil, nil, []*PolynomialNewtonVector{v}, nil, nil)
	contextTLSNearMinimum(t, r)
	before := r.Stats().LiveBlocks
	e = v.SourceFinalize()
	if !errors.Is(e, arena.ErrBounds) || v.data.raw != (memory.Block{}) || v.sourceStage != 1 || v.sourceFinalized || r.Stats().LiveBlocks+1 != before {
		t.Fatal("actual suffix physical release before stopped sync", e, v.sourceStage, r.State())
	}
	cause := v.sourceCause
	polynomialNewtonPhysicalAudit(t, m, nil, nil, nil, []*PolynomialNewtonVector{v}, nil, nil)
	used := b.Used()
	count, e := r.AllocationCount()
	serviceMust(t, e)
	if e = v.SourceFinalize(); e != cause || b.Used() != used {
		t.Fatal("literal source retry acknowledged owning release", e, cause)
	}
	afterCount, e := r.AllocationCount()
	serviceMust(t, e)
	if afterCount != count {
		t.Fatal("retry fabricated source event")
	}
	if _, e = v.Size(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("source-stopped ordinary size", e)
	}
	serviceMust(t, v.Close())
	if v.sourceFinalized || v.sourceCause != cause || v.sourceStage != 1 {
		t.Fatal("owning abort repaired source history")
	}
	contextTLSFinish(t, r, m.CloseChecked)
	// The source reference decrement executes once. Its stopped destructor
	// continuation belongs to explicit owning Close, never another DEC.
	beforeMM := r.Stats().LiveBlocks
	e = mm.DecManagerRef()
	if e != nil {
		if !errors.Is(e, arena.ErrBounds) || mm.managerRefs != 0 || r.Stats().LiveBlocks >= beforeMM {
			t.Fatal("source MM decrement/destructor prefix", e, mm.managerRefs, r.State())
		}
		contextTLSFinish(t, r, mm.CloseChecked)
	}
	contextTLSFinish(t, r, q.CloseChecked)
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("actual owning graph retained", b.Used())
	}
}
