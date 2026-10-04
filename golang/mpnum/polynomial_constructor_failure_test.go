// Source: src/math/polynomial/polynomial.cpp:2324-2344,2308-2321. Licensed under the MIT license.
package mpnum

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/utilid"
	"testing"
)

func TestPolynomialConstructorColdHeaderRefusalOwnership(t *testing.T) {
	for _, fault := range []int64{0, 1} {
		t.Run(string(rune('0'+fault)), func(t *testing.T) {
			q, b := rationalManager(t, false)
			mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: q.z.memory})
			serviceMust(t, e)
			serviceMust(t, mm.IncManagerRef())
			before := b.Used()
			count, e := q.z.memory.AllocationCount()
			serviceMust(t, e)
			b.FailAfter(fault)
			m, e := NewMultivariatePolynomialCore(MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Monomials: mm, Checkpoint: func() bool { t.Fatal("constructor checkpoint"); return false }, Cooperation: func() error { t.Fatal("constructor cooperation"); return nil }})
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal("actual cold reservation category", e)
			}
			after, e2 := q.z.memory.AllocationCount()
			serviceMust(t, e2)
			if after != count {
				t.Fatal("cold headers are not source malloc calls")
			}
			if fault == 0 {
				if m != nil || b.Used() != before {
					t.Fatal("unallocated owner retained")
				}
			} else {
				if m == nil || m.header == 0 || m.modular != nil || m.completed || m.originalCause != arena.ErrInjectedFailure || m.directoryInitialized || m.directorySourceDestroyed || m.modularSourceDestroyed {
					t.Fatal("exact retained owner/member prefix", m)
				}
				if b.Used() != before+m.header {
					t.Fatal("unused child header must not be charged")
				}
				if e = m.SourceFinalizePrefix(); e != arena.ErrClosed {
					t.Fatal("failed ctor is not completed destructor", e)
				}
				b.FailAfter(-1)
				serviceMust(t, m.CloseChecked())
				if b.Used() != before {
					t.Fatal("retained cold owner cleanup")
				}
			}
			b.FailAfter(-1)
			serviceMust(t, mm.DecManagerRef())
		})
	}
}

// Every real body reservation, through first complete selected owner. Source
// counters, physical roots and alreadyconsumed caller/header prefixes are
// inspected before changing injection. This does not claim full imp unwind.
func TestPolynomialConstructorBodyFailureNamedRoots(t *testing.T) {
	success := false
	for fault := int64(2); fault < 80 && !success; fault++ {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			b := arena.NewBudget(1 << 24)
			profile := memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
			q, e := NewRationalManager(RationalConfig{Integers: IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Profile: profile}})
			serviceMust(t, e)
			mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: q.z.memory, Profile: profile})
			serviceMust(t, e)
			serviceMust(t, mm.IncManagerRef())
			b.FailAfter(fault)
			m, e := NewMultivariatePolynomialCore(MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Monomials: mm, Checkpoint: func() bool { t.Fatal("body checkpoint"); return false }, Cooperation: func() error { t.Fatal("body cooperation"); return nil }})
			if e == nil {
				success = true
				if m == nil || !m.completed {
					t.Fatal("successful selected construction")
				}
			} else {
				if m == nil || !numericPhysicalReservationRefusal(e) || m.originalCause == nil || m.completed {
					t.Fatal("actual body error/owner", m, e)
				}
				if errors.Is(e, ErrPolynomialSourceMember) {
					t.Fatal("failed constructor substituted source destructor")
				}
			}
			polynomialPhysicalAudit(t, m)
			b.FailAfter(-1)
			serviceMust(t, m.CloseChecked())
			serviceMust(t, mm.DecManagerRef())
			serviceMust(t, q.CloseChecked())
			if b.Used() != 0 {
				t.Fatal("exact physical final cleanup", b.Used())
			}
		})
	}
	if !success {
		t.Fatal("bounded campaign never reached actual constructor success")
	}
}

func TestPolynomialConstructorCompletedMembersUnwindBeforeReturn(t *testing.T) {
	found := false
	for fault := int64(2); fault < 80 && !found; fault++ {
		b := arena.NewBudget(1 << 24)
		profile := memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
		q, e := NewRationalManager(RationalConfig{Integers: IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Profile: profile}})
		serviceMust(t, e)
		mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: q.z.memory, Profile: profile})
		serviceMust(t, e)
		serviceMust(t, mm.IncManagerRef())
		b.FailAfter(fault)
		m, e := NewMultivariatePolynomialCore(MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Monomials: mm, Checkpoint: func() bool { t.Fatal("unwind checkpoint"); return false }, Cooperation: func() error { t.Fatal("unwind cooperate"); return nil }})
		if e != nil && m.zero != (Polynomial{}) {
			found = true
			// The body-created zero is not an automatic destructor root. Completed
			// ID then directory members unwind independently, while that cell and the
			// actual body MM retain remain named for explicit owning abort.
			if _, x := m.ids.Range(); x != utilid.ErrOwner {
				t.Errorf("source ID destructor missing: %v", x)
			}
			if m.constructorUnwindStage != 3 || m.constructorUnwinding || m.cleanupCause != nil || !m.idsSourceDestroyed || !m.directorySourceDestroyed || !m.modularSourceDestroyed || m.constructorModularIndex != 7 {
				t.Error("actual completed selected-member cursor and original-only cause")
			}
			if m.sourceStarted || m.sourceStage != 0 {
				t.Error("member unwind substituted completed imp destructor")
			}
			if m.polynomials.raw != (memory.Block{}) {
				t.Error("published directory backing survived completed member unwind")
			}
			if m.header == 0 || m.modular == nil || m.modular.header == 0 || m.completed || !m.monomialRetained {
				t.Error("charged retained owner/member/body distinction")
			}
			l, x := m.polynomialLocation(m.zero)
			serviceMust(t, x)
			if !l.live || l.released {
				t.Error("failed constructor invoked completed imp body")
			}
			polynomialPhysicalAudit(t, m)
		}
		b.FailAfter(-1)
		serviceMust(t, m.CloseChecked())
		serviceMust(t, mm.DecManagerRef())
		serviceMust(t, q.CloseChecked())
		if b.Used() != 0 {
			t.Fatal("owning whole graph cleanup", b.Used())
		}
	}
	if !found {
		t.Fatal("campaign did not reach real post-zero body refusal")
	}
}
