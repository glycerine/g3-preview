package polynomial

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestUPolynomialNormalizeIntervalSeparateActions(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("debug actions erased")
	}
	for _, action := range []string{"T", "CT", "CCT", "CCC"} {
		t.Run(action, func(t *testing.T) {
			m, q, budget := upolySourceOwner(t)
			bqm := upolyIsolationBinaryOwner(t, q, budget)
			p := upolySourceNew(t, m, []int32{-1, 1})
			a, b := unumeral.NewBinaryInt(-2), unumeral.NewBinaryInt(2)
			t.Cleanup(func() { serviceMust(t, bqm.Del(&a)); serviceMust(t, bqm.Del(&b)) })
			polynomialAssertionActionsFixture(t, action)
			// Public supplied-sign overload with an independently supplied wrong sign;
			// actual evaluators and polynomial coefficients are never overwritten.
			ok, e := m.NormalizeIntervalCore(mustUPolyView(t, m, p), 1, bqm, &a, &b)
			if action == "CCC" {
				serviceMust(t, e)
				if !ok {
					t.Fatal("literal Continue return")
				}
			} else if ok || !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
				t.Fatal("normalization action", ok, e)
			}
			wantB := "2"
			if action == "CCT" || action == "CCC" {
				wantB = "0"
			}
			if upolyIsolationBinaryRat(t, m, bqm, &a).RatString() != "-2" || upolyIsolationBinaryRat(t, m, bqm, &b).RatString() != wantB {
				t.Fatal("source reset/assertion frontier")
			}
			// CCC produces an invalid bracket from an invalid supplied sign; the true
			// return is source continuation behavior, not a numerical correctness claim.
		})
	}
}
func TestUPolynomialNormalizeIntervalRuntimeOracles(t *testing.T) {
	for _, constant := range []int32{-1, 0, 1} {
		for _, disabled := range []bool{false, true} {
			t.Run(fmt.Sprint(constant, disabled), func(t *testing.T) {
				m, q, budget := upolySourceOwner(t)
				bqm := upolyIsolationBinaryOwner(t, q, budget)
				p := upolySourceNew(t, m, []int32{constant, 1})
				a, b := unumeral.NewBinaryInt(-2), unumeral.NewBinaryInt(2)
				t.Cleanup(func() { serviceMust(t, bqm.Del(&a)); serviceMust(t, bqm.Del(&b)) })
				polynomialAssertionActionsFixture(t, "")
				if polynomialSourceDebug && disabled {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				ok, e := m.NormalizeInterval(mustUPolyView(t, m, p), bqm, &a, &b)
				serviceMust(t, e)
				wa, wb := "-2", "2"
				if constant < 0 {
					wa = "0"
				}
				if constant > 0 {
					wb = "0"
				}
				if ok != (constant != 0) || upolyIsolationBinaryRat(t, m, bqm, &a).RatString() != wa || upolyIsolationBinaryRat(t, m, bqm, &b).RatString() != wb {
					t.Fatal("independent normalized interval", ok, wa, wb)
				}
			})
		}
	}
}
func TestUPolynomialNormalizeIntervalErasureAndEarlyReturn(t *testing.T) {
	m, q, budget := upolySourceOwner(t)
	bqm := upolyIsolationBinaryOwner(t, q, budget)
	a, b := unumeral.NewBinaryInt(0), unumeral.NewBinaryInt(2)
	t.Cleanup(func() { serviceMust(t, bqm.Del(&a)); serviceMust(t, bqm.Del(&b)) })
	polynomialAssertionActionsFixture(t, "")
	// Non-straddling source return precedes operand evaluation/supplied sign use.
	ok, e := m.NormalizeIntervalCore(UPolynomialView{}, 99, bqm, &a, &b)
	serviceMust(t, e)
	if !ok {
		t.Fatal("early return")
	}
	if polynomialSourceDebug {
		serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	}
	serviceMust(t, m.normalizeIntervalFinalSignAssertion(UPolynomialView{}, nil, nil))
}

func TestUPolynomialNormalizeIntervalNonzeroAction(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("debug actions erased")
	}
	for _, action := range []string{"CCT", "CCC"} {
		t.Run(action, func(t *testing.T) {
			m, q, budget := upolySourceOwner(t)
			bqm := upolyIsolationBinaryOwner(t, q, budget)
			p := upolySourceNew(t, m, []int32{0, 1})
			a, b := unumeral.NewBinaryInt(-2), unumeral.NewBinaryInt(2)
			t.Cleanup(func() { serviceMust(t, bqm.Del(&a)); serviceMust(t, bqm.Del(&b)) })
			polynomialAssertionActionsFixture(t, action)
			// The caller supplies zero independently; actual endpoint evaluations are -1,+1.
			ok, e := m.NormalizeIntervalCore(mustUPolyView(t, m, p), 0, bqm, &a, &b)
			if action == "CCT" {
				if ok || !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
					t.Fatal(ok, e)
				}
			} else {
				serviceMust(t, e)
				if ok {
					t.Fatal("zero root must return false")
				}
			}
			if upolyIsolationBinaryRat(t, m, bqm, &a).RatString() != "-2" || upolyIsolationBinaryRat(t, m, bqm, &b).RatString() != "2" {
				t.Fatal("endpoint mutation before zero-root return")
			}
		})
	}
}
