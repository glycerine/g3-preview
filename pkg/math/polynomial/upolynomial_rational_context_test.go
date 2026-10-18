// Source: src/math/polynomial/upolynomial.cpp, src/util/scoped_numeral_vector.h.
package polynomial

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func upolyContextOwner(t *testing.T) (*unumeral.RationalService, *UPolynomialManager, UPolynomial) {
	t.Helper()
	s, _ := serviceNew(t)
	m, e := NewUPolynomialManagerFromRationalService(s, UPolynomialConfig{Checkpoint: func() bool { panic("constant set must not checkpoint") }, Cooperation: func() error { panic("constant set must not cooperate") }})
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
	one, zero := unumeral.NewInt(1), unumeral.NewInt(0)
	serviceMust(t, s.Manager.SetParts(&a.Raw, &one, &zero))
	want := errors.New("source integer coefficient exception")
	called := false
	e = numericRationalService_WithUPolynomialRationals(s, m, func(c *UPolynomialRationalContext) error {
		return c.SetConstant(p, &a, func() error { called = true; return want })
	})
	if !called || !errors.Is(e, want) {
		t.Fatalf("Copy→IsInt→source exception: called=%v err=%v", called, e)
	}
	serviceAuditRationalRoots(t, s, &a.Raw)
	serviceMust(t, s.Del(&a))
}

func TestUPolynomialContextBorrowIdentity(t *testing.T) {
	s, m, p := upolyContextOwner(t)
	a := serviceValue(t, s, "3")
	var copied UPolynomialRationalContext
	serviceMust(t, numericRationalService_WithUPolynomialRationals(s, m, func(c *UPolynomialRationalContext) error { copied = *c; return nil }))
	if _, _, e := copied.Exponent(&a); !errors.Is(e, unumeral.ErrRationalServiceOwner) {
		t.Fatalf("copied borrowed context escaped lock: %v", e)
	}
	if e := copied.SetIdentity(p); !errors.Is(e, unumeral.ErrRationalServiceOwner) {
		t.Fatal(e)
	}
	if e := copied.SetConstant(p, &a, func() error { t.Fatal("invalid borrow reached callback"); return nil }); !errors.Is(e, unumeral.ErrRationalServiceOwner) {
		t.Fatal(e)
	}
	copied.StopVectors(arena.ErrBounds)
	if copied.Pending() || s.Lexical.Pending() || s.Lexical.Cause != nil {
		t.Fatal("escaped context poisoned actual owner")
	}

	serviceAuditRationalRoots(t, s, &a.Raw)
	serviceMust(t, s.Del(&a))
}

func TestUPolynomialContextVectorStopAPI(t *testing.T) {
	s, m, _ := upolyContextOwner(t)
	serviceMust(t, numericRationalService_WithUPolynomialRationals(s, m, func(c *UPolynomialRationalContext) error {
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
func contextTLSNearMinimum(t *testing.T, r *umemory.Runtime) {
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

func contextTLSOwner(t *testing.T) (*unumeral.RationalService, *UPolynomialManager, UPolynomial, *umemory.Runtime, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 25)
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(b)
	cfg.Integers.Memory = r
	cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	s, e := unumeral.NewRationalService(cfg)
	serviceMust(t, e)
	m, e := NewUPolynomialManagerFromRationalService(s, UPolynomialConfig{Checkpoint: func() bool { panic("constant route checkpoint") }, Cooperation: func() error { panic("constant route cooperation") }})
	serviceMust(t, e)
	p, e := m.NewPolynomial()
	serviceMust(t, e)
	t.Cleanup(func() {
		b.FailAfter(-1)
		serviceMust(t, numericRationalService_CleanupUPolynomialRationals(s, m, func() error { return nil }))
		contextTLSFinish(t, r, func() error { return m.Close() })
		contextTLSFinish(t, r, func() error { return s.Close() })
		contextTLSFinish(t, r, func() error { return r.Close() })
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	})
	return s, m, p, r, b
}

func contextBigOdd(t *testing.T, s *unumeral.RationalService, den int32) unumeral.OwnedRational {
	t.Helper()
	a, e := s.NewValue()
	serviceMust(t, e)
	limbs := make([]uint32, 32768)
	limbs[0] = 17
	limbs[len(limbs)-1] = 1
	serviceMust(t, s.Manager.Z.SetLimbs(&a.Raw.Num, limbs))
	a.Raw.Den = unumeral.NewInt(den)
	return a
}

func contextAudit(t *testing.T, s *unumeral.RationalService, a *unumeral.OwnedRational, m *UPolynomialManager, args ...UPolynomial) {
	t.Helper()
	contextAuditAll(t, s, a, m, args)
}
func contextAuditAll(t *testing.T, s *unumeral.RationalService, a *unumeral.OwnedRational, m *UPolynomialManager, args []UPolynomial, others ...*unumeral.OwnedRational) {
	t.Helper()
	roots := []*unumeral.Int{&a.Raw.Num, &a.Raw.Den}
	for _, v := range others {
		roots = append(roots, &v.Raw.Num, &v.Raw.Den)
	}
	for _, q := range []*unumeral.Rational{&s.ZeroField, &s.OneField, &s.MinusOneField, &s.InfZeroField.First, &s.InfZeroField.Second, &s.InfOneField.First, &s.InfOneField.Second, &s.InfMinusOneField.First, &s.InfMinusOneField.Second, &s.InfIntZeroField.First, &s.InfIntOneField.First, &s.InfIntMinusOneField.First} {
		roots = append(roots, &(*q.BorrowNumerator()), &(*q.BorrowDenominator()))
	}
	for i := range s.Cache {
		roots = append(roots, &s.Cache[i].Num, &s.Cache[i].Den)
	}
	for i := uint8(0); i < s.Lexical.Count; i++ {
		if s.Lexical.Done&(1<<i) == 0 {
			roots = append(roots, &s.Lexical.Roots[i].Num, &s.Lexical.Roots[i].Den)
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
	qopsAuditLiveOwnership(t, s.Manager, roots)
}

func TestUPolynomialContextExceptionBeforeCompletedDestructor(t *testing.T) {
	s, m, p, r, _ := contextTLSOwner(t)
	a := contextBigOdd(t, s, 2)
	original := errors.New("source integer coefficient exception")
	called := false
	e := numericRationalService_WithUPolynomialRationals(s, m, func(c *UPolynomialRationalContext) error {
		return c.SetConstant(p, &a, func() error { called = true; contextTLSNearMinimum(t, r); return original })
	})
	if !called || !errors.Is(e, original) || !errors.Is(e, arena.ErrBounds) {
		t.Fatalf("source exception missing: %v", e)
	}
	state := s.Lexical.State()
	if state.Roots != 1 || state.Completed != 1 || state.Aborts != 0 || !errors.Is(state.Cause, arena.ErrBounds) || !errors.Is(state.OriginalCause, original) {
		t.Fatal(state)
	}
	// The source raw free completed; its stopped descriptor is still the
	// completed local's actual retained root, with no independent registry root.
	ref := s.Lexical.Roots[0].Num.Cell
	cell := s.Manager.Z.Cell(ref)
	if !cell.Released {
		t.Fatal("physical free did not precede TLS guard", cell)
	}
	contextAudit(t, s, &a, m, p)
	serviceMust(t, numericRationalService_CleanupUPolynomialRationals(s, m, func() error { return nil }))
	contextTLSDelete(t, s, &a, r)
	serviceMust(t, m.DelPolynomial(&p))
}

func TestUPolynomialContextPartialCopyWaitsForStoppedVector(t *testing.T) {
	s, m, p, r, b := contextTLSOwner(t)
	a := contextBigOdd(t, s, 1)
	serviceMust(t, m.SetRationals(p, []unumeral.Rational{a.Raw}))
	contextTLSNearMinimum(t, r)
	b.FailAfter(0)
	var original, stopped error
	e := numericRationalService_WithUPolynomialRationals(s, m, func(c *UPolynomialRationalContext) error {
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
	state := s.Lexical.State()
	if state.Aborts != 1 || state.Completed != 0 || !errors.Is(state.Cause, stopped) || !errors.Is(state.OriginalCause, original) {
		t.Fatal(state)
	}
	if s.Lexical.Roots[0].Num.Cell != s.Manager.Z.FailedNullRef {
		t.Fatal("partial copy adapter drained before older vector")
	}
	contextAudit(t, s, &a, m, p)
	b.FailAfter(-1)
	serviceMust(t, s.Manager.Z.CleanupPendingAllocation())
	serviceMust(t, numericRationalService_CleanupUPolynomialRationals(s, m, func() error { return m.DelPolynomial(&p) }))
	if s.Lexical.Pending() {
		t.Fatal("cleanup did not finish")
	}
	contextTLSDelete(t, s, &a, r)

}

// Each ErrBounds is the actual postphysical TLS synchronization refusal. Retry
// acknowledges that exact release; no source counter or pointer is repaired.
func contextTLSDelete(t *testing.T, s *unumeral.RationalService, a *unumeral.OwnedRational, r *umemory.Runtime) {
	t.Helper()
	before := r.Stats().LiveBlocks
	e := s.Del(a)
	if e != nil {
		if !errors.Is(e, arena.ErrBounds) || r.Stats().LiveBlocks >= before {
			t.Fatal("nonconsumed teardown failure", e)
		}
		serviceAuditRationalRoots(t, s, &a.Raw)
		serviceMust(t, s.Del(a))
	}
}
func contextTLSFinish(t *testing.T, r *umemory.Runtime, close func() error) {
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
	serviceMust(t, m.SetRationals(p, []unumeral.Rational{a.Raw}))
	contextTLSNearMinimum(t, r)
	var stopped error
	e := numericRationalService_WithUPolynomialRationals(s, m, func(c *UPolynomialRationalContext) error {
		if s.Lexical.Count != 0 {
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
	if s.Lexical.Cause != nil || s.Lexical.Pending() {
		t.Fatal("vector-only failure poisoned unrelated service", s.Lexical.State())
	}
	zero, e := s.Zero()
	serviceMust(t, e)
	out, e := s.Add(&a, &zero)
	serviceMust(t, e)
	if roundPublicationDecode(t, s.Manager.Z, &a.Raw.Num).Cmp(roundPublicationDecode(t, s.Manager.Z, &out.Raw.Num)) != 0 || out.Raw.Den != unumeral.NewInt(1) {
		t.Fatal("independent scalar operation changed value")
	}
	contextAuditAll(t, s, &a, m, []UPolynomial{p}, &zero, &out)
	serviceMust(t, s.Del(&zero))
	contextTLSDelete(t, s, &out, r)
	serviceMust(t, numericRationalService_CleanupUPolynomialRationals(s, m, func() error { return m.DelPolynomial(&p) }))
	contextTLSDelete(t, s, &a, r)
}
