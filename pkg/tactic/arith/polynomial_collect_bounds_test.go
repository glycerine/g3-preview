package arith

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	goaltactic "github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

// Source: src/tactic/arith/propagate_ineqs_tactic.cpp:238-253.
func TestInequalityPolynomialBuilderCollectBoundsGoalOrderAndOwnership(t *testing.T) {
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
		v, e := q.NewValue()
		equationMust(t, e)
		equationMust(t, q.SetString(&v, text))
		h, e := m.ArithmeticNumeral(&v, false)
		equationMust(t, e)
		equationMust(t, q.Del(&v))
		return h
	}
	mkOrdered := func(op int32, lhs, rhs ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		h, e := m.MkFamilyApp(family, op, nil, []ast.ASTHandle{lhs, rhs})
		equationMust(t, e)
		return h
	}
	x, y := mkConst("collect-x", realSort), mkConst("collect-y", realSort)
	p := mkConst("collect-p", boolSort)
	lower := mkOrdered(3, x, mkNumeral("2"))
	upper := mkOrdered(2, y, mkNumeral("3"))
	unknown, err := m.MkEq(x, y)
	equationMust(t, err)
	input, err := goaltactic.NewGoal(m, false, false, false)
	equationMust(t, err)
	saved, err := goaltactic.NewGoal(m, false, false, false)
	equationMust(t, err)
	onlyUnknown, err := goaltactic.NewGoal(m, false, false, false)
	equationMust(t, err)
	onlySaved, err := goaltactic.NewGoal(m, false, false, false)
	equationMust(t, err)
	builder, err := NewPolynomialBuilder(m, store)
	equationMust(t, err)
	t.Cleanup(func() {
		for _, e := range []error{input.Close(), saved.Close(), onlyUnknown.Close(), onlySaved.Close(), builder.Close(), store.Close()} {
			equationMust(t, e)
		}
	})
	for _, formula := range []ast.ASTHandle{lower, p, upper, unknown} {
		equationMust(t, input.Assert(formula))
	}
	found, err := builder.CollectBounds(input, saved)
	equationMust(t, err)
	if !found {
		t.Fatal("recognized bounds were not consumed")
	}
	if size, e := input.Size(); e != nil || size != 4 {
		t.Fatal("borrowed input goal changed", size, e)
	}
	if size, e := saved.Size(); e != nil || size != 2 {
		t.Fatal("non-bound goal size", size, e)
	}
	for i, want := range []ast.ASTHandle{p, unknown} {
		got, e := saved.Form(uint32(i))
		equationMust(t, e)
		if got != want {
			t.Fatal("source non-bound formula order", i, got, want)
		}
	}
	for variable, lowerDirection := range []bool{true, false} {
		v := numeral.NewRational()
		var present bool
		if lowerDirection {
			present, _, _, err = store.Lower(uint32(variable), &v)
		} else {
			present, _, _, err = store.Upper(uint32(variable), &v)
		}
		equationMust(t, err)
		text, e := owner.String(&v)
		equationMust(t, e)
		if !present || text != []string{"2", "3"}[variable] {
			t.Fatal("collected source bound", variable, text, present)
		}
		equationMust(t, owner.Del(&v))
	}
	equationMust(t, builder.RestoreBounds(saved))
	if size, e := saved.Size(); e != nil || size != 4 {
		t.Fatal("restored bound count", size, e)
	}
	for i, want := range []ast.ASTHandle{lower, upper} {
		got, e := saved.Form(uint32(i + 2))
		equationMust(t, e)
		if got != want {
			t.Fatal("source variable-order bound restoration", i, got, want)
		}
	}
	equationMust(t, onlyUnknown.Assert(p))
	found, err = builder.CollectBounds(onlyUnknown, onlySaved)
	equationMust(t, err)
	if found {
		t.Fatal("non-bound-only goal reported a bound")
	}
	if size, e := onlySaved.Size(); e != nil || size != 1 {
		t.Fatal("non-bound-only copy", size, e)
	}
}
