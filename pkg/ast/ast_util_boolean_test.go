// Source: src/ast/ast_util.cpp:160-252; src/ast/ast_util.h:99-140.
package ast

import "testing"

func TestASTUtilBooleanFactoriesAndPushNot(t *testing.T) {
	m := astTestManager(t)
	s, e := m.MkBoolSort()
	astTestMust(t, e)
	a, e := m.MkVar(0, s)
	astTestMust(t, e)
	b, e := m.MkVar(1, s)
	astTestMust(t, e)
	c, e := m.MkVar(2, s)
	astTestMust(t, e)
	truth, e := m.MkTrue()
	astTestMust(t, e)
	falsehood, e := m.MkFalse()
	astTestMust(t, e)

	if got, err := m.MkAndArray(nil); err != nil || got != truth {
		t.Fatal("empty and", got, err)
	}
	if got, err := m.MkAndArray([]ASTHandle{a}); err != nil || got != a {
		t.Fatal("singleton and", got, err)
	}
	if got, err := m.MkOrArray(nil); err != nil || got != falsehood {
		t.Fatal("empty or", got, err)
	}

	notA, e := m.MkNot(a)
	astTestMust(t, e)
	if got, err := m.MkNotSimplified(notA); err != nil || got != a {
		t.Fatal("double negation", got, err)
	}
	if got, err := m.MkNotSimplified(truth); err != nil || got != falsehood {
		t.Fatal("not true", got, err)
	}
	if got, err := m.MkNotSimplified(falsehood); err != nil || got != truth {
		t.Fatal("not false", got, err)
	}

	conjunction, e := m.MkAndArgs([]ASTHandle{a, b, c})
	astTestMust(t, e)
	pushed, e := m.PushNot(conjunction)
	astTestMust(t, e)
	if ok, err := m.astUtilTestOperator(pushed, ASTOpOr, 3); err != nil || !ok {
		t.Fatal("not and", ok, err)
	}
	for i, want := range []ASTHandle{a, b, c} {
		child, err := m.GetArg(pushed, uint32(i))
		astTestMust(t, err)
		isNot, inner, err := m.IsNot(child)
		astTestMust(t, err)
		if !isNot || inner != want {
			t.Fatal("De Morgan child", i, child, inner, want)
		}
	}
	emptyAnd, e := m.MkAndArgs(nil)
	astTestMust(t, e)
	if got, err := m.PushNot(emptyAnd); err != nil || got != falsehood {
		t.Fatal("not empty and", got, err)
	}

	for n := 0; n <= 1; n++ {
		got, err := m.MkDistinctSimplified([]ASTHandle{a}[:n])
		if err != nil || got != truth {
			t.Fatal("small distinct", n, got, err)
		}
	}
	pair, e := m.MkDistinctSimplified([]ASTHandle{a, b})
	astTestMust(t, e)
	isNot, _, e := m.IsNot(pair)
	astTestMust(t, e)
	if !isNot {
		t.Fatal("binary distinct is not equality")
	}
	wide, e := m.MkDistinctSimplified([]ASTHandle{a, b, c})
	astTestMust(t, e)
	if ok, err := m.astUtilTestOperator(wide, ASTOpDistinct, 3); err != nil || !ok {
		t.Fatal("wide distinct", ok, err)
	}
}

func (m *ASTManager) astUtilTestOperator(h ASTHandle, op int32, arity uint32) (bool, error) {
	family, e := m.GetFamilyID(h)
	if e != nil {
		return false, e
	}
	kind, e := m.GetDeclKind(h)
	if e != nil {
		return false, e
	}
	n, e := m.GetNumArgs(h)
	return family == 0 && kind == op && n == arity, e
}
