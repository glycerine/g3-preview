// Source: src/math/polynomial/upolynomial_factorization.cpp:1030-1034.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestUPolynomialPrimitiveContentAssertionPublicActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			_, s := preparationPhysicalFixture(t)
			m := s.manager
			f := upolySourceNew(t, m, []int32{2, 6, 2})
			product := upolySourceNew(t, m, nil)
			fs, e := m.NewFactors()
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, s.CleanupRetained()); serviceMust(t, fs.Close()) })
			choice := action
			if action == "disabled" {
				choice = ""
			}
			polynomialAssertionActionsFixture(t, choice)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			completed, e := s.FactorSquareFreeDefault(mustUPolyView(t, m, f), fs, DefaultUPolynomialFactorParams())
			n, countErr := fs.DistinctFactors()
			serviceMust(t, countErr)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) || completed || n != 0 || s.scope != nil {
					t.Fatal("content Throw must stop before f_pp construction or factor publication", e, completed, n)
				}
			} else {
				serviceMust(t, e)
				// The source assumes primitive input; Continue does not extract content.
				// Modulo 3 this image is irreducible and publishes the original vector.
				if !completed || n != 1 {
					t.Fatal("literal irreducible continuation", completed, n)
				}
				serviceMust(t, fs.Multiply(product))
				same, e := m.Equal(f, product)
				serviceMust(t, e)
				if !same {
					t.Fatal("nonprimitive input was changed by assertion continuation")
				}
			}
			if m.factorTrace != nil {
				t.Fatal("assertion retained manager borrow")
			}
		})
	}
}

func TestUPolynomialPrimitiveDisabledAssertionStillComputesDebugContent(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source DEBUG_CODE is compiled out")
	}
	q, s := preparationPhysicalFixture(t)
	m := s.manager
	var large unumeral.Int
	serviceMust(t, q.Integers().SetLimbs(&large, []uint32{1, 0, 1}))
	t.Cleanup(func() { serviceMust(t, q.Integers().Del(&large)) })
	f, e := m.NewPolynomial()
	serviceMust(t, e)
	serviceMust(t, m.SetInts(f, []unumeral.Int{unumeral.NewInt(0), large, unumeral.NewInt(1)}))
	t.Cleanup(func() { serviceMust(t, m.DelPolynomial(&f)) })
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, s.CleanupRetained()); serviceMust(t, fs.Close()) })
	polynomialAssertionActionsFixture(t, "")
	serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	view := mustUPolyView(t, m, f)
	// gcd(0,BIG) reaches the real content-copy allocation before the runtime
	// assertion gate. Disabling SASSERT must not erase the DEBUG_CODE body.
	q.Budget().FailAfter(0)
	_, body := s.FactorSquareFreeDefault(view, fs, DefaultUPolynomialFactorParams())
	q.Budget().FailAfter(-1)
	if !numericPhysicalReservationRefusal(body) || s.diagnostic.invocation == 0 || s.scope != nil || fs.count != 0 {
		t.Fatal("disabled assertion skipped real debug GCD or entered f_pp early", body)
	}
}
