// Source: src/math/polynomial/upolynomial_factorization.cpp:1238-1273;
// src/math/polynomial/upolynomial.cpp:609-651; src/util/memory_manager.cpp:274-285.
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

// Actual source scalar-MUL leaf transport profile; no C++ malloc callback claim.
func TestUPolynomialRecombinationPublicationTransportProtectsActualCaller(t *testing.T) {
	for _, mode := range []string{"service", "close", "source", "set_z"} {
		t.Run(mode, func(t *testing.T) { recombinationPublicationTransportProtectsActualCaller(t, mode) })
	}
}
func recombinationPublicationTransportProtectsActualCaller(t *testing.T, mode string) {
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
			oldMode := prior.zpe.modular.z
			switch mode {
			case "service":
				stop = s.CleanupRetained()
			case "close":
				stop = prior.zpe.Close()
			case "source":
				stop = prior.zpe.SourceFinalize()
			case "set_z":
				stop = prior.zpe.SetZ()
			}
			if prior.zpe.modular.z != oldMode {
				t.Error("related mode changed during executing caller")
			}
			if stop != ErrUPolynomialCleanupPending || s.scope != prior || !reflect.DeepEqual(*prior, before) {
				t.Errorf("actual recombination scalar MUL caller changed before refusal: %v", stop)
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
	var value, prime Int
	defer func() {
		armed = false
		b.FailAfter(-1)
		serviceMust(t, rt.SetMaxSize(0))
		contextTLSFinish(t, rt, s.CleanupRetained)
		contextTLSFinish(t, rt, fs.Close)
		contextTLSFinish(t, rt, func() error { return q.z.Del(&value) })
		contextTLSFinish(t, rt, func() error { return q.z.Del(&prime) })
		contextTLSFinish(t, rt, func() error { return rt.Deallocate(&held) })
		contextTLSFinish(t, rt, s.Close)
		serviceMust(t, primes.Close())
		contextTLSFinish(t, rt, m.Close)
		contextTLSFinish(t, rt, q.CloseChecked)
		contextTLSFinish(t, rt, rt.Close)
		if b.Used() != 0 {
			t.Error("actual scalar MUL owner refunds", b.Used())
		}
	}()
	parent, e := s.newSquareFreeScope()
	serviceMust(t, e)
	// Construct the actual enclosing objects through the trial declaration;
	// a count alone would not establish their source destruction frontier.
	parent.zp, e = NewRelatedUPolynomialManager(m)
	serviceMust(t, e)
	parent.best, e = parent.zp.NewFactors()
	serviceMust(t, e)
	serviceMust(t, parent.ds.init(parent.best))
	parent.zpe, e = NewRelatedUPolynomialManager(m)
	serviceMust(t, e)
	parent.lifted, e = parent.zpe.NewFactors()
	serviceMust(t, e)
	var liftedSource upolyVectorState
	for _, constant := range []int32{1, 2} {
		serviceMust(t, parent.zpe.set(&liftedSource, []Int{NewInt(constant), NewInt(1)}))
		serviceMust(t, parent.lifted.PushCopy(parent.zpe.stateView(&liftedSource), 1))
	}
	serviceMust(t, parent.zpe.reset(&liftedSource))
	parent.zpe.freeBacking(&liftedSource)
	serviceMust(t, parent.it.init(parent.lifted, &parent.ds))
	pw := make([]uint32, 101)
	for i := range pw {
		pw[i] = ^uint32(0)
	}
	pw[100] = (1 << 17) - 1
	serviceMust(t, q.z.SetLimbs(&prime, pw))
	serviceMust(t, parent.zpe.SetZP(&prime))
	parent.constructed = 12
	parent.leading = NewInt(2)
	words := make([]uint32, 80)
	words[79] = 1
	serviceMust(t, q.z.SetLimbs(&value, words))
	serviceMust(t, m.set(&parent.trial, []Int{value, NewInt(1)}))
	pad := int64(99996) - rt.State().LocalSize - 12
	if pad <= 0 {
		t.Fatal("actual source threshold", rt.State())
	}
	held, e = rt.Allocate(uint64(pad))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	armed = true
	_, body := s.publishRecombinationTrial(fs, 1, true)
	armed = false
	if !numericPhysicalReservationRefusal(body) || calls != 1 {
		t.Fatal("actual source scalar MUL refusal", body, calls)
	}
	primeTrialPhysicalAuditAll(t, q, s, []*UPolynomialFactors{fs}, []memory.Block{held}, &value, &prime)
}

func recombinationPublicationFixture(t *testing.T) (*RationalManager, *UPolynomialFactorizationService, *UPolynomialFactors) {
	t.Helper()
	q, s := mignotteDirectFixture(t)
	m := s.manager
	c, e := s.newSquareFreeScope()
	serviceMust(t, e)
	c.zp, e = NewRelatedUPolynomialManager(m)
	serviceMust(t, e)
	c.best, e = c.zp.NewFactors()
	serviceMust(t, e)
	serviceMust(t, c.ds.init(c.best))
	c.zpe, e = NewRelatedUPolynomialManager(m)
	serviceMust(t, e)
	p := NewInt(7)
	serviceMust(t, c.zpe.SetZP(&p))
	c.lifted, e = c.zpe.NewFactors()
	serviceMust(t, e)
	var liftedSource upolyVectorState
	for _, constant := range []int32{1, 2} {
		serviceMust(t, c.zpe.set(&liftedSource, []Int{NewInt(constant), NewInt(1)}))
		serviceMust(t, c.lifted.PushCopy(c.zpe.stateView(&liftedSource), 1))
	}
	serviceMust(t, c.zpe.reset(&liftedSource))
	c.zpe.freeBacking(&liftedSource)
	serviceMust(t, c.it.init(c.lifted, &c.ds))
	c.constructed = 14
	c.leading = NewInt(2)
	serviceMust(t, m.set(&c.trial, []Int{NewInt(1), NewInt(1)}))
	serviceMust(t, m.set(&c.fpp, []Int{NewInt(8), NewInt(12), NewInt(4)}))
	fs, e := m.NewFactors()
	serviceMust(t, e)
	return q, s, fs
}

func TestUPolynomialRecombinationPublicationEveryActualPrefix(t *testing.T) {
	for _, left := range []bool{true, false} {
		for _, exact := range []bool{true, false} {
			completed := false
			for ordinal := int64(0); ordinal < 1024; ordinal++ {
				q, s, fs := recombinationPublicationFixture(t)
				m := s.manager
				c := s.scope
				if !exact {
					serviceMust(t, q.z.SetInt64(&m.values(&c.fpp)[0], 9))
				}
				q.budget.FailAfter(ordinal)
				remove, body := s.publishRecombinationTrial(fs, 3, left)
				primeTrialPhysicalAuditAll(t, q, s, []*UPolynomialFactors{fs}, nil)
				frame := &c.recombinationStep
				savedBody, savedSource, savedOwn := frame.bodyCause, frame.sourceCause, frame.ownCause
				if body == nil {
					if remove != exact {
						t.Fatal("independent exact-division result", left, exact, remove)
					}
					if exact {
						if fs.count != 1 || fs.totalFactors != 3 || c.fpp.size != 2 || c.leading != NewInt(1) {
							t.Fatal("literal partial factor/remaining publication", fs.count, fs.totalFactors, c.fpp.size, c.leading)
						}
						first := NewInt(1)
						remaining := NewInt(2)
						if !left {
							first, remaining = remaining, first
						}
						f, e := fs.factor(0)
						serviceMust(t, e)
						if f.size != 2 || m.values(f)[0] != first || m.values(f)[1] != NewInt(1) || m.values(&c.fpp)[0] != remaining || m.values(&c.fpp)[1] != NewInt(1) {
							t.Fatal("independent primitive/complement factor coefficients")
						}
					} else if fs.count != 0 || c.fpp.size != 3 || m.values(&c.fpp)[0] != NewInt(9) {
						t.Fatal("rejected source trial changed original/factors")
					}
				}
				q.budget.FailAfter(-1)
				serviceMust(t, frame.own(&frame.receipt, false))
				serviceMust(t, s.CleanupRetained())
				if frame.bodyCause != savedBody || frame.sourceCause != savedSource || frame.ownCause != savedOwn {
					t.Fatal("matched phases changed source history")
				}
				primeTrialPhysicalAuditAll(t, q, s, []*UPolynomialFactors{fs}, nil)
				serviceMust(t, fs.Close())
				if body == nil {
					completed = true
					t.Log("actual step first completion", left, exact, ordinal)
					break
				}
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("unexpected actual step prefix", left, exact, ordinal, body)
				}
			}
			if !completed {
				t.Fatal("missing actual full source step", left, exact)
			}
		}
	}
}

func TestUPolynomialRecombinationRemainingPublicationEveryActualPrefix(t *testing.T) {
	completed := false
	for ordinal := int64(0); ordinal < 1024; ordinal++ {
		q, s, fs := recombinationPublicationFixture(t)
		m := s.manager
		c := s.scope
		serviceMust(t, m.set(&c.fpp, []Int{NewInt(4), NewInt(2)}))
		q.budget.FailAfter(ordinal)
		body := s.publishRecombinationRemainder(fs, 5)
		primeTrialPhysicalAuditAll(t, q, s, []*UPolynomialFactors{fs}, nil)
		if body == nil {
			f, e := fs.factor(0)
			serviceMust(t, e)
			if fs.count != 1 || fs.totalFactors != 5 || f.size != 2 || m.values(f)[0] != NewInt(2) || m.values(f)[1] != NewInt(1) {
				t.Fatal("independent remaining primitive factor publication")
			}
		}
		q.budget.FailAfter(-1)
		serviceMust(t, c.recombinationStep.own(&c.recombinationStep.receipt, false))
		serviceMust(t, s.CleanupRetained())
		primeTrialPhysicalAuditAll(t, q, s, []*UPolynomialFactors{fs}, nil)
		serviceMust(t, fs.Close())
		if body == nil {
			completed = true
			t.Log("actual remainder first completion", ordinal)
			break
		}
		if !numericPhysicalReservationRefusal(body) {
			t.Fatal("unexpected remaining source prefix", ordinal, body)
		}
	}
	if !completed {
		t.Fatal("missing actual remaining publication")
	}
}

func TestUPolynomialRecombinationPublicationBIGGCDDivEveryActualPrefix(t *testing.T) {
	complete := false
	for ordinal := int64(0); ordinal < 2048; ordinal++ {
		q, s, fs := recombinationPublicationFixture(t)
		m := s.manager
		c := s.scope
		var prime, leading, square, middle, constant Int
		serviceMust(t, q.z.SetLimbs(&prime, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
		serviceMust(t, c.zpe.SetZP(&prime))
		serviceMust(t, q.z.SetLimbs(&leading, []uint32{0, 0, 2})) // L=2^65
		serviceMust(t, q.z.Mul(&leading, &leading, &square))
		two := NewInt(2)
		serviceMust(t, q.z.Mul(&square, &two, &middle))
		serviceMust(t, q.z.Add(&middle, &leading, &middle))
		serviceMust(t, q.z.Mul(&leading, &two, &constant))
		serviceMust(t, q.z.Copy(&c.leading, &leading))
		// Primitive square-free f=(L*x+1)*(x+2), source f_pp already scaled by L.
		serviceMust(t, m.set(&c.fpp, []Int{constant, middle, square}))
		serviceMust(t, m.set(&c.trial, []Int{NewInt(2), NewInt(1)}))
		q.budget.FailAfter(ordinal)
		remove, body := s.publishRecombinationTrial(fs, 2, true)
		primeTrialPhysicalAuditAll(t, q, s, []*UPolynomialFactors{fs}, nil, &prime, &leading, &square, &middle, &constant)
		if body == nil {
			f, e := fs.factor(0)
			serviceMust(t, e)
			if !remove || fs.count != 1 || fs.totalFactors != 2 || f.size != 2 || m.values(f)[0] != NewInt(2) || m.values(f)[1] != NewInt(1) || c.fpp.size != 2 {
				t.Fatal("independent BIG primitive/exact source result", remove, fs.count, c.fpp.size)
			}
			same, e := q.z.Equal(&m.values(&c.fpp)[0], &leading)
			serviceMust(t, e)
			if !same {
				t.Fatal("independent remaining L coefficient")
			}
			same, e = q.z.Equal(&m.values(&c.fpp)[1], &square)
			serviceMust(t, e)
			if !same {
				t.Fatal("independent remaining L squared coefficient")
			}
			same, e = q.z.Equal(&c.leading, &leading)
			serviceMust(t, e)
			if !same {
				t.Fatal("borrowed/new leading coefficient")
			}
		}
		q.budget.FailAfter(-1)
		frame := &c.recombinationStep
		serviceMust(t, frame.own(&frame.receipt, false))
		serviceMust(t, s.CleanupRetained())
		primeTrialPhysicalAuditAll(t, q, s, []*UPolynomialFactors{fs}, nil, &prime, &leading, &square, &middle, &constant)
		serviceMust(t, fs.Close())
		for _, v := range []*Int{&prime, &leading, &square, &middle, &constant} {
			serviceMust(t, q.z.Del(v))
		}
		if body == nil {
			complete = true
			t.Log("actual BIG GCD/Div first completion", ordinal)
			break
		}
		if !numericPhysicalReservationRefusal(body) {
			t.Fatal("unexpected BIG source prefix", ordinal, body)
		}
	}
	if !complete {
		t.Fatal("actual BIG source block never completed")
	}
}

// Exact metadata guard profile: actual frame entry, no substituted arithmetic.
func TestUPolynomialRecombinationPublicationExactCapturedContext(t *testing.T) {
	q, s, fs := recombinationPublicationFixture(t)
	c := s.scope
	m := s.manager
	frame, e := s.beginRecombinationStep(fs)
	serviceMust(t, e)
	r := frame.receipt
	context := m.modular.recombinationAllocationContext(frame)
	if context.roots(q.z) != &frame.numeric {
		t.Fatal("actual passed frame not recognized")
	}
	before := *frame
	if e := frame.own(&r, false); e != ErrUPolynomialCleanupPending || !reflect.DeepEqual(*frame, before) {
		t.Fatal("executing receipt changed before refusal", e)
	}
	if e := frame.finish(&r, nil); e != ErrUPolynomialCleanupPending || !reflect.DeepEqual(*frame, before) {
		t.Fatal("executing source changed before refusal", e)
	}
	copied := *frame
	bad := r
	bad.frame = &copied
	if e := copied.own(&bad, true); e != ErrUPolynomialOwner || !reflect.DeepEqual(*frame, before) {
		t.Fatal("copied frame has owning authority", e)
	}
	stale := context
	stale.recombination = &copied
	if stale.roots(q.z) != nil {
		t.Fatal("copied frame has allocation authority")
	}
	for _, owner := range frame.managers() {
		owner.callbackDepth++
	}
	if context.roots(q.z) != nil {
		t.Fatal("foreign callback acquired direct leaf")
	}
	for _, owner := range frame.managers() {
		owner.callbackDepth--
	}
	other, e := NewRelatedUPolynomialManager(m)
	serviceMust(t, e)
	if other.modular.recombinationAllocationContext(frame).roots(q.z) != nil {
		t.Fatal("foreign modular receiver acquired direct leaf")
	}
	serviceMust(t, other.Close())
	frame.executing = false
	serviceMust(t, frame.finish(&r, nil))
	serviceMust(t, frame.own(&r, true))
	newer, e := s.beginRecombinationStep(fs)
	serviceMust(t, e)
	nr := newer.receipt
	if newer != &c.recombinationStep || newer.invocation == r.invocation || context.roots(q.z) != nil {
		t.Fatal("old context retargeted reused actual frame")
	}
	before = *newer
	if e := newer.own(&r, true); e != ErrUPolynomialOwner || !reflect.DeepEqual(*newer, before) {
		t.Fatal("stale receipt changed new invocation", e)
	}
	newer.executing = false
	serviceMust(t, newer.finish(&nr, nil))
	serviceMust(t, newer.own(&nr, true))
	serviceMust(t, s.CleanupRetained())
	serviceMust(t, fs.Close())
	primeTrialPhysicalAudit(t, q, s)
}

// Qualified signed-host TLS owner-primitive destructor profile; no ordinary
// mathematical allocation failure is claimed at this explicit child boundary.
func TestUPolynomialRecombinationPublicationActualChildStopAndOwningHistory(t *testing.T) {
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
		parent, e := s.newSquareFreeScope()
		serviceMust(t, e)
		related, e := NewRelatedUPolynomialManager(m)
		serviceMust(t, e)
		parent.zpe = related // actual related binding, explicitly closed after older NUM
		parent.constructed = 2
		serviceMust(t, q.z.Copy(&parent.p, &q.z.two64))
		h, e := s.beginRecombinationStep(fs)
		serviceMust(t, e)
		r := h.receipt
		context := m.modular.recombinationAllocationContext(h)
		child, e := q.z.beginDividesNumeralInvocation(context)
		serviceMust(t, e)
		savedChild := *child
		childContext := integerAllocationInvocationContext{boolDivides: child, boolDividesInvocation: child.invocation}
		serviceMust(t, q.z.copyFromOwnerWithAllocationContext(&child.frame.numeral, q.z, &q.z.two64, nil, childContext))
		serviceMust(t, q.z.ensureOwnedCapacity(&child.frame.numeral, 40000))
		child.frame.executing = false
		beforeOlder := parent.p
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
			if parent.p != beforeOlder || h.sourceEnded || h.boolNumeral != savedChild || h.sourceCause != low {
				t.Fatal("younger compiled NUM stop consumed older enclosing numeral")
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
		parent.zpe = nil
		contextTLSFinish(t, rt, related.Close)
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

// Direct source Copy leaf guard profile: actual acquired descriptor tokens,
// independent replacement generation, and source-false versus own-true phases.
func TestUPolynomialRecombinationPublicationExactOldLeafPreservesForeignReplacement(t *testing.T) {
	for ordinal := int64(0); ordinal < 512; ordinal++ {
		found := false
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			q, s := mignotteDirectFixture(t)
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
			parent, e := s.newSquareFreeScope()
			serviceMust(t, e)
			related, e := NewRelatedUPolynomialManager(m)
			serviceMust(t, e)
			parent.zpe = related
			defer func() { parent.zpe = nil; serviceMust(t, related.Close()) }()
			fs, e := m.NewFactors()
			serviceMust(t, e)
			defer func() { serviceMust(t, fs.Close()) }()
			h, e := s.beginRecombinationStep(fs)
			serviceMust(t, e)
			r := h.receipt
			q.budget.FailAfter(ordinal / 16)
			body := m.modular.copyWithSourceContext(&value, &prime, m.modular.recombinationAllocationContext(h))
			h.executing = false
			primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{fs}, &prime, &foreign, &large, &value)
			old := h.numeric.cell
			if old.object != (memory.ObjectRef{}) {
				if q.z.pendingCell != old || !numericPhysicalReservationRefusal(body) {
					t.Fatal("actual direct pending leaf absent", ordinal, body, old, q.z.pendingCell)
				}
				_ = h.finish(&r, body)
				serviceMust(t, h.own(&r, false))
				if h.numeric.cell != old || q.z.pendingCell != old {
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

// Actual direct descriptor refusal is drained with the reservation budget still
// exhausted: completed child acknowledgment, older source NUM, then exact abort.
func TestUPolynomialRecombinationPublicationNumericAbortBeforeFaultLift(t *testing.T) {
	found := false
	for ordinal := int64(0); ordinal < 32; ordinal++ {
		q, s := mignotteDirectFixture(t)
		m := s.manager
		parent, e := s.newSquareFreeScope()
		serviceMust(t, e)
		related, e := NewRelatedUPolynomialManager(m)
		serviceMust(t, e)
		parent.zpe = related
		parent.constructed = 2
		serviceMust(t, q.z.Copy(&parent.p, &q.z.two64))
		fs, e := m.NewFactors()
		serviceMust(t, e)
		var value, input Int
		serviceMust(t, q.z.SetLimbs(&input, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
		h, e := s.beginRecombinationStep(fs)
		serviceMust(t, e)
		r := h.receipt
		q.budget.FailAfter(ordinal)
		body := m.modular.copyWithSourceContext(&value, &input, m.modular.recombinationAllocationContext(h))
		h.executing = false
		_ = h.finish(&r, body)
		primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{fs}, &value, &input)
		if h.numeric.cell.object != (memory.ObjectRef{}) {
			if !numericPhysicalReservationRefusal(body) {
				t.Fatal("actual descriptor refusal", body)
			}
			serviceMust(t, h.own(&r, false))
			if e := s.finishSquareFreeScopeSource(); e != ErrUPolynomialCleanupPending || parent.constructed != 0 || parent.p.cell != 0 {
				t.Fatal("older source suffix before exact abort", e, parent.constructed)
			}
			serviceMust(t, h.own(&r, true))
			if h.active || polynomialNegNumericPending(&h.numeric) || h.bodyCause != body || h.sourceCause != nil {
				t.Fatal("exact numeric abort lost roots/history")
			}
			primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{fs}, &value, &input)
			found = true
		} else {
			q.budget.FailAfter(-1)
			serviceMust(t, h.own(&r, false))
			serviceMust(t, h.own(&r, true))
		}
		q.budget.FailAfter(-1)
		parent.zpe = nil
		serviceMust(t, related.Close())
		serviceMust(t, s.CleanupRetained())
		serviceMust(t, fs.Close())
		serviceMust(t, q.z.cleanupRawNumeral(&value))
		serviceMust(t, q.z.Del(&input))
		primeTrialPhysicalAudit(t, q, s)
		if found {
			t.Log("actual before-lift descriptor ordinal", ordinal)
			return
		}
	}
	t.Fatal("no direct descriptor refusal")
}
