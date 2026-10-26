package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTModelGeneratorFallbackPreservesOnlyModelValues(t *testing.T) {
	m, config := smtModelGeneratorFixture(t)
	numeral, err := m.ArithmeticRealNumeralText("3")
	astTestMust(t, err)
	sort, err := m.GetSort(numeral)
	astTestMust(t, err)
	modelValue, err := m.MkModelValue(5, sort)
	astTestMust(t, err)
	context, err := NewSMTCheckerContext(m)
	astTestMust(t, err)
	state, err := NewSMTInternalizerState(m, context, nil)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, state.Close()) })
	family, err := m.ArithmeticFamilyID()
	astTestMust(t, err)
	dummy, err := NewSMTDummyTheory(family, "no arithmetic")
	astTestMust(t, err)
	astTestMust(t, state.RegisterTheory(dummy))
	var nodes []*SMTEnode
	for _, expression := range []ast.ASTHandle{numeral, modelValue} {
		node, err := state.MkEnode(expression, false, false, true)
		astTestMust(t, err)
		astTestMust(t, context.SetRelevant(expression, true))
		nodes = append(nodes, node)
	}
	g, err := NewSMTModelGenerator(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, g.Close()) })
	astTestMust(t, g.SetContext(context))
	astTestMust(t, g.InitializeModel(config))
	astTestMust(t, g.BuildStateValues(state))
	value, err := g.Value(nodes[0])
	astTestMust(t, err)
	private, err := m.IsModelValue(value)
	astTestMust(t, err)
	valueSort, err := m.GetSort(value)
	astTestMust(t, err)
	if !private || value == numeral || valueSort != sort {
		t.Fatal("disabled-theory numeral did not use model-value fallback", value, private, valueSort)
	}
	value, err = g.Value(nodes[1])
	astTestMust(t, err)
	if value != modelValue {
		t.Fatal("existing AST model value was replaced", value, modelValue)
	}
}
