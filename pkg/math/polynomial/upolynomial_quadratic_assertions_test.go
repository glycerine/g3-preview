package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestUPolynomialQuadraticPredicateAssertionActions(t *testing.T) {
	for _, product := range []bool{false, true} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(map[bool]string{false: "discriminant", true: "product"}[product]+"/"+action, func(t *testing.T) {
				m, _, _ := upolySourceOwner(t)
				a, b := upolySourceNew(t, m, []int32{-1, 0, 1}), upolySourceNew(t, m, []int32{-2, 0, 1})
				av, bv := mustUPolyView(t, m, a), mustUPolyView(t, m, b)
				choice := action
				if action == "disabled" {
					choice = ""
				}
				polynomialAssertionActionsFixture(t, choice)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				// Qualified direct invariant adapters: independent caller values/views;
				// no rewritten algorithm scratch or fabricated enclosing quadratic frame.
				zero := unumeral.NewInt(0)
				var e error
				if product {
					e = m.quadraticProductAssertion(av, bv)
				} else {
					e = m.quadraticDiscriminantAssertion(&zero)
				}
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
						t.Fatal("quadratic predicate Throw", e)
					}
				} else {
					serviceMust(t, e)
				}
				upolyGCDExpect(t, m, a, []int64{-1, 0, 1})
				upolyGCDExpect(t, m, b, []int64{-2, 0, 1})
			})
		}
	}
}
func TestUPolynomialQuadraticRuntimeAssertionOracles(t *testing.T) {
	for _, irreducible := range []bool{false, true} {
		for _, disabled := range []bool{false, true} {
			t.Run(map[bool]string{false: "split", true: "irreducible"}[irreducible]+"/"+map[bool]string{false: "enabled", true: "disabled"}[disabled], func(t *testing.T) {
				_, s := preparationPhysicalFixture(t)
				m := s.manager
				coeff := []int32{-1, 0, 1}
				want := []int64{-1, 0, 1}
				count := uint32(2)
				if irreducible {
					coeff = []int32{-2, 0, 1}
					want = []int64{-2, 0, 1}
					count = 1
				}
				p, out := upolySourceNew(t, m, coeff), upolySourceNew(t, m, nil)
				fs, e := m.NewFactors()
				serviceMust(t, e)
				t.Cleanup(func() { serviceMust(t, s.CleanupRetained()); serviceMust(t, fs.Close()) })
				polynomialAssertionActionsFixture(t, "")
				if polynomialSourceDebug && disabled {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				_, e = s.Factor(mustUPolyView(t, m, p), fs, DefaultUPolynomialFactorParams())
				serviceMust(t, e)
				if fs.count != count {
					t.Fatal("actual quadratic factors", fs.count, count)
				}
				peak := uint8(4)
				if !irreducible {
					peak = 6
					if polynomialSourceDebug {
						peak = 7
					}
				}
				if s.frontend.quadratic.peak != peak || s.frontend.quadratic.constructed != 0 {
					t.Fatal("runtime gate erased DEBUG product construction", s.frontend.quadratic.peak, peak)
				}
				serviceMust(t, fs.Multiply(out))
				upolyGCDExpect(t, m, out, want)
			})
		}
	}
}
func TestUPolynomialQuadraticPredicatesClosedAndErased(t *testing.T) {
	for _, product := range []bool{false, true} {
		t.Run(map[bool]string{false: "discriminant", true: "product"}[product], func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			p := upolySourceNew(t, m, []int32{-1, 0, 1})
			v := mustUPolyView(t, m, p)
			polynomialAssertionActionsFixture(t, "")
			if polynomialSourceDebug {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			// Invalid operands are skipped only by the erased numerical predicates.
			if product {
				serviceMust(t, m.quadraticProductAssertion(UPolynomialView{}, UPolynomialView{}))
			} else {
				serviceMust(t, m.quadraticDiscriminantAssertion(nil))
			}
			if !polynomialSourceDebug {
				return
			}
			serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
			var e error
			one := unumeral.NewInt(1)
			if product {
				e = m.quadraticProductAssertion(v, v)
			} else {
				e = m.quadraticDiscriminantAssertion(&one)
			}
			if !errors.Is(e, arena.ErrClosed) {
				t.Fatal("closed runtime accepted valid predicate", e)
			}
		})
	}
}
