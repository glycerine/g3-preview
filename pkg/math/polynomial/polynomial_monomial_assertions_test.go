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

func TestMonomialPredecessorSourceAssertionActions(t *testing.T) {
	for _, choice := range []string{"T", "C", "disabled"} {
		t.Run(choice, func(t *testing.T) {
			m := monomialFixture(t)
			actions := choice
			if choice == "disabled" {
				actions = "T"
			}
			polynomialAssertionActionsFixture(t, strings.Repeat(actions, 2))
			if polynomialSourceDebug && choice == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			// Actual raw variables, with the predecessor just above the matched
			// midpoint. No placement-constructor validity is fabricated.
			a := monomialReadProfileObject(t, m, 8)
			for i, variable := range []uint32{0, 2, 7, 6, 8, 10, 12, 14} {
				monomialReadWrite(t, m, a, 20+uint64(i)*8, variable)
			}
			v, e := m.MaxSmallerThan(a, 6)
			if polynomialSourceDebug && choice == "T" {
				if !errors.Is(e, ErrMonomialAssertion) {
					t.Fatal("midpoint assertion did not Throw", v, e)
				}
			} else {
				serviceMust(t, e)
				if v != 7 {
					t.Fatal("Continue/disabled/release changed reached predecessor", v)
				}
			}
			monomialPhysicalAudit(t, m)
		})
	}
}

func TestMonomialPredecessorResultAssertionActions(t *testing.T) {
	for _, choice := range []string{"T", "C", "disabled"} {
		t.Run(choice, func(t *testing.T) {
			m := monomialFixture(t)
			action := choice
			if choice == "disabled" {
				action = "T"
			}
			polynomialAssertionActionsFixture(t, action)
			if polynomialSourceDebug && choice == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}

			// The binary-search path returns 10, while the deliberately
			// malformed physical entry at index 1 is also below 12 and above
			// that result. This reaches the source assertion at line 368
			// through MaxSmallerThan itself; no helper-only failure is used.
			a := monomialReadProfileObject(t, m, 8)
			for i, variable := range []uint32{0, 11, 4, 6, 8, 9, 10, 14} {
				monomialReadWrite(t, m, a, 20+uint64(i)*8, variable)
			}
			v, e := m.MaxSmallerThan(a, 12)
			if polynomialSourceDebug && choice == "T" {
				if !errors.Is(e, ErrMonomialAssertion) {
					t.Fatal("result assertion did not Throw", v, e)
				}
			} else {
				serviceMust(t, e)
				if v != 10 {
					t.Fatal("Continue/disabled/release changed reached predecessor", v)
				}
			}
			monomialPhysicalAudit(t, m)
		})
	}
}

func TestMonomialPredecessorSourceNullAssertionAndClosedSwitch(t *testing.T) {
	for _, choice := range []string{"T", "C", "disabled"} {
		t.Run(choice, func(t *testing.T) {
			m := monomialFixture(t)
			actions := choice
			if choice == "disabled" {
				actions = "T"
			}
			polynomialAssertionActionsFixture(t, actions)
			if polynomialSourceDebug && choice == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			a := monomialReadProfileObject(t, m, 1)
			monomialReadWrite(t, m, a, 20, 7)
			v, e := m.MaxSmallerThan(a, ^uint32(0))
			if polynomialSourceDebug && choice == "T" {
				if !errors.Is(e, ErrMonomialAssertion) {
					t.Fatal("null variable assertion did not Throw", e)
				}
			} else {
				serviceMust(t, e)
				if v != 7 {
					t.Fatal("Continue/disabled/release predecessor", v)
				}
			}
			if polynomialSourceDebug {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
				if _, e := m.MaxSmallerThan(Monomial{}, 0); !errors.Is(e, arena.ErrClosed) {
					t.Fatal("entry assertion switch was not read before monomial size", e)
				}
			}
		})
	}
}

func TestMonomialPredecessorAssertionOperandReadGates(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source SASSERT erased")
	}
	for _, choice := range []string{"T", "C", "disabled"} {
		t.Run(choice, func(t *testing.T) {
			m := monomialFixture(t)
			actions := choice
			if choice == "disabled" {
				actions = "T"
			}
			polynomialAssertionActionsFixture(t, strings.Repeat(actions, 2))
			a := monomialReadProfileRawObject(t, m, 8)
			if choice == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				serviceMust(t, m.searchVariableAssert(a, 0, 1, true, 327, "get_var(low) < x"))
				serviceMust(t, m.searchPreviousAssert(a, 3, 6))
			}
			// Both source expressions short circuit before any uninitialized
			// header/array field. Continue must not evaluate the skipped operand.
			window := m.searchWindowAssert(a, 5, 7, 0, true)
			result := m.searchResultAssert(a, 0, ^uint32(0))
			for _, e := range []error{window, result} {
				if choice == "T" {
					if !errors.Is(e, ErrMonomialAssertion) {
						t.Fatal("short-circuited source assertion read an unrelated field", e)
					}
				} else {
					serviceMust(t, e)
				}
			}
		})
	}
}
