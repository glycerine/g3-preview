package smtcore

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

type smtModelDependencyProcedure struct {
	dependencies []SMTModelValueDependency
}

func (p *smtModelDependencyProcedure) Dependencies() []SMTModelValueDependency {
	return append([]SMTModelValueDependency(nil), p.dependencies...)
}
func (*smtModelDependencyProcedure) MakeValue([]ast.ASTHandle) (ast.ASTHandle, error) {
	return ast.ASTHandle{}, nil
}
func (*smtModelDependencyProcedure) IsFresh() bool { return false }

func TestSMTModelGeneratorFreshSourcesAndDependencyOrder(t *testing.T) {
	m := astTestManager(t)
	sort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "model-generator-sort"), nil)
	astTestMust(t, err)
	context, err := NewSMTCheckerContext(m)
	astTestMust(t, err)
	state, err := NewSMTInternalizerState(m, context, nil)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, state.Close()) })
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	nodes := make([]*SMTEnode, 3)
	for index, name := range []string{"a", "b", "c"} {
		nodeSort := sort
		if index == 2 {
			nodeSort = boolSort
		}
		expression, makeErr := m.MkConst(astTestSymbol(t, m, "model-generator-"+name), nodeSort)
		astTestMust(t, makeErr)
		nodes[index], makeErr = state.MkEnode(expression, false, false, true)
		astTestMust(t, makeErr)
	}
	generator, err := NewSMTModelGenerator(m)
	astTestMust(t, err)
	fresh, err := generator.MkExtraFreshValue(sort)
	astTestMust(t, err)
	freshProcedure, err := NewSMTFreshValueProcedure(fresh)
	astTestMust(t, err)
	bDependency, err := NewSMTEnodeValueDependency(nodes[1])
	astTestMust(t, err)
	procedures := map[*SMTEnode]SMTModelValueProcedure{
		nodes[0]: &smtModelDependencyProcedure{},
		nodes[1]: freshProcedure,
		nodes[2]: &smtModelDependencyProcedure{dependencies: []SMTModelValueDependency{bDependency}},
	}
	ordered, err := generator.topSortSources(nodes, procedures)
	astTestMust(t, err)
	if len(ordered) != 4 {
		t.Fatal("source count", len(ordered))
	}
	first, err := ordered[0].Enode()
	astTestMust(t, err)
	second, err := ordered[1].FreshValue()
	astTestMust(t, err)
	third, err := ordered[2].Enode()
	astTestMust(t, err)
	fourth, err := ordered[3].Enode()
	astTestMust(t, err)
	if first != nodes[0] || second != fresh || third != nodes[1] || fourth != nodes[2] {
		t.Fatal("dependency order", first, second, third, fourth)
	}
	if _, err = generator.MkExtraFreshValue(boolSort); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("finite fresh placeholder", err)
	}
}

func TestSMTModelGeneratorRejectsDependencyCycle(t *testing.T) {
	m := astTestManager(t)
	sort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "model-generator-cycle-sort"), nil)
	astTestMust(t, err)
	context, err := NewSMTCheckerContext(m)
	astTestMust(t, err)
	state, err := NewSMTInternalizerState(m, context, nil)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, state.Close()) })
	expressions := make([]ast.ASTHandle, 2)
	nodes := make([]*SMTEnode, 2)
	for index, name := range []string{"left", "right"} {
		expressions[index], err = m.MkConst(astTestSymbol(t, m, "model-cycle-"+name), sort)
		astTestMust(t, err)
		nodes[index], err = state.MkEnode(expressions[index], false, false, true)
		astTestMust(t, err)
	}
	left, err := NewSMTEnodeValueDependency(nodes[0])
	astTestMust(t, err)
	right, err := NewSMTEnodeValueDependency(nodes[1])
	astTestMust(t, err)
	procedures := map[*SMTEnode]SMTModelValueProcedure{
		nodes[0]: &smtModelDependencyProcedure{dependencies: []SMTModelValueDependency{right}},
		nodes[1]: &smtModelDependencyProcedure{dependencies: []SMTModelValueDependency{left}},
	}
	generator, err := NewSMTModelGenerator(m)
	astTestMust(t, err)
	if _, err = generator.topSortSources(nodes, procedures); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("dependency cycle", err)
	}
}
