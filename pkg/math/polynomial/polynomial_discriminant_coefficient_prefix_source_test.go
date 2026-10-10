// Source: src/math/polynomial/polynomial.cpp:5473-5500.
package polynomial

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

// p=y*x^2+1 reaches the polynomial leading-coefficient branch: Neg,
// the younger c reference, Coeff, then ExactDiv. Expected -4*y is the
// independent quadratic identity, not another Discriminant evaluation.
func TestPolynomialDiscriminantPolynomialLeadingCoefficientActualPrefixes(t *testing.T) {
	selected, success := false, false
	for prefix := int64(0); prefix < 256; prefix++ {
		m := polynomialDiscriminantFixture(t)
		c := m.core
		for i := 0; i < 2; i++ {
			_, e := c.monomials.MkVar()
			serviceMust(t, e)
		}
		xy, e := c.monomials.MkMonomial([]MonomialPower{{0, 2}, {1, 1}})
		serviceMust(t, e)
		unit, e := c.monomials.MkMonomial(nil)
		serviceMust(t, e)
		y, e := c.monomials.MkMonomial([]MonomialPower{{1, 1}})
		serviceMust(t, e)
		p, e := c.MkTerms([]unumeral.Int{unumeral.NewInt(1), unumeral.NewInt(1)}, []Monomial{xy, unit})
		serviceMust(t, e)
		p = polynomialCaller(t, c, p)
		expected, e := c.MkTerms([]unumeral.Int{unumeral.NewInt(-4)}, []Monomial{y})
		serviceMust(t, e)
		expected = polynomialCaller(t, c, expected)
		r, e := NewPolynomialRef(c)
		serviceMust(t, e)
		t.Cleanup(func() { serviceMust(t, m.CleanupDiscriminant()); serviceMust(t, r.Close()) })
		budget := m.originalBudget
		budget.FailAfter(prefix)
		e = m.Discriminant(p, 0, r)
		s := &m.discriminant
		polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, expected, r.root.value, r.root.incoming, s.refs[0].value, s.refs[1].value}, []*unumeral.Int{&s.lc}, r.header)
		if s.coefficient.invocation != 0 {
			selected = true
			if s.neg.invocation == 0 || s.scalar.invocation != 0 || s.constant.invocation == 0 {
				t.Fatal("polynomial coefficient suffix lost reached source order", prefix, e)
			}
		}
		if e == nil {
			success = true
		}
		if e != nil && (prefix == 15 || prefix == 16 || prefix == 21) {
			as, _ := c.cheap.as.size(c)
			ms, _ := c.cheap.ms.size(c)
			t.Logf("before own prefix=%d content=%+v cheap as=%d ms=%d append=%+v", prefix, c.contentLocal, as, ms, c.cheap.appendScope)
			for row := uint32(0); row < as && row < ms; row++ {
				a, ae := c.cheap.as.coefficient(c, row)
				term, me := c.cheap.ms.monomial(c, row)
				if ae == nil && me == nil {
					t.Logf("before own row=%d coefficient=%+v powers=%v", row, a, monomialRead(t, c.monomials, term))
				}
			}
		}
		if own := m.CleanupDiscriminant(); own != nil {
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, expected, r.root.value, r.root.incoming, s.refs[0].value, s.refs[1].value}, []*unumeral.Int{&s.lc}, r.header)
			// The owning suffix can reach a real ID-vector reservation in
			// either the exact failed polynomial factory or the append row's
			// actual last monomial DEC. Observe its named slot and stage.
			polynomialID := c.pending.idConsumed && c.matchesFactoryInvocation(&m.derivative.factory) && c.pending.generation == m.derivative.factory.invocation
			monomialID := false
			append := &c.cheap.appendScope
			if c.cheap.matchesAppendInvocation(&m.derivative.append) && append.coefficientCursor == append.coefficients && append.monomialCursor < append.monomials {
				i := append.monomialCursor
				term, read := c.cheap.ms.monomial(c, i)
				if read == nil && uint64(i) < uint64(len(c.cheap.monomialOwners)) && c.cheap.monomialOwners[i] == m.derivative.append.invocation {
					location, read := c.monomials.location(term)
					if read == nil && location.stage == 1 {
						rc, read := c.monomials.allocator.ReadUint32(location.raw, 0)
						monomialID = read == nil && rc == 0
					}
				}
			}
			if !numericPhysicalReservationRefusal(own) || (!polynomialID && !monomialID) {
				t.Fatalf("unexpected owning suffix stop prefix=%d body=%v own=%v depth=%d derivative=%d resultant=%d neg=%d coeff=%d exact=%d", prefix, e, own, s.depth, s.derivative.invocation, s.resultant.invocation, s.neg.invocation, s.coefficient.invocation, s.exact.invocation)
			}
			if monomialID {
				t.Log("actual last row DEC stopped at monomial ID recycle", prefix, append.monomialCursor)
			}
			c.integer.Budget().FailAfter(-1)
			serviceMust(t, m.CleanupDiscriminant())
		}
		polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, expected, r.root.value, r.root.incoming}, nil, r.header)
		budget.FailAfter(-1)
		if !success {
			prior := e
			if retry := m.Discriminant(p, 0, r); retry != nil {
				t.Logf("retry actual Integer tmp=%v ExactDiv active=%v stage=%d kind=%d sourceCause=%v", c.integer.Tmp, c.exactDivision.active, c.exactDivision.stage, c.exactDivision.kind, c.exactDivision.sourceCause)
				t.Fatalf("same owner retry prefix=%d original=%v retry=%v Neg=%+v Factory=%+v Append=%+v", prefix, prior, retry, c.polynomialNegScope, c.factoryInvocation, c.cheap.appendScope)
			}
			if prior == nil {
				t.Fatal("retry proof lost original refusal")
			}
			equal, err := c.Equal(r.root.value, expected)
			serviceMust(t, err)
			if !equal {
				t.Fatal("same actual owner/frame failed successful invocation after exact owning acknowledgment", prefix)
			}
		}
		// Equal has its own real scratch construction, outside the source
		// Discriminant horizon already audited and cleaned above.
		if success {
			equal, err := c.Equal(r.root.value, expected)
			serviceMust(t, err)
			if !equal {
				t.Fatal("quadratic discriminant is not -4*y", prefix)
			}
		}
		serviceMust(t, r.Close())
		if success {
			t.Log("first actual polynomial-leading Discriminant cold success", prefix)
			break
		}
	}
	if !selected || !success {
		t.Fatal("polynomial-leading suffix/first-success horizon missing", selected, success)
	}
}
