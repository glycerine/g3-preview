// Source: src/math/polynomial/algebraic_numbers.cpp:1565-1577;
// src/util/memory_manager.cpp:274-285. Returning-transport lifecycle fixture.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"reflect"
	"testing"
)

func TestAlgebraicNegActualTransportOwnerAndTupleRefusal(t *testing.T) {
	budget := arena.NewBudget(1 << 26)
	var c *AlgebraicRootCore
	armed := false
	calls := 0
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: budget, ThreadLocal: true, Transport: func(f memory.Failure) error {
		if !armed {
			return f
		}
		calls++
		q := &c.negation
		before := *q
		used := budget.Used()
		physical := c.config.Rationals.z.memory.Stats()
		r := algebraicNegInvocationReceipt{c, q, q.invocation}
		copiedFrame := *q
		fabricated := algebraicNegInvocationReceipt{c, &copiedFrame, q.invocation}
		if _, ok := c.negationHistoryFor(&fabricated); ok {
			t.Error("fabricated frame acquired history")
		}
		serviceMust(t, c.cleanupNegation(&fabricated, true))
		copiedContext := q.context(false)
		copiedContext.algebraicNeg = &copiedFrame
		if copiedContext.roots(c.config.Rationals.z) != nil {
			t.Error("copied frame acquired numeric authority")
		}
		stale := q.context(false)
		stale.algebraicNegInvocation--
		if stale.roots(c.config.Rationals.z) != nil {
			t.Error("stale epoch acquired numeric authority")
		}
		copiedOwner := *c
		copiedOwner.self = &copiedOwner
		copiedOwner.negation.owner = &copiedOwner
		foreign := copiedOwner.negation.context(false)
		if foreign.roots(c.config.Rationals.z) != nil {
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
		z := c.config.Rationals.z
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
		if !reflect.DeepEqual(before, *q) || used != budget.Used() || physical != z.memory.Stats() {
			t.Error("executing callback mutated charged state/actual tuple")
		}
		return f
	}})
	serviceMust(t, e)
	cfg := serviceConfig(budget)
	cfg.Integers.Memory = rt
	cfg.Integers.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	s, e := NewRationalService(cfg)
	serviceMust(t, e)
	p, e := s.NewUPolynomialManager(UPolynomialConfig{Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }})
	serviceMust(t, e)
	primes, e := NewPrimeGenerator(budget)
	serviceMust(t, e)
	f, e := NewUPolynomialFactorizationService(p, UPolynomialFactorizationConfig{Primes: primes})
	serviceMust(t, e)
	b, e := NewBinaryRationalManager(s.manager.z, BinaryRationalConfig{Diagnostics: s})
	serviceMust(t, e)
	bi, e := NewBinaryIntervalManager(b)
	serviceMust(t, e)
	allocator, e := memory.NewSmallAllocator(memory.SmallAllocatorConfig{Runtime: rt, Profile: cfg.Integers.Profile})
	serviceMust(t, e)
	c, e = NewAlgebraicRootCore(AlgebraicRootConfig{Rationals: s.manager, Polynomials: p, Factors: f, Binary: b, Intervals: bi, Allocator: allocator, Checkpoint: p.config.Checkpoint, Cooperation: p.config.Cooperation})
	serviceMust(t, e)
	var a Algebraic
	n := NewRational()
	var held memory.Block
	defer func() {
		armed = false
		budget.FailAfter(-1)
		serviceMust(t, rt.SetMaxSize(0))
		r := algebraicNegInvocationReceipt{c, &c.negation, c.negation.invocation}
		contextTLSFinish(t, rt, func() error { return c.cleanupNegation(&r, false) })
		contextTLSFinish(t, rt, func() error { return c.Del(&a) })
		contextTLSFinish(t, rt, func() error { return c.cleanupNegation(&r, true) })
		contextTLSFinish(t, rt, func() error { return s.manager.Del(&n) })
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
	serviceMust(t, s.manager.SetInt32(&n, math.MinInt32))
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
	algebraicCellSourcePhysical(t, c, nil, nil, []memory.Block{held}, &a)
}
