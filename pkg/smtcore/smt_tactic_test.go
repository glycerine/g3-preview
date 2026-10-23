package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/tactic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

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
	goal, err := tactic.NewGoal(m, false, false, false)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, goal.Close()) })
	var result tactic.TacticResult
	astTestMust(t, created.Apply(goal, &result))
	t.Cleanup(func() { astTestMust(t, result.Close()) })
	if len(result.Goals) != 1 || !goal.IsDecidedSat() {
		t.Fatal("SMT tactic factory result")
	}
}
