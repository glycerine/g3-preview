package polynomial

import (
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"strings"
	"testing"
)

func TestUPolynomialFactorHenselLiftTraceSeparateTagAndOwners(t *testing.T) {
	_, s := preparationPhysicalFixture(t)
	m := s.manager
	w := factorDegreeTraceProbe{t: t, service: s, checked: map[*UPolynomialManager]bool{}}
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("polynomial::factorization")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	p := upolySourceNew(t, m, []int32{2, 3, 1})
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, fs.Close()) })
	_, e = s.FactorSquareFreeDefault(mustUPolyView(t, m, p), fs, DefaultUPolynomialFactorParams())
	serviceMust(t, e)
	if !tags.ShouldTrace("polynomial::factorization") {
		if w.seen.Len() != 0 {
			t.Fatal("erased lift trace")
		}
		return
	}
	text := w.seen.String()
	if !strings.Contains(text, ":935 ") || !strings.Contains(text, "lifted to 16\nA = x + 2\nB = x + 1\n") || strings.Contains(text, "best zp factorization") || !w.headerChecked || !w.flushChecked || len(w.polynomialChecked) != 1 {
		t.Fatal("separate whole lift record/borrowed owners", text)
	}
}

func TestUPolynomialFactorHenselVectorTraceIndependentPublicCores(t *testing.T) {
	m, _, _, _, pf, lf, s, f, _, _ := upolyHenselPhysicalFixture(t)
	w := factorDegreeTraceProbe{t: t, service: s, checked: map[*UPolynomialManager]bool{}}
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("polynomial::factorization::bughunt")
	tags.Enable("polynomial::factorization")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	serviceMust(t, s.HenselLiftFactors(f, pf, 16, lf))
	if tags.ShouldTrace("polynomial::factorization") {
		for _, record := range []string{":744 ", ":771 ", ":886 ", ":960 ", ":906 ", ":921 ", ":927 ", ":935 "} {
			if !strings.Contains(w.seen.String(), record) {
				t.Fatal("missing independent-core source record", record)
			}
		}
		if !w.headerChecked || !w.flushChecked || len(w.polynomialChecked) != 1 {
			t.Fatal("independent-core trace did not protect actual borrowers")
		}
	} else if w.seen.Len() != 0 {
		t.Fatal("erased independent trace")
	}
	if s.scope != nil {
		t.Fatal("public Hensel adapter retained after successful traced source completion")
	}
	n, e := lf.DistinctFactors()
	serviceMust(t, e)
	if n != 3 {
		t.Fatal("independent lifted factors", n)
	}
}
