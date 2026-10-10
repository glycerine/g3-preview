// Source: src/math/polynomial/upolynomial.cpp:1906-1951;
// src/util/memory_manager.cpp:274-285. Qualified returning-transport guard fixture.
package polynomial

import (
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestUPolynomialRootUpperBoundActualCallbackOwnerAndTupleRefusal(t *testing.T) {
	for _, mode := range []string{"source", "own"} {
		t.Run(mode, func(t *testing.T) {
			var c *AlgebraicRootCore
			var p *UPolynomialManager
			armed := false
			calls := 0
			transport := func(f umemory.Failure) error {
				if !armed {
					return f
				}
				calls++
				q := &p.rootUpperBound
				z := c.config.Rationals.Integers()
				before := *q
				used := z.Budget().Used()
				physical := z.RuntimeOwner().Stats()
				context := q.context()
				r := p.rootUpperBoundReceipt()
				copiedFrame := *q
				bad := context
				bad.rootUpperBound = &copiedFrame
				if (bad.roots(z)) != nil {
					t.Error("copied frame acquired roots")
				}
				bad = context
				bad.rootUpperBoundInvocation++
				if (bad.roots(z)) != nil {
					t.Error("stale nonce acquired roots")
				}
				copiedOwner := *p
				copiedOwner.self = &copiedOwner
				copiedOwner.rootUpperBound.owner = &copiedOwner
				bad = context
				bad.rootUpperBound = &copiedOwner.rootUpperBound
				if (bad.roots(z)) != nil {
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
				n := z.Naturals()
				stage, closing, finalizing, closed := n.SourceStage, n.Closing, n.SourceFinalizing, n.ClosedField
				var e error
				if mode == "source" {
					e = n.SourceFinalize()
				} else {
					e = n.CloseChecked()
				}
				if e != arena.ErrInvalidSlot || n.SourceStage != stage || n.Closing != closing || n.SourceFinalizing != finalizing || n.ClosedField != closed {
					t.Errorf("Natural %s crossed actual executing caller: %v %v %v %v %v", mode, e, n.SourceStage, n.Closing, n.SourceFinalizing, n.ClosedField)
				}
				if !reflect.DeepEqual(before, *q) || used != z.Budget().Used() || physical != z.RuntimeOwner().Stats() {
					t.Error("refused callback changed exact charged frame/physical state")
				}
				return f
			}
			var rt *umemory.Runtime
			c, p, rt = algebraicNegTLSFixture(t, transport)
			z := c.config.Rationals.Integers()
			poly, e := p.NewPolynomial()
			serviceMust(t, e)
			var big util.Int
			serviceMust(t, z.SetLimbs(&big, []uint32{1, 2, 3, 4}))
			serviceMust(t, p.SetInts(poly, []util.Int{big, util.NewInt(1)}))
			serviceMust(t, z.Del(&big))
			v, e := p.wholeView(poly)
			serviceMust(t, e)
			var out util.Int
			held, e := rt.Allocate(uint64(int64(100000) - rt.State().LocalSize - 12))
			serviceMust(t, e)
			serviceMust(t, rt.SetMaxSize(1))
			armed = true
			body := p.RootUpperBound(v, &out)
			armed = false
			if body == nil || calls == 0 {
				t.Fatal("actual returning transport not reached", body, calls)
			}
			rootUpperActualGraph(t, c, p, poly, &out, []umemory.Block{held})
			serviceMust(t, rt.SetMaxSize(0))
			serviceMust(t, p.cleanupRootUpperBound(false))
			serviceMust(t, z.Del(&out))
			serviceMust(t, p.cleanupRootUpperBound(true))
			serviceMust(t, rt.Deallocate(&held))
			serviceMust(t, p.DelPolynomial(&poly))
		})
	}
}
