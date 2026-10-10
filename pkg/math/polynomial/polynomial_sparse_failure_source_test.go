// Source: src/math/polynomial/polynomial.cpp:3134-3182.
package polynomial

import (
	"fmt"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialSparseInterpolationEveryActualReservationPrefix(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 160 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			m := full.core
			serviceMust(t, m.modular.SetZPUint64(97))
			p := polynomialQueryTerms(t, m, []int32{1, 3, 2}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})
			sk, e := NewPolynomialSkeleton(m, p, 0)
			serviceMust(t, e)
			s, e := NewPolynomialSparseInterpolator(full, sk)
			serviceMust(t, e)
			t.Cleanup(func() {
				m.integer.Budget().FailAfter(-1)
				serviceMust(t, s.CloseChecked())
				serviceMust(t, sk.CloseChecked())
			})
			polys := []Polynomial{p}
			for _, x := range []int32{0, 1, 2} {
				in := unumeral.NewInt(x)
				q := polynomialQueryTerms(t, m, []int32{x*x + 3*x + 2}, [][]MonomialPower{nil})
				polys = append(polys, q)
				ok, e := s.Add(&in, q)
				serviceMust(t, e)
				if !ok {
					t.Fatal("matching sample support")
				}
			}
			out, e := NewPolynomialRef(m)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, out.Close()) })
			m.integer.Budget().FailAfter(ordinal)
			ok, e := s.Interpolate(out)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual source allocation category", ordinal, e)
			}
			polynomialSparsePhysical(t, s, out, polys)
			m.integer.Budget().FailAfter(-1)
			serviceMust(t, s.CleanupInterpolation())
			polynomialSparsePhysical(t, s, out, polys)
			if e == nil {
				if !ok {
					t.Fatal("source Vandermonde nonsingular")
				}
				equal, x := m.Equal(out.root.value, p)
				serviceMust(t, x)
				if !equal {
					t.Fatal("independent polynomial identity")
				}
				success = true
				t.Log("every refusal before success", ordinal)
			}
		})
	}
	if !success {
		t.Fatal("no complete actual prefix endpoint")
	}
}
