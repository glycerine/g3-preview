package polynomial

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestUPolynomialFactorSquareFreeDefaultPublicInterface(t *testing.T) {
	_, s := preparationPhysicalFixture(t)
	m := s.manager
	var f upolyVectorState
	serviceMust(t, m.set(&f, []unumeral.Int{unumeral.NewInt(-6), unumeral.NewInt(1), unumeral.NewInt(1)}))
	defer func() { serviceMust(t, m.reset(&f)); m.freeBacking(&f) }()
	fs, e := m.NewFactors()
	serviceMust(t, e)
	defer func() { serviceMust(t, fs.Close()) }()
	completed, e := s.FactorSquareFreeDefault(m.stateView(&f), fs, DefaultUPolynomialFactorParams())
	serviceMust(t, e)
	n, e := fs.DistinctFactors()
	serviceMust(t, e)
	if !completed || n != 2 {
		t.Fatal("known (x-2)*(x+3) factorization", completed, n)
	}
	seen := map[int32]bool{}
	for i := uint32(0); i < n; i++ {
		k, e := fs.Multiplicity(i)
		serviceMust(t, e)
		v, e := fs.FactorView(i)
		serviceMust(t, e)
		if k != 1 || v.count != 2 {
			t.Fatal("default multiplicity/linear degree", k, v.count)
		}
		constant, e := m.read(v, 0)
		serviceMust(t, e)
		lead, e := m.read(v, 1)
		serviceMust(t, e)
		if !constant.IsSmall() || !lead.IsSmall() || lead.Val != 1 {
			t.Fatal("expected primitive monic linear factors")
		}
		seen[constant.Val] = true
	}
	if !seen[-2] || !seen[3] {
		t.Fatal("independent linear factors", seen)
	}
}
