package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

type smtModelFreshFallbackTheory struct {
	*smtInternalizerTestTheory
	modelValueCalls int
}

func (*smtModelFreshFallbackTheory) InitializeModel(*SMTModelGenerator) error { return nil }

func (t *smtModelFreshFallbackTheory) ModelValue(*SMTEnode, *SMTModelGenerator) (SMTModelValueProcedure, error) {
	t.modelValueCalls++
	return nil, ast.ErrASTDomain
}

func TestSMTModelGeneratorUsesFreshValueForUnattachedTheoryRoot(t *testing.T) {
	m, config := smtModelGeneratorFixture(t)
	family, err := m.ArithmeticFamilyID()
	astTestMust(t, err)
	sort, err := m.MkFamilySort(family, 0, nil)
	astTestMust(t, err)
	numeral, err := m.ArithmeticRealNumeralText("3")
	astTestMust(t, err)
	context, err := NewSMTCheckerContext(m)
	astTestMust(t, err)
	state, err := NewSMTInternalizerState(m, context, nil)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, state.Close()) })
	base, err := NewSMTTheoryBase(family)
	astTestMust(t, err)
	theory := &smtModelFreshFallbackTheory{smtInternalizerTestTheory: &smtInternalizerTestTheory{SMTTheoryBase: base}}
	astTestMust(t, state.RegisterTheory(theory))
	node, err := state.MkEnode(numeral, false, false, true)
	astTestMust(t, err)
	astTestMust(t, context.SetRelevant(numeral, true))
	if node.TheoryVar(family) != SMTNullTheoryVar {
		t.Fatal("fixture root unexpectedly has a theory variable")
	}
	generator, err := NewSMTModelGenerator(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, generator.Close()) })
	astTestMust(t, generator.SetContext(context))
	astTestMust(t, generator.InitializeModel(config))
	astTestMust(t, generator.BuildStateValues(state))
	value, err := generator.Value(node)
	astTestMust(t, err)
	private, err := m.IsModelValue(value)
	astTestMust(t, err)
	valueSort, err := m.GetSort(value)
	astTestMust(t, err)
	if theory.modelValueCalls != 0 || !private || value == numeral || valueSort != sort {
		t.Fatal("unattached theory root did not use the fresh-value procedure", theory.modelValueCalls, value, private, valueSort)
	}
}

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
