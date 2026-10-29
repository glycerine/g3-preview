package ufbv

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/cooperate"
	uparams "github.com/glycerine/g3/pkg/util/params"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func macroTacticFixture(t *testing.T, proofs bool) (*ast.ASTManager, model.ModelEvaluatorConfig) {
	t.Helper()
	mode := ast.ASTProofDisabled
	if proofs {
		mode = ast.ASTProofEnabled
	}
	m := cacheFixtureProofMode(t, mode)
	context, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget(), PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: m.Budget()}}})
	cacheMust(t, err)
	catalog, err := uparams.NewGlobalParamsCatalog(m.Symbols, uparams.GlobalParamsCatalogConfig{Budget: m.Budget(), Descriptions: arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: m.Budget()}}, Text: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: m.Budget()}}})
	cacheMust(t, err)
	globals, err := uparams.NewGlobalParamsService(m.Symbols, context, uparams.GlobalParamsConfig{Budget: m.Budget(), Catalog: catalog, DescriptorPageSize: 2, DescriptorWordsPageSize: 16})
	cacheMust(t, err)
	cacheMust(t, globals.InitializeSource())
	t.Cleanup(func() {
		cacheMust(t, globals.Close())
		cacheMust(t, catalog.Close())
		cacheMust(t, context.Close())
	})
	cooperation, err := cooperate.NewCooperation(m.Budget())
	cacheMust(t, err)
	t.Cleanup(func() { cacheMust(t, cooperation.Close()) })
	return m, model.ModelEvaluatorConfig{Context: context, Globals: globals, Memory: m.NumeralService().RuntimeOwner(), Cooperation: cooperation}
}

func macroTacticFormula(t *testing.T, m *ast.ASTManager, quasi bool) (ast.ASTHandle, ast.ASTHandle, ast.ASTHandle) {
	t.Helper()
	boolean, err := m.MkBoolSort()
	cacheMust(t, err)
	variable, err := m.MkVar(0, boolean)
	cacheMust(t, err)
	name, err := m.Symbols.Intern("macro-tactic-f")
	cacheMust(t, err)
	domain := []ast.ASTHandle{boolean}
	if quasi {
		domain = append(domain, boolean)
	}
	declaration, err := m.MkFuncDecl(name, domain, boolean, ast.ASTFuncDeclInfo{})
	cacheMust(t, err)
	arguments := []ast.ASTHandle{variable}
	ground, err := m.MkConst(usymbol.Numerical(7002), boolean)
	cacheMust(t, err)
	if quasi {
		arguments = append(arguments, ground)
	}
	head, err := m.MkApp(declaration, arguments)
	cacheMust(t, err)
	definition, err := m.MkNot(variable)
	cacheMust(t, err)
	body, err := m.MkEq(head, definition)
	cacheMust(t, err)
	quantifier, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolean}, Names: []usymbol.Symbol{usymbol.Numerical(7001)}, Body: body})
	cacheMust(t, err)
	constant, err := m.MkConst(usymbol.Numerical(7003), boolean)
	cacheMust(t, err)
	applicationArgs := []ast.ASTHandle{constant}
	if quasi {
		applicationArgs = append(applicationArgs, ground)
	}
	application, err := m.MkApp(declaration, applicationArgs)
	cacheMust(t, err)
	return quantifier, application, constant
}

func TestMacroFinderTacticGoalProofCoreAndModelBoundary(t *testing.T) {
	m, services := macroTacticFixture(t, true)
	quantifier, application, constant := macroTacticFormula(t, m, false)
	dependencyTerm, err := m.MkConst(usymbol.Numerical(7004), mustBoolSort(t, m))
	cacheMust(t, err)
	dependency, err := m.DependencyLeaf(dependencyTerm)
	cacheMust(t, err)
	quantifierProof, err := m.MkProofAsserted(quantifier)
	cacheMust(t, err)
	applicationProof, err := m.MkProofAsserted(application)
	cacheMust(t, err)
	goal, err := tactic.NewGoal(m, true, true, true)
	cacheMust(t, err)
	t.Cleanup(func() { cacheMust(t, goal.Close()) })
	cacheMust(t, goal.AssertWithProof(quantifier, quantifierProof, dependency))
	cacheMust(t, goal.AssertWithProof(application, applicationProof, dependency))
	runner, err := NewMacroFinderTactic(m, services.Context, services.Globals, nil, services)
	cacheMust(t, err)
	t.Cleanup(func() { cacheMust(t, runner.Close()) })
	var result tactic.TacticResult
	t.Cleanup(func() { cacheMust(t, result.Close()) })
	cacheMust(t, runner.Apply(goal, &result))
	if size, err := goal.Size(); err != nil || size != 1 {
		t.Fatal("macro-finder residue", size, err)
	}
	got, err := goal.Form(0)
	cacheMust(t, err)
	want, err := m.MkNot(constant)
	cacheMust(t, err)
	if got != want {
		t.Fatal("macro-finder expansion", got, want)
	}
	if proof, err := goal.Proof(0); err != nil || proof.IsNull() {
		t.Fatal("macro-finder proof", proof, err)
	}
	if core, err := goal.Dependency(0); err != nil || core.IsNull() {
		t.Fatal("macro-finder dependency", core, err)
	}
	if result.ModelConverter == nil {
		t.Fatal("macro-finder omitted model reconstruction")
	}
}

func TestQuasiMacrosTacticRunsToFixedPoint(t *testing.T) {
	m, services := macroTacticFixture(t, false)
	quantifier, _, _ := macroTacticFormula(t, m, true)
	goal, err := tactic.NewGoal(m, false, false, false)
	cacheMust(t, err)
	t.Cleanup(func() { cacheMust(t, goal.Close()) })
	cacheMust(t, goal.Assert(quantifier))
	runner, err := NewQuasiMacrosTactic(m, services.Context, services.Globals, nil, services)
	cacheMust(t, err)
	t.Cleanup(func() { cacheMust(t, runner.Close()) })
	var result tactic.TacticResult
	t.Cleanup(func() { cacheMust(t, result.Close()) })
	cacheMust(t, runner.Apply(goal, &result))
	if depth, err := goal.Depth(); err != nil || depth != 1 {
		t.Fatal("quasi-macros depth", depth, err)
	}
	if size, err := goal.Size(); err != nil || size != 0 {
		t.Fatal("quasi-macros fixed-point result", size, err)
	}
	if sorted, err := goal.IsWellSorted(); err != nil || !sorted {
		t.Fatal("quasi-macros result sorting", sorted, err)
	}
}

func mustBoolSort(t *testing.T, m *ast.ASTManager) ast.ASTHandle {
	t.Helper()
	sort, err := m.MkBoolSort()
	cacheMust(t, err)
	return sort
}
