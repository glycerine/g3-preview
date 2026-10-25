package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	protomodel "github.com/glycerine/g3/pkg/smtcore/proto_model"
	"github.com/glycerine/g3/pkg/util/cooperate"
	"github.com/glycerine/g3/pkg/util/lbool"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	uparams "github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMTModelCheckerValidatesGroundModelWithNoActiveQuantifiers(t *testing.T) {
	m, cfg := smtModelGeneratorFixture(t)
	s, err := NewSMTSolver(m, nil, "AUFLIA", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, s.Close()) })
	state := s.kernel.internalizer
	checker := state.quantifierManager.modelChecker
	a := smtRelevancyBool(t, m, "inactive-quantifier-invalid-model")
	astTestMust(t, state.Assert(a, ast.ASTHandle{}, 0))
	astTestMust(t, state.context.SetRelevant(a, true))
	proto, err := protomodel.NewProtoModel(m, cfg)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, proto.Close()) })
	decl, err := m.GetDecl(a)
	astTestMust(t, err)
	no, err := m.MkFalse()
	astTestMust(t, err)
	astTestMust(t, proto.Core().RegisterConstant(decl, no))
	// No registered quantifiers: preserve the source fast path.
	accepted, err := checker.Check(proto, nil)
	astTestMust(t, err)
	if !accepted {
		t.Fatal("empty registry did not use source fast path")
	}
	q, _ := smtRecursiveTestDefinition(t, m, "inactive-definition")
	astTestMust(t, state.Internalize(q, true))
	astTestMust(t, state.context.SetAssignment(q, lbool.Undef))
	active, err := state.quantifierManager.activeQuantifiers()
	astTestMust(t, err)
	if len(active) != 0 || checker.registry.Empty() {
		t.Fatal("expected filtered-empty nonempty registry")
	}
	initial := checker.maxCexs
	accepted, err = checker.Check(proto, active)
	astTestMust(t, err)
	if accepted || checker.iteration != 0 || checker.maxCexs != initial+checker.initialMaxCexs {
		t.Fatal("inactive registry bypassed ground validation or changed iteration", accepted, checker.iteration, checker.maxCexs)
	}
	// The same filtered-empty registry still observes the source iteration cap.
	astTestMust(t, checker.ConfigureLifecycle(1, 0))
	accepted, err = checker.Check(proto, active)
	astTestMust(t, err)
	if accepted || !checker.Exhausted() {
		t.Fatal("filtered-empty registry bypassed iteration cap")
	}
}

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
	if satisfied || !checker.HasNewInstances() || registry.schedule.HasWork() {
		t.Fatal("false Boolean binding did not queue a counterexample", satisfied)
	}
	if checker.iteration != 1 || checker.maxCexs != 2 {
		t.Fatal("first model-check lifecycle", checker.iteration, checker.maxCexs)
	}
	astTestMust(t, checker.Restart())
	astTestMust(t, manager.Propagate())
	if registry.schedule.Stats().Instances != 1 {
		t.Fatal("counterexample was not materialized", registry.schedule.Stats())
	}
	satisfied, err = checker.Check(proto, []ast.ASTHandle{quantifier})
	astTestMust(t, err)
	if satisfied || registry.schedule.Stats().Instances != 1 {
		t.Fatal("duplicate counterexample changed instance count or incorrectly validated the model", satisfied, registry.schedule.Stats())
	}
	if checker.iteration != 2 || checker.maxCexs != 3 {
		t.Fatal("second model-check lifecycle", checker.iteration, checker.maxCexs)
	}
	astTestMust(t, checker.ConfigureLifecycle(1, 0))
	astTestMust(t, checker.InitSearch())
	satisfied, err = checker.Check(proto, []ast.ASTHandle{quantifier})
	astTestMust(t, err)
	if satisfied || !checker.Exhausted() || checker.iteration != 0 || checker.maxCexs != 1 {
		t.Fatal("exhausted model-check lifecycle", satisfied, checker.Exhausted(), checker.iteration, checker.maxCexs)
	}
}
