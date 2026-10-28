package polynomial

// Source: src/math/polynomial/polynomial.cpp:1821-1850,2303-2370,2460-2461,3185,3214,3875,4070-4071,6234,6344,6865.

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// Independent physical/default-state assertions for the real complete imp
// composition. These do not credit the registered PSC driver or its algorithms.
func TestPolynomialImpBorrowedMonomialConstructorHierarchy(t *testing.T) {
	q, budget := rationalManager(t, false)
	mm, err := NewMonomialManager(MonomialManagerConfig{Budget: budget, Memory: q.Integers().Memory})
	serviceMust(t, err)
	serviceMust(t, mm.IncManagerRef())
	defer func() { serviceMust(t, mm.DecManagerRef()) }()
	before := mm.managerRefs
	m, err := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{
		Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm,
			Checkpoint:  func() bool { panic("constructor source never checkpoints") },
			Cooperation: func() error { panic("constructor source never cooperates") }},
	})
	serviceMust(t, err)
	if m.core == nil || m.upm == nil || m.upm.modular == m.core.modular || m.upm.config.Integer != q.Integers() || m.core.monomials != mm || mm.managerRefs != before+1 {
		t.Fatal("actual separate UP/coefficient managers or MM body retain missing")
	}
	if m.core.tmp1.powers != nil || &m.core.tmp1 == &mm.temps[1] {
		t.Fatal("native imp scratch is a distinct empty member")
	}

	if !m.useSparseGCD || m.usePRSGCD || !m.core.completed {
		t.Fatal("post-init source flag/body publication missing")
	}
	p := m.core.unit
	info, err := m.core.Info(p)
	serviceMust(t, err)
	if info.Size != 1 || info.RefCount != 1 {
		t.Fatal("source unit private ownership", info)
	}
	serviceMust(t, m.CloseChecked())
	if mm.managerRefs != before {
		t.Fatal("borrowed MM retain not released")
	}
}

// The second source constructor allocates a real MM from the borrowed allocator;
// its own destruction must leave that supplied allocator and Runtime usable.
func TestPolynomialImpBorrowedAllocatorConstructorHierarchy(t *testing.T) {
	q, budget := rationalManager(t, false)
	allocator, e := umemory.NewSmallAllocator(umemory.SmallAllocatorConfig{Runtime: q.Integers().Memory, Profile: umemory.SmallAllocatorProfile{PointerBits: 64}})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{
		Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q,
			Checkpoint:  func() bool { panic("unreached source ctor checkpoint") },
			Cooperation: func() error { panic("unreached source ctor cooperation") }},
		Allocator: allocator,
	})
	serviceMust(t, e)
	if m.core.monomials == nil || m.core.monomials.allocator != allocator || m.core.monomials.managerRefs != 1 || m.core.monomials.memory != q.Integers().Memory {
		t.Fatal("second source constructor did not create/retain the actual borrowed-allocator MM")
	}
	// Source set_zp only changes m_manager, not distinct independently Z m_upm.
	serviceMust(t, m.core.modular.SetZPUint64(7))
	if !m.upm.modular.IntegerMode() {
		t.Fatal("imp set_zp accidentally changed distinct UP member")
	}
	serviceMust(t, m.CloseChecked())
	if e = q.Integers().Memory.CheckOpen(); e != nil {
		t.Fatal("borrowed Runtime destroyed", e)
	}
	raw, e := allocator.Allocate(20)
	serviceMust(t, e)
	serviceMust(t, allocator.Deallocate(20, raw))
	serviceMust(t, allocator.Close())
	_ = budget
}

// Literal source destruction releases the body fields, drops MM before reverse
// members, then leaves the genuinely charged Go owner for explicit completion.
func TestPolynomialImpSourceDestructorSeparateOwningClose(t *testing.T) {
	q, b := rationalManager(t, false)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: q.Integers().Memory})
	serviceMust(t, e)
	serviceMust(t, mm.IncManagerRef())
	defer func() { serviceMust(t, mm.DecManagerRef()) }()
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { panic("unreached") }, Cooperation: func() error { panic("unreached") }}})
	serviceMust(t, e)
	before := mm.managerRefs
	m.reserveCompositionDegrees(17)
	m.lexPermutation = []uint32{1, 0}
	m.lexBuckets = [][]uint32{{0}, {1}}
	m.varDegrees = []uint32{1, 2, 3}
	serviceMust(t, m.core.substitution.found.reserve(5))
	serviceMust(t, m.SourceFinalize())
	if !m.sourceFinalized || m.closed || m.bodyStage != 11 || m.memberStage != 23 || m.header == 0 || !m.upm.sourceFinalized || m.core.tmp1.powers != nil || mm.managerRefs != before-1 || m.varDegrees != nil || m.degree2pos != nil || m.lexPermutation != nil || m.lexBuckets != nil {
		t.Fatal("literal complete body/member order or retained owner", m.bodyStage, m.memberStage)
	}
	count, e := q.Integers().Memory.AllocationCount()
	serviceMust(t, e)
	serviceMust(t, m.SourceFinalize())
	after, e := q.Integers().Memory.AllocationCount()
	serviceMust(t, e)
	if after != count {
		t.Fatal("source completion replay")
	}
	if _, e = m.core.MkVariable(0, 1); e != arena.ErrClosed {
		t.Fatal("source begun core guard", e)
	}
	serviceMust(t, m.CloseChecked())
}

func TestPolynomialImpConstructorEveryReservationUnwindsMembersOnly(t *testing.T) {
	failures := 0
	success := false
	for ordinal := int64(0); ordinal < 80; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			q, b := rationalManager(t, false)
			mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: q.Integers().Memory})
			serviceMust(t, e)
			serviceMust(t, mm.IncManagerRef())
			defer func() { serviceMust(t, mm.DecManagerRef()) }()
			beforeRefs := mm.managerRefs
			beforeBytes := b.Used()
			beforeRaw := q.Integers().Memory.Stats().LiveBlocks
			b.FailAfter(ordinal)
			m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { panic("unreached") }, Cooperation: func() error { panic("unreached") }}})
			if e != nil {
				failures++
				if !numericPhysicalReservationRefusal(errors.Unwrap(e)) && !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual refusal category", e)
				}
				if m == nil {
					if b.Used() != beforeBytes || q.Integers().Memory.Stats().LiveBlocks != beforeRaw || mm.managerRefs != beforeRefs {
						t.Fatal("first raw Go header refusal changed roots")
					}
				} else {
					if m.originalCause == nil || m.completed || m.bodyStage != 0 || m.sourceStarted {
						t.Fatal("failed constructor ran completed imp destructor")
					}
					if m.core == nil {
						if b.Used() != beforeBytes+m.header || q.Integers().Memory.Stats().LiveBlocks != beforeRaw || mm.managerRefs != beforeRefs {
							t.Fatal("failed child raw header roots")
						}
					} else {
						polynomialImpPhysicalAudit(t, m)
					}
				}
			} else {
				success = true
				polynomialImpPhysicalAudit(t, m)
			}
			// Immediate physical census precedes lifting the actual reservation refusal.
			b.FailAfter(-1)
			if m != nil {
				serviceMust(t, m.CloseChecked())
			}
			if mm.managerRefs != beforeRefs {
				t.Fatal("owning abort did not release only actual body MM retain")
			}
		})
		if success {
			break
		}
	}
	if failures == 0 || !success {
		t.Fatal("bounded real constructor campaign incomplete", failures, success)
	}
}

func TestPolynomialImpAllocatorAndDefaultConstructorEveryReservation(t *testing.T) {
	for _, borrowed := range []bool{true, false} {
		t.Run(fmt.Sprint("borrowedAllocator=", borrowed), func(t *testing.T) {
			failures, success := 0, false
			for ordinal := int64(0); ordinal < 100; ordinal++ {
				t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
					q, b := rationalManager(t, false)
					var a *umemory.SmallAllocator
					var e error
					if borrowed {
						a, e = umemory.NewSmallAllocator(umemory.SmallAllocatorConfig{Runtime: q.Integers().Memory, Profile: umemory.SmallAllocatorProfile{PointerBits: 64}})
						serviceMust(t, e)
					}
					baseline := b.Used()
					b.FailAfter(ordinal)
					m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { panic("unreached constructor checkpoint") }, Cooperation: func() error { panic("unreached constructor cooperation") }}, Allocator: a})
					if e != nil {
						failures++
						if !numericPhysicalReservationRefusal(e) {
							t.Fatal("actual constructor refusal", e)
						}
						if m != nil {
							if m.completed || m.bodyStage != 0 || m.sourceStarted || m.originalCause == nil {
								t.Fatal("failed constructor executed source body")
							}
							if m.core != nil {
								polynomialImpPhysicalAudit(t, m, a)
							} else if b.Used() != baseline+m.header {
								t.Fatal("failed core header census")
							}
						} else if b.Used() != baseline {
							t.Fatal("first header refusal retained unnamed root")
						}
					} else {
						success = true
						polynomialImpPhysicalAudit(t, m, a)
					}
					b.FailAfter(-1)
					if m != nil {
						serviceMust(t, m.CloseChecked())
					}
					if a != nil {
						serviceMust(t, a.Close())
					}
				})
				if success {
					break
				}
			}
			if failures == 0 || !success {
				t.Fatal("bounded full constructor campaign incomplete", failures, success)
			}
		})
	}
}

func TestPolynomialImpOwningCloseRejectsActualCallerBeforeDestruction(t *testing.T) {
	q, b := rationalManager(t, false)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: q.Integers().Memory})
	serviceMust(t, e)
	serviceMust(t, mm.IncManagerRef())
	defer func() { serviceMust(t, mm.DecManagerRef()) }()
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { panic("unreached") }, Cooperation: func() error { panic("unreached") }}})
	serviceMust(t, e)
	_, e = mm.MkVar()
	serviceMust(t, e)
	p, e := m.core.MkVariable(0, 1)
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(p))
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
	before := b.Used()
	raw := q.Integers().Memory.Stats()
	refs := mm.managerRefs
	if e = m.CloseChecked(); e != ErrPolynomialLive || m.closing || m.bodyStage != 0 || m.memberStage != 0 || b.Used() != before || q.Integers().Memory.Stats() != raw || mm.managerRefs != refs {
		t.Fatal("caller ownership preguard changed source graph", e)
	}
	if _, e = m.core.Info(p); e != nil {
		t.Fatal("caller refusal closed ordinary owner", e)
	}
	serviceMust(t, m.core.DecRef(p))
	serviceMust(t, m.CloseChecked())
	if e = m.ready(); e != arena.ErrClosed {
		t.Fatal("closed owner readiness", e)
	}
	copied := *m
	if e = copied.ready(); e != ErrPolynomialOwner {
		t.Fatal("copied owner readiness", e)
	}
}

func TestPolynomialImpUPStoppedLeafPreservesParentSuffix(t *testing.T) {
	b := arena.NewBudget(1 << 25)
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: unumeral.IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64}, Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
	serviceMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: r})
	serviceMust(t, e)
	serviceMust(t, mm.IncManagerRef())
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { panic("unreached") }, Cooperation: func() error { panic("unreached") }}})
	serviceMust(t, e)
	digits := make([]uint32, 40000)
	digits[0] = 17
	digits[len(digits)-1] = 1
	serviceMust(t, q.Integers().SetLimbs(&(*m.upm.modular.BorrowModulus()), digits))
	old := (*m.upm.modular.BorrowModulus()).Cell
	polynomialImpPhysicalAudit(t, m)
	contextTLSNearMinimum(t, r)
	e = m.SourceFinalize()
	if !errors.Is(e, arena.ErrBounds) || m.memberStage != 22 || m.upm.sourceStage != 4 || m.upm.sourceIndex != 0 || m.sourceFinalized || m.core.modularSourceDestroyed {
		t.Fatal("actual UP leaf stop/suffix", e, m.memberStage, m.upm.sourceStage, m.upm.sourceIndex)
	}
	cell := q.Integers().Cell(old)
	if !cell.Released || !cell.CleanupPending || (*m.upm.modular.BorrowModulus()).Cell != old {
		t.Fatal("retained actual consumed UP descriptor", cell)
	}
	polynomialImpPhysicalAudit(t, m)
	state := r.State()
	if again := m.SourceFinalize(); again != e || r.State() != state {
		t.Fatal("parent source retried actual child release", again)
	}
	contextTLSFinish(t, r, m.CloseChecked)
	if m.sourceFinalized || m.memberStage != 22 || m.sourceCause != e || m.upm.sourceCause != e {
		t.Fatal("owning continuation changed stopped source history")
	}
	if drop := mm.DecManagerRef(); drop != nil && !errors.Is(drop, arena.ErrBounds) {
		t.Fatal(drop)
	}
	contextTLSFinish(t, r, mm.CloseChecked)
	contextTLSFinish(t, r, q.CloseChecked)
	contextTLSFinish(t, r, r.Close)
	if b.Used() != 0 {
		t.Fatal("actual retained owner graph", b.Used())
	}
}
