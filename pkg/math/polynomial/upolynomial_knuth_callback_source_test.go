package polynomial

// Source: src/math/polynomial/upolynomial.cpp:1594-1602,2029-2074.
// Qualified actual returning-transport callback guard.

import (
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"math"
	"reflect"
	"testing"
)

func TestUPolynomialKnuthMinimumActualOwnerCallback(t *testing.T) {
	var p *UPolynomialManager
	armed, reached := false, false
	var oldContext integerAllocationInvocationContext
	transport := func(f umemory.Failure) error {
		if !armed {
			return f
		}
		armed = false
		reached = true
		q := &p.knuth
		if oldContext.knuth != nil && (oldContext.roots(p.config.Integer)) != nil {
			t.Error("actual old context retargeted later live call")
		}
		beforeQ := *q
		used := p.config.Integer.Budget().Used()
		physical := p.config.Integer.RuntimeOwner().Stats()
		r := p.knuthReceipt()
		for _, abort := range []bool{false, true} {
			if e := p.cleanupKnuthWithReceipt(&r, abort); e != ErrUPolynomialCleanupPending {
				t.Error("executing own", e)
			}
		}
		if e := q.finish(nil); e != ErrUPolynomialCleanupPending {
			t.Error("executing source", e)
		}
		bad := q.context()
		copied := *q
		bad.knuth = &copied
		if (bad.roots(p.config.Integer)) != nil {
			t.Error("copied frame acquired roots")
		}
		bad = q.context()
		bad.knuthInvocation++
		if (bad.roots(p.config.Integer)) != nil {
			t.Error("stale invocation acquired roots")
		}
		if e := p.pMinusXWithSourceContext(q.target, bad); e != ErrUPolynomialOwner {
			t.Error("stale passed transform context", e)
		}
		for _, end := range []func() error{p.Close, p.config.Integer.SourceFinalize, p.config.Integer.CloseChecked, p.config.Integer.Naturals().SourceFinalize, p.config.Integer.Naturals().CloseChecked} {
			if e := end(); e == nil {
				t.Error("executing owner accepted retirement")
			}
		}
		if !reflect.DeepEqual(*q, beforeQ) || used != p.config.Integer.Budget().Used() || physical != p.config.Integer.RuntimeOwner().Stats() {
			t.Error("refused exact calls changed graph/history")
		}
		before, stage := p.sourceFinalizing, p.sourceStage
		e := p.SourceFinalize()
		if e != ErrUPolynomialCleanupPending || p.sourceFinalizing != before || p.sourceStage != stage {
			t.Errorf("actual MIN Neg callback retired caller before return: %v flags %v/%v stage %v/%v", e, before, p.sourceFinalizing, stage, p.sourceStage)
		}
		return f
	}
	c, actual, rt := algebraicNegTLSFixture(t, transport)
	p = actual
	z := c.config.Rationals.Integers()
	poly := algebraicPolynomial(t, p, 1, math.MinInt32, 1)
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	held, e := rt.Allocate(uint64(int64(100000) - rt.State().LocalSize - 12))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	armed = true
	_, body := p.KnuthNegativeRootUpperBound(v)
	armed = false
	serviceMust(t, rt.SetMaxSize(0))
	if !reached || body == nil {
		t.Fatal("actual MIN allocation callback missing", reached, body)
	}
	algebraicCellSourcePhysical(t, c, []UPolynomial{poly}, nil, []umemory.Block{held})
	serviceMust(t, p.cleanupKnuth(false))
	serviceMust(t, p.cleanupKnuth(true))
	serviceMust(t, rt.Deallocate(&held))
	oldContext = p.knuth.context()
	later := algebraicPolynomial(t, p, 1, math.MinInt32, 1)
	lv, e := p.wholeView(later)
	serviceMust(t, e)
	held, e = rt.Allocate(uint64(int64(100000) - rt.State().LocalSize - 12))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	armed = true
	_, body = p.KnuthNegativeRootUpperBound(lv)
	armed = false
	serviceMust(t, rt.SetMaxSize(0))
	if body == nil {
		t.Fatal("later actual transport failure missing")
	}
	algebraicCellSourcePhysical(t, c, []UPolynomial{poly, later}, nil, []umemory.Block{held})
	serviceMust(t, p.cleanupKnuth(false))
	serviceMust(t, p.cleanupKnuth(true))
	serviceMust(t, rt.Deallocate(&held))

	_ = z
}
