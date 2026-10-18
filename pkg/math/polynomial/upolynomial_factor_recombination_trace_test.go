package polynomial

import (
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"strings"
	"testing"
)

func TestUPolynomialFactorRecombinationTraceInexactDivision(t *testing.T) {
	_, s := preparationPhysicalFixture(t)
	m := s.manager
	var text strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&text)
	tags.Enable("polynomial::factorization::bughunt")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	p := upolySourceNew(t, m, []int32{1, 0, 0, 0, 1})
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, fs.Close()) })
	_, e = s.FactorSquareFreeDefault(mustUPolyView(t, m, p), fs, DefaultUPolynomialFactorParams())
	serviceMust(t, e)
	if tags.ShouldTrace("polynomial::factorization::bughunt") {
		out := text.String()
		if !strings.Contains(out, "result = false\n") || !strings.Contains(out, "trial_factor_quo = ") || !strings.Contains(out, ":1274 ") {
			t.Fatal("inexact source division record", out)
		}
	} else if text.Len() != 0 {
		t.Fatal("erased recombination trace")
	}
	n, e := fs.DistinctFactors()
	serviceMust(t, e)
	if n != 1 {
		t.Fatal("irreducible result changed", n)
	}
	product, e := m.NewPolynomial()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.DelPolynomial(&product)) })
	serviceMust(t, fs.Multiply(product))
	equal, e := m.Equal(p, product)
	serviceMust(t, e)
	if !equal {
		t.Fatal("trace changed factor product")
	}
}
