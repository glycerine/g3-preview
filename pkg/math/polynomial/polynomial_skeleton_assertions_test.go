package polynomial

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestPolynomialSkeletonPowerAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C"} {
		t.Run(action, func(t *testing.T) {
			m := polynomialCellFixture(t)
			p, _ := polynomialSkeletonInput(t, m)
			s, e := NewPolynomialSkeleton(m, p, 0)
			serviceMust(t, e)
			a, e := s.Entry(0)
			serviceMust(t, e)
			index := a.FirstPower + a.NumPowers
			if uint64(index) >= uint64(len(s.powers)) {
				t.Fatal("fixture needs a following physical source field")
			}
			polynomialAssertionActionsFixture(t, strings.Repeat(action, 3))
			power, pe := s.Power(a, a.NumPowers)
			original, oe := s.Original(a, a.NumPowers)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(pe, ErrPolynomialAssertion) || !errors.Is(oe, ErrPolynomialAssertion) {
					t.Fatal("ith accessor assertion did not throw", pe, oe)
				}
			} else {
				serviceMust(t, pe)
				serviceMust(t, oe)
				if power != s.powers[index] || original != s.originals[index] {
					t.Fatal("Continue/release did not read following physical fields", power, original)
				}
			}
			serviceMust(t, s.CloseChecked())
			serviceMust(t, m.DecRef(p))
		})
	}
}

func TestPolynomialSkeletonVectorBoundsAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C"} {
		t.Run(action, func(t *testing.T) {
			m := polynomialCellFixture(t)
			v, e := NewPolynomialSkeletonVector(m)
			serviceMust(t, e)
			polynomialAssertionActionsFixture(t, strings.Repeat(action, 3))
			_, e = v.Get(0)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrPolynomialAssertion) {
					t.Fatal("vector index assertion did not throw", e)
				}
			} else if !errors.Is(e, arena.ErrBounds) {
				t.Fatal("Continue/release did not reach native bounds refusal", e)
			}
			e = v.Pop()
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrPolynomialAssertion) {
					t.Fatal("empty pop assertion did not throw", e)
				}
			} else if !errors.Is(e, arena.ErrBounds) {
				t.Fatal("Continue/release did not reach wrapped vector bound", e)
			}
			serviceMust(t, v.CloseChecked())
		})
	}
}
