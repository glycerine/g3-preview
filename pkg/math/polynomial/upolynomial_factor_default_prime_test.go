package polynomial

import (
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	primegen "github.com/glycerine/g3/pkg/util/prime"
)

func TestUPolynomialFactorDefaultProcessPrimeBinding(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	g, e := primegen.BorrowProcessGenerator()
	if e != nil {
		t.Fatal(e)
	}
	s, e := NewUPolynomialFactorizationService(m, UPolynomialFactorizationConfig{})
	if e != nil {
		t.Fatal(e)
	}
	if s.config.Primes != g {
		t.Fatal("source default factor service did not borrow process prime cache")
	}
	fs, e := m.NewFactors()
	if e != nil {
		t.Fatal(e)
	}
	p, e := m.NewPolynomial()
	if e != nil {
		t.Fatal(e)
	}
	if e := m.SetInts(p, []unumeral.Int{unumeral.NewInt(-1), unumeral.NewInt(0), unumeral.NewInt(1)}); e != nil {
		t.Fatal(e)
	}
	v := mustUPolyView(t, m, p)
	if _, e := s.Factor(v, fs, DefaultUPolynomialFactorParams()); e != nil {
		t.Fatal(e)
	}
	if n, e := fs.DistinctFactors(); e != nil || n != 2 {
		t.Fatal("source default factorization", n, e)
	}
	if e := fs.Close(); e != nil {
		t.Fatal(e)
	}
	if e := m.DelPolynomial(&p); e != nil {
		t.Fatal(e)
	}
	if e := s.Close(); e != nil {
		t.Fatal(e)
	}
	if e := g.Check(); e != nil {
		t.Fatal("factor service closed borrowed process generator", e)
	}
}
