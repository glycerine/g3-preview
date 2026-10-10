// Source: src/math/polynomial/upolynomial.cpp:1734-1764;
// src/util/mpz.h:393-407. Qualified public signed-host TLS destructor fixture.
package polynomial

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	primegen "github.com/glycerine/g3/pkg/util/prime"
)

func algebraicNegTLSFixture(t *testing.T, transport umemory.ErrorTransport) (*AlgebraicRootCore, *UPolynomialManager, *umemory.Runtime) {
	t.Helper()
	budget := arena.NewBudget(1 << 26)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, Transport: transport})
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
	c, e := NewAlgebraicRootCore(AlgebraicRootConfig{Rationals: s.Manager, Polynomials: p, Factors: f, Binary: b, Intervals: bi, Allocator: allocator, Checkpoint: p.config.Checkpoint, Cooperation: p.config.Cooperation})
	serviceMust(t, e)
	t.Cleanup(func() {
		budget.FailAfter(-1)
		serviceMust(t, rt.SetMaxSize(0))
		contextTLSFinish(t, rt, c.CleanupNegation)
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
			t.Fatal("actual owner refunds", budget.Used())
		}
	})
	return c, p, rt
}

func TestAlgebraicNegActualHornerBodyAndYoungerDestructorStop(t *testing.T) {
	algebraicNegDTCase(t, false)
}
func TestAlgebraicNegCoreCloseAfterStickySourceStop(t *testing.T) { algebraicNegDTCase(t, true) }
func algebraicNegDTCase(t *testing.T, closeCore bool) {
	reached := false
	for offset := int64(12); offset < 400 && !reached; offset += 4 {
		t.Run(fmt.Sprint(offset), func(t *testing.T) {
			var c *AlgebraicRootCore
			var rt *umemory.Runtime
			var lane, held umemory.Block
			armed := false
			transport := func(f umemory.Failure) error {
				if !armed || c.negation.phase != 4 || c.negation.eval.addMul.Frame == nil || c.negation.eval.addMul.Frame.Tmp.Cell == 0 || c.config.Rationals.Integers().IsFailureRef(c.negation.eval.addMul.Frame.Tmp.Cell) {
					return f
				}
				// The actual failed ADD allocation has returned through Runtime transport;
				// the source AddMul body has a published TMP and no reached explicit DEL.
				armed = false
				reached = true
				serviceMust(t, rt.SetMaxSize(0))
				contextTLSNearMinimum(t, rt)
				var e error
				lane, e = rt.Allocate(8)
				serviceMust(t, e)
				delta := int64(-1000000)
				if e = rt.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
					t.Fatal(e)
				}
				return f
			}
			var p *UPolynomialManager
			c, p, rt = algebraicNegTLSFixture(t, transport)
			z := c.config.Rationals.Integers()
			original := algebraicPolynomial(t, p, 1, math.MinInt32, 1)
			v, e := p.wholeView(original)
			serviceMust(t, e)
			var lower, upper unumeral.BinaryRational
			var upperInt unumeral.Int
			serviceMust(t, c.config.Binary.SetInt32(&lower, math.MaxInt32))
			serviceMust(t, z.SetUint32(&upperInt, uint32(math.MaxInt32)+3))
			serviceMust(t, c.config.Binary.SetInteger(&upper, &upperInt))
			var a Algebraic
			serviceMust(t, c.SetRoot(&a, v, &lower, &upper, true))
			serviceMust(t, c.config.Binary.Del(&upper))
			serviceMust(t, c.config.Binary.Del(&lower))
			serviceMust(t, z.Del(&upperInt))
			defer func() {
				armed = false
				serviceMust(t, rt.SetMaxSize(0))
				if !c.closed {
					r := algebraicNegInvocationReceipt{c, &c.negation, c.negation.invocation}
					contextTLSFinish(t, rt, func() error { return c.cleanupNegation(&r, false) })
					contextTLSFinish(t, rt, func() error { return c.Del(&a) })
					contextTLSFinish(t, rt, func() error { return c.cleanupNegation(&r, true) })
				}
				if lane != (umemory.Block{}) {
					contextTLSFinish(t, rt, func() error {
						if lane == (umemory.Block{}) {
							return nil
						}
						return rt.Deallocate(&lane)
					})
				}
				contextTLSFinish(t, rt, func() error {
					if held == (umemory.Block{}) {
						return nil
					}
					return rt.Deallocate(&held)
				})
			}()
			pad := int64(100000) - rt.State().LocalSize - offset
			if pad <= 0 {
				t.Fatal(rt.State())
			}
			held, e = rt.Allocate(uint64(pad))
			serviceMust(t, e)
			serviceMust(t, rt.SetMaxSize(1))
			armed = true
			body := c.Neg(&a)
			armed = false
			if !reached {
				if body != nil && !numericPhysicalReservationRefusal(body) {
					t.Fatal(body)
				}
				return
			}
			q := &c.negation
			r := algebraicNegInvocationReceipt{c, q, q.invocation}
			if !errors.Is(body, arena.ErrBounds) || q.bodyCause == nil || q.eval.sourceCause == nil || q.eval.sourceRemaining != 1 {
				t.Fatal("actual body plus r NUM DT frontier after inline ak DT", body, q.eval.sourceRemaining)
			}
			if q.eval.addMul.Frame.Tmp.Cell == 0 || q.eval.addMul.Frame.SourceCause != nil {
				t.Fatal("failed-body RAW TMP was incorrectly source-destructed")
			}
			extra := []*unumeral.Int{&q.eval.values[0], &q.eval.values[1], &q.eval.addMul.Frame.Tmp}
			algebraicCellSourcePhysical(t, c, []UPolynomial{original}, extra, []umemory.Block{lane, held}, &a)
			t.Logf("captured lane=%+v root=%+v eval=%+v tmp=%+v", lane, q.numeric, q.eval.numeric, q.eval.addMul.Frame.Numeric)
			before := q.diagnostic()
			physical := rt.Stats()
			callerBefore := a
			usedBefore := z.Budget().Used()
			if e := c.CleanupValue(&a); e != ErrAlgebraicPending {
				t.Fatal("own caller abort crossed younger stopped NUM", e)
			}
			if e := c.delClosing(&a); e != ErrAlgebraicPending {
				t.Fatal("private source caller DT crossed younger stopped NUM", e)
			}
			if e := c.Del(&a); e != ErrAlgebraicPending {
				t.Fatal("direct caller DT crossed younger stopped NUM", e)
			}
			if a != callerBefore || before != q.diagnostic() || physical != rt.Stats() || usedBefore != z.Budget().Used() {
				t.Fatal("refused direct caller DT changed physical/cold graph")
			}

			again := q.finishBinarySource(nil)
			if !errors.Is(again, q.eval.sourceCause) || before != q.diagnostic() || physical != rt.Stats() {
				t.Fatal("stopped source replay changed cursor/roots", again)
			}
			if closeCore {
				olderFactors, olderRoots, olderUppers, olderLowers := c.factors, c.roots, c.uppers, c.lowers
				if e := c.Close(); e != ErrAlgebraicLive {
					t.Fatal("own core close crossed caller-value DT", e)
				}
				if c.closing || c.closed || c.factors != olderFactors || c.roots != olderRoots || c.uppers != olderUppers || c.lowers != olderLowers || before != q.diagnostic() || physical != rt.Stats() {
					t.Fatal("refused owner close changed older members")
				}
			}
			contextTLSFinish(t, rt, func() error { return c.cleanupNegation(&r, false) })
			contextTLSFinish(t, rt, func() error { return c.Del(&a) })
			if closeCore {
				oldHeader, oldHistory := c.header, len(c.negationHistory)
				oldSource := q.sourceCause
				stop := c.Close()
				if !errors.Is(stop, arena.ErrBounds) || c.closed || c.closing || c.header != oldHeader || len(c.negationHistory) != oldHistory || q.sourceCause != oldSource {
					t.Fatal("younger own stop failed to protect older owner storage", stop)
				}
				algebraicCellSourcePhysical(t, c, []UPolynomial{original}, extra, []umemory.Block{lane, held})
				contextTLSFinish(t, rt, c.Close)
				if q.active || !c.closed || q.bodyCause != before.bodyCause || q.sourceCause != before.sourceCause || q.eval.sourceRemaining != before.sourceRemaining {
					t.Fatal("core owning continuation erased source history")
				}
				algebraicCellSourcePhysical(t, c, []UPolynomial{original}, nil, []umemory.Block{lane, held})
				return
			}

			contextTLSFinish(t, rt, func() error { return c.cleanupNegation(&r, true) })
			t.Logf("after-own lane=%+v read=%v", lane, func() error { _, e := rt.Read(lane, 0); return e }())
			if !errors.Is(q.cleanupCause, arena.ErrBounds) {
				t.Fatal("matched real first-own stop missing", q.cleanupCause)
			}
			history, ok := c.negationHistoryFor(&r)
			if !ok || history != q.diagnostic() {
				t.Fatal("final child tuples/own history lost")
			}
			if q.active || q.bodyCause != before.bodyCause || q.sourceCause != before.sourceCause {
				t.Fatal("own acknowledgment rewrote stopped histories")
			}
		})
	}
	if !reached {
		t.Fatal("no actual later ADD allocation transport with published BIG TMP")
	}
}
