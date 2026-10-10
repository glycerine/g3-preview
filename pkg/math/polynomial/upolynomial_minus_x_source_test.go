// Source: src/math/polynomial/upolynomial.cpp:1594-1602.
// Actual public returning-allocation callback and destructive-prefix fixtures.
package polynomial

import (
	"fmt"
	"math"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestUPolynomialPMinusXActualMinimumCallbackProtectsCaller(t *testing.T) {
	var p *UPolynomialManager
	armed, reached := false, false
	var oldContext integerAllocationInvocationContext
	var foreignView UPolynomialView
	var current *UPolynomial
	transport := func(f umemory.Failure) error {
		if !armed {
			return f
		}
		armed = false
		reached = true
		q := &p.knuth
		if q.kind != 3 {
			t.Fatal("public transform did not bind actual kind3 caller", q.kind)
		}
		if oldContext.knuth != nil && (oldContext.roots(p.config.Integer)) != nil {
			t.Error("actual old context retargeted later live call")
		}
		used := p.config.Integer.Budget().Used()
		physical := p.config.Integer.RuntimeOwner().Stats()
		beforeQ := *q
		if e := p.DelPolynomial(current); e != ErrUPolynomialCleanupPending {
			t.Error("executing borrowed input DT", e)
		}
		if e := p.pMinusXWithSourceContext(foreignView, q.context()); e != ErrUPolynomialOwner {
			t.Error("captured context accepted foreign borrowed input", e)
		}
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
		for _, end := range []func() error{p.modular.Close, p.Close, p.config.Integer.SourceFinalize, p.config.Integer.CloseChecked, p.config.Integer.Naturals().SourceFinalize, p.config.Integer.Naturals().CloseChecked} {
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
	foreign := algebraicPolynomial(t, p, 1, 7, 1)
	foreignView, e := p.wholeView(foreign)
	serviceMust(t, e)
	poly := algebraicPolynomial(t, p, 1, math.MinInt32, 1)
	current = &poly
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	held, e := rt.Allocate(uint64(int64(100000) - rt.State().LocalSize - 12))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	armed = true
	body := p.PMinusXViewInPlace(v)
	armed = false
	if !reached || body == nil {
		t.Fatal("actual MIN allocation callback missing", reached, body)
	}
	algebraicCellSourcePhysical(t, c, []UPolynomial{poly, foreign}, nil, []umemory.Block{held})
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, p.cleanupKnuth(false))
	serviceMust(t, p.cleanupKnuth(true))
	serviceMust(t, rt.Deallocate(&held))
	oldContext = p.knuth.context()
	later := algebraicPolynomial(t, p, 1, math.MinInt32, 1)
	current = &later
	lv, e := p.wholeView(later)
	serviceMust(t, e)
	held, e = rt.Allocate(uint64(int64(100000) - rt.State().LocalSize - 12))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	armed = true
	body = p.PMinusXViewInPlace(lv)
	armed = false
	if body == nil {
		t.Fatal("later actual transport failure missing")
	}
	algebraicCellSourcePhysical(t, c, []UPolynomial{poly, later, foreign}, nil, []umemory.Block{held})
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, p.cleanupKnuth(false))
	serviceMust(t, p.cleanupKnuth(true))
	serviceMust(t, rt.Deallocate(&held))
}

func TestUPolynomialPMinusXActualMinimumEveryPrefixAndInlineReuse(t *testing.T) {
	acquired, success := false, false
	for prefix := int64(0); prefix < 64 && !success; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, p := algebraicFixture(t)
			z := c.config.Rationals.Integers()
			poly, e := p.NewPolynomial()
			serviceMust(t, e)
			serviceMust(t, p.SetInts(poly, []util.Int{util.NewInt(1), util.NewInt(7), util.NewInt(2), util.NewInt(math.MinInt32), util.NewInt(1)}))
			v, e := p.wholeView(poly)
			serviceMust(t, e)
			baseline := z.Budget().Used() - integerActualOwnedBytes(z) - p.knuthHistoryBytes
			z.Budget().FailAfter(prefix)
			body := p.PMinusXViewInPlace(v)
			q := &p.knuth
			r := p.knuthReceipt()
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal(body)
			}
			coefficient, e := p.read(v, 1)
			serviceMust(t, e)
			if *coefficient != util.NewInt(-7) {
				t.Fatal("earlier source odd NEG was restored/skipped", *coefficient)
			}
			algebraicCellSourcePhysical(t, c, []UPolynomial{poly}, nil, nil)
			if expected := baseline + integerActualOwnedBytes(z) + p.knuthHistoryBytes; expected != z.Budget().Used() {
				t.Fatal("before-lift exact cold", expected, z.Budget().Used())
			}
			if polynomialNegNumericPending(&q.numeric) {
				acquired = true
			}
			z.Budget().FailAfter(-1)
			serviceMust(t, p.cleanupKnuthWithReceipt(&r, false))
			serviceMust(t, p.DelPolynomial(&poly))
			serviceMust(t, p.cleanupKnuthWithReceipt(&r, true))
			small := algebraicPolynomial(t, p, 1, 7, 2)
			sv, e := p.wholeView(small)
			serviceMust(t, e)
			empty := algebraicPolynomial(t, p)
			constant := algebraicPolynomial(t, p, 9)
			ev, e := p.wholeView(empty)
			serviceMust(t, e)
			cv, e := p.wholeView(constant)
			serviceMust(t, e)
			var oldHistory upolyKnuthHistory
			if r.history != nil {
				oldHistory = *r.history
			}
			beforeUsed := z.Budget().Used()
			beforePhysical := z.RuntimeOwner().Stats()
			z.Budget().FailAfter(0)
			serviceMust(t, p.PMinusXViewInPlace(ev))
			serviceMust(t, p.PMinusXViewInPlace(cv))
			e = p.PMinusXViewInPlace(sv)
			serviceMust(t, e)
			e = p.PMinusXViewInPlace(sv)
			z.Budget().FailAfter(-1)
			serviceMust(t, e)
			if beforeUsed != z.Budget().Used() || beforePhysical != z.RuntimeOwner().Stats() {
				t.Fatal("inline repeated NEG allocated history/payload")
			}
			term, e := p.read(sv, 1)
			serviceMust(t, e)
			if *term != util.NewInt(7) {
				t.Fatal("inline double transform did not restore literal value", *term)
			}
			before := p.knuth
			serviceMust(t, p.cleanupKnuthWithReceipt(&r, false))
			serviceMust(t, p.cleanupKnuthWithReceipt(&r, true))
			if !reflect.DeepEqual(before, p.knuth) || (r.history != nil && *r.history != oldHistory) {
				t.Fatal("old tuple changed later inline frame/history")
			}
			serviceMust(t, p.CleanupKnuthBounds())
			success = body == nil
		})
	}
	if !acquired || !success {
		t.Fatal("actual leaf/success prefix missing", acquired, success)
	}
}

func TestUPolynomialPMinusXOldAcknowledgmentPreservesLaterActualRoots(t *testing.T) {
	c, p := algebraicFixture(t)
	z := c.config.Rationals.Integers()
	var foreign util.Int
	foreign = util.NewInt(17)
	savedForeign := foreign
	defer func() { serviceMust(t, z.Del(&foreign)) }()
	var old upolyKnuthInvocationReceipt
	foundOld, foundNew := false, false
	for prefix := int64(0); prefix < 64 && (!foundOld || !foundNew); prefix++ {
		poly, e := p.NewPolynomial()
		serviceMust(t, e)
		serviceMust(t, p.SetInts(poly, []util.Int{util.NewInt(1), util.NewInt(math.MinInt32), util.NewInt(1)}))
		v, e := p.wholeView(poly)
		serviceMust(t, e)
		z.Budget().FailAfter(prefix)
		body := p.PMinusXViewInPlace(v)
		q := &p.knuth
		r := p.knuthReceipt()
		algebraicCellSourcePhysical(t, c, []UPolynomial{poly}, []*util.Int{&foreign}, nil)
		if foreign != savedForeign {
			t.Fatal("unrelated actual BIG numeral changed")
		}
		retained := body != nil && q.history != nil && polynomialNegNumericPending(&q.numeric)
		if foundOld && q.history != nil && (retained || body == nil) {
			before := *q
			used := z.Budget().Used()
			physical := z.RuntimeOwner().Stats()
			history := *old.history
			serviceMust(t, p.cleanupKnuthWithReceipt(&old, false))
			serviceMust(t, p.cleanupKnuthWithReceipt(&old, true))
			if !reflect.DeepEqual(before, *q) || used != z.Budget().Used() || physical != z.RuntimeOwner().Stats() || *old.history != history || foreign != savedForeign {
				t.Fatal("old ACK consumed later/foreign actual roots or history")
			}
			algebraicCellSourcePhysical(t, c, []UPolynomial{poly}, []*util.Int{&foreign}, nil)
			foundNew = true
		}
		z.Budget().FailAfter(-1)
		serviceMust(t, p.cleanupKnuthWithReceipt(&r, false))
		serviceMust(t, p.DelPolynomial(&poly))
		serviceMust(t, p.cleanupKnuthWithReceipt(&r, true))
		if retained && !foundOld {
			old = r
			foundOld = true
			serviceMust(t, z.SetLimbs(&foreign, []uint32{17, 23, 31, 47}))
			savedForeign = foreign
			prefix = -1
		}
		if foreign != savedForeign {
			t.Fatal("own cleanup crossed foreign actual numeral")
		}
	}
	if !foundOld || !foundNew {
		t.Fatal("actual stopped leaf then later allocating published/retained invocation not reached", foundOld, foundNew)
	}
}
