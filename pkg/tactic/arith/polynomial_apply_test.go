package arith

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	goaltactic "github.com/glycerine/g3/pkg/tactic"
)

// Source: src/tactic/arith/propagate_ineqs_tactic.cpp:483-549;
// src/tactic/goal.cpp:79-88.
func TestInequalityPolynomialBuilderApplyGoalBranchesAndConfiguration(t *testing.T) {
	m, symbols, q, _ := polynomialASTManager(t)
	owner, _ := equationManager(t)
	store, err := NewBoundEquationStore(owner)
	equationMust(t, err)
	family, err := m.RegisterArithmeticPlugin()
	equationMust(t, err)
	realSort, err := m.MkFamilySort(family, 0, nil)
	equationMust(t, err)
	boolSort, err := m.MkBoolSort()
	equationMust(t, err)
	mkConst := func(name string, sort ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		sym, e := symbols.Intern(name)
		equationMust(t, e)
		h, e := m.MkConst(sym, sort)
		equationMust(t, e)
		return h
	}
	mkNumeral := func(text string) ast.ASTHandle {
		t.Helper()
		value, e := q.NewValue()
		equationMust(t, e)
		equationMust(t, q.SetString(&value, text))
		h, e := m.ArithmeticNumeral(&value, false)
		equationMust(t, e)
		equationMust(t, q.Del(&value))
		return h
	}
	mkBound := func(op int32, lhs, rhs ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		h, e := m.MkFamilyApp(family, op, nil, []ast.ASTHandle{lhs, rhs})
		equationMust(t, e)
		return h
	}
	x := mkConst("apply-x", realSort)
	p := mkConst("apply-p", boolSort)
	lower := mkBound(3, x, mkNumeral("2"))
	upper := mkBound(2, x, mkNumeral("5"))
	noBounds, err := goaltactic.NewGoal(m, false, true, false)
	equationMust(t, err)
	input, err := goaltactic.NewGoal(m, false, true, false)
	equationMust(t, err)
	coreGoal, err := goaltactic.NewGoal(m, false, false, true)
	equationMust(t, err)
	builder, err := NewPolynomialBuilder(m, store)
	equationMust(t, err)
	var result *goaltactic.Goal
	t.Cleanup(func() {
		if result != nil {
			equationMust(t, result.Close())
		}
		for _, e := range []error{noBounds.Close(), input.Close(), coreGoal.Close(), builder.Close(), store.Close()} {
			equationMust(t, e)
		}
	})
	equationMust(t, noBounds.Assert(p))
	equationMust(t, noBounds.SetDepth(9))
	borrowed, err := builder.Apply(noBounds)
	equationMust(t, err)
	if borrowed != noBounds {
		t.Fatal("no-bound source branch must return original goal")
	}
	if depth, e := noBounds.Depth(); e != nil || depth != 9 {
		t.Fatal("no-bound branch changed original depth", depth, e)
	}
	if result, err := builder.Apply(coreGoal); result != nil || err == nil || err.Error() != "propagate-ineqs does not support unsat core production" {
		t.Fatal("source core guard", result, err)
	}
	for _, formula := range []ast.ASTHandle{lower, p, upper} {
		equationMust(t, input.Assert(formula))
	}
	equationMust(t, input.SetDepth(7))
	equationMust(t, input.SetPrecision(goaltactic.GoalUnder))
	result, err = builder.Apply(input)
	equationMust(t, err)
	if result == nil || result == input {
		t.Fatal("recognized bound did not return a new goal")
	}
	if depth, e := result.Depth(); e != nil || depth != 8 {
		t.Fatal("source copied goal depth", depth, e)
	}
	if precision, e := result.Precision(); e != nil || precision != goaltactic.GoalUnder {
		t.Fatal("source copied goal precision", precision, e)
	}
	if models, e := result.ModelsEnabled(); e != nil || !models {
		t.Fatal("source copied model flag", models, e)
	}
	if size, e := result.Size(); e != nil || size != 3 {
		t.Fatal("bound result size", size, e)
	}
	for i, want := range []ast.ASTHandle{p, lower, upper} {
		got, e := result.Form(uint32(i))
		equationMust(t, e)
		if got != want {
			t.Fatal("non-bound then variable-order bounds", i, got, want)
		}
	}
	if size, e := input.Size(); e != nil || size != 3 {
		t.Fatal("borrowed input formulas changed", size, e)
	}
	if _, e := builder.Apply(nil); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("invalid goal guard", e)
	}
	conflictStore, err := NewBoundEquationStore(owner)
	equationMust(t, err)
	conflictBuilder, err := NewPolynomialBuilder(m, conflictStore)
	equationMust(t, err)
	conflictInput, err := goaltactic.NewGoal(m, false, false, false)
	equationMust(t, err)
	var conflictResult *goaltactic.Goal
	t.Cleanup(func() {
		if conflictResult != nil {
			equationMust(t, conflictResult.Close())
		}
		for _, e := range []error{conflictInput.Close(), conflictBuilder.Close(), conflictStore.Close()} {
			equationMust(t, e)
		}
	})
	equationMust(t, conflictInput.Assert(mkBound(3, x, mkNumeral("7"))))
	equationMust(t, conflictInput.Assert(mkBound(2, x, mkNumeral("3"))))
	conflictResult, err = conflictBuilder.Apply(conflictInput)
	equationMust(t, err)
	if conflictResult == nil || conflictResult == conflictInput {
		t.Fatal("inconsistent result did not own a new goal")
	}
	falsehood, err := m.MkFalse()
	equationMust(t, err)
	if size, e := conflictResult.Size(); e != nil || size != 1 {
		t.Fatal("inconsistent result size", size, e)
	}
	if formula, e := conflictResult.Form(0); e != nil || formula != falsehood {
		t.Fatal("source inconsistent branch must publish only false", formula, e)
	}
	if size, e := conflictInput.Size(); e != nil || size != 2 {
		t.Fatal("inconsistent input goal was mutated", size, e)
	}
}
