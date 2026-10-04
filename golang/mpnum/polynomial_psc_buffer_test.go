// Source: src/util/ref_buffer.h:38-99; src/util/buffer.h:39-46,146-151;
// src/math/polynomial/polynomial.cpp:5704. Licensed under the MIT license.
// Source author: Leonardo de Moura (leonardo).
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialPSCBufferInlineAndUnjoinedPush(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{7}, [][]MonomialPower{{{0, 1}}})
	b := &m.pscBuffer
	serviceMust(t, b.construct(m))
	before := m.memory.Stats()
	for i := 0; i < 16; i++ {
		serviceMust(t, b.push(m, p))
	}
	if b.count != 16 || m.memory.Stats() != before {
		t.Fatal("actual source inline16 acquired heap storage")
	}
	rc, e := m.readWord(p, 0)
	serviceMust(t, e)
	if rc != 17 {
		t.Fatal("sixteen source INCs", rc)
	}
	m.integer.budget.FailAfter(0)
	e = b.push(m, p)
	if !numericPhysicalReservationRefusal(e) || b.count != 16 || b.incoming != p {
		t.Fatal("INC before actual growth refusal", e, b.count, b.incoming)
	}
	rc, e = m.readWord(p, 0)
	serviceMust(t, e)
	if rc != 18 {
		t.Fatal("unjoined INC must remain actual root", rc)
	}
	polynomialPSCPhysical(t, m, nil, []Polynomial{p}, nil)
	m.integer.budget.FailAfter(-1)
	serviceMust(t, b.finalize(m, false))
	rc, e = m.readWord(p, 0)
	serviceMust(t, e)
	if rc != 2 || b.incoming != p {
		t.Fatal("ordinary dtor must not acknowledge unjoined INC", rc, b.incoming)
	}
	serviceMust(t, b.finalize(m, true))
	rc, e = m.readWord(p, 0)
	serviceMust(t, e)
	if rc != 1 || b.incoming != (Polynomial{}) {
		t.Fatal("owning abort did not reach unjoined root", rc)
	}
}
func TestPolynomialPSCBufferHeapAndCapturedInitializedRange(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{3}, [][]MonomialPower{{{0, 2}}})
	b := &m.pscBuffer
	serviceMust(t, b.construct(m))
	for i := 0; i < 33; i++ {
		serviceMust(t, b.push(m, p))
	}
	if b.capacity != 64 || b.count != 33 {
		t.Fatal("source capacity doubles", b.capacity, b.count)
	}
	for i := uint32(0); i < 33; i++ {
		r, e := b.get(m, i)
		serviceMust(t, e)
		if r != p {
			t.Fatal(i, r)
		}
	}
	_, e := b.get(m, 64)
	if !errors.Is(e, arena.ErrBounds) && !errors.Is(e, ErrPolynomialAssertion) {
		t.Fatal("actual read guard", e)
	}
	serviceMust(t, b.finalize(m, false))
	rc, e := m.readWord(p, 0)
	serviceMust(t, e)
	if rc != 1 {
		t.Fatal("ascending source dtor range", rc)
	}
}

func TestPolynomialPSCBufferReturnedAllocationBeforeSourceDestruction(t *testing.T) {
	budget := arena.NewBudget(1 << 28)
	runtime, e := memory.NewRuntime(memory.RuntimeConfig{Budget: budget, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	profile := memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := NewRationalManager(RationalConfig{Integers: IntegerConfig{Memory: runtime, Profile: profile, Cells: arena.Config{PageSize: 2, Budget: budget}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: budget}}}})
	serviceMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: budget, Memory: runtime, Profile: profile})
	serviceMust(t, e)
	serviceMust(t, mm.IncManagerRef())
	m, e := NewMultivariatePolynomialCore(MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Monomials: mm, Checkpoint: func() bool { t.Fatal("buffer checkpoint"); return false }, Cooperation: func() error { t.Fatal("buffer cooperation"); return nil }})
	serviceMust(t, e)
	_, e = mm.MkVar()
	serviceMust(t, e)
	p, e := m.MkVariable(0, 1)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(p))
	b := &m.pscBuffer
	serviceMust(t, b.construct(m))
	for i := 0; i < 16; i++ {
		serviceMust(t, b.push(m, p))
	}
	local := runtime.State().LocalSize
	if local < 0 || local >= 99972 {
		t.Fatal("fixture actual local", runtime.State())
	}
	prep, e := runtime.Allocate(uint64(99980 - local - 8))
	serviceMust(t, e)
	before := runtime.Stats()
	serviceMust(t, runtime.SetMaxSize(1))
	e = b.push(m, p)
	var failure memory.Failure
	if !errors.As(e, &failure) || failure.Kind != memory.FailureMemout || !runtime.State().OutOfMemory || b.pending == (memory.Block{}) {
		t.Fatal("actual allocated-then-refused prefix", e, runtime.State(), b.pending)
	}
	if b.raw != (memory.Block{}) || b.count != 16 || b.incoming != p || runtime.Stats().LiveBlocks != before.LiveBlocks+1 {
		t.Fatal("actual roots/published range", b, runtime.Stats())
	}
	rc, e := m.readWord(p, 0)
	serviceMust(t, e)
	if rc != 18 {
		t.Fatal(rc)
	}
	polynomialPSCPhysical(t, m, nil, []Polynomial{p}, []memory.Block{prep})
	serviceMust(t, b.finalize(m, false))
	rc, e = m.readWord(p, 0)
	serviceMust(t, e)
	if rc != 2 || b.pending == (memory.Block{}) || b.incoming != p || runtime.Stats().LiveBlocks != before.LiveBlocks+1 {
		t.Fatal("source dtor consumed failed-push owning roots", rc, b, runtime.Stats())
	}
	polynomialPSCPhysical(t, m, nil, []Polynomial{p}, []memory.Block{prep})
	// Lift only the explicit fixture limit, then perform owning abort separately.
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, b.finalize(m, true))
	rc, e = m.readWord(p, 0)
	serviceMust(t, e)
	if rc != 1 || runtime.Stats().LiveBlocks != before.LiveBlocks {
		t.Fatal("owning abort", rc, runtime.Stats())
	}
	serviceMust(t, runtime.Deallocate(&prep))
	serviceMust(t, m.DecRef(p))
	serviceMust(t, m.CloseChecked())
	serviceMust(t, mm.DecManagerRef())
	serviceMust(t, q.CloseChecked())
	serviceMust(t, runtime.Close())
}

func TestPolynomialPSCBufferPendingLocalBlocksOwningCoreClose(t *testing.T) {
	m := polynomialCellFixture(t)
	p, e := m.MkVariable(0, 1)
	serviceMust(t, e)
	b := &m.pscBuffer
	serviceMust(t, b.construct(m))
	serviceMust(t, b.push(m, p))
	t.Cleanup(func() { serviceMust(t, b.finalize(m, true)) })
	// The only extra polynomial reference is the real completed H local.
	if e = m.CloseChecked(); !errors.Is(e, ErrPolynomialPending) {
		t.Fatal("owning close must identify actual pending local before caller guard", e)
	}
	serviceMust(t, b.finalize(m, false))
}
