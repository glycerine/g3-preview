// Source: src/util/ref_buffer.h; src/util/buffer.h;
// src/math/polynomial/polynomial.cpp:3895-4015.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

// Public oversized reallocations reach the guarded signed-host/overflow domain.
// This is neither defined C++ failure credit nor normal mathematical exhaustion.
func TestPolynomialGCDMonomialBufferConsumedOldGrowth(t *testing.T) {
	budget := arena.NewBudget(1 << 26)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: budget, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	q, e := NewRationalManager(RationalConfig{Integers: IntegerConfig{Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 8, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}}})
	serviceMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: budget, Memory: r})
	serviceMust(t, e)
	serviceMust(t, mm.IncManagerRef())
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Monomials: mm, Checkpoint: func() bool { panic("unreached") }, Cooperation: func() error { panic("unreached") }}})
	serviceMust(t, e)
	defer func() {
		contextTLSFinish(t, r, m.CleanupGCD)
		polynomialTLSCloseFull(t, m, r, mm)
		if e := mm.DecManagerRef(); e != nil && !errors.Is(e, arena.ErrBounds) {
			t.Fatal(e)
		}
		contextTLSFinish(t, r, mm.CloseChecked)
		contextTLSFinish(t, r, q.CloseChecked)
		contextTLSFinish(t, r, r.Close)
		if budget.Used() != 0 {
			t.Fatal("actual owning bytes", budget.Used())
		}
	}()
	s, e := m.beginGCDFunction()
	serviceMust(t, e)
	b := &s.monomialBuffer
	b.construct()
	s.stack[s.depth] = gcdMonomialBufferID
	s.depth++
	for i := 0; i < 8192; i++ {
		serviceMust(t, b.push(m.core, mm.unit))
	}
	old := b.raw
	contextTLSNearMinimum(t, r)
	lane, e := r.Allocate(8)
	serviceMust(t, e)
	delta := int64(-300000) - r.State().LocalSize
	request := uint64(16) + uint64(delta) - 8
	if e = r.Reallocate(&lane, request); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("real wrapped realloc prefix", e)
	}
	e = b.push(m.core, mm.unit)
	if !errors.Is(e, arena.ErrBounds) || b.raw != (memory.Block{}) || b.pending == (memory.Block{}) || b.copied != 8192 {
		t.Fatal("actual old free before synchronization stop", e, b.raw, b.pending, b.copied, r.State())
	}
	if _, e = r.ReadUint32(old, 0); e == nil {
		t.Fatal("old physical storage remains live")
	}
	if _, e = r.ReadUint32(b.pending, 0); e != nil {
		t.Fatal("captured copied payload is not live", e)
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, nil, nil, 0, []memory.Block{lane})
	// The source still points to consumed storage, not its inline array or the
	// owning-only copied expansion. Only owning cleanup may traverse that copy.
	if _, e = b.get(m.core, 0); e == nil {
		t.Error("source get silently selected inline storage after physical old free")
	}
	m.gcdDepth--
	m.core.pscLeave()
	e = m.finishGCDFunctionSource(s)
	if e == nil {
		t.Error("source destruction unexpectedly acknowledged stopped growth")
	}
	cause := e
	state := r.State()
	if retry := m.finishGCDFunctionSource(s); retry != cause || r.State() != state {
		t.Error("source retry changed stopped frontier", retry, cause)
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, nil, nil, 0, []memory.Block{lane})
	contextTLSFinish(t, r, m.CleanupGCD)
	if lane != (memory.Block{}) {
		contextTLSFinish(t, r, func() error {
			if lane == (memory.Block{}) {
				return nil
			}
			return r.Deallocate(&lane)
		})
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, nil, nil, 0, []memory.Block{lane})
}

// Exact captured allocation progress permits signed-host teardown continuation.
func polynomialTLSCloseFull(t *testing.T, m *MultivariatePolynomialManager, r *memory.Runtime, mm *MonomialManager) {
	t.Helper()
	for attempts := 0; ; attempts++ {
		if attempts > 64 {
			t.Fatal("bounded actual owning continuation")
		}
		oldDirectory := m.core.polynomials.raw
		var oldCells [2]*polynomialLocation
		var oldObjects [2]memory.ObjectRef
		for i, p := range [2]Polynomial{m.core.zero, m.core.unit} {
			if p != (Polynomial{}) {
				l, e := m.core.polynomialLocation(p)
				serviceMust(t, e)
				if !l.released {
					oldCells[i] = l
					oldObjects[i] = l.raw
				}
			}
		}
		live := r.Stats().LiveBlocks
		e := m.CloseChecked()
		if e == nil {
			break
		}
		if !errors.Is(e, arena.ErrBounds) {
			t.Fatal("actual reached owner refusal", e)
		}
		consumed := oldDirectory != (memory.Block{}) && m.core.polynomials.raw == (memory.Block{})
		for i, l := range oldCells {
			if l != nil && l.released {
				if _, readErr := mm.allocator.ReadUint32(oldObjects[i], 0); readErr == nil {
					t.Fatal("captured cached polynomial allocation remains readable after reached physical release")
				}
				consumed = true
			}
		}
		if consumed {
			if oldDirectory != (memory.Block{}) && m.core.polynomials.raw == (memory.Block{}) {
				if _, readErr := r.ReadUint32(oldDirectory, 0); readErr == nil {
					t.Fatal("old directory not physically consumed")
				}
			}
			if roots := m.core.ids.RawRoots(); roots[0] == (memory.Block{}) {
				t.Fatal("actual new ID backing missing", roots)
			}
			polynomialImpPhysicalAuditNamedRaw(t, m, nil, nil, 0, nil)
		}
		if !consumed && r.Stats().LiveBlocks >= live {
			t.Fatal("no reached physical owning progress", e, m.cleanupStage, m.core.closeStage, m.core.zero, m.core.unit, m.core.ids.RawRoots(), oldDirectory, m.core.polynomials.raw, live, r.Stats().LiveBlocks)
		}
	}
}
