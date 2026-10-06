// Source: src/math/polynomial/upolynomial.cpp:1801-1820.
// Qualified returning-transport and signed-host TLS destructor guard fixture.
package mpnum

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"testing"
)

func TestUPolynomialEvalSignIntPublicBodyAndNumeralDT(t *testing.T) {
	reached := false
	for offset := int64(12); offset < 400 && !reached; offset += 4 {
		t.Run(fmt.Sprint(offset), func(t *testing.T) {
			var z *IntegerManager
			var rt *memory.Runtime
			var lane, held memory.Block
			armed := false
			transport := func(f memory.Failure) error {
				if !armed || z == nil || z.addMulInvocation.tmp.cell == 0 || z.isFailureRef(z.addMulInvocation.tmp.cell) {
					return f
				}
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
			c, p, runtime := algebraicNegTLSFixture(t, transport)
			rt = runtime
			z = c.config.Rationals.z
			poly := algebraicPolynomial(t, p, 1, math.MaxInt32, 1)
			v, e := p.wholeView(poly)
			serviceMust(t, e)
			var b Int
			serviceMust(t, z.SetInt32(&b, math.MaxInt32))
			defer func() {
				armed = false
				serviceMust(t, rt.SetMaxSize(0))
				contextTLSFinish(t, rt, func() error { return p.cleanupIntegerEvaluation(false) })
				contextTLSFinish(t, rt, z.CleanupAddMulRaw)
				contextTLSFinish(t, rt, func() error { return z.Del(&b) })
				contextTLSFinish(t, rt, func() error { return p.cleanupIntegerEvaluation(true) })
				for _, block := range []*memory.Block{&lane, &held} {
					contextTLSFinish(t, rt, func() error {
						if *block == (memory.Block{}) {
							return nil
						}
						return rt.Deallocate(block)
					})
				}
			}()
			pad := int64(100000) - rt.State().LocalSize - offset
			if pad <= 0 {
				t.Fatal(rt.State())
			}
			held, e = rt.Allocate(uint64(pad))
			serviceMust(t, e)
			serviceMust(t, rt.SetMaxSize(1))
			armed = true
			baseline := z.budget.Used() - integerActualOwnedBytes(z) - p.integerEvaluationHistoryBytes
			_, body := p.EvalSignInt(v, &b)
			armed = false
			if !reached {
				if body != nil && !numericPhysicalReservationRefusal(body) {
					t.Fatal(body)
				}
				return
			}
			if !errors.Is(body, arena.ErrBounds) {
				t.Fatal("public Horner discarded actual r NUM destructor cause", body)
			}
			q := &p.integerEvaluation
			if q.bodyCause == nil || q.sourceCause == nil || q.sourceRemaining != 1 || q.addMul.frame == nil || q.addMul.frame.tmp.cell == 0 {
				t.Fatal("actual source/RAW frontier", q.sourceRemaining)
			}
			algebraicCellSourcePhysical(t, c, []UPolynomial{poly}, []*Int{&b, &q.values[0], &q.addMul.frame.tmp}, []memory.Block{lane, held})
			if want := baseline + integerActualOwnedBytes(z) + p.integerEvaluationHistoryBytes; want != z.budget.Used() {
				t.Fatal("actual cold graph", want, z.budget.Used())
			}
			before := *q
			used := z.budget.Used()
			physical := rt.Stats()
			if e := p.SourceFinalize(); e != q.sourceCause {
				t.Fatal("older UPM source crossed stopped Horner", e)
			}
			if again := q.finish(nil); !errors.Is(again, q.sourceCause) || q.sourceRemaining != before.sourceRemaining || used != z.budget.Used() || physical != rt.Stats() {
				t.Fatal("source retry replayed stopped NUM", again)
			}
			polyBefore := poly
			if e := p.DelPolynomial(&poly); e != ErrUPolynomialCleanupPending || poly != polyBefore || used != z.budget.Used() || physical != rt.Stats() {
				t.Fatal("direct older coefficient owner crossed stopped NUM", e)
			}
			h := *q.history
			serviceMust(t, p.cleanupIntegerEvaluation(false))
			serviceMust(t, z.Del(&b))
			firstOwn := p.cleanupIntegerEvaluation(true)
			if !errors.Is(firstOwn, arena.ErrBounds) || q.ownCause != firstOwn {
				t.Fatal("actual unjoined TMP owning stop", firstOwn)
			}
			contextTLSFinish(t, rt, func() error { return p.cleanupIntegerEvaluation(true) })
			if q.history.bodyCause != h.bodyCause || q.history.sourceCause != h.sourceCause || q.history.ownCause != firstOwn || !errors.Is(body, h.sourceCause) {
				t.Fatal("own ACK erased original history")
			}

		})
	}
	if !reached {
		t.Fatal("actual failed ADD with retained TMP did not reach TLS destructor frontier")
	}
}
