// Source: algebraic_numbers.cpp:1761-1762.
package polynomial

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestAlgebraicComparePostRefinementAssertionActions(t *testing.T) {
	for _, line := range []int32{1761, 1762} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(fmt.Sprint(line, "/", action), func(t *testing.T) {
				c, _, _ := algebraicCompareFixtureProfile(t, true)
				var zero Algebraic
				choices := action
				if action == "disabled" {
					choices = ""
				}
				polynomialAssertionActionsFixture(t, choices)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				expression := "!a.is_basic()"
				if line == 1762 {
					expression = "!b.is_basic()"
				}
				body := c.assertComparisonNonbasic(&zero, line, expression)
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(body, ErrAlgebraicAssertion) || !linearThrownAssertion(body) {
						t.Fatal("post-refinement SASSERT did not honor Throw", body)
					}
				} else {
					serviceMust(t, body)
				}
			})
		}
	}
}

func TestAlgebraicCompareRefinementReachesConcreteAssertionThrow(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions erased")
	}
	for _, first := range []bool{false, true} {
		t.Run(fmt.Sprint(first), func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			rational := algebraicCompareRoot(t, c, []int32{-1, 0, 1}, 0, 0, 2, 0, true)
			irrational := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
			polynomialAssertionActionsFixture(t, "T")
			a, b := &irrational, &rational
			if first {
				a, b = &rational, &irrational
			}
			_, body := c.Compare(a, b)
			if !errors.Is(body, ErrAlgebraicAssertion) || !linearThrownAssertion(body) || c.compareRefine != 1 {
				t.Fatal("minimal comparison did not reach its actual source SASSERT", body, c.compareRefine)
			}
			l, e := c.location(&rational)
			serviceMust(t, e)
			if l == nil || l.kind != 0 {
				t.Fatal("assertion preceded actual rational promotion")
			}
			serviceMust(t, c.CleanupRetained())
			serviceMust(t, c.Del(&irrational))
			serviceMust(t, c.Del(&rational))
		})
	}
}

func TestAlgebraicCompareAssertionGatePrecedesOperandRead(t *testing.T) {
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	polynomialAssertionActionsFixture(t, "")
	if polynomialSourceDebug {
		serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	}
	serviceMust(t, c.assertComparisonNonbasic(nil, 1761, "!a.is_basic()"))
	serviceMust(t, c.assertComparisonNonbasic(nil, 1762, "!b.is_basic()"))
	if !polynomialSourceDebug {
		return
	}
	d := diagnostic.BorrowProcessDebugContext()
	serviceMust(t, d.EnableAssertions(true))
	serviceMust(t, d.Close())
	if e := c.assertComparisonNonbasic(nil, 1761, "!a.is_basic()"); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("invalid assertion switch read operand first", e)
	}
}

func TestAlgebraicCompareAssertionStatementsRequeryDynamicGate(t *testing.T) {
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	polynomialAssertionActionsFixture(t, "C")
	var zero Algebraic
	serviceMust(t, c.assertComparisonNonbasic(&zero, 1761, "!a.is_basic()"))
	if polynomialSourceDebug {
		serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	}
	serviceMust(t, c.assertComparisonNonbasic(nil, 1762, "!b.is_basic()"))
}

func TestAlgebraicCompareEntryAssertionShortCircuitAndGate(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			choices := action
			if action == "disabled" {
				choices = ""
			}
			polynomialAssertionActionsFixture(t, choices)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			var zero Algebraic
			body := c.assertComparisonPair(&zero, nil)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(body, ErrAlgebraicAssertion) || !linearThrownAssertion(body) {
					t.Fatal("entry SASSERT did not short circuit to source Throw", body)
				}
			} else {
				serviceMust(t, body)
			}
			if !polynomialSourceDebug || action == "disabled" {
				serviceMust(t, c.assertComparisonPair(nil, nil))
			}
		})
	}
}

func TestAlgebraicCompareEntrySwitchPrecedesCellMembers(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions erased")
	}
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	a := algebraicQueryRawCell(t, c, 1)
	b := algebraicQueryRawCell(t, c, 1)
	polynomialAssertionActionsFixture(t, "")
	serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	if _, e := c.Compare(&a, &b); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("comparison read source members before entry assertion switch", e)
	}
	for _, value := range []*Algebraic{&a, &b} {
		l, e := c.locationCleanup(value)
		serviceMust(t, e)
		for offset := uint64(0); offset < 56; offset += 4 {
			if _, e := c.config.Allocator.ReadUint32(l.raw, offset); !errors.Is(e, umemory.ErrRuntimeUninitialized) {
				t.Fatal("entry switch initialized source member", offset, e)
			}
		}
	}
}
