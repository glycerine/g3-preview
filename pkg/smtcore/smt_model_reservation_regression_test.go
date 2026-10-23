package smtcore

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTModelGeneratorReservesExistingValuesBeforeFreshAllocation(t *testing.T) {
	for _, existingFirst := range []bool{false, true} {
		t.Run(fmt.Sprint(existingFirst), func(t *testing.T) {
			m, config := smtModelGeneratorFixture(t)
			sort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "reservation-sort"), nil)
			astTestMust(t, err)
			value, err := m.MkModelValue(0, sort)
			astTestMust(t, err)
			constant, err := m.MkConst(astTestSymbol(t, m, "reservation-constant"), sort)
			astTestMust(t, err)
			context, err := NewSMTCheckerContext(m)
			astTestMust(t, err)
			state, err := NewSMTInternalizerState(m, context, nil)
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, state.Close()) })
			ordered := []ast.ASTHandle{constant, value}
			if existingFirst {
				ordered[0], ordered[1] = ordered[1], ordered[0]
			}
			var freshNode, existingNode *SMTEnode
			for _, expression := range ordered {
				node, err := state.MkEnode(expression, false, false, true)
				astTestMust(t, err)
				astTestMust(t, context.SetRelevant(expression, true))
				if expression == constant {
					freshNode = node
				} else {
					existingNode = node
				}
			}
			generator, err := NewSMTModelGenerator(m)
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, generator.Close()) })
			astTestMust(t, generator.SetContext(context))
			astTestMust(t, generator.InitializeModel(config))
			astTestMust(t, generator.BuildStateValues(state))
			freshValue, err := generator.Value(freshNode)
			astTestMust(t, err)
			existingValue, err := generator.Value(existingNode)
			astTestMust(t, err)
			if existingValue != value {
				t.Fatal("existing model value was changed")
			}
			if freshValue == value {
				t.Fatal("fresh fallback value collided with an existing model value")
			}
		})
	}
}
