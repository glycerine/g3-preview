// Source: src/math/polynomial/upolynomial.cpp:2029-2035,2052-2074.
package mpnum

import (
	"math"
	"reflect"
	"testing"
)

func TestUPolynomialKnuthOldAcknowledgmentPreservesLaterActualRoots(t *testing.T) {
	c, p := algebraicFixture(t)
	z := c.config.Rationals.z
	var foreign Int
	foreign = NewInt(17)
	savedForeign := foreign
	defer func() { serviceMust(t, z.Del(&foreign)) }()
	var old upolyKnuthInvocationReceipt
	foundOld, foundNew := false, false
	for prefix := int64(0); prefix < 64 && (!foundOld || !foundNew); prefix++ {
		poly, e := p.NewPolynomial()
		serviceMust(t, e)
		serviceMust(t, p.SetInts(poly, []Int{NewInt(1), NewInt(math.MinInt32), NewInt(1)}))
		v, e := p.wholeView(poly)
		serviceMust(t, e)
		z.budget.FailAfter(prefix)
		_, body := p.KnuthNegativeRootUpperBound(v)
		q := &p.knuth
		r := p.knuthReceipt()
		algebraicCellSourcePhysical(t, c, []UPolynomial{poly}, []*Int{&foreign}, nil)
		if foreign != savedForeign {
			t.Fatal("unrelated actual BIG numeral changed")
		}
		retained := body != nil && q.history != nil && polynomialNegNumericPending(&q.numeric)
		if foundOld && q.history != nil && (retained || body == nil) {
			before := *q
			used := z.budget.Used()
			physical := z.memory.Stats()
			history := *old.history
			serviceMust(t, p.cleanupKnuthWithReceipt(&old, false))
			serviceMust(t, p.cleanupKnuthWithReceipt(&old, true))
			if !reflect.DeepEqual(before, *q) || used != z.budget.Used() || physical != z.memory.Stats() || *old.history != history || foreign != savedForeign {
				t.Fatal("old ACK consumed later/foreign actual roots or history")
			}
			algebraicCellSourcePhysical(t, c, []UPolynomial{poly}, []*Int{&foreign}, nil)
			foundNew = true
		}
		z.budget.FailAfter(-1)
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
