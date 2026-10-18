package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestUPolynomialSturmSignAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			choice := action
			if choice == "disabled" {
				choice = ""
			}
			polynomialAssertionActionsFixture(t, choice)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			// Qualified invariant adapter: retained sign evaluators produce -1,0,1.
			// No sequence coefficients or evaluator results are mutated to create 2.
			e := m.sturmVariationSignAssertion(2)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
					t.Fatal(e)
				}
			} else {
				serviceMust(t, e)
			}
			if m.factorTrace != nil {
				t.Fatal("sign assertion retained numeric borrow")
			}
		})
	}
}
func TestUPolynomialSturmVariationAssertionProfiles(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "enabled", true: "disabled"}[disabled], func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			p := upolySourceNew(t, m, []int32{-1, 0, 1})
			s := upolySturmSequence(t, m)
			polynomialAssertionActionsFixture(t, "")
			if polynomialSourceDebug && disabled {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			serviceMust(t, m.SturmSequence(mustUPolyView(t, m, p), s))
			for i, call := range []func() (uint32, error){func() (uint32, error) { return m.SignVariationsMinusInfinity(s) }, func() (uint32, error) { return m.SignVariationsPlusInfinity(s) }, func() (uint32, error) { return m.SignVariationsZero(s) }, func() (uint32, error) { b := unumeral.NewBinaryInt(2); return m.SignVariationsBinary(s, &b) }} {
				got, e := call()
				serviceMust(t, e)
				if got != []uint32{2, 0, 1, 0}[i] {
					t.Fatal("independent variations for x^2-1", i, got)
				}
			}
		})
	}
}
func TestUPolynomialSturmSignClosedGateAndEmptyReturn(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions compiled out")
	}
	m, _, _ := upolySourceOwner(t)
	s := upolySturmSequence(t, m)
	polynomialAssertionActionsFixture(t, "")
	serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	if e := m.sturmVariationSignAssertion(1); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("valid sign skipped runtime gate", e)
	}
	n, e := m.SignVariationsPlusInfinity(s)
	serviceMust(t, e)
	if n != 0 {
		t.Fatal("empty sequence did not take literal early return", n)
	}
}
