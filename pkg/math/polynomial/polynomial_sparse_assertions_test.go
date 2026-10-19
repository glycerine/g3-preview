package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialSparseNullSkeletonAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			s, e := NewPolynomialSparseInterpolator(full, nil)
			serviceMust(t, e)
			choices := action + action
			if action == "disabled" {
				choices = ""
			}
			polynomialAssertionActionsFixture(t, choices)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			in := unumeral.NewInt(1)
			_, addErr := s.Add(&in, Polynomial{})
			out, e := NewPolynomialRef(full.core)
			serviceMust(t, e)
			_, mkErr := s.Interpolate(out)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(addErr, ErrPolynomialAssertion) || !errors.Is(mkErr, ErrPolynomialAssertion) {
					t.Fatal("null skeleton assertion did not throw", addErr, mkErr)
				}
			} else if !errors.Is(addErr, ErrPolynomialOwner) || !errors.Is(mkErr, ErrPolynomialOwner) {
				t.Fatal("Continue/release did not reach safe borrowed-owner boundary", addErr, mkErr)
			}
			serviceMust(t, out.Close())
			serviceMust(t, s.CloseChecked())
		})
	}
}

func TestPolynomialSparseInputLimitAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			m := full.core
			serviceMust(t, m.modular.SetZPUint64(97))
			p := polynomialQueryTerms(t, m, []int32{1}, [][]MonomialPower{{{0, 1}}})
			q := polynomialQueryTerms(t, m, []int32{2}, [][]MonomialPower{nil})
			sk, e := NewPolynomialSkeleton(m, p, 0)
			serviceMust(t, e)
			s, e := NewPolynomialSparseInterpolator(full, sk)
			serviceMust(t, e)
			in := unumeral.NewInt(2)
			ok, e := s.Add(&in, q)
			serviceMust(t, e)
			if !ok {
				t.Fatal("first source sample")
			}
			choices := action
			if action == "disabled" {
				choices = ""
			}
			polynomialAssertionActionsFixture(t, choices)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			ok, e = s.Add(&in, q)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrPolynomialAssertion) || ok || len(s.inputs) != 1 {
					t.Fatal("input limit assertion did not throw before publication", ok, e, len(s.inputs))
				}
			} else if e != nil || !ok || len(s.inputs) != 2 {
				t.Fatal("Continue/release did not publish the following input", ok, e, len(s.inputs))
			}
			serviceMust(t, s.CloseChecked())
			serviceMust(t, sk.CloseChecked())
		})
	}
}
