package polynomial

// Source: src/math/polynomial/upolynomial.cpp:1906-1951.
// Qualified returning transport / signed-host TLS destructor frontier.

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestUPolynomialRootUpperBoundActualBodyAndNumeralDT(t *testing.T) {
	reached := false
	for offset := int64(12); offset < 500 && !reached; offset += 4 {
		t.Run(fmt.Sprint(offset), func(t *testing.T) {
			var z *unumeral.IntegerManager
			var rt *umemory.Runtime
			var lane, held umemory.Block
			armed := false
			var transportBlocks uint32
			transport := func(f umemory.Failure) error {
				if !armed {
					return f
				}
				armed = false
				serviceMust(t, rt.SetMaxSize(0))
				contextTLSNearMinimum(t, rt)
				var e error
				lane, e = rt.Allocate(8)
				serviceMust(t, e)
				delta := int64(-1000000)
				if e = rt.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
					t.Fatal(e)
				}
				transportBlocks = rt.Stats().LiveBlocks
				return f
			}
			c, p, runtime := algebraicNegTLSFixture(t, transport)
			rt = runtime
			z = c.config.Rationals.Integers()
			poly, e := p.NewPolynomial()
			serviceMust(t, e)
			var big unumeral.Int
			serviceMust(t, z.SetLimbs(&big, []uint32{1, 2, 3, 4}))
			serviceMust(t, p.SetInts(poly, []unumeral.Int{big, unumeral.NewInt(1)}))
			serviceMust(t, z.Del(&big))
			v, e := p.wholeView(poly)
			serviceMust(t, e)
			var out unumeral.Int
			defer func() {
				armed = false
				serviceMust(t, rt.SetMaxSize(0))
				contextTLSFinish(t, rt, func() error { return p.cleanupRootUpperBound(false) })
				contextTLSFinish(t, rt, func() error { return z.Del(&out) })
				contextTLSFinish(t, rt, func() error { return p.cleanupRootUpperBound(true) })
				for _, b := range []*umemory.Block{&lane, &held} {
					contextTLSFinish(t, rt, func() error {
						if *b == (umemory.Block{}) {
							return nil
						}
						return rt.Deallocate(b)
					})
				}
				contextTLSFinish(t, rt, func() error { return p.DelPolynomial(&poly) })
			}()
			pad := int64(100000) - rt.State().LocalSize - offset
			if pad <= 0 {
				t.Fatal(rt.State())
			}
			held, e = rt.Allocate(uint64(pad))
			serviceMust(t, e)
			serviceMust(t, rt.SetMaxSize(1))
			armed = true
			baseline := z.Budget().Used() - integerActualOwnedBytes(z) - p.rootUpperBoundHistoryBytes
			body := p.RootUpperBound(v, &out)
			armed = false
			if lane == (umemory.Block{}) {
				return
			}
			if body == nil {
				t.Fatal("failure callback did not stop source body")
			}
			// Real public allocations followed by NUM DT trigger the signed TLS free guard.
			if rt.Stats().LiveBlocks >= transportBlocks {
				return
			}
			reached = true
			if !errors.Is(body, arena.ErrBounds) {
				t.Fatal("Cauchy bound discarded actual reached NUM destructor cause", body)
			}
			q := &p.rootUpperBound
			if q.sourceCause == nil || q.bodyCause == nil || q.constructed != 5 || q.sourceRemaining != 5 {
				t.Fatal("actual c DT source frontier", q.constructed, q.sourceRemaining, q.sourceCause)
			}
			extra := []*unumeral.Int{&out}
			for i := range q.values {
				extra = append(extra, &q.values[i])
			}
			for _, r := range []*unumeral.IntegerRawInvocationReceipt{&q.divisionRaw, &q.extendedRaw} {
				if r.Frame != nil {
					for i := uint8(0); i < r.Frame.Count; i++ {
						extra = append(extra, &r.Frame.Values[i])
					}
				}
			}
			algebraicCellSourcePhysical(t, c, []UPolynomial{poly}, extra, []umemory.Block{lane, held})
			if want := baseline + integerActualOwnedBytes(z) + p.rootUpperBoundHistoryBytes; want != z.Budget().Used() {
				t.Fatal("before-lift exact cold", want, z.Budget().Used())
			}
			before := *q
			used := z.Budget().Used()
			physical := rt.Stats()
			if e := p.SourceFinalize(); e != q.sourceCause {
				t.Fatal("older UPM source crossed c DT", e)
			}
			if e := q.finish(nil); !errors.Is(e, q.sourceCause) || q.sourceRemaining != before.sourceRemaining || used != z.Budget().Used() || physical != rt.Stats() {
				t.Fatal("source retry replayed stopped c DT", e)
			}
			polyBefore := poly
			if e := p.DelPolynomial(&poly); e != ErrUPolynomialCleanupPending || poly != polyBefore || used != z.Budget().Used() || physical != rt.Stats() {
				t.Fatal("older actual coefficient DT crossed source stop", e)
			}
			h := *q.history
			serviceMust(t, p.cleanupRootUpperBound(false))
			serviceMust(t, z.Del(&out))
			firstOwn := p.cleanupRootUpperBound(true)
			if !errors.Is(firstOwn, arena.ErrBounds) || q.ownCause != firstOwn {
				t.Fatal("exact unjoined leaf first-own stop", firstOwn)
			}
			contextTLSFinish(t, rt, func() error { return p.cleanupRootUpperBound(true) })
			if q.history.bodyCause != h.bodyCause || q.history.sourceCause != h.sourceCause || q.history.ownCause != firstOwn || q.history.sourceRemaining != h.sourceRemaining || !errors.Is(body, h.sourceCause) {
				t.Fatal("own cleanup changed first source/body/cursor")
			}

		})
	}
	if !reached {
		t.Fatal("actual body and source DT frontier not reached")
	}
}
