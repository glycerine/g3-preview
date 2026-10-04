// Source: src/math/polynomial/polynomial.cpp:193,507-544,1821-1850,2308-2344. Licensed under the MIT license.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialMemberPreparationBeforeBody(t *testing.T) {
	q, b := rationalManager(t, false)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: q.z.memory})
	serviceMust(t, e)
	serviceMust(t, mm.IncManagerRef())
	defer func() { serviceMust(t, mm.DecManagerRef()) }()
	refs := mm.managerRefs
	c := MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Monomials: mm, Checkpoint: func() bool { panic("unreached") }, Cooperation: func() error { panic("unreached") }}
	m, e := preparePolynomialCore(c, true)
	serviceMust(t, e)
	if m.modular == nil || m.directoryInitialized || m.idsInitialized || m.monomialRetained || m.completed || m.zero != (Polynomial{}) || m.unit != (Polynomial{}) || mm.managerRefs != refs {
		t.Fatal("preparation executed source body")
	}
	serviceMust(t, m.constructPolynomialCoreDefaults())
	if !m.directoryInitialized || !m.idsInitialized || m.monomialRetained || mm.managerRefs != refs {
		t.Fatal("default members executed body")
	}
	serviceMust(t, m.initializePolynomialCoreBody())
	if !m.completed || !m.monomialRetained || mm.managerRefs != refs+1 {
		t.Fatal("source body retain/init missing")
	}
	serviceMust(t, m.CloseChecked())
	if mm.managerRefs != refs {
		t.Fatal("borrowed monomial retain not released")
	}
}

func TestPolynomialRuntimeTempDefaultAndFailure(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	serviceMust(t, e)
	var tmp monomialTemp
	serviceMust(t, monomialTempConstruct(r, &tmp))
	if !tmp.completed || tmp.capacity != 128 || tmp.pending != (memory.Block{}) || tmp.raw == (memory.Block{}) {
		t.Fatal("actual default tmp")
	}
	for i, want := range [5]uint32{0, ^uint32(0), 0, 0, 0} {
		got, e := r.ReadUint32(tmp.raw, uint64(i)*4)
		serviceMust(t, e)
		if got != want {
			t.Fatal(i, got, want)
		}
	}
	if _, e = r.ReadUint32(tmp.raw, 20); e == nil {
		t.Fatal("uninitialized powers synthesized")
	}
	serviceMust(t, r.Deallocate(&tmp.raw))
	b.FailAfter(0)
	var failed monomialTemp
	e = monomialTempConstruct(r, &failed)
	if !numericPhysicalReservationRefusal(e) || failed.completed || failed.raw != (memory.Block{}) || failed.capacity != 0 {
		t.Fatal("failed placement published completed default", e, failed)
	}
	b.FailAfter(-1)
	if failed.pending != (memory.Block{}) {
		serviceMust(t, r.Deallocate(&failed.pending))
	}
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("actual runtime cleanup", b.Used())
	}
}

func TestPolynomialRuntimeTempReachedPlacementGuard(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	serviceMust(t, e)
	var tmp monomialTemp
	// uint32 source get_obj_size wraps to four actual requested bytes. RC is
	// initialized before the reached ID word guard; no early capacity preflight.
	e = monomialTempAllocate(r, &tmp, 0x1ffffffe)
	if e != arena.ErrBounds || tmp.raw != (memory.Block{}) || tmp.pending == (memory.Block{}) || tmp.completed {
		t.Fatal("actual endangered placement prefix", e, tmp)
	}
	rc, e := r.ReadUint32(tmp.pending, 0)
	serviceMust(t, e)
	if rc != 0 || r.Stats().LiveBlocks != 1 {
		t.Fatal("first actual header word/root", rc, r.Stats())
	}
	serviceMust(t, r.Deallocate(&tmp.pending))
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("explicit pending cleanup", b.Used())
	}
}
