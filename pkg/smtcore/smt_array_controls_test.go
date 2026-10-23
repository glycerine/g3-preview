package smtcore

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTArrayBooleanGateEnodesRetainAntecedentsAndScope(t *testing.T) {
	for _, operation := range []int32{ast.ASTArrayOpSelect, ast.ASTArrayOpDefault} {
		for _, truth := range []bool{false, true} {
			t.Run(fmt.Sprint(operation, "/", truth), func(t *testing.T) {
				m := astTestManager(t)
				family, err := m.RegisterArrayPlugin()
				astTestMust(t, err)
				_, _, state := smtInternalizerFixture(t, m, nil)
				theory, err := NewSMTArrayTheory(family)
				astTestMust(t, err)
				astTestMust(t, state.RegisterTheory(theory))
				boolean, err := m.MkBoolSort()
				astTestMust(t, err)
				sort, err := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{boolean}), boolean)
				astTestMust(t, err)
				array, err := m.MkConst(astTestSymbol(t, m, "array-Boolean-gate"), sort)
				astTestMust(t, err)
				arguments := []ast.ASTHandle{array}
				if operation == ast.ASTArrayOpSelect {
					index, err := m.MkTrue()
					astTestMust(t, err)
					arguments = append(arguments, index)
				}
				expression, err := m.MkFamilyApp(family, operation, nil, arguments)
				astTestMust(t, err)
				astTestMust(t, state.PushScope())
				astTestMust(t, state.Internalize(expression, true))
				literal, err := state.Literal(expression)
				astTestMust(t, err)
				if !truth {
					literal = literal.Negated()
				}
				data, err := state.BooleanData(literal.Var())
				astTestMust(t, err)
				if !data.IsENode() {
					t.Fatal("Boolean array term lost its source enode flag in gate context")
				}
				astTestMust(t, state.AssignLiteral(literal, NewSMTBooleanAxiom()))
				_, err = state.PropagateCore()
				astTestMust(t, err)
				resolver, err := NewSMTConflictResolver(state)
				astTestMust(t, err)
				t.Cleanup(func() { astTestMust(t, resolver.Close()) })
				premise, err := resolver.expander.LiteralForENode(state.context.enodes[expression])
				astTestMust(t, err)
				if premise != literal {
					t.Fatal("Boolean enode antecedent polarity", premise, literal)
				}
				astTestMust(t, state.PopScopes(1))
				if _, found := state.BoolVar(expression); found || state.context.enodes[expression] != nil {
					t.Fatal("child array gate survived scope pop")
				}
			})
		}
	}
}
