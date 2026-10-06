// Source: src/math/polynomial/upolynomial_factorization.cpp:1030-1050;
// src/util/mpz.h:124-149; src/util/memory_manager.cpp:274-285.
// Licensed under the MIT license. Source authors: Dejan (t-dejanj), 2011-11-15; Leonardo de Moura.
package mpnum

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
	"testing"
)

func TestUPolynomialFactorPreparationActualCopyTransport(t *testing.T) {
	for _, mode := range []string{"service", "close", "source", "mode"} {
		t.Run(mode, func(t *testing.T) { preparationActualCopyTransport(t, mode) })
	}
}
func preparationActualCopyTransport(t *testing.T, mode string) {
	b := arena.NewBudget(1 << 26)
	var s *UPolynomialFactorizationService
	armed := false
	calls := 0
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true, Transport: func(f memory.Failure) error {
		if armed {
			calls++
			prior := s.scope
			before := *prior
			var stop error
			switch mode {
			case "service":
				stop = s.CleanupRetained()
			case "close":
				stop = s.manager.Close()
			case "source":
				stop = s.manager.SourceFinalize()
			case "mode":
				stop = s.manager.SetZ()
			}
			if stop != ErrUPolynomialCleanupPending || s.scope != prior || !reflect.DeepEqual(*prior, before) {
				t.Errorf("actual preparation coefficient SET caller changed before refusal: %v", stop)
			}
		}
		return f
	}})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	cfg.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := NewRationalManager(RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.z, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }})
	serviceMust(t, e)
	primes, e := NewPrimeGenerator(b)
	serviceMust(t, e)
	s, e = NewUPolynomialFactorizationService(m, UPolynomialFactorizationConfig{Primes: primes})
	serviceMust(t, e)
	var held memory.Block
	fs, e := m.NewFactors()
	serviceMust(t, e)
	var value Int
	defer func() {
		armed = false
		b.FailAfter(-1)
		serviceMust(t, rt.SetMaxSize(0))
		contextTLSFinish(t, rt, s.CleanupRetained)
		contextTLSFinish(t, rt, fs.Close)
		contextTLSFinish(t, rt, func() error { return q.z.Del(&value) })
		contextTLSFinish(t, rt, func() error { return rt.Deallocate(&held) })
		contextTLSFinish(t, rt, s.Close)
		serviceMust(t, primes.Close())
		contextTLSFinish(t, rt, m.Close)
		contextTLSFinish(t, rt, q.CloseChecked)
		contextTLSFinish(t, rt, rt.Close)
		if b.Used() != 0 {
			t.Error("actual coefficient SET owner refunds", b.Used())
		}
	}()
	parent, e := s.newSquareFreeScope()
	serviceMust(t, e)
	parent.constructed = 1 // actual f_pp vector declaration; no p/related fields yet
	words := make([]uint32, 80)
	words[79] = 1
	serviceMust(t, q.z.SetLimbs(&value, words))
	var original upolyVectorState
	serviceMust(t, m.set(&original, []Int{value, NewInt(1)}))
	serviceMust(t, fs.PushCopy(m.stateView(&original), 1))
	serviceMust(t, m.reset(&original))
	m.freeBacking(&original)
	state, e := fs.factor(0)
	serviceMust(t, e)
	// Warm only the literal vector reservation, isolating the next actual BIG
	// coefficient SET allocator leaf rather than a preceding vector request.
	serviceMust(t, m.reserve(&parent.fpp, 2))
	pad := int64(99996) - rt.State().LocalSize - 12
	if pad <= 0 {
		t.Fatal("actual source threshold", rt.State())
	}
	held, e = rt.Allocate(uint64(pad))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	armed = true
	body := s.prepareSquareFree(m.stateView(state), fs, 1)
	armed = false
	if !numericPhysicalReservationRefusal(body) || calls != 1 {
		t.Fatal("actual source coefficient SET refusal", body, calls)
	}
	primeTrialPhysicalAuditAll(t, q, s, []*UPolynomialFactors{fs}, []memory.Block{held}, &value)
}

// Actual DEBUG helper before the enclosing f_pp declaration; no parent
// owner or numeral is constructed early to grant acquisition authority.
func TestUPolynomialFactorPreparationDiagnosticBeforeParentTransport(t *testing.T) {
	for _, mode := range []string{"service", "close", "source", "mode"} {
		t.Run(mode, func(t *testing.T) { preparationDiagnosticBeforeParentTransport(t, mode) })
	}
}
func preparationDiagnosticBeforeParentTransport(t *testing.T, mode string) {
	b := arena.NewBudget(1 << 26)
	var s *UPolynomialFactorizationService
	armed := false
	calls := 0
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true, Transport: func(f memory.Failure) error {
		if armed {
			calls++
			before := *s
			var stop error
			switch mode {
			case "service":
				stop = s.Close()
			case "close":
				stop = s.manager.Close()
			case "source":
				stop = s.manager.SourceFinalize()
			case "mode":
				stop = s.manager.SetZ()
			}
			if stop != ErrUPolynomialCleanupPending || !reflect.DeepEqual(*s, before) {
				t.Errorf("actual preparation coefficient SET caller changed before refusal: %v", stop)
			}
		}
		return f
	}})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	cfg.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := NewRationalManager(RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.z, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }})
	serviceMust(t, e)
	primes, e := NewPrimeGenerator(b)
	serviceMust(t, e)
	s, e = NewUPolynomialFactorizationService(m, UPolynomialFactorizationConfig{Primes: primes})
	serviceMust(t, e)
	var held memory.Block
	fs, e := m.NewFactors()
	serviceMust(t, e)
	var value Int
	defer func() {
		armed = false
		b.FailAfter(-1)
		serviceMust(t, rt.SetMaxSize(0))
		contextTLSFinish(t, rt, s.CleanupRetained)
		contextTLSFinish(t, rt, fs.Close)
		contextTLSFinish(t, rt, func() error { return q.z.Del(&value) })
		contextTLSFinish(t, rt, func() error { return rt.Deallocate(&held) })
		if !s.closed {
			contextTLSFinish(t, rt, s.Close)
		}
		serviceMust(t, primes.Close())
		contextTLSFinish(t, rt, m.Close)
		contextTLSFinish(t, rt, q.CloseChecked)
		contextTLSFinish(t, rt, rt.Close)
		if b.Used() != 0 {
			t.Error("actual coefficient SET owner refunds", b.Used())
		}
	}()
	words := make([]uint32, 80)
	words[79] = 1
	serviceMust(t, q.z.SetLimbs(&value, words))
	var original upolyVectorState
	serviceMust(t, m.set(&original, []Int{value, NewInt(1)}))
	serviceMust(t, fs.PushCopy(m.stateView(&original), 1))
	serviceMust(t, m.reset(&original))
	m.freeBacking(&original)
	state, e := fs.factor(0)
	serviceMust(t, e)
	// The DEBUG scoped content precedes f_pp and every enclosing source field.
	if s.scope != nil {
		t.Fatal("diagnostic constructed parent early")
	}
	pad := int64(99996) - rt.State().LocalSize - 12
	if pad <= 0 {
		t.Fatal("actual source threshold", rt.State())
	}
	held, e = rt.Allocate(uint64(pad))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	armed = true
	body := s.primitiveDiagnosticWithPreparation(m.stateView(state))
	armed = false
	if !numericPhysicalReservationRefusal(body) || calls != 1 {
		t.Fatal("actual source coefficient SET refusal", body, calls)
	}
	primeTrialPhysicalAuditAll(t, q, s, []*UPolynomialFactors{fs}, []memory.Block{held}, &value)
}

// Primitive cubic source input, caller content BIG, and every actual
// reservation prefix through conditional c publication and source destruction.
func TestUPolynomialFactorPreparationConditionalEveryActualPrefix(t *testing.T) {
	for _, negative := range []bool{false, true} {
		for _, k := range []uint32{1, 2} {
			completed := false
			for ordinal := int64(0); ordinal < 512; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%v/%d/%d", negative, k, ordinal), func(t *testing.T) {
					q, s := mignotteDirectFixture(t)
					m := s.manager
					fs, e := m.NewFactors()
					serviceMust(t, e)
					var large, constant, want Int
					serviceMust(t, q.z.SetLimbs(&large, []uint32{0, 0, 2}))
					serviceMust(t, q.z.SetLimbs(&want, []uint32{0, 0, 2}))
					serviceMust(t, q.z.Copy(&constant, &q.z.two64))
					serviceMust(t, fs.SetConstant(&constant))
					if negative {
						serviceMust(t, q.z.Neg(&large))
					}
					small := NewInt(1)
					if negative {
						small = NewInt(-1)
					}
					var original upolyVectorState
					serviceMust(t, m.set(&original, []Int{small, NewInt(0), NewInt(0), large}))
					serviceMust(t, fs.PushCopy(m.stateView(&original), 1))
					serviceMust(t, m.reset(&original))
					m.freeBacking(&original)
					originalState, e := fs.factor(0)
					serviceMust(t, e)
					view := m.stateView(originalState)
					parent, e := s.newSquareFreeScope()
					serviceMust(t, e)
					parent.constructed = 1
					q.budget.FailAfter(ordinal)
					body := s.prepareSquareFree(view, fs, k)
					primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{fs}, &large, &constant, &want)
					h := &s.preparation
					r := h.receipt
					source, own := h.sourceCause, h.ownCause
					if body == nil {
						expected := NewInt(1)
						actual, e := m.read(m.stateView(&parent.fpp), 0)
						serviceMust(t, e)
						same, e := q.z.Equal(actual, &expected)
						serviceMust(t, e)
						if !same {
							t.Fatal("source sign-loop independent constant", actual)
						}
						top, e := m.read(m.stateView(&parent.fpp), 3)
						serviceMust(t, e)
						same, e = q.z.Equal(top, &want)
						serviceMust(t, e)
						if !same {
							t.Fatal("source sign-loop independent leading coefficient")
						}
						sign := int32(1)
						if negative && k%2 == 1 {
							sign = -1
						}
						positive, e := q.z.IsNeg(&fs.constant)
						serviceMust(t, e)
						if positive != (sign < 0) {
							t.Fatal("actual caller constant publication", negative, k, fs.constant)
						}
						done = true
					} else if !numericPhysicalReservationRefusal(body) {
						t.Fatal("actual source reservation", body)
					}
					if (!negative || k%2 == 0) && h.numeral != (Int{}) {
						t.Fatal("erased c branch acquired numeral")
					}
					q.budget.FailAfter(-1)
					serviceMust(t, h.own(&r, false))
					serviceMust(t, s.CleanupRetained())
					if h.active || h.bodyCause != body || h.sourceCause != source || h.ownCause != own {
						t.Fatal("exact preparation history/retirement", h.active, h.bodyCause, body)
					}
					primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{fs}, &large, &constant, &want)
					serviceMust(t, fs.Close())
					serviceMust(t, q.z.Del(&large))
					serviceMust(t, q.z.Del(&constant))
					serviceMust(t, q.z.Del(&want))
					primeTrialPhysicalAudit(t, q, s)
				})
				if done {
					completed = true
					t.Log("first complete", negative, k, ordinal)
					break
				}
			}
			if !completed {
				t.Fatal("no actual source completion", negative, k)
			}
		}
	}
}

// Real declared conditional NUM and actual source/own release; signed-host TLS
// guard composition isolates the destructor, not ordinary math OOM reachability.
func TestUPolynomialFactorPreparationActualConditionalDestructorHistory(t *testing.T) {
	for _, source := range []bool{true, false} {
		t.Run(fmt.Sprint(source), func(t *testing.T) {
			b := arena.NewBudget(1 << 26)
			rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
			serviceMust(t, e)
			cfg := serviceConfig(b).Integers
			cfg.Memory = rt
			cfg.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
			zq, e := NewRationalManager(RationalConfig{Integers: cfg})
			serviceMust(t, e)
			m, e := NewUPolynomialManager(UPolynomialConfig{Integer: zq.z, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }})
			serviceMust(t, e)
			primes, e := NewPrimeGenerator(b)
			serviceMust(t, e)
			s, e := NewUPolynomialFactorizationService(m, UPolynomialFactorizationConfig{Primes: primes})
			serviceMust(t, e)
			fs, e := m.NewFactors()
			serviceMust(t, e)
			parent, e := s.newSquareFreeScope()
			serviceMust(t, e)
			parent.constructed = 1
			serviceMust(t, m.set(&parent.fpp, []Int{NewInt(1), NewInt(1)}))
			h, e := s.beginFactorPreparation(false)
			serviceMust(t, e)
			r := h.receipt
			context := m.modular.preparationAllocationContext(h)
			h.constructed = true
			serviceMust(t, m.modular.copyWithSourceContext(&h.numeral, &zq.z.two64, context))
			serviceMust(t, zq.z.ensureOwnedCapacity(&h.numeral, 40000))
			h.executing = false
			original := parent.fpp
			contextTLSNearMinimum(t, rt)
			blocks := rt.Stats().LiveBlocks
			var first error
			if source {
				first = h.finish(&r, nil)
			} else {
				first = h.own(&r, false)
			}
			if !errors.Is(first, arena.ErrBounds) || rt.Stats().LiveBlocks+1 != blocks {
				t.Fatal("actual conditional NUM release", first)
			}
			if parent.fpp != original {
				t.Fatal("younger conditional DT changed older vector")
			}
			primeTrialPhysicalAuditOwners(t, zq, s, []*UPolynomialFactors{fs})
			if source {
				before := *h
				again := s.finishSquareFreeScopeSource()
				if !errors.Is(again, h.sourceCause) || parent.fpp != original || !reflect.DeepEqual(*h, before) {
					t.Fatal("source retry replayed older suffix", again)
				}
			} else if h.ownCause != first || h.ownEnded {
				t.Fatal("first own stop")
			}
			recovery, e := rt.Allocate(400000)
			serviceMust(t, e)
			contextTLSFinish(t, rt, s.CleanupRetained)
			if source && !errors.Is(first, h.sourceCause) || !source && h.ownCause != first {
				t.Fatal("matched owning acknowledgment lost immutable history")
			}
			primeTrialPhysicalAuditAll(t, zq, s, []*UPolynomialFactors{fs}, []memory.Block{recovery})
			contextTLSFinish(t, rt, fs.Close)
			e = rt.Deallocate(&recovery)
			if e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (memory.Block{})) {
				t.Fatal("recovery release", e)
			}
			contextTLSFinish(t, rt, s.Close)
			serviceMust(t, primes.Close())
			contextTLSFinish(t, rt, m.Close)
			contextTLSFinish(t, rt, zq.CloseChecked)
			contextTLSFinish(t, rt, rt.Close)
			if b.Used() != 0 {
				t.Fatal("actual refunds", b.Used())
			}
		})
	}
}

// Invalid Go metadata guards and actual frame reuse are distinct from source
// allocation-prefix credit. Old context must not follow a reused member.
func TestUPolynomialFactorPreparationExactContextHistoryGuards(t *testing.T) {
	q, s := mignotteDirectFixture(t)
	parent, e := s.newSquareFreeScope()
	serviceMust(t, e)
	parent.constructed = 1
	h, e := s.beginFactorPreparation(false)
	serviceMust(t, e)
	r := h.receipt
	c := s.manager.modular.preparationAllocationContext(h)
	if c.roots(q.z) != &h.numeric {
		t.Fatal("actual captured context missing")
	}
	before := *h
	if e = h.own(&r, false); e != ErrUPolynomialCleanupPending || !reflect.DeepEqual(*h, before) || h.ownCause != nil {
		t.Fatal("executing refusal poisoned history", e)
	}
	if e = h.finish(&r, nil); e != ErrUPolynomialCleanupPending || !reflect.DeepEqual(*h, before) {
		t.Fatal("executing source refusal changed tuple", e)
	}
	copyFrame := *h
	bad := r
	bad.frame = &copyFrame
	if e = copyFrame.own(&bad, true); e != ErrUPolynomialOwner || !reflect.DeepEqual(*h, before) {
		t.Fatal("copied frame owned actual source fields", e)
	}
	copiedContext := c
	copiedContext.preparation = &copyFrame
	if copiedContext.roots(q.z) != nil {
		t.Fatal("copied frame acquired leaf")
	}
	s.manager.callbackDepth++
	if c.roots(q.z) != nil {
		t.Fatal("callback acquired passed frame leaf")
	}
	s.manager.callbackDepth--
	other, e := NewRelatedUPolynomialManager(s.manager)
	serviceMust(t, e)
	if other.modular.preparationAllocationContext(h).roots(q.z) != nil {
		t.Fatal("foreign modular receiver acquired leaf")
	}
	serviceMust(t, other.Close())
	h.executing = false
	serviceMust(t, h.finish(&r, nil))
	serviceMust(t, h.own(&r, true))
	newer, e := s.beginFactorPreparation(false)
	serviceMust(t, e)
	nr := newer.receipt
	if nr.invocation == r.invocation || c.roots(q.z) != nil {
		t.Fatal("old context followed reused actual member")
	}
	before = *newer
	if e = newer.own(&r, true); e != ErrUPolynomialOwner || !reflect.DeepEqual(*newer, before) {
		t.Fatal("stale owner changed new fields", e)
	}
	newer.executing = false
	serviceMust(t, newer.finish(&nr, nil))
	serviceMust(t, s.CleanupRetained())
	primeTrialPhysicalAudit(t, q, s)
}

// Valid primitive sparse input reaches DEBUG gcd(0,BIG) direct SET before
// the later coefficient one; old leaf ownership never adopts a replacement.
func TestUPolynomialFactorPreparationDiagnosticOldLeafPreservesForeignReplacement(t *testing.T) {
	for ordinal := int64(0); ordinal < 512; ordinal++ {
		found := false
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			q, s := preparationPhysicalFixture(t)
			m := s.manager
			var prime, foreign, large, value Int
			serviceMust(t, q.z.SetLimbs(&prime, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
			words := make([]uint32, 25000)
			words[24999] = 1
			serviceMust(t, q.z.SetLimbs(&large, words))
			defer func() {
				serviceMust(t, q.z.cleanupRawNumeral(&foreign))
				serviceMust(t, q.z.cleanupRawNumeral(&value))
				serviceMust(t, q.z.Del(&prime))
				serviceMust(t, q.z.Del(&large))
			}()
			fs, e := m.NewFactors()
			serviceMust(t, e)
			defer func() { serviceMust(t, fs.Close()) }()
			var original upolyVectorState
			serviceMust(t, m.set(&original, []Int{NewInt(0), prime, NewInt(0), NewInt(1)}))
			serviceMust(t, fs.PushCopy(m.stateView(&original), 1))
			serviceMust(t, m.reset(&original))
			m.freeBacking(&original)
			state, e := fs.factor(0)
			serviceMust(t, e)
			// Real held NUM consumes the slot actually freed by the source vector.
			// This is a cold owner-primitive profile, not a fabricated reservation.
			serviceMust(t, q.z.Copy(&value, &prime))
			q.budget.FailAfter(ordinal / 16)
			body := s.primitiveDiagnosticWithPreparation(m.stateView(state))
			h := &s.diagnostic
			r := h.receipt

			primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{fs}, &prime, &foreign, &large, &value)
			owned := &h.numeric
			if !polynomialNegNumericPending(owned) && q.z.matchesRawInvocation(&h.gcdRaw) {
				owned = &h.gcdRaw.frame.numeric
			}
			old := owned.cell
			oldRoots := *owned
			if polynomialNegNumericPending(owned) {
				if (old.object != (memory.ObjectRef{}) && q.z.pendingCell != old) || !numericPhysicalReservationRefusal(body) {
					t.Fatal("actual direct pending leaf absent", ordinal, body, old, q.z.pendingCell)
				}
				_ = h.finish(&r, body)
				serviceMust(t, h.own(&r, false))
				if *owned != oldRoots || (old.object != (memory.ObjectRef{}) && q.z.pendingCell != old) {
					t.Fatal("false phase retired unjoined raw leaf")
				}
				q.budget.FailAfter(-1)
				serviceMust(t, q.z.CleanupPendingAllocation())
				// Actual independent nil-context leaf starts after the old token is consumed.
				// Search its real reservation prefixes rather than minting a pending token.
				foreignFound := false
				for next := ordinal % 16; next < ordinal%16+1; next++ {
					q.budget.FailAfter(next)
					failure := q.z.Copy(&foreign, &large)
					primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{fs}, &prime, &foreign, &large, &value)
					laterRaw, x := q.z.allocator.PendingRoots()
					serviceMust(t, x)
					if q.z.pendingCell.object != (memory.ObjectRef{}) || laterRaw != ([3]memory.Block{}) {
						replacement := q.z.pendingCell
						if (replacement.object != (memory.ObjectRef{}) && replacement == old) || !numericPhysicalReservationRefusal(failure) {
							t.Fatal("not a genuine later generation", next, failure, replacement, old)
						}
						q.budget.FailAfter(-1)
						serviceMust(t, h.own(&r, true))
						afterRaw, x := q.z.allocator.PendingRoots()
						serviceMust(t, x)
						if q.z.pendingCell != replacement || afterRaw != laterRaw {
							t.Fatal("old exact abort consumed foreign replacement", replacement, q.z.pendingCell)
						}
						primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{fs}, &prime, &foreign, &large, &value)
						serviceMust(t, h.own(&r, true))
						afterRaw, x = q.z.allocator.PendingRoots()
						serviceMust(t, x)
						if q.z.pendingCell != replacement || afterRaw != laterRaw {
							t.Fatal("stale old abort consumed later pending")
						}
						serviceMust(t, q.z.CleanupPendingAllocation())
						serviceMust(t, q.z.cleanupRawNumeral(&foreign))
						foreignFound = true
						found = true
						t.Log("actual old/foreign pending reservation prefixes", ordinal/16, next)
						break
					}
					q.budget.FailAfter(-1)
					serviceMust(t, q.z.CleanupPendingAllocation())
					serviceMust(t, q.z.cleanupRawNumeral(&foreign))
				}
				if !foreignFound {
					serviceMust(t, h.own(&r, true))
				}
			} else {
				q.budget.FailAfter(-1)
				_ = h.finish(&r, body)
				serviceMust(t, h.own(&r, false))
				serviceMust(t, h.own(&r, true))
			}
		})
		if found {
			return
		}
	}
	t.Fatal("no actual direct old descriptor prefix")
}

func preparationPhysicalFixture(t *testing.T) (*RationalManager, *UPolynomialFactorizationService) {
	t.Helper()
	b := arena.NewBudget(1 << 26)
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Cells.PageSize = 1
	cfg.Memory = rt
	cfg.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := NewRationalManager(RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.z, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }})
	serviceMust(t, e)
	primes, e := NewPrimeGenerator(b)
	serviceMust(t, e)
	s, e := NewUPolynomialFactorizationService(m, UPolynomialFactorizationConfig{Primes: primes})
	serviceMust(t, e)
	t.Cleanup(func() {
		b.FailAfter(-1)
		serviceMust(t, s.Close())
		serviceMust(t, primes.Close())
		serviceMust(t, m.Close())
		serviceMust(t, q.CloseChecked())
		serviceMust(t, rt.Close())
		if b.Used() != 0 {
			t.Error("actual cold fixture refunds", b.Used())
		}
	})
	return q, s
}

// Actual DEBUG source NUM precedes f_pp; every cold reservation prefix
// names the direct/GCD child roots before both fault lift and owning rescue.
func TestUPolynomialFactorPreparationDiagnosticEveryActualPrefix(t *testing.T) {
	completed := false
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		done := false
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			q, s := preparationPhysicalFixture(t)
			m := s.manager
			var prime, held Int
			serviceMust(t, q.z.SetLimbs(&prime, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
			defer func() { serviceMust(t, q.z.cleanupRawNumeral(&held)); serviceMust(t, q.z.Del(&prime)) }()
			fs, e := m.NewFactors()
			serviceMust(t, e)
			defer func() { serviceMust(t, fs.Close()) }()
			var original upolyVectorState
			serviceMust(t, m.set(&original, []Int{NewInt(0), prime, NewInt(0), NewInt(1)}))
			serviceMust(t, fs.PushCopy(m.stateView(&original), 1))
			serviceMust(t, m.reset(&original))
			m.freeBacking(&original)
			st, e := fs.factor(0)
			serviceMust(t, e)
			serviceMust(t, q.z.Copy(&held, &prime))
			q.budget.FailAfter(ordinal)
			body := s.primitiveDiagnosticWithPreparation(m.stateView(st))
			h := &s.diagnostic
			r := h.receipt
			firstBody, firstSource := h.bodyCause, h.sourceCause
			if s.scope != nil {
				t.Fatal("DEBUG created enclosing f_pp")
			}
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal("actual DEBUG refusal", body)
			}
			primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{fs}, &prime, &held)
			// Source completed NUM and exact younger children first, no enclosing locals exist.
			stop := h.own(&r, false)
			if stop != nil && !numericPhysicalReservationRefusal(stop) {
				t.Fatal("false owning phase", stop)
			}
			primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{fs}, &prime, &held)
			if stop == nil {
				stop = h.own(&r, true)
			}
			if stop != nil && !numericPhysicalReservationRefusal(stop) {
				t.Fatal("true owning phase", stop)
			}
			primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{fs}, &prime, &held)
			q.budget.FailAfter(-1)
			serviceMust(t, h.own(&r, false))
			serviceMust(t, h.own(&r, true))
			if h.active || h.bodyCause != firstBody || h.sourceCause != firstSource {
				t.Fatal("exact immutable DEBUG history")
			}
			primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{fs}, &prime, &held)
			done = body == nil
		})
		if done {
			completed = true
			t.Log("first complete DEBUG source", ordinal)
			break
		}
	}
	if !completed {
		t.Fatal("no actual DEBUG completion")
	}
}
