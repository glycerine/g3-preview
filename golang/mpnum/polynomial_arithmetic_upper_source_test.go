// Source: src/math/polynomial/polynomial.cpp:2665-2744.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import "testing"

func TestPolynomialRegularAddMulActualFactoryReturnEveryPrefix(t *testing.T) {
	for prefix := int64(0); prefix < 80; prefix++ {
		full := polynomialArithmeticDirectNumericFixture(t)
		c := full.core
		seven, five := NewInt(7), NewInt(5)
		p, e := c.MkConstantTakeCopy(&seven)
		serviceMust(t, e)
		serviceMust(t, c.IncRef(p))
		q, e := c.MkConstantTakeCopy(&five)
		serviceMust(t, e)
		serviceMust(t, c.IncRef(q))
		one := NewInt(1)
		var r polynomialArithmeticInvocationReceipt
		c.integer.budget.FailAfter(prefix)
		out, e := c.addMulWithReceipt(&one, c.monomials.unit, p, &one, c.monomials.unit, q, &r)
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, out}, nil, 0)
		if e != nil && !numericPhysicalReservationRefusal(e) {
			t.Fatal("source failure", prefix, e)
		}
		serviceMust(t, c.cleanupArithmeticInvocation(&r, false))
		c.integer.budget.FailAfter(-1)
		if e != nil {
			serviceMust(t, c.cleanupArithmeticInvocation(&r, true))
		} else {
			if r.frame.active || r.frame.factory.frame.result != (Polynomial{}) {
				t.Fatal("direct return retained child authority")
			}
			serviceMust(t, c.cleanupArithmeticInvocation(&r, true))
			n, x := c.readWord(out, 8)
			serviceMust(t, x)
			a, x := c.coefficientCore(out, 0)
			serviceMust(t, x)
			if n != 1 || a != NewInt(12) {
				t.Fatal("independent 7+5", n, a)
			}
			serviceMust(t, c.IncRef(out))
			serviceMust(t, c.DecRef(out))
		}
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q}, nil, 0)
		serviceMust(t, c.DecRef(p))
		serviceMust(t, c.DecRef(q))
		if e == nil {
			return
		}
	}
	t.Fatal("actual source horizon did not reach success")
}

func TestPolynomialScaleEarlyAliasesDoNotAcquireOlderRegularFrame(t *testing.T) {
	full := polynomialArithmeticDirectNumericFixture(t)
	c := full.core
	seven := NewInt(7)
	p, e := c.MkConstantTakeCopy(&seven)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	var old polynomialArithmeticInvocationReceipt
	serviceMust(t, c.som.acquireArithmeticInvocation(&old))
	serviceMust(t, c.som.addPolynomialWithArithmeticReceipt(p, &old))
	zero, one := NewInt(0), NewInt(1)
	var r polynomialArithmeticInvocationReceipt
	out, e := c.scaleWithReceipt(&zero, Monomial{}, Polynomial{}, &r)
	serviceMust(t, e)
	if out != c.zero || r.invocation != 0 {
		t.Fatal("zero branch acquired/read erased source operands")
	}
	out, e = c.scaleWithReceipt(&one, c.monomials.unit, p, &r)
	serviceMust(t, e)
	if out != p || r.invocation != 0 || !c.matchesArithmeticInvocation(&old) {
		t.Fatal("alias branch acquired older frame")
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, nil, 0)
	serviceMust(t, c.cleanupArithmeticInvocation(&old, true))
	serviceMust(t, c.DecRef(p))
}

func TestPolynomialArithmeticSourceWrappersEveryActualPrefix(t *testing.T) {
	for op := 0; op < 5; op++ {
		success := false
		for prefix := int64(0); prefix < 200; prefix++ {
			full := polynomialArithmeticDirectNumericFixture(t)
			c := full.core
			seven, five := NewInt(7), NewInt(5)
			p, e := c.MkConstantTakeCopy(&seven)
			serviceMust(t, e)
			serviceMust(t, c.IncRef(p))
			q, e := c.MkConstantTakeCopy(&five)
			serviceMust(t, e)
			serviceMust(t, c.IncRef(q))
			var r polynomialArithmeticInvocationReceipt
			var out Polynomial
			want := int32(0)
			c.integer.budget.FailAfter(prefix)
			switch op {
			case 0:
				out, e = c.addWithReceipt(p, q, &r)
				want = 12
			case 1:
				out, e = c.subWithReceipt(p, q, &r)
				want = 2
			case 2:
				out, e = c.mulWithReceipt(p, q, &r)
				want = 35
			case 3:
				three := NewInt(3)
				out, e = c.scaleWithReceipt(&three, c.monomials.unit, p, &r)
				want = 21
			case 4:
				two := NewInt(2)
				out, e = c.mulAddWithReceipt(p, q, &two, &r)
				want = 37
			}
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, out}, nil, 0)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual wrapper source error", op, prefix, e)
			}
			if x := c.cleanupArithmeticInvocation(&r, false); x != nil {
				t.Fatal("false phase", op, prefix, e, x)
			}
			c.integer.budget.FailAfter(-1)
			if x := c.cleanupArithmeticInvocation(&r, true); x != nil {
				t.Fatal("true phase", op, prefix, e, x)
			}
			if e == nil {
				coefficient, x := c.coefficientCore(out, 0)
				serviceMust(t, x)
				if coefficient != NewInt(want) {
					t.Fatal("independent scalar coefficient", op, prefix, coefficient, want)
				}
				if c.matchesArithmeticInvocation(&r) {
					t.Fatal("direct returned token retained owning frame", op, prefix)
				}
				serviceMust(t, c.CleanupValue(&out))
				success = true
			}
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q}, nil, 0)
			serviceMust(t, c.DecRef(p))
			serviceMust(t, c.DecRef(q))
			if success {
				t.Log("actual first success", op, prefix)
				break
			}
		}
		if !success {
			t.Fatal("actual source wrapper did not reach success", op)
		}
	}
}
