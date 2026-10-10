// Source: src/math/polynomial/polynomial.cpp:3453-3530,4352-4506.
package polynomial

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialICCPIndependentContentAndPrimitiveBranches(t *testing.T) {
	for route := uint32(0); route < 5; route++ {
		full := polynomialPSCFullFixture(t)
		m := full.core
		p, cwant, ppwant := m.zero, m.unit, m.zero
		iwant := int32(0)
		switch route {
		case 1:
			p = polynomialQueryTerms(t, m, []int32{-7}, [][]MonomialPower{nil})
			ppwant = m.unit
			iwant = -7
		case 2:
			p = polynomialQueryTerms(t, m, []int32{-6, 10}, [][]MonomialPower{{{1, 1}}, nil})
			cwant = polynomialQueryTerms(t, m, []int32{-3, 5}, [][]MonomialPower{{{1, 1}}, nil})
			ppwant = m.unit
			iwant = 2
		case 3:
			p = polynomialQueryTerms(t, m, []int32{6, 12, 18}, [][]MonomialPower{{{0, 2}}, {{0, 1}, {1, 1}}, {{1, 2}}})
			ppwant = polynomialQueryTerms(t, m, []int32{1, 2, 3}, [][]MonomialPower{{{0, 2}}, {{0, 1}, {1, 1}}, {{1, 2}}})
			iwant = 6
		case 4:
			p = polynomialQueryTerms(t, m, []int32{6, 12}, [][]MonomialPower{{{0, 1}, {1, 1}}, {{1, 2}}})
			cwant = polynomialQueryTerms(t, m, []int32{1}, [][]MonomialPower{{{1, 1}}})
			ppwant = polynomialQueryTerms(t, m, []int32{1, 2}, [][]MonomialPower{{{0, 1}}, {{1, 1}}})
			iwant = 6
		}
		i := unumeral.NewInt(99)
		c, e := NewPolynomialRef(m)
		serviceMust(t, e)
		pp, e := NewPolynomialRef(m)
		serviceMust(t, e)
		serviceMust(t, c.Assign(m.zero))
		serviceMust(t, pp.Assign(m.unit))
		serviceMust(t, full.IntegerContentPrimitiveParts(p, 0, &i, c, pp))
		if i != unumeral.NewInt(iwant) {
			t.Fatal("independent integer component", route, i, iwant)
		}
		equal, e := m.Equal(c.root.value, cwant)
		serviceMust(t, e)
		if !equal {
			t.Fatal("independent content", route)
		}
		equal, e = m.Equal(pp.root.value, ppwant)
		serviceMust(t, e)
		if !equal {
			t.Fatal("independent primitive part", route)
		}
		serviceMust(t, c.Close())
		serviceMust(t, pp.Close())
		serviceMust(t, m.modular.Del(&i))
	}
}

func TestPolynomialGeneralGCDIndependentCommonContent(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	// u=(x+y)*(x+1), v=(x+y)*(y+2), whose primitive GCD is x+y.
	u := polynomialQueryTerms(t, m, []int32{1, 1, 1, 1}, [][]MonomialPower{{{0, 2}}, {{0, 1}, {1, 1}}, {{0, 1}}, {{1, 1}}})
	v := polynomialQueryTerms(t, m, []int32{1, 1, 2, 2}, [][]MonomialPower{{{0, 1}, {1, 1}}, {{1, 2}}, {{0, 1}}, {{1, 1}}})
	want := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, {{1, 1}}})
	r, e := NewPolynomialRef(m)
	serviceMust(t, e)
	serviceMust(t, full.GCD(u, v, r))
	equal, e := m.Equal(r.root.value, want)
	serviceMust(t, e)
	if !equal {
		t.Fatal("independent common linear factor")
	}
	serviceMust(t, r.Close())
}

// Exercise a coefficient filter beyond both former inline capacities. Expected
// content and primitive parts are independently constructed polynomials.
func TestPolynomialICCPNativeHighDegreeFilters(t *testing.T) {
	for _, degree := range []uint32{33, 129, 257} {
		full := polynomialPSCFullFixture(t)
		c := full.core
		for i := 0; i < 2; i++ {
			_, e := c.monomials.MkVar()
			serviceMust(t, e)
		}
		p := polynomialQueryTerms(t, c, []int32{2, 4}, [][]MonomialPower{{{0, degree}, {1, 1}}, {{0, 3}, {1, 1}}})
		wantContent := polynomialQueryTerms(t, c, []int32{1}, [][]MonomialPower{{{1, 1}}})
		wantPrimitive := polynomialQueryTerms(t, c, []int32{1, 2}, [][]MonomialPower{{{0, degree}}, {{0, 3}}})
		var integer unumeral.Int
		var content, primitive polynomialReference
		for repeat := 0; repeat < 2; repeat++ {
			serviceMust(t, full.iccpParts(p, 0, &integer, &content, &primitive))
			if integer != unumeral.NewInt(2) {
				t.Fatal("integer content", degree, integer)
			}
			same, e := c.Equal(content.value, wantContent)
			serviceMust(t, e)
			if !same {
				t.Fatal("coefficient content", degree)
			}
			same, e = c.Equal(primitive.value, wantPrimitive)
			serviceMust(t, e)
			if !same {
				t.Fatal("primitive part", degree)
			}
			for _, s := range full.iccpFrames {
				if s.active || s.filter != nil || s.powers != nil {
					t.Fatal("completed ICCP scratch retained", degree, s.active, len(s.filter), len(s.powers))
				}
			}
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, wantContent, wantPrimitive, content.value, primitive.value}, []*unumeral.Int{&integer}, 0)
			serviceMust(t, content.close(c))
			serviceMust(t, primitive.close(c))
			serviceMust(t, c.integer.Del(&integer))
		}
	}
}
