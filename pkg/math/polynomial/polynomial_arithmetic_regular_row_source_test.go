// Source: src/math/polynomial/polynomial.cpp:1988-2039.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialRegularActualINCBindsRowBeforeNumericFailure(t *testing.T) {
	for prefix := int64(0); prefix < 32; prefix++ {
		full := polynomialArithmeticDirectNumericFixture(t)
		c := full.core
		var seven util.Int
		words := make([]uint32, 2048)
		words[len(words)-1] = 1
		serviceMust(t, c.integer.SetLimbs(&seven, words))
		p, e := c.MkConstantTakeCopy(&seven)
		serviceMust(t, e)
		serviceMust(t, c.IncRef(p))
		var r polynomialArithmeticInvocationReceipt
		serviceMust(t, c.som.acquireArithmeticInvocation(&r))
		before, e := c.monomialRefCount(c.monomials.unit)
		serviceMust(t, e)
		one := util.NewInt(1)
		c.integer.Budget().FailAfter(prefix)
		e = c.som.addSourceInputWithArithmeticReceipt(&one, c.monomials.unit, polynomialSOMInput{polynomial: p}, false, &r)
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, []*util.Int{&seven}, 0)
		ms, x := c.som.ms.size(c)
		serviceMust(t, x)
		as, x := c.som.as.size(c)
		serviceMust(t, x)
		after, x := c.monomialRefCount(c.monomials.unit)
		serviceMust(t, x)

		if e != nil && ms == 1 && as == 1 && after == before+1 {
			if !c.som.matchesAppendInvocation(&r.frame.append) || len(c.som.monomialOwners) == 0 || c.som.monomialOwners[0] != r.frame.append.invocation || r.frame.append.frame.monomialClaims != 1 {
				t.Fatal("reached source INC lacks exact per-slot authority", prefix, e)
			}
			c.integer.Budget().FailAfter(-1)
			serviceMust(t, c.cleanupArithmeticInvocation(&r, true))
			afterOwn, x := c.monomialRefCount(c.monomials.unit)
			serviceMust(t, x)
			position, x := c.som.positions.Get(c.monomials.unit)
			serviceMust(t, x)
			if afterOwn != before || position != ^uint32(0) {
				t.Fatal("matched regular row abort retained source position/INC", afterOwn, position)
			}
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, []*util.Int{&seven}, 0)
			serviceMust(t, c.som.addMul(&one, c.monomials.unit, p))
			serviceMust(t, c.som.close())
			serviceMust(t, c.DecRef(p))
			serviceMust(t, c.integer.Del(&seven))
			return
		}
		c.integer.Budget().FailAfter(-1)
		serviceMust(t, c.som.close())
		serviceMust(t, c.DecRef(p))
		serviceMust(t, c.integer.Del(&seven))
	}
	t.Fatal("did not reach actual regular ms-push/INC before actual numeric Copy/Mul failure")
}
