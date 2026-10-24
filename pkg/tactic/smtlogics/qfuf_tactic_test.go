package smtlogics_test

import (
	"testing"

	"github.com/glycerine/g3/pkg/cmd_context"
	"github.com/glycerine/g3/pkg/tactic"
)

func TestQFUFTacticModelSurvivesStrategyClose(t *testing.T) {
	c, err := cmd_context.NewStandaloneContext()
	qfaufliaMust(t, err)
	t.Cleanup(func() { qfaufliaMust(t, c.Close()) })
	qfaufliaMust(t, c.SetLogic("QF_UF"))
	m, err := c.Manager()
	qfaufliaMust(t, err)
	runner, err := c.NewBuiltinTactic("qfuf")
	qfaufliaMust(t, err)
	qfaufliaMust(t, runner.IncRef())
	closed := false
	t.Cleanup(func() {
		if !closed {
			qfaufliaMust(t, runner.DecRef())
		}
	})
	sort, err := m.MkBoolSort()
	qfaufliaMust(t, err)
	name, err := m.Symbols.Intern("qfuf-model-x")
	qfaufliaMust(t, err)
	x, err := m.MkConst(name, sort)
	qfaufliaMust(t, err)
	qfaufliaPin(t, m, x)
	goal, err := tactic.NewGoal(m, false, true, false)
	qfaufliaMust(t, err)
	t.Cleanup(func() { qfaufliaMust(t, goal.Close()) })
	qfaufliaMust(t, goal.Assert(x))
	var result tactic.TacticResult
	t.Cleanup(func() { qfaufliaMust(t, result.Close()) })
	qfaufliaMust(t, runner.Apply(goal, &result))
	if !goal.IsDecidedSat() || result.ModelConverter == nil {
		t.Fatal("missing solved goal/model converter")
	}
	qfaufliaMust(t, runner.Cleanup())
	qfaufliaMust(t, runner.DecRef())
	closed = true
	var md tactic.ModelRef
	t.Cleanup(func() { qfaufliaMust(t, md.Close()) })
	qfaufliaMust(t, tactic.ModelConverterToModel(m, result.ModelConverter, &md))
	model, err := md.Get()
	qfaufliaMust(t, err)
	decl, err := m.GetDecl(x)
	qfaufliaMust(t, err)
	value, err := model.ConstantInterpretation(decl)
	qfaufliaMust(t, err)
	if !m.IsTrue(value) {
		t.Fatal("strategy close lost reconstructed Boolean value", value)
	}
}
