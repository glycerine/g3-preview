// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"testing"
)

func upolyContextOwner(t *testing.T) (*RationalService, *UPolynomialManager, UPolynomial) {
	t.Helper()
	s, _ := serviceNew(t)
	m, e := s.NewUPolynomialManager(UPolynomialConfig{Checkpoint: func() bool { panic("constant set must not checkpoint") }, Cooperation: func() error { panic("constant set must not cooperate") }})
	serviceMust(t, e)
	p, e := m.NewPolynomial()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.DelPolynomial(&p)); serviceMust(t, m.Close()) })
	return s, m, p
}

func TestUPolynomialContextRawConstantCopy(t *testing.T) {
	s, m, p := upolyContextOwner(t)
	a, e := s.NewValue()
	serviceMust(t, e)
	// Public source fraction setter publishes finite 1/0; ordinary arithmetic reading
	// remains invalid, but constant conversion copies before structural IsInt.
	one, zero := NewInt(1), NewInt(0)
	serviceMust(t, s.manager.SetParts(&a.raw, &one, &zero))
	want := errors.New("source integer coefficient exception")
	called := false
	e = s.WithUPolynomialRationals(m, func(c *UPolynomialRationalContext) error {
		return c.SetConstant(p, &a, func() error { called = true; return want })
	})
	if !called || !errors.Is(e, want) {
		t.Fatalf("Copy→IsInt→source exception: called=%v err=%v", called, e)
	}
	serviceAuditRationalRoots(t, s, &a.raw)
	serviceMust(t, s.Del(&a))
}

func TestUPolynomialContextBorrowIdentity(t *testing.T) {
	s, m, p := upolyContextOwner(t)
	a := serviceValue(t, s, "3")
	var copied UPolynomialRationalContext
	serviceMust(t, s.WithUPolynomialRationals(m, func(c *UPolynomialRationalContext) error { copied = *c; return nil }))
	if _, _, e := copied.Exponent(&a); !errors.Is(e, ErrRationalServiceOwner) {
		t.Fatalf("copied borrowed context escaped lock: %v", e)
	}
	if e := copied.SetIdentity(p); !errors.Is(e, ErrRationalServiceOwner) {
		t.Fatal(e)
	}
	if e := copied.SetConstant(p, &a, func() error { t.Fatal("invalid borrow reached callback"); return nil }); !errors.Is(e, ErrRationalServiceOwner) {
		t.Fatal(e)
	}
	copied.StopVectors(arena.ErrBounds)
	if copied.Pending() || s.lexical.pending() || s.lexical.cause != nil {
		t.Fatal("escaped context poisoned actual owner")
	}

	serviceAuditRationalRoots(t, s, &a.raw)
	serviceMust(t, s.Del(&a))
}

func TestUPolynomialContextVectorStopAPI(t *testing.T) {
	s, m, _ := upolyContextOwner(t)
	serviceMust(t, s.WithUPolynomialRationals(m, func(c *UPolynomialRationalContext) error {
		c.StopVectors(nil)
		if c.Pending() {
			t.Fatal("nil stop must not invent pending cleanup")
		}
		return nil
	}))
}

// Public realloc operations publish the actual LP64 TLS lane before the host
// extent guard. This fixture deliberately reaches source signed-overflow UB;
// it is not ordinary exhaustion and never writes private Runtime counters.
func contextTLSNearMinimum(t *testing.T, r *memory.Runtime) {
	t.Helper()
	first, e := r.Allocate(8)
	serviceMust(t, e)
	second, e := r.Allocate(8)
	serviceMust(t, e)
	delta := int64(-1000000)
	request := uint64(16) + uint64(int64(delta)) - 8
	if e = r.Reallocate(&first, request); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	serviceMust(t, r.Deallocate(&first))
	state := r.State()
	if state.LocalSize != 0 || state.GlobalSize >= 0 {
		t.Fatal("first real TLS synchronization", state)
	}
	target := int64(math.MinInt64) + 1000 - state.GlobalSize + 16
	request = uint64(target) + 16 - 8
	if e = r.Reallocate(&second, request); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	serviceMust(t, r.Deallocate(&second))
	if state = r.State(); state.GlobalSize != math.MinInt64+1000 || state.LocalSize != 0 {
		t.Fatal("public TLS fixture", state)
	}
}

func contextTLSOwner(t *testing.T) (*RationalService, *UPolynomialManager, UPolynomial, *memory.Runtime, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 25)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(b)
	cfg.Integers.Memory = r
	cfg.Integers.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	s, e := NewRationalService(cfg)
	serviceMust(t, e)
	m, e := s.NewUPolynomialManager(UPolynomialConfig{Checkpoint: func() bool { panic("constant route checkpoint") }, Cooperation: func() error { panic("constant route cooperation") }})
	serviceMust(t, e)
	p, e := m.NewPolynomial()
	serviceMust(t, e)
	t.Cleanup(func() {
		b.FailAfter(-1)
		serviceMust(t, s.CleanupUPolynomialRationals(func() error { return nil }))
		contextTLSFinish(t, r, func() error { return m.Close() })
		contextTLSFinish(t, r, func() error { return s.Close() })
		contextTLSFinish(t, r, func() error { return r.Close() })
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	})
	return s, m, p, r, b
}

func contextBigOdd(t *testing.T, s *RationalService, den int32) OwnedRational {
	t.Helper()
	a, e := s.NewValue()
	serviceMust(t, e)
	limbs := make([]uint32, 32768)
	limbs[0] = 17
	limbs[len(limbs)-1] = 1
	serviceMust(t, s.manager.z.SetLimbs(&a.raw.num, limbs))
	a.raw.den = NewInt(den)
	return a
}

func contextAudit(t *testing.T, s *RationalService, a *OwnedRational, m *UPolynomialManager, args ...UPolynomial) {
	t.Helper()
	contextAuditAll(t, s, a, m, args)
}
func contextAuditAll(t *testing.T, s *RationalService, a *OwnedRational, m *UPolynomialManager, args []UPolynomial, others ...*OwnedRational) {
	t.Helper()
	roots := []*Int{&a.raw.num, &a.raw.den}
	for _, v := range others {
		roots = append(roots, &v.raw.num, &v.raw.den)
	}
	for _, q := range []*Rational{&s.zero, &s.one, &s.minusOne, &s.infZero.first, &s.infZero.second, &s.infOne.first, &s.infOne.second, &s.infMinusOne.first, &s.infMinusOne.second, &s.infIntZero.first, &s.infIntOne.first, &s.infIntMinusOne.first} {
		roots = append(roots, &q.num, &q.den)
	}
	for i := range s.cache {
		roots = append(roots, &s.cache[i].num, &s.cache[i].den)
	}
	for i := uint8(0); i < s.lexical.count; i++ {
		if s.lexical.done&(1<<i) == 0 {
			roots = append(roots, &s.lexical.roots[i].num, &s.lexical.roots[i].den)
		}
	}
	for _, p := range args {
		d, e := m.descriptor(p)
		serviceMust(t, e)
		values := m.values(&d.state)
		for i := uint32(0); i < d.state.size; i++ {
			roots = append(roots, &values[i])
		}
	}
	qopsAuditLiveOwnership(t, s.manager, roots)
}

func TestUPolynomialContextExceptionBeforeCompletedDestructor(t *testing.T) {
	s, m, p, r, _ := contextTLSOwner(t)
	a := contextBigOdd(t, s, 2)
	original := errors.New("source integer coefficient exception")
	called := false
	e := s.WithUPolynomialRationals(m, func(c *UPolynomialRationalContext) error {
		return c.SetConstant(p, &a, func() error { called = true; contextTLSNearMinimum(t, r); return original })
	})
	if !called || !errors.Is(e, original) || !errors.Is(e, arena.ErrBounds) {
		t.Fatalf("source exception missing: %v", e)
	}
	state := s.lexical.state()
	if state.Roots != 1 || state.Completed != 1 || state.Aborts != 0 || !errors.Is(state.Cause, arena.ErrBounds) || !errors.Is(state.OriginalCause, original) {
		t.Fatal(state)
	}
	// The source raw free completed; its stopped descriptor is still the
	// completed local's actual retained root, with no independent registry root.
	ref := s.lexical.roots[0].num.cell
	cell := s.manager.z.cell(ref)
	if !cell.released {
		t.Fatal("physical free did not precede TLS guard", cell)
	}
	contextAudit(t, s, &a, m, p)
	serviceMust(t, s.CleanupUPolynomialRationals(func() error { return nil }))
	contextTLSDelete(t, s, &a, r)
	serviceMust(t, m.DelPolynomial(&p))
}

func TestUPolynomialContextPartialCopyWaitsForStoppedVector(t *testing.T) {
	s, m, p, r, b := contextTLSOwner(t)
	a := contextBigOdd(t, s, 1)
	serviceMust(t, m.SetRationals(p, []Rational{a.raw}))
	contextTLSNearMinimum(t, r)
	b.FailAfter(0)
	var original, stopped error
	e := s.WithUPolynomialRationals(m, func(c *UPolynomialRationalContext) error {
		original = c.SetConstant(p, &a, func() error { t.Fatal("copy refused before integer predicate"); return nil })
		if !numericPhysicalReservationRefusal(original) {
			t.Fatal(original)
		}
		if c.Pending() {
			t.Fatal("partial copy alone stops completed vector unwind")
		}
		stopped = m.DelPolynomial(&p)
		if !errors.Is(stopped, arena.ErrBounds) {
			t.Fatal(stopped)
		}
		c.StopVectors(stopped)
		if !c.Pending() {
			t.Fatal("actual stopped vector not retained")
		}
		return errors.Join(original, stopped)
	})
	if !errors.Is(e, original) || !errors.Is(e, stopped) {
		t.Fatal(e)
	}
	state := s.lexical.state()
	if state.Aborts != 1 || state.Completed != 0 || !errors.Is(state.Cause, stopped) || !errors.Is(state.OriginalCause, original) {
		t.Fatal(state)
	}
	if s.lexical.roots[0].num.cell != s.manager.z.failedRef {
		t.Fatal("partial copy adapter drained before older vector")
	}
	contextAudit(t, s, &a, m, p)
	b.FailAfter(-1)
	serviceMust(t, s.manager.z.CleanupPendingAllocation())
	serviceMust(t, s.CleanupUPolynomialRationals(func() error { return m.DelPolynomial(&p) }))
	if s.lexical.pending() {
		t.Fatal("cleanup did not finish")
	}
	contextTLSDelete(t, s, &a, r)

}

// Each ErrBounds is the actual postphysical TLS synchronization refusal. Retry
// acknowledges that exact release; no source counter or pointer is repaired.
func contextTLSDelete(t *testing.T, s *RationalService, a *OwnedRational, r *memory.Runtime) {
	t.Helper()
	before := r.Stats().LiveBlocks
	e := s.Del(a)
	if e != nil {
		if !errors.Is(e, arena.ErrBounds) || r.Stats().LiveBlocks >= before {
			t.Fatal("nonconsumed teardown failure", e)
		}
		serviceAuditRationalRoots(t, s, &a.raw)
		serviceMust(t, s.Del(a))
	}
}
func contextTLSFinish(t *testing.T, r *memory.Runtime, close func() error) {
	t.Helper()
	for {
		before := r.Stats().LiveBlocks
		e := close()
		if e == nil {
			return
		}
		if !errors.Is(e, arena.ErrBounds) || r.Stats().LiveBlocks >= before {
			t.Fatal("teardown did not consume actual block", e, r.State())
		}
	}
}

func TestUPolynomialContextVectorOnlyStopDoesNotPoisonService(t *testing.T) {
	s, m, p, r, _ := contextTLSOwner(t)
	a := contextBigOdd(t, s, 1)
	serviceMust(t, m.SetRationals(p, []Rational{a.raw}))
	contextTLSNearMinimum(t, r)
	var stopped error
	e := s.WithUPolynomialRationals(m, func(c *UPolynomialRationalContext) error {
		if s.lexical.count != 0 {
			t.Fatal("unexpected rational local")
		}
		stopped = m.DelPolynomial(&p)
		if !errors.Is(stopped, arena.ErrBounds) {
			t.Fatal(stopped)
		}
		c.StopVectors(stopped)
		return stopped
	})
	if !errors.Is(e, stopped) {
		t.Fatal(e)
	}
	contextAudit(t, s, &a, m, p)
	if s.lexical.cause != nil || s.lexical.pending() {
		t.Fatal("vector-only failure poisoned unrelated service", s.lexical.state())
	}
	zero, e := s.Zero()
	serviceMust(t, e)
	out, e := s.Add(&a, &zero)
	serviceMust(t, e)
	if roundPublicationDecode(t, s.manager.z, &a.raw.num).Cmp(roundPublicationDecode(t, s.manager.z, &out.raw.num)) != 0 || out.raw.den != NewInt(1) {
		t.Fatal("independent scalar operation changed value")
	}
	contextAuditAll(t, s, &a, m, []UPolynomial{p}, &zero, &out)
	serviceMust(t, s.Del(&zero))
	contextTLSDelete(t, s, &out, r)
	serviceMust(t, s.CleanupUPolynomialRationals(func() error { return m.DelPolynomial(&p) }))
	contextTLSDelete(t, s, &a, r)
}
