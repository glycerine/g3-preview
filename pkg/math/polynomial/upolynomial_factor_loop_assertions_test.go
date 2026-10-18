package polynomial

import (
	"errors"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"testing"
)

func TestUPolynomialFactorLoopPredicateActions(t *testing.T) {
	for _, route := range []string{"nonconstant", "size", "unit", "final-size", "final-value"} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(route+"/"+action, func(t *testing.T) {
				m, _, _ := upolySourceOwner(t)
				coeff := []int32{2}
				if route == "size" || route == "final-size" {
					coeff = nil
				}
				p := upolySourceNew(t, m, coeff)
				d, e := m.descriptor(p)
				serviceMust(t, e)
				choice := action
				if action == "disabled" {
					choice = ""
				}
				polynomialAssertionActionsFixture(t, choice)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				// Qualified direct invariant adapters use independent actual polynomial
				// storage, without altering an executing factorization's numerical scratch.
				switch route {
				case "nonconstant":
					e = m.factorNonconstantAssertion(&d.state)
				case "size":
					e = m.factorConstantSizeAssertion(&d.state)
				case "unit":
					e = m.factorConstantUnitAssertion(&d.state)
				default:
					e = m.factorFinalOneAssertion(&d.state)
				}
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
						t.Fatal("factor-loop Throw", e)
					}
				} else {
					serviceMust(t, e)
				}
				want := []int64{2}
				if len(coeff) == 0 {
					want = nil
				}
				upolyGCDExpect(t, m, p, want)
			})
		}
	}
}
func TestUPolynomialFactorConstantSeparateActionsAndErasure(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	empty := upolySourceNew(t, m, nil)
	d, e := m.descriptor(empty)
	serviceMust(t, e)
	polynomialAssertionActionsFixture(t, "C")
	serviceMust(t, m.factorConstantSizeAssertion(&d.state))
	e = m.factorConstantUnitAssertion(&d.state)
	if polynomialSourceDebug {
		if !errors.Is(e, ErrUPolynomialOwner) {
			t.Fatal("Continue did not reach actual C[0] guard", e)
		}
	} else {
		serviceMust(t, e)
	}
	if polynomialSourceDebug {
		serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	}
	serviceMust(t, m.factorNonconstantAssertion(nil))
	serviceMust(t, m.factorConstantSizeAssertion(nil))
	serviceMust(t, m.factorConstantUnitAssertion(nil))
	serviceMust(t, m.factorFinalOneAssertion(nil))
}
func TestUPolynomialFactorLoopRuntimeOracles(t *testing.T) {
	for _, repeated := range []bool{false, true} {
		for _, disabled := range []bool{false, true} {
			t.Run(map[bool]string{false: "squarefree", true: "cube"}[repeated]+"/"+map[bool]string{false: "enabled", true: "disabled"}[disabled], func(t *testing.T) {
				_, s := preparationPhysicalFixture(t)
				m := s.manager
				coeff := []int32{-1, 0, 1}
				want := []int64{-1, 0, 1}
				count := uint32(2)
				if repeated {
					coeff = []int32{-1, 3, -3, 1}
					want = []int64{-1, 3, -3, 1}
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
					t.Fatal("actual source factor count", fs.count, count)
				}
				if repeated && (fs.totalFactors != 3 || fs.degrees[0] != 3) {
					t.Fatal("actual cube multiplicity", fs.totalFactors, fs.degrees[0])
				}
				serviceMust(t, fs.Multiply(out))
				upolyGCDExpect(t, m, out, want)
			})
		}
	}
}
