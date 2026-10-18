package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestUPolynomialHenselPairPreconditionSourceActions(t *testing.T) {
	for _, route := range []string{"product-degree", "bezout-degree"} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(route+"/"+action, func(t *testing.T) {
				_, s := preparationPhysicalFixture(t)
				m := s.manager
				a, b := upolySourceNew(t, m, []int32{0, 1}), upolySourceNew(t, m, []int32{1, 1})
				uc, cc := []int32{-1}, []int32{1, 1}
				if route == "bezout-degree" {
					uc, cc = []int32{0, 1}, []int32{3, 1, 1}
				}
				u, v, c := upolySourceNew(t, m, uc), upolySourceNew(t, m, []int32{1}), upolySourceNew(t, m, cc)
				al, bl := upolySourceNew(t, m, []int32{71}), upolySourceNew(t, m, []int32{72})
				choice := action
				if action == "C" && route == "product-degree" {
					choice = "CCC"
				}
				if action == "disabled" {
					choice = ""
				}
				polynomialAssertionActionsFixture(t, choice)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				one, three := unumeral.NewInt(1), unumeral.NewInt(3)
				e := s.HenselLiftPair(&one, &three, &three, mustUPolyView(t, m, u), mustUPolyView(t, m, a), mustUPolyView(t, m, v), mustUPolyView(t, m, b), mustUPolyView(t, m, c), al, bl)
				// Source mpz::div rounds a negative non-exact coefficient downward:
				// f=-x^2, hence T=-x and B_lifted=1-2*x.
				wantA, wantB := []int32{0, 1}, []int32{1, -2}
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
						t.Fatal("source precondition Throw", e)
					}
					wantA, wantB = []int32{71}, []int32{72}
				} else {
					serviceMust(t, e)
					if route == "bezout-degree" {
						wantA, wantB = []int32{3, 1}, []int32{1, 4}
					}
				}
				pa, pb := upolySourceNew(t, m, wantA), upolySourceNew(t, m, wantB)
				for _, pair := range [2][2]UPolynomial{{al, pa}, {bl, pb}} {
					same, e := m.Equal(pair[0], pair[1])
					serviceMust(t, e)
					if !same {
						t.Fatal("precondition action changed source output frontier", route, action, upolyDivisionValues(t, m, al), upolyDivisionValues(t, m, bl))
					}
				}
				serviceMust(t, s.CleanupRetained())
				if m.factorTrace != nil {
					t.Fatal("precondition retained assertion borrow")
				}
			})
		}
	}
}

func TestUPolynomialHenselFactorAssertionActionsAndModulusFrontier(t *testing.T) {
	for _, route := range []string{"empty", "constant", "multiplicity", "gcd-unit"} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(route+"/"+action, func(t *testing.T) {
				_, s := preparationPhysicalFixture(t)
				m := s.manager
				zp, e := NewRelatedUPolynomialManager(m)
				serviceMust(t, e)
				zpe, e := NewRelatedUPolynomialManager(m)
				serviceMust(t, e)
				pf, e := zp.NewFactors()
				serviceMust(t, e)
				lf, e := zpe.NewFactors()
				serviceMust(t, e)
				t.Cleanup(func() {
					serviceMust(t, s.CleanupRetained())
					serviceMust(t, lf.Close())
					serviceMust(t, pf.Close())
					serviceMust(t, zpe.Close())
					serviceMust(t, zp.Close())
				})
				five, twentyFive := unumeral.NewInt(5), unumeral.NewInt(25)
				serviceMust(t, zp.SetZP(&five))
				serviceMust(t, zpe.SetZP(&twentyFive))
				fcoeff := []int32{-1, 0, 1}
				if route != "empty" {
					for i, coeff := range [][]int32{{-1, 1}, {1, 1}} {
						if route == "gcd-unit" && i == 1 {
							coeff = []int32{-1, 1}
						}
						poly := upolySourceNew(t, m, coeff)
						degree := uint32(1)
						if route == "multiplicity" && i == 0 {
							degree = 2
						}
						serviceMust(t, pf.PushCopy(mustUPolyView(t, m, poly), degree))
					}
				}
				if route == "constant" {
					two := unumeral.NewInt(2)
					serviceMust(t, pf.SetConstant(&two))
				}
				if route == "gcd-unit" {
					fcoeff = []int32{1, -2, 1}
				}
				if route == "multiplicity" {
					fcoeff = []int32{1, -1, -1, 1}
				}
				f := upolySourceNew(t, m, fcoeff)
				choice := action
				if action == "C" && route == "multiplicity" {
					choice = "CC"
				}
				if action == "disabled" {
					choice = ""
				}
				polynomialAssertionActionsFixture(t, choice)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				e = s.HenselLiftFactors(mustUPolyView(t, m, f), pf, 1, lf)
				throwing := polynomialSourceDebug && action == "T"
				if throwing {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
						t.Fatal("factor source Throw", route, e)
					}
				} else {
					serviceMust(t, e)
				}
				count, e := lf.DistinctFactors()
				serviceMust(t, e)
				p, e := zpe.modular.BorrowedModulus()
				serviceMust(t, e)
				modulus, e := m.config.Integer.Int64(p)
				serviceMust(t, e)
				wantMod := int64(5)
				if throwing && (route == "empty" || route == "constant") {
					wantMod = 25
				}
				if modulus != wantMod {
					t.Fatal("source modulus mutation frontier", route, action, modulus, wantMod)
				}
				if throwing {
					if count != 0 {
						t.Fatal("Throw passed lifted-factor publication", count)
					}
				} else {
					want := uint32(2)
					if route == "empty" {
						want = 1
					}
					if count != want {
						t.Fatal("continued source factors", route, count)
					}
					product := upolySourceNew(t, zpe, nil)
					serviceMust(t, lf.Multiply(product))
					expected := []int32(nil)
					if route == "constant" {
						expected = []int32{-2, 0, 2}
					}
					if route == "multiplicity" || route == "gcd-unit" {
						expected = fcoeff
					}
					p := upolySourceNew(t, zpe, expected)
					same, e := zpe.Equal(product, p)
					serviceMust(t, e)
					if !same {
						t.Fatal("continued literal factor product", route, action, upolyDivisionValues(t, zpe, product))
					}
				}
				serviceMust(t, s.CleanupRetained())
			})
		}
	}
}
