// Source: src/math/polynomial/polynomial.cpp:5473-5500;
// src/util/mpzzp.h:58-68,131.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import "testing"

func TestPolynomialDiscriminantEvenUpperActualNegNormalizeSuffixPrefixes(t *testing.T) {
	selected, success := false, false
	for prefix := int64(0); prefix < 64; prefix++ {
		m := polynomialDiscriminantFixture(t)
		c := m.core
		_, e := c.monomials.MkVar()
		serviceMust(t, e)
		var modulus, upper, carrier Int
		digits := make([]uint32, 64)
		digits[63] = 1
		serviceMust(t, c.integer.SetLimbs(&modulus, digits))
		serviceMust(t, c.modular.SetZP(&modulus))
		digits = make([]uint32, 63)
		digits[62] = 0x80000000
		serviceMust(t, c.integer.SetLimbs(&upper, digits))
		x2, e := c.monomials.MkVariable(0, 2)
		serviceMust(t, e)
		unit, e := c.monomials.MkMonomial(nil)
		serviceMust(t, e)
		p, e := c.MkTerms([]Int{upper, NewInt(1)}, []Monomial{x2, unit})
		serviceMust(t, e)
		p = polynomialCaller(t, c, p)
		// Real deletion warms only the owning ID recycle vectors.
		eleven := NewInt(11)
		doomed, e := c.MkConstantTakeCopy(&eleven)
		serviceMust(t, e)
		serviceMust(t, c.IncRef(doomed))
		serviceMust(t, c.DecRef(doomed))
		doomedMonomial, e := c.monomials.MkVariable(0, 3)
		serviceMust(t, e)
		serviceMust(t, c.monomials.IncRef(doomedMonomial))
		serviceMust(t, c.monomials.DecRef(doomedMonomial))
		thirteen := NewInt(13)
		old, e := c.MkConstantTakeCopy(&thirteen)
		serviceMust(t, e)
		r, e := NewPolynomialRef(c)
		serviceMust(t, e)
		serviceMust(t, r.Assign(old))
		calls := 0
		h, e := NewPolynomialDeletionHandler(c.integer.budget, func(_ *PolynomialDeletionHandler, core *MultivariatePolynomialCore, value Polynomial) error {
			s := &m.discriminant
			if calls != 0 || !s.active || !s.executing || s.depth != 1 || value != old || !r.root.assigning || r.root.incoming != core.zero {
				return nil
			}
			calls++
			two := NewInt(2)
			if e := core.integer.Mul(&upper, &two, &carrier); e != nil {
				return e
			}
			if core.integer.cell(core.integer.tmp[0]).capacity != 6 {
				t.Fatal("real caller MUL did not replace published large scratch")
			}
			core.integer.budget.FailAfter(prefix)
			return nil
		})
		serviceMust(t, e)
		serviceMust(t, c.AddDeletionHandler(h))
		e = m.Discriminant(p, 0, r)
		s := &m.discriminant
		polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, r.root.value, r.root.incoming, s.refs[0].value, s.refs[1].value}, []*Int{&modulus, &upper, &carrier, &s.lc}, r.header+h.header)
		if e != nil && s.leadingNeg.invocation != 0 && s.leadingNeg.frame.stage == 2 && s.scalar.invocation == 0 {
			selected = true
			zero := NewInt(0)
			negative, signErr := c.integer.Less(&s.lc, &zero)
			serviceMust(t, signErr)
			if !negative || s.scalar.invocation != 0 || r.root.value != c.zero || !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual sign/PN lower correction suffix order", prefix, e)
			}
		}
		serviceMust(t, m.CleanupDiscriminant())
		polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, r.root.value, r.root.incoming}, []*Int{&modulus, &upper, &carrier}, r.header+h.header)
		c.integer.budget.FailAfter(-1)
		serviceMust(t, c.RemoveDeletionHandler(h))
		serviceMust(t, h.CloseChecked())
		if e == nil {
			success = true
			zero, zerr := c.IsZero(r.root.value)
			serviceMust(t, zerr)
			if calls != 1 || !zero {
				t.Fatal("even-modulus derivative zero resultant must stay zero", calls)
			}
		}
		serviceMust(t, r.Close())
		for _, n := range []*Int{&modulus, &upper, &carrier} {
			serviceMust(t, c.integer.Del(n))
		}
		if success {
			t.Log("first actual Discriminant even-upper suffix success", prefix)
			break
		}
	}
	if !selected || !success {
		t.Fatal("actual parent lower Add refusal and first-success horizon missing", selected, success)
	}
}
