package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func TestSMTTacticPublishesIndependentModelsAcrossApplications(t *testing.T) {
	m, evaluator := smtModelGeneratorFixture(t)
	created, err := NewSMTTacticUsing(m, nil, "QF_UF", false, SMTSolverConfig{Params: evaluator.Context, Globals: evaluator.Globals, ModelEvaluator: evaluator})
	astTestMust(t, err)
	astTestMust(t, created.IncRef())
	t.Cleanup(func() { astTestMust(t, created.DecRef()) })
	atom := smtRelevancyBool(t, m, "smt-tactic-model-atom")
	astTestPin(t, m, atom)
	declaration, err := m.GetDecl(atom)
	astTestMust(t, err)
	var saved tactic.ModelRef
	t.Cleanup(func() { astTestMust(t, saved.Close()) })
	for _, positive := range []bool{true, false} {
		goal, err := tactic.NewGoal(m, false, true, false)
		astTestMust(t, err)
		formula := atom
		if !positive {
			formula, err = m.MkNot(atom)
			astTestMust(t, err)
		}
		astTestMust(t, goal.Assert(formula))
		var result tactic.TacticResult
		func() {
			defer func() { astTestMust(t, result.Close()); astTestMust(t, goal.Close()) }()
			astTestMust(t, created.Apply(goal, &result))
			if !goal.IsDecidedSat() || result.ModelConverter == nil {
				t.Fatal("SMT tactic did not publish a model for its satisfiable goal", positive)
			}
			var current tactic.ModelRef
			defer func() { astTestMust(t, current.Close()) }()
			astTestMust(t, result.ModelConverter.ApplyUnindexed(&current))
			model, err := current.Get()
			astTestMust(t, err)
			if model == nil {
				t.Fatal("SMT tactic converter returned an empty model")
			}
			value, err := model.ConstantInterpretation(declaration)
			astTestMust(t, err)
			if positive && !m.IsTrue(value) || !positive && !m.IsFalse(value) {
				t.Fatal("SMT tactic model lost the asserted Boolean value", positive, value)
			}
			if positive {
				astTestMust(t, saved.SetRef(&current))
			}
		}()
	}
	previous, err := saved.Get()
	astTestMust(t, err)
	value, err := previous.ConstantInterpretation(declaration)
	astTestMust(t, err)
	if !m.IsTrue(value) {
		t.Fatal("later tactic application changed the previously published model")
	}
	// The original term remains independently owned after both local SMT
	// contexts and their result converters have been destroyed.
	if sort, err := m.GetSort(atom); err != nil || sort == (ast.ASTHandle{}) {
		t.Fatal("tactic teardown invalidated its caller's AST", err)
	}
}

func TestNewSMTTacticUsingCopiesAndFixesAutoConfig(t *testing.T) {
	m := astTestManager(t)
	params, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget()})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, params.Close()) })
	input, err := params.NewValue()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, params.Del(&input)) })
	created, err := NewSMTTacticUsing(m, &input, "QF_UF", true, SMTSolverConfig{Params: params})
	astTestMust(t, err)
	astTestMust(t, created.IncRef())
	t.Cleanup(func() { astTestMust(t, created.DecRef()) })
	autoName := "auto_config"
	present, err := params.ContainsName(&input, &autoName)
	astTestMust(t, err)
	if present {
		t.Fatal("factory mutated caller parameters")
	}
	goal, err := tactic.NewGoal(m, false, true, false)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, goal.Close()) })
	var result tactic.TacticResult
	astTestMust(t, created.Apply(goal, &result))
	t.Cleanup(func() { astTestMust(t, result.Close()) })
	if len(result.Goals) != 1 || !goal.IsDecidedSat() {
		t.Fatal("SMT tactic factory result")
	}
	if result.ModelConverter == nil {
		t.Fatal("SMT tactic with default evaluator services did not publish a model")
	}
}
