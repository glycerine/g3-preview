package polynomial

// Source: src/math/polynomial/algebraic_numbers.cpp:1565-1577;
// src/util/memory_manager.cpp:274-285. Returning-transport lifecycle fixture.

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	primegen "github.com/glycerine/g3/pkg/util/prime"
)

func TestAlgebraicNegActualTransportOwnerAndTupleRefusal(t *testing.T) {
	budget := arena.NewBudget(1 << 26)
	var c *AlgebraicRootCore
	armed := false
	calls := 0
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, Transport: func(f umemory.Failure) error {
		if !armed {
			return f
		}
		calls++
		q := &c.negation
		before := *q
		used := budget.Used()
		physical := c.config.Rationals.Integers().RuntimeOwner().Stats()
		r := algebraicNegInvocationReceipt{c, q, q.invocation}
		copiedFrame := *q
		fabricated := algebraicNegInvocationReceipt{c, &copiedFrame, q.invocation}
		if _, ok := c.negationHistoryFor(&fabricated); ok {
			t.Error("fabricated frame acquired history")
		}
		serviceMust(t, c.cleanupNegation(&fabricated, true))
		copiedContext := q.context(false)
		copiedContext.algebraicNeg = &copiedFrame
		if (copiedContext.roots(c.config.Rationals.Integers())) != nil {
			t.Error("copied frame acquired numeric authority")
		}
		stale := q.context(false)
		stale.algebraicNegInvocation--
		if (stale.roots(c.config.Rationals.Integers())) != nil {
			t.Error("stale epoch acquired numeric authority")
		}
		copiedOwner := *c
		copiedOwner.self = &copiedOwner
		copiedOwner.negation.owner = &copiedOwner
		foreign := copiedOwner.negation.context(false)
		if (foreign.roots(c.config.Rationals.Integers())) != nil {
			t.Error("copied owner acquired numeric authority")
		}

		for _, abort := range []bool{false, true} {
			if e := c.cleanupNegation(&r, abort); e != ErrAlgebraicPending {
				t.Errorf("executing exact child acknowledged: %v", e)
			}
		}
		if e := c.CleanupRetained(); e != ErrAlgebraicPending {
			t.Errorf("core owning callback: %v", e)
		}
		if e := c.Close(); e != ErrAlgebraicPending {
			t.Errorf("core close callback: %v", e)
		}
		z := c.config.Rationals.Integers()
		p := c.config.Polynomials
		if e := z.CloseChecked(); e != ErrPolynomialPending {
			t.Errorf("Integer owning callback: %v", e)
		}
		if e := z.SourceFinalize(); e != ErrPolynomialPending {
			t.Errorf("Integer source callback: %v", e)
		}
		if e := c.config.Rationals.SourceFinalize(); e != ErrPolynomialPending {
			t.Errorf("Rational source callback: %v", e)
		}
		if e := c.config.Rationals.CloseChecked(); e != ErrPolynomialPending {
			t.Errorf("Rational owning callback: %v", e)
		}
		if e := p.Close(); e != ErrUPolynomialCleanupPending {
			t.Errorf("UP owning callback: %v", e)
		}
		if e := p.SourceFinalize(); e != ErrUPolynomialCleanupPending {
			t.Errorf("UP source callback: %v", e)
		}
		if !reflect.DeepEqual(before, *q) || used != budget.Used() || physical != z.RuntimeOwner().Stats() {
			t.Error("executing callback mutated charged state/actual tuple")
		}
		return f
	}})
	serviceMust(t, e)
	cfg := serviceConfig(budget)
	cfg.Integers.Memory = rt
	cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	s, e := unumeral.NewRationalService(cfg)
	serviceMust(t, e)
	p, e := NewUPolynomialManagerFromRationalService(s, UPolynomialConfig{Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }})
	serviceMust(t, e)
	primes, e := primegen.NewGenerator(budget)
	serviceMust(t, e)
	f, e := NewUPolynomialFactorizationService(p, UPolynomialFactorizationConfig{Primes: primes})
	serviceMust(t, e)
	b, e := unumeral.NewBinaryRationalManager(s.Manager.Z, unumeral.BinaryRationalConfig{Diagnostics: s})
	serviceMust(t, e)
	bi, e := unumeral.NewBinaryIntervalManager(b)
	serviceMust(t, e)
	allocator, e := umemory.NewSmallAllocator(umemory.SmallAllocatorConfig{Runtime: rt, Profile: cfg.Integers.Profile})
	serviceMust(t, e)
	c, e = NewAlgebraicRootCore(AlgebraicRootConfig{Rationals: s.Manager, Polynomials: p, Factors: f, Binary: b, Intervals: bi, Allocator: allocator, Checkpoint: p.config.Checkpoint, Cooperation: p.config.Cooperation})
	serviceMust(t, e)
	var a Algebraic
	n := unumeral.NewRational()
	var held umemory.Block
	defer func() {
		armed = false
		budget.FailAfter(-1)
		serviceMust(t, rt.SetMaxSize(0))
		r := algebraicNegInvocationReceipt{c, &c.negation, c.negation.invocation}
		contextTLSFinish(t, rt, func() error { return c.cleanupNegation(&r, false) })
		contextTLSFinish(t, rt, func() error { return c.CleanupValue(&a) })
		contextTLSFinish(t, rt, func() error { return c.cleanupNegation(&r, true) })
		contextTLSFinish(t, rt, func() error { return s.Manager.Del(&n) })
		contextTLSFinish(t, rt, func() error { return rt.Deallocate(&held) })
		contextTLSFinish(t, rt, c.Close)
		contextTLSFinish(t, rt, allocator.Close)
		bi.Close()
		contextTLSFinish(t, rt, b.CloseChecked)
		contextTLSFinish(t, rt, f.Close)
		serviceMust(t, primes.Close())
		contextTLSFinish(t, rt, p.Close)
		contextTLSFinish(t, rt, s.Close)
		contextTLSFinish(t, rt, rt.Close)
		if budget.Used() != 0 {
			t.Error("actual owner refund", budget.Used())
		}
	}()
	serviceMust(t, s.Manager.SetInt32(&n, math.MinInt32))
	serviceMust(t, c.SetRational(&a, &n))
	// Source TLS synchronization reaches a real Runtime refusal at MIN promotion.
	pad := int64(99996) - rt.State().LocalSize - 12
	if pad <= 0 {
		t.Fatal(rt.State())
	}
	held, e = rt.Allocate(uint64(pad))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	armed = true
	body := c.Neg(&a)
	armed = false
	if !numericPhysicalReservationRefusal(body) || calls != 1 {
		t.Fatal("actual MIN allocation transport", body, calls)
	}
	if c.negation.cleanupCause != nil {
		t.Fatal("reentry refusal poisoned first-own history")
	}
	if !errors.Is(body, c.negation.bodyCause) {
		t.Fatal("immutable body cause missing")
	}
	algebraicCellSourcePhysical(t, c, nil, nil, []umemory.Block{held}, &a)
}
