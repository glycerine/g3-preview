package smtcore

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTFailedConsequentConflictRetainsTrueOppositeAndFalseLemma(t *testing.T) {
	for _, binaryPropagation := range []bool{false, true} {
		t.Run(fmt.Sprint(binaryPropagation), func(t *testing.T) {
			m := astTestManager(t)
			_, _, state := smtInternalizerFixture(t, m, nil)
			astTestMust(t, state.SetBinaryClauseOptimization(true))
			a, b := smtRelevancyBool(t, m, "polarity-a"), smtRelevancyBool(t, m, "polarity-b")
			astTestMust(t, state.Internalize(a, true))
			astTestMust(t, state.Internalize(b, true))
			la, err := state.Literal(a)
			astTestMust(t, err)
			lb, err := state.Literal(b)
			astTestMust(t, err)
			if binaryPropagation {
				astTestMust(t, state.addClause([]SMTLiteral{la.Negated(), lb.Negated()}, false, ast.ASTHandle{}))
			}
			astTestMust(t, state.PushScope())
			astTestMust(t, state.assignDecisionLiteral(la))
			astTestMust(t, state.PushScope())
			astTestMust(t, state.assignDecisionLiteral(lb))
			if binaryPropagation {
				consistent, err := state.PropagateCore()
				astTestMust(t, err)
				if consistent {
					t.Fatal("binary contradiction was not detected")
				}
			} else {
				// b justifies ~a, but a is already assigned true.
				astTestMust(t, state.AssignLiteral(la.Negated(), NewSMTBooleanBinaryClause(lb)))
			}
			conflict, found := state.Conflict()
			if !found {
				t.Fatal("missing failed-consequent conflict")
			}
			notLiteral := state.ConflictLiteral()
			assignment, err := state.Assignment(notLiteral)
			astTestMust(t, err)
			if assignment != lbool.True {
				t.Fatal("source not_l must be true, opposite to the failed consequent", notLiteral, assignment)
			}
			resolver, err := NewSMTConflictResolver(state)
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, resolver.Close()) })
			resolution, err := resolver.Resolve(conflict, notLiteral, 0)
			astTestMust(t, err)
			if !resolution.Resolved || resolution.ConflictLevel != 2 || resolution.NewScopeLevel != 1 || !reflect.DeepEqual(resolution.Lemma, []SMTLiteral{lb.Negated(), la.Negated()}) {
				t.Fatal("source first-UIP lemma or backjump level", resolution)
			}
			for _, literal := range resolution.Lemma {
				value, err := state.Assignment(literal)
				astTestMust(t, err)
				if value != lbool.False {
					t.Fatal("conflict analysis produced a non-false lemma literal", literal, value)
				}
			}
			astTestMust(t, state.PopScopes(1))
			astTestMust(t, state.AddLearnedClause(resolution.Lemma))
			value, err := state.Assignment(lb.Negated())
			astTestMust(t, err)
			if value != lbool.True {
				t.Fatal("learned clause failed to assert after backjump", value)
			}
		})
	}
}
