package polynomial

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestMonomialPowerSourceAssertionActions(t *testing.T) {
	for _, choice := range []string{"T", "C"} {
		t.Run(choice, func(t *testing.T) {
			m := monomialFixture(t)
			polynomialAssertionActionsFixture(t, strings.Repeat(choice, 3))
			a := monomialReadProfileObject(t, m, 2)
			monomialReadWrite(t, m, a, 12, 1)
			monomialReadWrite(t, m, a, 28, 7)
			monomialReadWrite(t, m, a, 32, 9)
			power, pe := m.ReadPower(a, 1)
			variable, ve := m.varCore(a, 1)
			degree, de := m.degreeAtCore(a, 1)
			if polynomialSourceDebug && choice == "T" {
				for _, e := range []error{pe, ve, de} {
					if !errors.Is(e, ErrMonomialAssertion) {
						t.Fatal("source Throw failed to stop the power read", e)
					}
				}
			} else {
				serviceMust(t, pe)
				serviceMust(t, ve)
				serviceMust(t, de)
				if power != (MonomialPower{Var: 7, Degree: 9}) || variable != 7 || degree != 9 {
					t.Fatal("Continue/release lost physical power fields", power, variable, degree)
				}
			}
			monomialPhysicalAudit(t, m)
		})
	}
}

func TestMonomialPowerSourceAssertionSwitchAndClosedDebug(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source SASSERT erased")
	}
	m := monomialFixture(t)
	polynomialAssertionActionsFixture(t, "T")
	d := diagnostic.BorrowProcessDebugContext()
	serviceMust(t, d.EnableAssertions(false))
	a := monomialReadProfileRawObject(t, m, 1)
	monomialReadWrite(t, m, a, 24, 9)
	degree, e := m.degreeAtCore(a, 0)
	serviceMust(t, e)
	if degree != 9 {
		t.Fatal("disabled assertion read the logical size or variable", degree)
	}
	serviceMust(t, d.Close())
	if _, e := m.degreeAtCore(a, 0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed switch owner was not read before the degree", e)
	}
}
