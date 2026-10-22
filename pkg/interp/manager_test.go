package interp

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func interpManagerFixture(t *testing.T) (*ast.ASTManager, *Manager) {
	t.Helper()
	m := interpASTFixture(t)
	interpMust(t, m.RegisterDeclarationPlugins())
	w, err := NewManager(m)
	interpMust(t, err)
	t.Cleanup(func() { interpMust(t, w.Close()) })
	return m, w
}

func TestInterpolationManagerOperatorConstructionAndAccess(t *testing.T) {
	m, w := interpManagerFixture(t)
	boolSort, err := m.MkBoolSort()
	interpMust(t, err)
	p, err := w.MakeVariable("manager-p", boolSort)
	interpMust(t, err)
	p = interpPin(t, m, p)
	q, err := w.MakeVariable("manager-q", boolSort)
	interpMust(t, err)
	q = interpPin(t, m, q)
	conjunction, err := w.Make(OperatorAnd, p, q)
	interpMust(t, err)
	conjunction = interpPin(t, m, conjunction)
	if op, e := w.Operator(conjunction); e != nil || op != OperatorAnd {
		t.Fatal(op, e)
	}
	args, err := w.Args(conjunction)
	interpMust(t, err)
	if len(args) != 2 || args[0] != p || args[1] != q {
		t.Fatal(args)
	}
	decl, err := w.Symbol(p)
	interpMust(t, err)
	rebuilt, err := w.MakeDeclaration(decl)
	interpMust(t, err)
	if rebuilt != p {
		t.Fatal("declaration application is not shared", rebuilt, p)
	}
	if op, e := w.Operator(p); e != nil || op != OperatorUninterpreted {
		t.Fatal(op, e)
	}

	integer, err := m.ArithmeticIntegerNumeralText("7")
	interpMust(t, err)
	integer = interpPin(t, m, integer)
	if op, e := w.Operator(integer); e != nil || op != OperatorNumeral {
		t.Fatal(op, e)
	}
	plus, err := w.Make(OperatorPlus, integer, integer)
	interpMust(t, err)
	plus = interpPin(t, m, plus)
	if op, e := w.Operator(plus); e != nil || op != OperatorPlus {
		t.Fatal(op, e)
	}
	intSort, err := w.Type(integer)
	interpMust(t, err)
	variable, err := m.MkVar(3, intSort)
	interpMust(t, err)
	variable = interpPin(t, m, variable)
	if op, e := w.Operator(variable); e != nil || op != OperatorVariable {
		t.Fatal(op, e)
	}
	if n, e := w.NumArgs(variable); e != nil || n != 0 {
		t.Fatal(n, e)
	}
}

func TestInterpolationManagerBooleanSimplifiers(t *testing.T) {
	m, w := interpManagerFixture(t)
	truth, err := w.Make(OperatorTrue)
	interpMust(t, err)
	falsehood, err := w.Make(OperatorFalse)
	interpMust(t, err)
	boolSort, err := m.MkBoolSort()
	interpMust(t, err)
	p, err := w.MakeVariable("manager-simplify", boolSort)
	interpMust(t, err)
	p = interpPin(t, m, p)
	must := func(h ast.ASTHandle, err error) ast.ASTHandle {
		t.Helper()
		interpMust(t, err)
		return h
	}

	checks := []struct {
		name string
		got  ast.ASTHandle
		want ast.ASTHandle
	}{
		{"and-true", must(w.MkAnd(truth, p)), p},
		{"and-false", must(w.MkAnd(p, falsehood)), falsehood},
		{"or-false", must(w.MkOr(falsehood, p)), p},
		{"or-true", must(w.MkOr(p, truth)), truth},
		{"implies-true", must(w.MkImplies(truth, p)), p},
		{"equal-true", must(w.MkEqual(truth, p)), p},
		{"ite-true", must(w.MkIte(truth, p, falsehood)), p},
		{"ite-bool", must(w.MkIte(p, truth, falsehood)), p},
	}
	for _, check := range checks {
		if check.got != check.want {
			t.Errorf("%s = %v, want %v", check.name, check.got, check.want)
		}
	}
	notP, err := w.MkNot(p)
	interpMust(t, err)
	notP = interpPin(t, m, notP)
	if got, e := w.MkNot(notP); e != nil || got != p {
		t.Fatal("double negation", got, e)
	}
	if got, e := w.MkAndSlice([]ast.ASTHandle{truth, p, truth}); e != nil || got != p {
		t.Fatal("filtered conjunction", got, e)
	}
	if got, e := w.MkOrSlice([]ast.ASTHandle{falsehood, p}); e != nil || got != p {
		t.Fatal("folded disjunction", got, e)
	}
}

func TestInterpolationManagerBorrowAndCopyGuards(t *testing.T) {
	_, w := interpManagerFixture(t)
	copyOwner := *w
	if _, err := copyOwner.ASTManager(); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("copied manager", err)
	}
	interpMust(t, w.Close())
	if _, err := w.ASTManager(); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("closed manager", err)
	}
}
