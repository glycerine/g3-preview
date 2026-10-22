package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	protomodel "github.com/glycerine/g3/pkg/smtcore/proto_model"
	"github.com/glycerine/g3/pkg/util/cooperate"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	uparams "github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMTModelCheckerFindsFiniteBooleanCounterexample(t *testing.T) {
	m, state, registry, manager := smtQuantifierManagerFixture(t, 10, 20, SMTQuantifierManagerConfig{})
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	variable, err := m.MkVar(0, boolean)
	astTestMust(t, err)
	quantifier, err := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true, Sorts: []ast.ASTHandle{boolean}, Names: []symbol.Symbol{astTestSymbol(t, m, "model-check-x")}, Body: variable,
	})
	astTestMust(t, err)
	astTestMust(t, state.Internalize(quantifier, true))
	context, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget()})
	astTestMust(t, err)
	catalog, err := uparams.NewGlobalParamsCatalog(m.Symbols, uparams.GlobalParamsCatalogConfig{Budget: m.Budget()})
	astTestMust(t, err)
	globals, err := uparams.NewGlobalParamsService(m.Symbols, context, uparams.GlobalParamsConfig{Budget: m.Budget(), Catalog: catalog})
	astTestMust(t, err)
	astTestMust(t, globals.InitializeSource())
	params, err := context.NewValue()
	astTestMust(t, err)
	memory, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: m.Budget()})
	astTestMust(t, err)
	cooperation, err := cooperate.NewCooperation(m.Budget())
	astTestMust(t, err)
	t.Cleanup(func() {
		astTestMust(t, cooperation.Close())
		astTestMust(t, memory.Close())
		astTestMust(t, context.Del(&params))
		astTestMust(t, globals.Close())
		astTestMust(t, catalog.Close())
		astTestMust(t, context.Close())
	})
	proto, err := protomodel.NewProtoModel(m, model.ModelEvaluatorConfig{
		Context: context, Globals: globals, Params: &params, Memory: memory, Cooperation: cooperation,
	})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, proto.Close()) })
	checker, err := NewSMTModelChecker(state, 1)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, checker.Close()) })
	satisfied, err := checker.Check(proto, []ast.ASTHandle{quantifier})
	astTestMust(t, err)
	if satisfied || !registry.schedule.HasWork() {
		t.Fatal("false Boolean binding did not queue a counterexample", satisfied)
	}
	astTestMust(t, manager.Propagate())
	if registry.schedule.Stats().Instances != 1 {
		t.Fatal("counterexample was not materialized", registry.schedule.Stats())
	}
	satisfied, err = checker.Check(proto, []ast.ASTHandle{quantifier})
	astTestMust(t, err)
	if !satisfied {
		t.Fatal("duplicate counterexample was reported as new")
	}
}
