// Source: src/math/polynomial/upolynomial.cpp:1324-1329,432-513;
// src/util/mpz.h:393-426. Qualified real unjoined RAW SubMul own TLS release.
package mpnum

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestUPolynomialRemoveOneHalfRootActualUnjoinedOwnStop(t *testing.T) {
	reached := false
	for prefix := int64(0); prefix < 100 && !reached; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, p, rt := algebraicNegTLSFixture(t, nil)
			z := p.config.Integer
			var b, fourB, neg Int
			serviceMust(t, z.SetLimbs(&b, []uint32{1, 2, 3, 4}))
			four := NewInt(4)
			serviceMust(t, z.Mul(&b, &four, &fourB))
			serviceMust(t, z.Copy(&neg, &b))
			serviceMust(t, z.Neg(&neg))
			input, e := p.NewPolynomial()
			serviceMust(t, e)
			serviceMust(t, p.SetInts(input, []Int{neg, NewInt(0), fourB}))
			out, e := p.NewPolynomial()
			serviceMust(t, e)
			serviceMust(t, z.Del(&b))
			serviceMust(t, z.Del(&fourB))
			serviceMust(t, z.Del(&neg))
			v, e := p.wholeView(input)
			serviceMust(t, e)
			z.budget.FailAfter(prefix)
			body := p.RemoveOneHalfRoot(v, out)
			q := &p.removeOneHalfRoot
			r := p.removeOneHalfRootReceipt()
			extra := []*Int{&q.values[0], &p.oneHalfRoot.values[0], &p.oneHalfRoot.values[1]}
			if z.matchesAddMulInvocation(&q.addMul) {
				extra = append(extra, &q.addMul.frame.tmp)
			}
			algebraicCellSourcePhysical(t, c, []UPolynomial{input, out}, extra, nil)
			reached = body != nil && z.matchesAddMulInvocation(&q.addMul) && q.addMul.frame.stage < 3 && q.addMul.frame.tmp.cell != 0 && !z.isFailureRef(q.addMul.frame.tmp.cell)
			z.budget.FailAfter(-1)
			serviceMust(t, p.cleanupRemoveOneHalfRootWithReceipt(&r, false))
			serviceMust(t, p.DelPolynomial(&input))
			serviceMust(t, p.DelPolynomial(&out))
			if reached {
				oldBody, oldSource, child := q.bodyCause, q.sourceCause, q.addMul
				contextTLSNearMinimum(t, rt)
				lane, e := rt.Allocate(8)
				serviceMust(t, e)
				delta := int64(-1000000)
				if e = rt.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
					t.Fatal(e)
				}
				first := p.cleanupRemoveOneHalfRootWithReceipt(&r, true)
				if !errors.Is(first, arena.ErrBounds) || q.ownCause != first || q.sourceCause != oldSource || q.bodyCause != oldBody || q.addMul != child {
					t.Fatal("actual first-own stop/history", first, q.ownCause)
				}
				algebraicCellSourcePhysical(t, c, nil, []*Int{&q.addMul.frame.tmp}, []memory.Block{lane})
				contextTLSFinish(t, rt, func() error { return p.cleanupRemoveOneHalfRootWithReceipt(&r, true) })
				contextTLSFinish(t, rt, func() error {
					if lane == (memory.Block{}) {
						return nil
					}
					return rt.Deallocate(&lane)
				})
				if q.ownCause != first || q.active || q.sourceCause != oldSource || q.bodyCause != oldBody || q.addMul != child || !errors.Is(body, oldBody) {
					t.Fatal("own retry changed returned source tuple")
				}
			} else {
				serviceMust(t, p.cleanupRemoveOneHalfRootWithReceipt(&r, true))
			}
		})
	}
	if !reached {
		t.Fatal("real BIG RAW SubMul retained path not reached")
	}
}
