package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"testing"
)

func TestUPolynomialRefinementSourceAssertionActions(t *testing.T) {
	for _, route := range []string{"refine", "core", "precision", "core-precision"} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(route+"/"+action, func(t *testing.T) {
				m, q, budget := upolySourceOwner(t)
				bqm := upolyIsolationBinaryOwner(t, q, budget)
				// The zero polynomial and zero-width interval deliberately violate the
				// source sign precondition. Continue must still execute the literal body.
				p := upolySourceNew(t, m, []int32{0})
				v := mustUPolyView(t, m, p)
				a, b := unumeral.NewBinaryInt(0), unumeral.NewBinaryInt(0)
				choices := action
				if action == "C" && (route == "refine" || route == "precision") {
					choices = "CC"
				}
				if action == "disabled" {
					choices = ""
				}
				polynomialAssertionActionsFixture(t, choices)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				checkpoints := 0
				m.config.Checkpoint = func() bool { checkpoints++; return true }
				var interval bool
				var e error
				switch route {
				case "refine":
					interval, e = m.Refine(v, bqm, &a, &b)
				case "core":
					interval, e = m.RefineCore(v, 0, bqm, &a, &b)
				case "precision":
					interval, e = m.RefineToPrecision(v, bqm, &a, &b, 0)
				case "core-precision":
					interval, e = m.RefineCoreToPrecision(v, 0, bqm, &a, &b, 0)
				}
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) || checkpoints != 0 {
						t.Fatal("source Throw did not precede body", interval, e, checkpoints)
					}
				} else {
					serviceMust(t, e)
					precision := route == "precision" || route == "core-precision"
					if interval != precision || checkpoints != map[bool]int{false: 0, true: 1}[precision] {
						t.Fatal("continued/erased literal body", interval, checkpoints)
					}
				}
				serviceMust(t, bqm.Del(&a))
				serviceMust(t, bqm.Del(&b))
			})
		}
	}
}

func TestUPolynomialRefinementDisabledAssertionsEraseNumericalOperands(t *testing.T) {
	m, q, budget := upolySourceOwner(t)
	bqm := upolyIsolationBinaryOwner(t, q, budget)
	p := upolySourceNew(t, m, []int32{-2, 0, 1})
	v := mustUPolyView(t, m, p)
	a, b := unumeral.NewBinaryInt(1), unumeral.NewBinaryInt(2)
	polynomialAssertionActionsFixture(t, "")
	if polynomialSourceDebug {
		serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	}
	checkpoints := 0
	m.config.Checkpoint = func() bool { checkpoints++; return false }
	before, used := m.binaryEvaluation, budget.Used()
	budget.FailAfter(0)
	_, e := m.RefineCoreToPrecision(v, -1, bqm, &a, &b, 40)
	budget.FailAfter(-1)
	if !errors.Is(e, ErrUPolynomialCanceled) || checkpoints != 1 {
		t.Fatal("erased checks did not reach first checkpoint", e, checkpoints)
	}
	if !reflect.DeepEqual(before, m.binaryEvaluation) || used != budget.Used() {
		t.Fatal("erased SASSERT evaluated Horner operands or acquired storage")
	}
	serviceMust(t, bqm.Del(&a))
	serviceMust(t, bqm.Del(&b))
}

func TestUPolynomialRefinementEvaluatedAssertionContinuesLiteralBranch(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m, q, budget := upolySourceOwner(t)
			bqm := upolyIsolationBinaryOwner(t, q, budget)
			p := upolySourceNew(t, m, []int32{-2, 0, 1})
			v := mustUPolyView(t, m, p)
			a, b := unumeral.NewBinaryInt(1), unumeral.NewBinaryInt(2)
			choices := "CC"
			if action == "T" {
				choices = "T"
			}
			if action == "disabled" {
				choices = ""
			}
			polynomialAssertionActionsFixture(t, choices)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			// Deliberately supply the opposite sign. Continue must follow sign_mid ==
			// sign_a and publish 3/2 in a, even though it is no longer a valid bracket.
			interval, e := m.RefineCore(v, 1, bqm, &a, &b)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
					t.Fatal("evaluated source assertion did not Throw", e)
				}
				if a.K() != 0 || a.BorrowNumerator().Val != 1 {
					t.Fatal("Throw changed bound")
				}
			} else {
				serviceMust(t, e)
				if !interval || a.K() != 1 || a.BorrowNumerator().Val != 3 || b.K() != 0 || b.BorrowNumerator().Val != 2 {
					t.Fatal("continued source branch changed", interval, a, b)
				}
			}
			serviceMust(t, bqm.Del(&a))
			serviceMust(t, bqm.Del(&b))
		})
	}
}
