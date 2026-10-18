package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"testing"
)

func TestAlgebraicPredicateSourceAssertionActions(t *testing.T) {
	for _, rational := range []bool{false, true} {
		for _, action := range []string{"T", "C", "disabled"} {
			name := "integer/" + action
			if rational {
				name = "rational/" + action
			}
			t.Run(name, func(t *testing.T) {
				c, _, _ := algebraicCompareFixture(t)
				serviceMust(t, c.beginPredicate())
				two, upper := unumeral.NewBinaryInt(2), unumeral.NewBinaryInt(1)
				serviceMust(t, c.floorPredicateCandidate(&two))
				if rational {
					serviceMust(t, c.beginPredicateQ())
					one := unumeral.NewInt(1)
					serviceMust(t, c.setPredicateQParts(&c.predicate.candidate, &one))
				}
				choices := action
				if action == "disabled" {
					choices = ""
				}
				polynomialAssertionActionsFixture(t, choices)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				e := c.predicateUpperAssert(&upper, rational)
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrAlgebraicAssertion) || !linearThrownAssertion(e) {
						t.Fatal("SASSERT did not honor Throw", e)
					}
				} else {
					serviceMust(t, e)
				}
				if c.predicateDiagnosticRunning || c.predicate.arithmeticExecuting || c.predicateQ.executing {
					t.Fatal("assertion execution span retained")
				}
				if rational {
					e = c.finishPredicateQ(e)
				}
				_ = c.finishPredicate(e)
				serviceMust(t, c.CleanupRetained())
			})
		}
	}
}
func TestAlgebraicPredicateBasicAssertionActions(t *testing.T) {
	for _, line := range []int32{251, 306} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(action, func(t *testing.T) {
				c, _, _ := algebraicCompareFixture(t)
				root := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
				choices := action
				if action == "disabled" {
					choices = ""
				}
				polynomialAssertionActionsFixture(t, choices)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				e := c.predicateBasicAssert(&root, line)
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrAlgebraicAssertion) || !linearThrownAssertion(e) {
						t.Fatal("basic SASSERT did not honor Throw", e)
					}
				} else {
					serviceMust(t, e)
				}
				serviceMust(t, c.Del(&root))
			})
		}
	}
}
func TestAlgebraicPredicateDisabledAssertionsEraseNumericalOperands(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	polynomialAssertionActionsFixture(t, "")
	if polynomialSourceDebug {
		serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	}
	before, local := c.predicate, c.predicateQ
	used := c.config.Rationals.Budget().Used()
	c.config.Rationals.Budget().FailAfter(0)
	// No predicate/candidate is constructed, and nil operands cannot be read.
	for _, rational := range []bool{false, true} {
		serviceMust(t, c.predicateUpperAssert(nil, rational))
	}
	serviceMust(t, c.predicateBasicAssert(nil, 251))
	c.config.Rationals.Budget().FailAfter(-1)
	if !reflect.DeepEqual(before, c.predicate) || !reflect.DeepEqual(local, c.predicateQ) || used != c.config.Rationals.Budget().Used() {
		t.Fatal("erased assertion entered numerical scope or acquired allocation")
	}
}

func TestAlgebraicPredicateEnabledAssertionOwnsNumericalRefusal(t *testing.T) {
	c, _, runtime := algebraicNegTLSFixture(t, nil)
	z, b := c.config.Rationals.Integers(), c.config.Binary
	var n unumeral.Int
	var upper unumeral.BinaryRational
	serviceMust(t, z.SetString(&n, "36893488147419103235"))
	serviceMust(t, b.SetIntegerPower(&upper, &n, 1))
	serviceMust(t, z.Del(&n))
	serviceMust(t, c.beginPredicate())
	serviceMust(t, c.floorPredicateCandidate(&upper))
	polynomialAssertionActionsFixture(t, "")
	pad := algebraicCellSourceTLSBoundary(t, c)
	e := c.predicateUpperAssert(&upper, false)
	if polynomialSourceDebug {
		if e == nil || !numericPhysicalReservationRefusal(e) || !z.MatchesCopyAllocationInvocation(&c.predicate.integerComparison) {
			t.Fatal("enabled SASSERT did not reach actual comparison copy refusal", e)
		}
	} else {
		serviceMust(t, e)
		if c.predicate.integerComparison != (unumeral.IntegerCopyAllocationReceipt{}) {
			t.Fatal("erased SASSERT acquired comparison")
		}
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	_ = c.finishPredicate(e)
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, b.Del(&upper))
}

func TestAlgebraicMagnitudeSignAssertionActionsAndShortCircuit(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			c, _, _ := algebraicCompareFixture(t)
			choices := action
			if action == "disabled" {
				choices = ""
			}
			polynomialAssertionActionsFixture(t, choices)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			lower, upper := unumeral.NewBinaryInt(-1), unumeral.NewBinaryInt(1)
			e := c.magnitudeSignAssert(&lower, &upper)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrAlgebraicAssertion) || !linearThrownAssertion(e) {
					t.Fatal("magnitude SASSERT did not honor Throw", e)
				}
			} else {
				serviceMust(t, e)
			}
			if c.predicateDiagnosticRunning {
				t.Fatal("assertion retained execution guard")
			}
			zero := unumeral.NewBinaryInt(0)
			// A true first operand must not inspect the invalid second operand.
			serviceMust(t, c.magnitudeSignAssert(&zero, nil))
			if action == "disabled" || !polynomialSourceDebug {
				serviceMust(t, c.magnitudeSignAssert(nil, nil))
			}
		})
	}
}
