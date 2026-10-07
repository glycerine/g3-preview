// Source: src/math/polynomial/polynomial.cpp:5473-5500.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialDiscriminantActualResultantBeforeLeadingNUMPrefixes(t *testing.T) {
	selected, success := false, false
	for prefix := int64(0); prefix < 128; prefix++ {
		m := polynomialDiscriminantFixture(t)
		c := m.core
		_, e := c.monomials.MkVar()
		serviceMust(t, e)
		var leading util.Int
		digits := make([]uint32, 64)
		digits[0] = 7
		digits[63] = 1
		serviceMust(t, c.integer.SetLimbs(&leading, digits))
		x2, e := c.monomials.MkMonomial([]MonomialPower{{0, 2}})
		serviceMust(t, e)
		unit, e := c.monomials.MkMonomial(nil)
		serviceMust(t, e)
		p, e := c.MkTerms([]util.Int{leading, util.NewInt(1)}, []Monomial{x2, unit})
		serviceMust(t, e)
		p = polynomialCaller(t, c, p)
		r, e := NewPolynomialRef(c)
		serviceMust(t, e)
		t.Cleanup(func() { serviceMust(t, m.CleanupDiscriminant()); serviceMust(t, r.Close()) })
		budget := m.originalBudget
		budget.FailAfter(prefix)
		e = m.Discriminant(p, 0, r)
		s := &m.discriminant
		values := []Polynomial{p, r.root.value, s.refs[0].value, s.refs[1].value}
		polynomialImpPhysicalAuditNamed(t, m, values, []*util.Int{&leading, &s.lc}, r.header)
		if e != nil && s.constant.invocation != 0 {
			selected = true
			if r.root.value == (Polynomial{}) {
				t.Fatal("leading NUM read preceded actual caller Resultant publication", prefix, e)
			}
			t.Log("actual caller Resultant published before leading NUM refusal", prefix)
		}
		if e == nil {
			success = true
			if s.active {
				t.Fatal("ordinary completion retained unexpected parent owner", prefix)
			}
		}
		// The borrowed output stays named across parent cleanup. Its public
		// reference destructor may need real ID-vector growth outside this function.
		serviceMust(t, m.CleanupDiscriminant())
		polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, r.root.value, r.root.incoming}, []*util.Int{&leading}, r.header)
		budget.FailAfter(-1)
		serviceMust(t, r.Close())
		serviceMust(t, c.integer.Del(&leading))
		if success {
			t.Log("first actual Discriminant cold success", prefix)
			break
		}
	}
	if !selected || !success {
		t.Fatal("actual leading NUM/Resultant prefix horizon missing", selected, success)
	}
}
