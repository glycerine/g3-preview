// Source: src/math/polynomial/upolynomial_factorization.cpp:1213-1240;
// src/math/polynomial/upolynomial_factorization_int.h:345-413;
// src/util/memory_manager.cpp:274-285.
// Licensed under the MIT license. Source authors: Dejan (t-dejanj), 2011-11-15; Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
	"testing"
)

// This is the actual supported ErrorTransport callback profile at a real
// lower Copy allocation. It does not claim a C++ allocator callback.
func TestUPolynomialRecombinationTrialTransportProtectsActualCaller(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	var s *UPolynomialFactorizationService
	armed := false
	calls := 0
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true, Transport: func(f memory.Failure) error {
		if armed {
			calls++
			prior := s.scope
			before := *prior
			stop := s.CleanupRetained()
			if stop != ErrUPolynomialCleanupPending || s.scope != prior || !reflect.DeepEqual(*prior, before) {
				t.Errorf("actual executing trial owner changed before refusal: %v", stop)
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
	var leading Int
	var held memory.Block
	fs, e := m.NewFactors()
	serviceMust(t, e)
	defer func() {
		armed = false
		b.FailAfter(-1)
		serviceMust(t, rt.SetMaxSize(0))
		contextTLSFinish(t, rt, s.CleanupRetained)
		contextTLSFinish(t, rt, fs.Close)
		contextTLSFinish(t, rt, func() error { return q.z.Del(&leading) })
		contextTLSFinish(t, rt, func() error { return rt.Deallocate(&held) })
		contextTLSFinish(t, rt, s.Close)
		serviceMust(t, primes.Close())
		contextTLSFinish(t, rt, m.Close)
		contextTLSFinish(t, rt, q.CloseChecked)
		contextTLSFinish(t, rt, rt.Close)
		if b.Used() != 0 {
			t.Error("actual owner refunds", b.Used())
		}
	}()
	parent, e := s.newSquareFreeScope()
	serviceMust(t, e)
	parent.constructed = 2
	serviceMust(t, q.z.Copy(&parent.p, &q.z.two64))
	var source upolyVectorState
	serviceMust(t, m.set(&source, []Int{NewInt(1), NewInt(1)}))
	serviceMust(t, fs.PushCopy(m.stateView(&source), 1))
	serviceMust(t, fs.PushCopy(m.stateView(&source), 1))
	serviceMust(t, m.reset(&source))
	m.freeBacking(&source)
	serviceMust(t, parent.it.init(fs, &parent.ds))
	defer parent.it.close()
	parent.it.size = 1
	parent.it.current[0] = 0
	parent.it.current[1] = 2
	words := make([]uint32, 80)
	words[79] = 1
	serviceMust(t, q.z.SetLimbs(&leading, words))
	// Put the next actual raw request across the existing TLS synchronization
	// threshold, with a real public allocation and the normal malloc limit.
	pad := int64(99996) - rt.State().LocalSize - 12
	if pad <= 0 {
		t.Fatal("actual threshold", rt.State())
	}
	held, e = rt.Allocate(uint64(pad))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	armed = true
	used, body := s.selectFactorTrial(&parent.it, &leading, &parent.fpp, &parent.trial, true)
	armed = false
	if used || !numericPhysicalReservationRefusal(body) || calls != 1 {
		t.Fatal("actual tail Copy refusal", used, body, calls)
	}
	primeTrialPhysicalAuditAll(t, q, s, []*UPolynomialFactors{fs}, []memory.Block{held}, &leading)
	helper := &parent.trialHelper
	if !polynomialNegNumericPending(&helper.numeric) {
		t.Fatal("actual Copy leaf missing captured authority")
	}
	if e := s.finishSquareFreeScopeSource(); e != ErrUPolynomialCleanupPending || parent.constructed != 0 || parent.p.cell != 0 || helper.bodyCause != body || helper.sourceCause != nil {
		t.Fatal("completed helper DT did not precede older enclosing source suffix", e, parent.constructed, parent.p)
	}
	primeTrialPhysicalAuditAll(t, q, s, []*UPolynomialFactors{fs}, []memory.Block{held}, &leading)

}

func TestUPolynomialRecombinationTrialEveryActualBranchPrefix(t *testing.T) {
	for _, left := range []bool{true, false} {
		for _, accept := range []bool{true, false} {
			completed := false
			for ordinal := int64(0); ordinal < 1024; ordinal++ {
				q, s := mignotteDirectFixture(t)
				m := s.manager
				parent, e := s.newSquareFreeScope()
				serviceMust(t, e)
				fs, e := m.NewFactors()
				serviceMust(t, e)
				var source upolyVectorState
				trial, fpp := &parent.trial, &parent.fpp
				for _, v := range [][]Int{{NewInt(1), NewInt(1)}, {NewInt(2), NewInt(1)}} {
					serviceMust(t, m.set(&source, v))
					serviceMust(t, fs.PushCopy(m.stateView(&source), 1))
				}
				serviceMust(t, m.reset(&source))
				m.freeBacking(&source)
				serviceMust(t, parent.it.init(fs, &parent.ds))
				parent.it.size = 1
				parent.it.current[0] = 0
				parent.it.current[1] = 2
				var leading, constant Int
				serviceMust(t, q.z.Copy(&leading, &q.z.two64))
				serviceMust(t, q.z.Mul(&leading, func() *Int { v := NewInt(2); return &v }(), &constant))
				if !accept {
					serviceMust(t, q.z.SetInt64(&constant, 7))
				}
				serviceMust(t, m.set(fpp, []Int{constant}))
				q.budget.FailAfter(ordinal)
				used, body := s.selectFactorTrial(&parent.it, &leading, fpp, trial, left)
				primeTrialPhysicalAuditAll(t, q, s, []*UPolynomialFactors{fs}, nil, &leading, &constant)
				// Driver-owned vector fields are named separately from the concrete parent.
				// Their actual storage is charged below by the enclosing fixture graph.
				helper := &parent.trialHelper
				savedBody, savedSource, savedOwn := helper.bodyCause, helper.sourceCause, helper.ownCause
				if body == nil {
					if used != accept {
						t.Fatal("independent constant divisibility", left, accept, used)
					}
					if accept {
						want := NewInt(1)
						if !left {
							want = NewInt(2)
						}
						v := m.values(trial)
						if trial.size != 2 || v[0] != want || v[1] != NewInt(1) {
							t.Fatal("independent selected complement", left, trial.size)
						}
					} else if trial.size != 0 {
						t.Fatal("erased branch materialized trial")
					}
				}
				q.budget.FailAfter(-1)
				serviceMust(t, helper.own(&helper.receipt, false))
				serviceMust(t, m.reset(trial))
				m.freeBacking(trial)
				serviceMust(t, m.reset(fpp))
				m.freeBacking(fpp)
				parent.it.close()
				serviceMust(t, s.CleanupRetained())
				if helper.bodyCause != savedBody || helper.sourceCause != savedSource || helper.ownCause != savedOwn {
					t.Fatal("exact acknowledgment altered source history")
				}
				serviceMust(t, q.z.Del(&leading))
				serviceMust(t, q.z.Del(&constant))
				serviceMust(t, fs.Close())
				primeTrialPhysicalAudit(t, q, s)
				if body == nil {
					t.Log("actual branch completion", left, accept, ordinal)
					completed = true
					break
				}
			}
			if !completed {
				t.Fatal("actual branch never completes", left, accept)
			}
		}
	}
}

// Reached signed-accounting source/own release stops are guarded helper
// profiles, not claimed normal factorization OOM recipes.
func TestUPolynomialRecombinationTrialActualReleaseHistories(t *testing.T) {
	for _, source := range []bool{true, false} {
		b := arena.NewBudget(1 << 26)
		rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
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
		s, e := NewUPolynomialFactorizationService(m, UPolynomialFactorizationConfig{Primes: primes})
		serviceMust(t, e)
		fs, e := m.NewFactors()
		serviceMust(t, e)
		_, e = s.newSquareFreeScope()
		serviceMust(t, e)
		h, e := s.beginRecombinationTrial(fs)
		serviceMust(t, e)
		r := h.receipt
		context := m.modular.trialAllocationContext(h)
		serviceMust(t, m.modular.copyWithSourceContext(&h.tmp, &q.z.two64, context))
		serviceMust(t, q.z.ensureOwnedCapacity(&h.tmp, 40000))
		h.executing = false
		contextTLSNearMinimum(t, rt)
		blocks := rt.Stats().LiveBlocks
		var first error
		if source {
			first = h.finish(&r, nil)
		} else {
			first = h.own(&r, false)
		}
		if !errors.Is(first, arena.ErrBounds) || rt.Stats().LiveBlocks+1 != blocks {
			t.Fatal("actual consumed release", source, first)
		}
		primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{fs})
		if source {
			if !errors.Is(first, h.sourceCause) || h.sourceEnded {
				t.Fatal("first source stop history")
			}
			before := *h
			if !errors.Is(h.finish(&r, nil), h.sourceCause) || !reflect.DeepEqual(*h, before) {
				t.Fatal("source retry consumed older state")
			}
		} else if h.ownCause != first || h.ownEnded {
			t.Fatal("first owning stop history")
		}
		recovery, e := rt.Allocate(400000)
		serviceMust(t, e)
		contextTLSFinish(t, rt, func() error { return h.own(&r, false) })
		contextTLSFinish(t, rt, func() error { return h.own(&r, true) })
		if source && !errors.Is(first, h.sourceCause) || !source && h.ownCause != first {
			t.Fatal("acknowledgment lost history")
		}
		primeTrialPhysicalAuditAll(t, q, s, []*UPolynomialFactors{fs}, []memory.Block{recovery})
		contextTLSFinish(t, rt, s.CleanupRetained)
		contextTLSFinish(t, rt, fs.Close)
		e = rt.Deallocate(&recovery)
		if e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (memory.Block{})) {
			t.Fatal("actual consumed recovery block", e)
		}
		contextTLSFinish(t, rt, s.Close)
		serviceMust(t, primes.Close())
		contextTLSFinish(t, rt, m.Close)
		contextTLSFinish(t, rt, q.CloseChecked)
		contextTLSFinish(t, rt, rt.Close)
		if b.Used() != 0 {
			t.Fatal("actual refunds", b.Used())
		}
	}
}

func TestUPolynomialRecombinationTrialExactReceiptAndCapturedContext(t *testing.T) {
	q, s := mignotteDirectFixture(t)
	m := s.manager
	parent, e := s.newSquareFreeScope()
	serviceMust(t, e)
	fs, e := m.NewFactors()
	serviceMust(t, e)
	h, e := s.beginRecombinationTrial(fs)
	serviceMust(t, e)
	r := h.receipt
	c := m.modular.trialAllocationContext(h)
	if c.trialFrame(q.z) != h || c.roots(q.z) != &h.numeric {
		t.Fatal("actual embedded frame refused")
	}
	copyFrame := *h
	bad := r
	bad.frame = &copyFrame
	before := *h
	if h.finish(&bad, nil) != ErrUPolynomialOwner || h.own(&bad, true) != ErrUPolynomialOwner || !reflect.DeepEqual(*h, before) {
		t.Fatal("copied receipt frame modified actual source/own tuple")
	}
	copyOwner := *parent
	bad = r
	bad.owner = &copyOwner
	if h.finish(&bad, nil) != ErrUPolynomialOwner || h.own(&bad, true) != ErrUPolynomialOwner || !reflect.DeepEqual(*h, before) {
		t.Fatal("copied parent acquired cleanup authority")
	}
	foreign, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.z, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }})
	serviceMust(t, e)
	if foreign.modular.trialAllocationContext(h) != (integerAllocationInvocationContext{}) {
		t.Fatal("foreign same-Integer modular receiver acquired parent")
	}
	m.callbackDepth++
	if c.roots(q.z) != nil {
		t.Fatal("callback acquired old concrete parent numeric roots")
	}
	m.callbackDepth--
	h.executing = false
	serviceMust(t, h.finish(&r, nil))
	serviceMust(t, h.own(&r, true))
	newer, e := s.beginRecombinationTrial(fs)
	serviceMust(t, e)
	if c.trialFrame(q.z) != nil || c.roots(q.z) != nil {
		t.Fatal("stale context retargeted reused actual member")
	}
	newBefore := *newer
	if newer.finish(&r, nil) != ErrUPolynomialOwner || newer.own(&r, true) != ErrUPolynomialOwner || !reflect.DeepEqual(*newer, newBefore) {
		t.Fatal("old receipt retired new generation")
	}
	nr := newer.receipt
	newer.executing = false
	serviceMust(t, newer.finish(&nr, nil))
	serviceMust(t, newer.own(&nr, true))
	serviceMust(t, s.CleanupRetained())
	serviceMust(t, fs.Close())
	serviceMust(t, foreign.Close())
	primeTrialPhysicalAudit(t, q, s)
}

func TestUPolynomialRecombinationTrialFiniteBIGEveryActualPrefix(t *testing.T) {
	for _, left := range []bool{true, false} {
		complete := false
		for ordinal := int64(0); ordinal < 1024; ordinal++ {
			q, s := mignotteDirectFixture(t)
			m := s.manager
			parent, e := s.newSquareFreeScope()
			serviceMust(t, e)
			zpe, e := NewRelatedUPolynomialManager(m)
			serviceMust(t, e)
			var p, b, b1, tail, middle Int
			serviceMust(t, q.z.SetLimbs(&p, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
			serviceMust(t, zpe.SetZP(&p))
			serviceMust(t, q.z.Copy(&b, &q.z.two64))
			one := NewInt(1)
			two := NewInt(2)
			serviceMust(t, q.z.Add(&b, &one, &b1))
			serviceMust(t, q.z.Add(&b, &two, &tail))
			serviceMust(t, q.z.Mul(&b, &two, &middle))
			serviceMust(t, q.z.Add(&middle, &one, &middle))
			fs, e := zpe.NewFactors()
			serviceMust(t, e)
			var source upolyVectorState
			for _, a := range []*Int{&b, &b1, &b, &b1} {
				serviceMust(t, zpe.set(&source, []Int{*a, one}))
				serviceMust(t, fs.PushCopy(zpe.stateView(&source), 1))
			}
			serviceMust(t, zpe.reset(&source))
			zpe.freeBacking(&source)
			serviceMust(t, parent.it.init(fs, &parent.ds))
			parent.it.size = 2
			parent.it.current[0] = 0
			parent.it.current[1] = 1
			parent.it.current[2] = 4
			serviceMust(t, m.set(&parent.fpp, []Int{tail}))
			q.budget.FailAfter(ordinal)
			used, body := s.selectFactorTrial(&parent.it, &one, &parent.fpp, &parent.trial, left)
			primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{fs}, &p, &b, &b1, &tail, &middle)
			if body == nil {
				if !used || parent.trial.size != 3 {
					t.Fatal("actual BIG trial route", left, used, parent.trial.size)
				}
				v := zpe.values(&parent.trial)
				for i, a := range []*Int{&tail, &middle, &one} {
					same, e := q.z.Equal(&v[i], a)
					serviceMust(t, e)
					if !same {
						t.Fatal("independent GF127 convolution", left, i)
					}
				}
			}
			q.budget.FailAfter(-1)
			h := &parent.trialHelper
			r := h.receipt
			serviceMust(t, h.own(&r, false))
			serviceMust(t, zpe.reset(&parent.trial))
			zpe.freeBacking(&parent.trial)
			serviceMust(t, m.reset(&parent.fpp))
			m.freeBacking(&parent.fpp)
			parent.it.close()
			serviceMust(t, s.CleanupRetained())
			serviceMust(t, fs.Close())
			serviceMust(t, zpe.Close())
			for _, a := range []*Int{&p, &b, &b1, &tail, &middle} {
				serviceMust(t, q.z.Del(a))
			}
			primeTrialPhysicalAudit(t, q, s)
			if body == nil {
				t.Log("actual GF127 branch completion", left, ordinal)
				complete = true
				break
			}
		}
		if !complete {
			t.Fatal("actual finite BIG trial never completes", left)
		}
	}
}

func TestUPolynomialRecombinationTrialRelatedOwnerTransport(t *testing.T) {
	for _, mode := range []string{"close", "source", "set_z"} {
		t.Run(mode, func(t *testing.T) {
			b := arena.NewBudget(1 << 26)
			var s *UPolynomialFactorizationService
			var target *UPolynomialManager
			armed := false
			calls := 0
			rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true, Transport: func(f memory.Failure) error {
				if armed {
					calls++
					prior := s.scope
					before := *prior
					oldMode := target.modular.z
					var stop error
					switch mode {
					case "close":
						stop = target.Close()
					case "source":
						stop = target.SourceFinalize()
					case "set_z":
						stop = target.SetZ()
					}
					if stop != ErrUPolynomialCleanupPending || s.scope != prior || !reflect.DeepEqual(*prior, before) || target.modular.z != oldMode {
						t.Errorf("actual executing trial owner changed before refusal: %v", stop)
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
			var leading Int
			var held memory.Block
			target, e = NewRelatedUPolynomialManager(m)
			serviceMust(t, e)
			var prime Int
			serviceMust(t, q.z.SetLimbs(&prime, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
			serviceMust(t, target.SetZP(&prime))
			fs, e := target.NewFactors()
			serviceMust(t, e)
			defer func() {
				armed = false
				b.FailAfter(-1)
				serviceMust(t, rt.SetMaxSize(0))
				contextTLSFinish(t, rt, s.CleanupRetained)
				contextTLSFinish(t, rt, fs.Close)
				contextTLSFinish(t, rt, target.Close)
				contextTLSFinish(t, rt, func() error { return q.z.Del(&prime) })
				contextTLSFinish(t, rt, func() error { return q.z.Del(&leading) })
				contextTLSFinish(t, rt, func() error { return rt.Deallocate(&held) })
				contextTLSFinish(t, rt, s.Close)
				serviceMust(t, primes.Close())
				contextTLSFinish(t, rt, m.Close)
				contextTLSFinish(t, rt, q.CloseChecked)
				contextTLSFinish(t, rt, rt.Close)
				if b.Used() != 0 {
					t.Error("actual owner refunds", b.Used())
				}
			}()
			parent, e := s.newSquareFreeScope()
			serviceMust(t, e)
			parent.constructed = 2
			serviceMust(t, q.z.Copy(&parent.p, &q.z.two64))
			var source upolyVectorState
			serviceMust(t, target.set(&source, []Int{NewInt(1), NewInt(1)}))
			serviceMust(t, fs.PushCopy(target.stateView(&source), 1))
			serviceMust(t, fs.PushCopy(target.stateView(&source), 1))
			serviceMust(t, target.reset(&source))
			target.freeBacking(&source)
			serviceMust(t, parent.it.init(fs, &parent.ds))
			defer parent.it.close()
			parent.it.size = 1
			parent.it.current[0] = 0
			parent.it.current[1] = 2
			words := make([]uint32, 80)
			words[79] = 1
			serviceMust(t, q.z.SetLimbs(&leading, words))
			// Put the next actual raw request across the existing TLS synchronization
			// threshold, with a real public allocation and the normal malloc limit.
			pad := int64(99996) - rt.State().LocalSize - 12
			if pad <= 0 {
				t.Fatal("actual threshold", rt.State())
			}
			held, e = rt.Allocate(uint64(pad))
			serviceMust(t, e)
			serviceMust(t, rt.SetMaxSize(1))
			armed = true
			used, body := s.selectFactorTrial(&parent.it, &leading, &parent.fpp, &parent.trial, true)
			armed = false
			if used || !numericPhysicalReservationRefusal(body) || calls != 1 {
				t.Fatal("actual tail Copy refusal", used, body, calls)
			}
			primeTrialPhysicalAuditAll(t, q, s, []*UPolynomialFactors{fs}, []memory.Block{held}, &leading, &prime)
			helper := &parent.trialHelper
			if !polynomialNegNumericPending(&helper.numeric) {
				t.Fatal("actual Copy leaf missing captured authority")
			}
			if e := s.finishSquareFreeScopeSource(); e != ErrUPolynomialCleanupPending || parent.constructed != 0 || parent.p.cell != 0 || helper.bodyCause != body || helper.sourceCause != nil {
				t.Fatal("completed helper DT did not precede older enclosing source suffix", e, parent.constructed, parent.p)
			}
			primeTrialPhysicalAuditAll(t, q, s, []*UPolynomialFactors{fs}, []memory.Block{held}, &leading, &prime)

		})
	}
}

// This source-DT composition uses the actual guarded compiled NUM frame and
// actual Copy/publication plus capacity acquisition to reach its real destructor.
// It is an owner-primitive guard profile, not a substituted Rem/math driver.
func TestUPolynomialRecombinationTrialActualCompiledChildSourceStopProtectsTmp(t *testing.T) {
	for _, source := range []bool{true} {
		b := arena.NewBudget(1 << 26)
		rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
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
		s, e := NewUPolynomialFactorizationService(m, UPolynomialFactorizationConfig{Primes: primes})
		serviceMust(t, e)
		fs, e := m.NewFactors()
		serviceMust(t, e)
		_, e = s.newSquareFreeScope()
		serviceMust(t, e)
		h, e := s.beginRecombinationTrial(fs)
		serviceMust(t, e)
		r := h.receipt
		context := m.modular.trialAllocationContext(h)
		serviceMust(t, m.modular.copyWithSourceContext(&h.tmp, &q.z.two64, context))
		child, e := q.z.beginDividesNumeralInvocation(context)
		serviceMust(t, e)
		savedChild := *child
		childContext := integerAllocationInvocationContext{boolDivides: child, boolDividesInvocation: child.invocation}
		serviceMust(t, q.z.copyFromOwnerWithAllocationContext(&child.frame.numeral, q.z, &q.z.two64, nil, childContext))
		serviceMust(t, q.z.ensureOwnedCapacity(&child.frame.numeral, 40000))
		child.frame.executing = false
		beforeTmp := h.tmp
		h.executing = false
		contextTLSNearMinimum(t, rt)
		blocks := rt.Stats().LiveBlocks
		var first error
		if source {
			low := q.z.finishDividesNumeralSource(child.frame)
			if !errors.Is(low, arena.ErrBounds) {
				t.Fatal("actual compiled NUM child source stop", low)
			}
			first = h.finish(&r, nil)
			if h.tmp != beforeTmp || h.sourceEnded || h.boolNumeral != savedChild || h.sourceCause != low {
				t.Fatal("younger compiled NUM stop consumed older tmp")
			}
		} else {
			first = h.own(&r, false)
		}
		if !errors.Is(first, arena.ErrBounds) || rt.Stats().LiveBlocks+1 != blocks {
			t.Fatal("actual consumed release", source, first)
		}
		primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{fs})
		if source {
			if !errors.Is(first, h.sourceCause) || h.sourceEnded {
				t.Fatal("first source stop history")
			}
			before := *h
			if !errors.Is(h.finish(&r, nil), h.sourceCause) || !reflect.DeepEqual(*h, before) {
				t.Fatal("source retry consumed older state")
			}
		} else if h.ownCause != first || h.ownEnded {
			t.Fatal("first owning stop history")
		}
		recovery, e := rt.Allocate(400000)
		serviceMust(t, e)
		contextTLSFinish(t, rt, func() error { return h.own(&r, false) })
		contextTLSFinish(t, rt, func() error { return h.own(&r, true) })
		if source && !errors.Is(first, h.sourceCause) || !source && h.ownCause != first {
			t.Fatal("acknowledgment lost history")
		}
		primeTrialPhysicalAuditAll(t, q, s, []*UPolynomialFactors{fs}, []memory.Block{recovery})
		contextTLSFinish(t, rt, s.CleanupRetained)
		contextTLSFinish(t, rt, fs.Close)
		e = rt.Deallocate(&recovery)
		if e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (memory.Block{})) {
			t.Fatal("actual consumed recovery block", e)
		}
		contextTLSFinish(t, rt, s.Close)
		serviceMust(t, primes.Close())
		contextTLSFinish(t, rt, m.Close)
		contextTLSFinish(t, rt, q.CloseChecked)
		contextTLSFinish(t, rt, rt.Close)
		if b.Used() != 0 {
			t.Fatal("actual refunds", b.Used())
		}
	}
}
