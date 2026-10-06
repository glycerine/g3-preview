// Source: src/math/polynomial/upolynomial.cpp:1906-1951;
// src/util/memory_manager.cpp:274-285. Qualified returning-transport guard fixture.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
	"testing"
)

func TestUPolynomialRootUpperBoundActualCallbackOwnerAndTupleRefusal(t *testing.T) {
	for _, mode := range []string{"source", "own"} {
		t.Run(mode, func(t *testing.T) {
			var c *AlgebraicRootCore
			var p *UPolynomialManager
			armed := false
			calls := 0
			transport := func(f memory.Failure) error {
				if !armed {
					return f
				}
				calls++
				q := &p.rootUpperBound
				z := c.config.Rationals.z
				before := *q
				used := z.budget.Used()
				physical := z.memory.Stats()
				context := q.context()
				r := p.rootUpperBoundReceipt()
				copiedFrame := *q
				bad := context
				bad.rootUpperBound = &copiedFrame
				if bad.roots(z) != nil {
					t.Error("copied frame acquired roots")
				}
				bad = context
				bad.rootUpperBoundInvocation++
				if bad.roots(z) != nil {
					t.Error("stale nonce acquired roots")
				}
				copiedOwner := *p
				copiedOwner.self = &copiedOwner
				copiedOwner.rootUpperBound.owner = &copiedOwner
				bad = context
				bad.rootUpperBound = &copiedOwner.rootUpperBound
				if bad.roots(z) != nil {
					t.Error("copied owner acquired roots")
				}
				fake := r
				fake.frame = &copiedFrame
				if e := p.cleanupRootUpperBoundWithReceipt(&fake, true); e != nil {
					t.Error("fabricated tuple", e)
				}
				for _, abort := range []bool{false, true} {
					if e := p.cleanupRootUpperBoundWithReceipt(&r, abort); e != ErrUPolynomialCleanupPending {
						t.Error("executing own", e)
					}
				}
				if e := q.finish(nil); e != ErrUPolynomialCleanupPending {
					t.Error("executing source", e)
				}
				if e := p.SourceFinalize(); e != ErrUPolynomialCleanupPending {
					t.Error("UP source", e)
				}
				if e := p.Close(); e != ErrUPolynomialCleanupPending {
					t.Error("UP own", e)
				}
				if e := z.SourceFinalize(); e != ErrPolynomialPending {
					t.Error("Integer source", e)
				}
				if e := z.CloseChecked(); e != ErrPolynomialPending {
					t.Error("Integer own", e)
				}
				if e := c.config.Rationals.SourceFinalize(); e != ErrPolynomialPending {
					t.Error("Rational source", e)
				}
				if e := c.config.Rationals.CloseChecked(); e != ErrPolynomialPending {
					t.Error("Rational own", e)
				}
				n := z.natural
				stage, closing, finalizing, closed := n.sourceStage, n.closing, n.sourceFinalizing, n.closed
				var e error
				if mode == "source" {
					e = n.SourceFinalize()
				} else {
					e = n.CloseChecked()
				}
				if e != arena.ErrInvalidSlot || n.sourceStage != stage || n.closing != closing || n.sourceFinalizing != finalizing || n.closed != closed {
					t.Errorf("Natural %s crossed actual executing caller: %v %v %v %v %v", mode, e, n.sourceStage, n.closing, n.sourceFinalizing, n.closed)
				}
				if !reflect.DeepEqual(before, *q) || used != z.budget.Used() || physical != z.memory.Stats() {
					t.Error("refused callback changed exact charged frame/physical state")
				}
				return f
			}
			var rt *memory.Runtime
			c, p, rt = algebraicNegTLSFixture(t, transport)
			z := c.config.Rationals.z
			poly, e := p.NewPolynomial()
			serviceMust(t, e)
			var big Int
			serviceMust(t, z.SetLimbs(&big, []uint32{1, 2, 3, 4}))
			serviceMust(t, p.SetInts(poly, []Int{big, NewInt(1)}))
			serviceMust(t, z.Del(&big))
			v, e := p.wholeView(poly)
			serviceMust(t, e)
			var out Int
			held, e := rt.Allocate(uint64(int64(100000) - rt.State().LocalSize - 12))
			serviceMust(t, e)
			serviceMust(t, rt.SetMaxSize(1))
			armed = true
			body := p.RootUpperBound(v, &out)
			armed = false
			if body == nil || calls == 0 {
				t.Fatal("actual returning transport not reached", body, calls)
			}
			rootUpperActualGraph(t, c, p, poly, &out, []memory.Block{held})
			serviceMust(t, rt.SetMaxSize(0))
			serviceMust(t, p.cleanupRootUpperBound(false))
			serviceMust(t, z.Del(&out))
			serviceMust(t, p.cleanupRootUpperBound(true))
			serviceMust(t, rt.Deallocate(&held))
			serviceMust(t, p.DelPolynomial(&poly))
		})
	}
}
