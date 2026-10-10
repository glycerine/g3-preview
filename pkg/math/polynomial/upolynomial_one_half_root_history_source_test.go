// Source: src/math/polynomial/upolynomial.cpp:1298-1321.
package polynomial

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"math"
	"reflect"
	"testing"
)

func TestUPolynomialHasOneHalfRootOldAcknowledgmentPreservesLaterActualRoots(t *testing.T) {
	c, p := algebraicFixture(t)
	z := c.config.Rationals.Integers()
	var foreign unumeral.Int
	foreign = unumeral.NewInt(17)
	savedForeign := foreign
	defer func() { serviceMust(t, z.Del(&foreign)) }()
	var old upolyOneHalfRootInvocationReceipt
	foundOld, foundNew := false, false
	for prefix := int64(0); prefix < 64 && (!foundOld || !foundNew); prefix++ {
		poly, e := p.NewPolynomial()
		serviceMust(t, e)
		if foundOld {
			// A later MIN call can reuse the now-warm checked scratch without an
			// allocation. A larger actual leading SET reaches a fresh source leaf.
			var leading unumeral.Int
			limbs := make([]uint32, 40)
			limbs[0], limbs[len(limbs)-1] = 17, 1
			serviceMust(t, z.SetLimbs(&leading, limbs))
			serviceMust(t, p.SetInts(poly, []unumeral.Int{unumeral.NewInt(1), unumeral.NewInt(3), leading}))
			serviceMust(t, z.Del(&leading))
		} else {
			serviceMust(t, p.SetInts(poly, []unumeral.Int{unumeral.NewInt(1), unumeral.NewInt(math.MinInt32), unumeral.NewInt(1)}))
		}
		v, e := p.wholeView(poly)
		serviceMust(t, e)
		if foundOld {
			// The later assertion is against one actual allocating successful
			// invocation, not repeated failed scratch lifetimes in this manager.
			z.Budget().FailAfter(-1)
		} else {
			z.Budget().FailAfter(prefix)
		}
		_, body := p.HasOneHalfRoot(v)
		q := &p.oneHalfRoot
		r := p.oneHalfRootReceipt()
		algebraicCellSourcePhysical(t, c, []UPolynomial{poly}, []*unumeral.Int{&foreign}, nil)
		if foreign != savedForeign {
			t.Fatal("unrelated actual BIG numeral changed")
		}
		retained := body != nil && q.history != nil && polynomialNegNumericPending(&q.numeric)
		if foundOld && q.history != nil && (retained || body == nil) {
			before := *q
			used := z.Budget().Used()
			physical := z.RuntimeOwner().Stats()
			history := *old.history
			serviceMust(t, p.cleanupOneHalfRootWithReceipt(&old, false))
			serviceMust(t, p.cleanupOneHalfRootWithReceipt(&old, true))
			if !reflect.DeepEqual(before, *q) || used != z.Budget().Used() || physical != z.RuntimeOwner().Stats() || *old.history != history || foreign != savedForeign {
				t.Fatal("old ACK consumed later/foreign actual roots or history")
			}
			algebraicCellSourcePhysical(t, c, []UPolynomial{poly}, []*unumeral.Int{&foreign}, nil)
			foundNew = true
		}
		z.Budget().FailAfter(-1)
		serviceMust(t, p.cleanupOneHalfRootWithReceipt(&r, false))
		serviceMust(t, p.DelPolynomial(&poly))
		serviceMust(t, p.cleanupOneHalfRootWithReceipt(&r, true))
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
