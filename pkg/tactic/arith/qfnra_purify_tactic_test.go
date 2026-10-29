package arith

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/tactic"
)

func qfnraPurifier(t *testing.T, proofs bool, complete bool) (*QFNRAPurifyTactic, *ast.ASTManager) {
	t.Helper()
	base, m := fixDLFixture(t, proofs)
	params, err := base.context.NewValue()
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, base.context.Del(&params)) })
	name := "complete"
	equationMust(t, base.context.SetBoolName(&params, &name, complete))
	runner, err := NewQFNRAPurifyTactic(m, base.context, base.globals, &params, base.services)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, runner.Close()) })
	return runner, m
}

func TestQFNRAPurifyDivisionProofCoreAndModelFilter(t *testing.T) {
	runner, m := qfnraPurifier(t, true, false)
	x := fixDLConst(t, m, false, "purify-div-x")
	y := fixDLConst(t, m, false, "purify-div-y")
	two := fixDLNumber(t, m, false, "2")
	division := fixDLApp(t, m, qfnraPurifyDiv, x, y)
	formula, err := m.MkEq(division, two)
	equationMust(t, err)
	fixDLPin(t, m, formula)
	asserted, err := m.MkProofAsserted(formula)
	equationMust(t, err)
	dependency, err := m.DependencyLeaf(formula)
	equationMust(t, err)
	goal, err := tactic.NewGoal(m, true, true, true)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, goal.Close()) })
	equationMust(t, goal.AssertWithProof(formula, asserted, dependency))
	var result tactic.TacticResult
	t.Cleanup(func() { equationMust(t, result.Close()) })
	equationMust(t, runner.Apply(goal, &result))
	size, err := goal.Size()
	equationMust(t, err)
	if size != 2 || result.ModelConverter == nil {
		t.Fatal("division purification result", size, result.ModelConverter)
	}
	rewritten, err := goal.Form(0)
	equationMust(t, err)
	fresh, err := m.GetArg(rewritten, 0)
	equationMust(t, err)
	decl, err := m.GetDecl(fresh)
	equationMust(t, err)
	rewrittenProof, err := goal.Proof(0)
	equationMust(t, err)
	fact, err := m.GetProofFact(rewrittenProof)
	equationMust(t, err)
	if fact != rewritten {
		t.Fatal("division rewrite proof conclusion")
	}
	leaves, err := m.DependencyLinearize(mustGoalDependency(t, goal, 0), nil)
	equationMust(t, err)
	if len(leaves) != 1 || leaves[0] != formula {
		t.Fatal("division rewrite changed dependency", leaves)
	}
	leaves, err = m.DependencyLinearize(mustGoalDependency(t, goal, 1), nil)
	equationMust(t, err)
	if len(leaves) != 0 {
		t.Fatal("purification constraint acquired dependency", leaves)
	}
	md, err := model.NewModel(m)
	equationMust(t, err)
	equationMust(t, md.CoreField.RegisterConstant(decl, two))
	var output tactic.ModelRef
	equationMust(t, output.Set(&md.CoreField))
	t.Cleanup(func() { equationMust(t, output.Close()) })
	equationMust(t, result.ModelConverter.ApplyModel(&output, 0))
	filtered, err := output.Get()
	equationMust(t, err)
	present, err := filtered.HasInterpretation(decl)
	equationMust(t, err)
	if present {
		t.Fatal("purification fresh declaration leaked into model")
	}
}

func mustGoalDependency(t *testing.T, goal *tactic.Goal, index uint32) ast.ASTDependency {
	t.Helper()
	dependency, err := goal.Dependency(index)
	equationMust(t, err)
	return dependency
}

func TestQFNRAPurifyZeroOddAndEvenPowers(t *testing.T) {
	for _, row := range []struct {
		name, exponent string
		constraints    uint32
	}{
		{"zero", "0", 2},
		{"odd-root", "1/3", 1},
		{"even-root", "1/2", 2},
	} {
		t.Run(row.name, func(t *testing.T) {
			runner, m := qfnraPurifier(t, false, false)
			x := fixDLConst(t, m, false, "purify-power-x")
			y := fixDLConst(t, m, false, "purify-power-y")
			exponent := fixDLNumber(t, m, false, row.exponent)
			power := fixDLApp(t, m, qfnraPurifyPower, x, exponent)
			formula, err := m.MkEq(power, y)
			equationMust(t, err)
			fixDLPin(t, m, formula)
			goal, err := tactic.NewGoal(m, false, false, false)
			equationMust(t, err)
			t.Cleanup(func() { equationMust(t, goal.Close()) })
			equationMust(t, goal.Assert(formula))
			var result tactic.TacticResult
			t.Cleanup(func() { equationMust(t, result.Close()) })
			equationMust(t, runner.Apply(goal, &result))
			size, err := goal.Size()
			equationMust(t, err)
			if size != 1+row.constraints {
				t.Fatal("power purification constraint count", size)
			}
			rewritten, err := goal.Form(0)
			equationMust(t, err)
			left, err := m.GetArg(rewritten, 0)
			equationMust(t, err)
			if family, err := m.GetFamilyID(left); err != nil || family == runner.arithmetic {
				if err != nil {
					t.Fatal(err)
				}
				if op, e := m.GetDeclKind(left); e != nil || op == qfnraPurifyPower {
					t.Fatal("power remained in rewritten assertion", op, e)
				}
			}
		})
	}
}

func TestQFNRAPurifyLeavesPositiveIntegerPower(t *testing.T) {
	runner, m := qfnraPurifier(t, false, false)
	x := fixDLConst(t, m, false, "purify-square-x")
	two := fixDLNumber(t, m, false, "2")
	square := fixDLApp(t, m, qfnraPurifyPower, x, two)
	formula, err := m.MkEq(square, two)
	equationMust(t, err)
	fixDLPin(t, m, formula)
	goal, err := tactic.NewGoal(m, false, false, false)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, goal.Close()) })
	equationMust(t, goal.Assert(formula))
	var result tactic.TacticResult
	t.Cleanup(func() { equationMust(t, result.Close()) })
	equationMust(t, runner.Apply(goal, &result))
	rewritten, err := goal.Form(0)
	equationMust(t, err)
	if rewritten != formula {
		t.Fatal("positive integer power was purified")
	}
}
