// Source: src/math/polynomial/polynomial.cpp:2692-2698,2530-2536;
// src/util/rational.h:54; src/util/mpq.h:186-189.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialMulAddConstantShortcutActualAcquisitionsPreserveOlderRegular(t *testing.T) {
	saw := false
	for prefix := int64(0); prefix < 80; prefix++ {
		full := polynomialArithmeticDirectNumericFixture(t)
		c := full.core
		seven := util.NewInt(7)
		p, e := c.MkConstantTakeCopy(&seven)
		serviceMust(t, e)
		serviceMust(t, c.IncRef(p))
		var older polynomialArithmeticInvocationReceipt
		serviceMust(t, c.som.acquireArithmeticInvocation(&older))
		serviceMust(t, c.som.addPolynomialWithArithmeticReceipt(p, &older))
		var big util.Int
		limbs := make([]uint32, 80)
		limbs[0], limbs[79] = 19, 1
		serviceMust(t, c.integer.SetLimbs(&big, limbs))
		var r polynomialArithmeticInvocationReceipt
		c.integer.Budget().FailAfter(prefix)
		out, e := c.mulAddWithReceipt(c.zero, Polynomial{}, &big, &r)
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, out}, []*util.Int{&big}, 0)
		if e != nil && !numericPhysicalReservationRefusal(e) {
			t.Fatal("actual shortcut failure", prefix, e)
		}
		if r.frame != &c.constantScope.arithmeticScope || !c.matchesArithmeticInvocation(&older) {
			t.Fatal("fake SOM/older frame acquisition", prefix)
		}
		if c.integer.PendingAllocationObject() != (util.SourceCellBacking{}).Object && r.frame.numeric.Cell == c.integer.PendingCell {
			saw = true
		}
		serviceMust(t, c.cleanupArithmeticInvocation(&r, false))
		c.integer.Budget().FailAfter(-1)
		serviceMust(t, c.cleanupArithmeticInvocation(&r, true))
		row, x := c.som.as.coefficient(c, 0)
		serviceMust(t, x)
		n, x := c.som.ms.size(c)
		serviceMust(t, x)
		if n != 1 || row != seven || !c.matchesArithmeticInvocation(&older) {
			t.Fatal("constant abort changed older rows", prefix, n, row)
		}
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, out}, []*util.Int{&big}, 0)
		if e == nil {
			a, x := c.coefficientCore(out, 0)
			serviceMust(t, x)
			equal, x := c.integer.Equal(&a, &big)
			serviceMust(t, x)
			if !equal {
				t.Fatal("constant value")
			}
			serviceMust(t, c.IncRef(out))
			serviceMust(t, c.DecRef(out))
		}
		serviceMust(t, c.cleanupArithmeticInvocation(&older, true))
		serviceMust(t, c.DecRef(p))
		serviceMust(t, c.integer.Del(&big))
		if e == nil {
			if !saw {
				t.Fatal("no actual numeric acquisition refusal")
			}
			return
		}
	}
	t.Fatal("real constant source horizon not reached")
}
