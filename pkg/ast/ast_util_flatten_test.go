package ast

// Source: src/ast/ast_util.cpp:254-366; src/ast/ast_util.h:146-153.

import "testing"

func TestASTUtilFlattenAndOrSourceReprocessing(t *testing.T) {
	m := astTestManager(t)
	s, e := m.MkBoolSort()
	astTestMust(t, e)
	vars := make([]ASTHandle, 4)
	for i := range vars {
		vars[i], e = m.MkVar(uint32(i), s)
		astTestMust(t, e)
	}
	a, b, c, d := vars[0], vars[1], vars[2], vars[3]
	truth, e := m.MkTrue()
	astTestMust(t, e)
	falsehood, e := m.MkFalse()
	astTestMust(t, e)
	or, e := m.MkOrArgs([]ASTHandle{a, b})
	astTestMust(t, e)
	notOr, e := m.MkNot(or)
	astTestMust(t, e)
	imp, e := m.MkImplies(c, d)
	astTestMust(t, e)
	notImp, e := m.MkNot(imp)
	astTestMust(t, e)
	innerNot, e := m.MkNot(a)
	astTestMust(t, e)
	double, e := m.MkNot(innerNot)
	astTestMust(t, e)

	andResult, e := m.FlattenAnd([]ASTHandle{truth, notOr, notImp, double})
	astTestMust(t, e)
	if len(andResult) != 5 {
		t.Fatal("flatten and size", andResult)
	}
	wantPositive := map[ASTHandle]bool{a: true, c: true}
	wantNegative := map[ASTHandle]bool{a: true, b: true, d: true}
	positive, negative := make(map[ASTHandle]bool), make(map[ASTHandle]bool)
	for _, h := range andResult {
		isNot, inner, err := m.IsNot(h)
		astTestMust(t, err)
		if isNot {
			negative[inner] = true
		} else {
			positive[h] = true
		}
	}
	if !positive[a] || !positive[c] || len(positive) != len(wantPositive) || !negative[a] || !negative[b] || !negative[d] || len(negative) != len(wantNegative) {
		t.Fatal("flatten and terms", andResult, positive, negative, wantPositive, wantNegative)
	}
	if got, err := m.FlattenAnd([]ASTHandle{a, falsehood, b}); err != nil || len(got) != 1 || got[0] != falsehood {
		t.Fatal("and annihilator", got, err)
	}

	and, e := m.MkAndArgs([]ASTHandle{a, b})
	astTestMust(t, e)
	notAnd, e := m.MkNot(and)
	astTestMust(t, e)
	orResult, e := m.FlattenOr([]ASTHandle{falsehood, notAnd, imp, double})
	astTestMust(t, e)
	if len(orResult) != 5 {
		t.Fatal("flatten or size", orResult)
	}
	positive = make(map[ASTHandle]bool)
	negative = make(map[ASTHandle]bool)
	for _, h := range orResult {
		isNot, inner, err := m.IsNot(h)
		astTestMust(t, err)
		if isNot {
			negative[inner] = true
		} else {
			positive[h] = true
		}
	}
	if !positive[a] || !positive[d] || len(positive) != 2 || !negative[a] || !negative[b] || !negative[c] || len(negative) != 3 {
		t.Fatal("flatten or terms", orResult, positive, negative)
	}
	if got, err := m.FlattenOr([]ASTHandle{a, truth, b}); err != nil || len(got) != 1 || got[0] != truth {
		t.Fatal("or annihilator", got, err)
	}
}
