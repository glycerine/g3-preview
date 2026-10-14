package arith

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/numeral"
)

// Source: src/tactic/arith/propagate_ineqs_tactic.cpp:175-249.
func TestInequalityPolynomialBuilderProcessBoundSourceNormalization(t *testing.T) {
	m, symbols, q, _ := polynomialASTManager(t)
	owner, _ := equationManager(t)
	store, err := NewBoundEquationStore(owner)
	equationMust(t, err)
	defer func() { equationMust(t, store.Close()) }()
	family, err := m.RegisterArithmeticPlugin()
	equationMust(t, err)
	sort, err := m.MkFamilySort(family, 0, nil)
	equationMust(t, err)
	mkVar := func(name string) ast.ASTHandle {
		t.Helper()
		symbol, e := symbols.Intern(name)
		equationMust(t, e)
		h, e := m.MkConst(symbol, sort)
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
	x, y := mkVar("x"), mkVar("y")
	three, four, six, two, eight := mkNumeral("3"), mkNumeral("4"), mkNumeral("6"), mkNumeral("2"), mkNumeral("8")
	builder, err := NewPolynomialBuilder(m, store)
	equationMust(t, err)
	defer func() { equationMust(t, builder.Close()) }()
	apply := func(h ast.ASTHandle, want bool) {
		t.Helper()
		got, e := builder.ProcessBound(h)
		equationMust(t, e)
		if got != want {
			t.Fatalf("processed=%v want %v", got, want)
		}
	}
	apply(mkOrdered(2, three, x), true) // numeral-left swap: x >= 3
	leFour := mkOrdered(2, x, four)
	notLeFour, err := m.MkNot(leFour)
	equationMust(t, err)
	apply(notLeFour, true) // x > 4
	apply(mkOrdered(2, x, six), true)
	eqY, err := m.MkEq(y, two)
	equationMust(t, err)
	apply(eqY, true)
	notEq, err := m.MkNot(eqY)
	equationMust(t, err)
	apply(notEq, false)
	apply(mkOrdered(2, x, y), false)
	if len(builder.var2expr) != 2 || len(store.constraints) != 0 {
		t.Fatal("unrecognized bounds changed variable or equation ownership")
	}
	check := func(variable uint32, lower bool, want string, strictWant bool) {
		t.Helper()
		v := numeral.NewRational()
		defer func() { equationMust(t, owner.Del(&v)) }()
		var found, strict bool
		var e error
		if lower {
			found, strict, _, e = store.Lower(variable, &v)
		} else {
			found, strict, _, e = store.Upper(variable, &v)
		}
		equationMust(t, e)
		text, e := owner.String(&v)
		equationMust(t, e)
		if !found || text != want || strict != strictWant {
			t.Fatalf("bound %d lower=%v: %v %s strict=%v", variable, lower, found, text, strict)
		}
	}
	check(0, true, "4", true)
	check(0, false, "6", false)
	check(1, true, "2", false)
	check(1, false, "2", false)
	sum := mkOrdered(6, x, y)
	apply(mkOrdered(2, sum, eight), true)
	if len(builder.var2expr) != 3 || len(store.constraints) != 1 || builder.coeff.Size() != 3 {
		t.Fatal("sum bound did not reach the concrete rational coefficient buffer and equation store")
	}
}
