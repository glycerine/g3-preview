package smtcore

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/util/cooperate"
	"github.com/glycerine/g3/pkg/util/lbool"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

type smtModelDependencyProcedure struct {
	dependencies []SMTModelValueDependency
}

func (p *smtModelDependencyProcedure) Dependencies() []SMTModelValueDependency {
	return append([]SMTModelValueDependency(nil), p.dependencies...)
}
func (*smtModelDependencyProcedure) MakeValue(*SMTModelGenerator, []ast.ASTHandle) (ast.ASTHandle, error) {
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

func smtModelGeneratorFixture(t *testing.T) (*ast.ASTManager, model.ModelEvaluatorConfig) {
	t.Helper()
	symbols, numerals, budget := astTestOwners(t)
	context, err := uparams.NewParamsContext(symbols, numerals, uparams.ParamsConfig{Budget: budget})
	astTestMust(t, err)
	catalog, err := uparams.NewGlobalParamsCatalog(symbols, uparams.GlobalParamsCatalogConfig{Budget: budget})
	astTestMust(t, err)
	globals, err := uparams.NewGlobalParamsService(symbols, context, uparams.GlobalParamsConfig{Budget: budget, Catalog: catalog})
	astTestMust(t, err)
	astTestMust(t, globals.InitializeSource())
	params, err := context.NewValue()
	astTestMust(t, err)
	memory, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget})
	astTestMust(t, err)
	cooperation, err := cooperate.NewCooperation(budget)
	astTestMust(t, err)
	manager, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget})
	astTestMust(t, err)
	t.Cleanup(func() {
		astTestMust(t, manager.Close())
		astTestMust(t, cooperation.Close())
		astTestMust(t, memory.Close())
		astTestMust(t, context.Del(&params))
		astTestMust(t, globals.Close())
		astTestMust(t, catalog.Close())
		astTestMust(t, context.Close())
	})
	return manager, model.ModelEvaluatorConfig{Context: context, Globals: globals, Params: &params, Memory: memory, Cooperation: cooperation}
}

func TestSMTModelGeneratorBuildsAndPinsRootValues(t *testing.T) {
	m, config := smtModelGeneratorFixture(t)
	sort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "model-build-sort"), nil)
	astTestMust(t, err)
	expression, err := m.MkConst(astTestSymbol(t, m, "model-build-root"), sort)
	astTestMust(t, err)
	value, err := m.MkModelValue(0, sort)
	astTestMust(t, err)
	context, err := NewSMTCheckerContext(m)
	astTestMust(t, err)
	state, err := NewSMTInternalizerState(m, context, nil)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, state.Close()) })
	node, err := state.MkEnode(expression, false, false, true)
	astTestMust(t, err)
	procedure, err := NewSMTExpressionValueProcedure(value)
	astTestMust(t, err)
	generator, err := NewSMTModelGenerator(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, generator.Close()) })
	astTestMust(t, generator.SetContext(context))
	astTestMust(t, generator.InitializeModel(config))
	astTestMust(t, generator.BuildValues([]*SMTEnode{node}, map[*SMTEnode]SMTModelValueProcedure{node: procedure}))
	if got, getErr := generator.Value(node); getErr != nil || got != value {
		t.Fatal("built root value", got, getErr)
	}
	if err = generator.BuildValues([]*SMTEnode{node}, map[*SMTEnode]SMTModelValueProcedure{node: procedure}); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("duplicate build", err)
	}
}

func TestSMTModelGeneratorPublishesBooleanConstant(t *testing.T) {
	m, config := smtModelGeneratorFixture(t)
	sort, err := m.MkBoolSort()
	astTestMust(t, err)
	expression, err := m.MkConst(astTestSymbol(t, m, "model-build-boolean"), sort)
	astTestMust(t, err)
	context, err := NewSMTCheckerContext(m)
	astTestMust(t, err)
	state, err := NewSMTInternalizerState(m, context, nil)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, state.Close()) })
	variable, err := state.MkBoolVar(expression)
	astTestMust(t, err)
	_, err = state.MkEnode(expression, false, false, true)
	astTestMust(t, err)
	astTestMust(t, state.SetAssignment(variable, lbool.True))
	astTestMust(t, context.SetRelevant(expression, true))
	generator, err := NewSMTModelGenerator(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, generator.Close()) })
	astTestMust(t, generator.SetContext(context))
	astTestMust(t, generator.InitializeModel(config))
	astTestMust(t, generator.BuildStateValues(state))
	finalModel, err := generator.FinalModel()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, finalModel.Close()) })
	declaration, err := m.GetDecl(expression)
	astTestMust(t, err)
	value, err := finalModel.Core().ConstantInterpretation(declaration)
	astTestMust(t, err)
	if !m.IsTrue(value) {
		t.Fatal("Boolean model interpretation", value)
	}
}
