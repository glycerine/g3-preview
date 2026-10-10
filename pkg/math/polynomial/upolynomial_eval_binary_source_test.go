// Source: src/math/polynomial/upolynomial.cpp:1734-1764.
// Qualified returning-transport and signed-host TLS destructor guard fixture.
package polynomial

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestUPolynomialEvalSignBinaryPublicBodyAndNumeralDT(t *testing.T) {
	reached := false
	for offset := int64(12); offset < 400 && !reached; offset += 4 {
		t.Run(fmt.Sprint(offset), func(t *testing.T) {
			var z *util.IntegerManager
			var rt *umemory.Runtime
			var lane, held umemory.Block
			armed := false
			transport := func(f umemory.Failure) error {
				if !armed || z == nil || z.AddMulInvocation.Tmp.Cell == 0 || z.IsFailureRef(z.AddMulInvocation.Tmp.Cell) {
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
			z = c.config.Rationals.Integers()
			poly := algebraicPolynomial(t, p, 1, math.MaxInt32, 1)
			v, e := p.wholeView(poly)
			serviceMust(t, e)
			var b util.BinaryRational
			serviceMust(t, c.config.Binary.SetInt32(&b, math.MaxInt32))
			defer func() {
				armed = false
				serviceMust(t, rt.SetMaxSize(0))
				contextTLSFinish(t, rt, func() error { return p.cleanupBinaryEvaluation(false) })
				contextTLSFinish(t, rt, z.CleanupAddMulRaw)
				contextTLSFinish(t, rt, func() error { return c.config.Binary.Del(&b) })
				contextTLSFinish(t, rt, func() error { return p.cleanupBinaryEvaluation(true) })
				for _, block := range []*umemory.Block{&lane, &held} {
					contextTLSFinish(t, rt, func() error {
						if *block == (umemory.Block{}) {
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
			baseline := z.Budget().Used() - integerActualOwnedBytes(z) - p.binaryEvaluationHistoryBytes
			_, body := p.EvalSignBinary(v, &b)
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
			q := &p.binaryEvaluation
			if q.bodyCause == nil || q.sourceCause == nil || q.sourceRemaining != 1 || q.addMul.Frame == nil || q.addMul.Frame.Tmp.Cell == 0 {
				t.Fatal("actual source/RAW frontier", q.sourceRemaining)
			}
			algebraicCellSourcePhysical(t, c, []UPolynomial{poly}, []*util.Int{&(*b.BorrowNumerator()), &q.values[0], &q.values[1], &q.addMul.Frame.Tmp}, []umemory.Block{lane, held})
			if want := baseline + integerActualOwnedBytes(z) + p.binaryEvaluationHistoryBytes; want != z.Budget().Used() {
				t.Fatal("actual cold graph", want, z.Budget().Used())
			}
			before := *q
			used := z.Budget().Used()
			physical := rt.Stats()
			if e := p.SourceFinalize(); e != q.sourceCause {
				t.Fatal("older UPM source crossed stopped Horner", e)
			}
			if again := q.finish(nil); !errors.Is(again, q.sourceCause) || q.sourceRemaining != before.sourceRemaining || used != z.Budget().Used() || physical != rt.Stats() {
				t.Fatal("source retry replayed stopped NUM", again)
			}
			polyBefore := poly
			if e := p.DelPolynomial(&poly); e != ErrUPolynomialCleanupPending || poly != polyBefore || used != z.Budget().Used() || physical != rt.Stats() {
				t.Fatal("direct older coefficient owner crossed stopped NUM", e)
			}
			h := *q.history
			serviceMust(t, p.cleanupBinaryEvaluation(false))
			serviceMust(t, c.config.Binary.Del(&b))
			firstOwn := p.cleanupBinaryEvaluation(true)
			if !errors.Is(firstOwn, arena.ErrBounds) || q.ownCause != firstOwn {
				t.Fatal("actual unjoined TMP owning stop", firstOwn)
			}
			contextTLSFinish(t, rt, func() error { return p.cleanupBinaryEvaluation(true) })
			if q.history.bodyCause != h.bodyCause || q.history.sourceCause != h.sourceCause || q.history.ownCause != firstOwn || !errors.Is(body, h.sourceCause) {
				t.Fatal("own ACK erased original history")
			}

		})
	}
	if !reached {
		t.Fatal("actual failed ADD with retained TMP did not reach TLS destructor frontier")
	}
}
