// Source: src/ast/factor_equivs.cpp; src/ast/factor_equivs.h.
package ast

import "testing"

func TestASTFactorEqualitiesArithmeticNormalizationAndRemainder(t *testing.T) {
	m := astTestManager(t)
	fid, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	intSort := m.external(m.arithmeticPlugin.fields[1])
	x, e := m.MkConst(astTestSymbol(t, m, "factor-x"), intSort)
	astTestMust(t, e)
	y, e := m.MkConst(astTestSymbol(t, m, "factor-y"), intSort)
	astTestMust(t, e)
	p, e := m.MkConst(astTestSymbol(t, m, "factor-p"), m.external(m.boolSort))
	astTestMust(t, e)
	for _, h := range []ASTHandle{x, y, p} {
		astTestPin(t, m, h)
	}
	q := m.numerals
	n, e := q.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, q.Del(&n)) }()
	numeral := func(text string) ASTHandle {
		astTestMust(t, q.SetString(&n, text))
		h, err := m.ArithmeticNumeral(&n, true)
		astTestMust(t, err)
		return h
	}
	zero, minusOne := numeral("0"), numeral("-1")
	negX, e := m.MkFamilyApp(fid, 9, nil, []ASTHandle{minusOne, x})
	astTestMust(t, e)
	difference, e := m.MkFamilyApp(fid, 6, nil, []ASTHandle{y, negX})
	astTestMust(t, e)
	eq, e := m.MkEq(difference, zero)
	astTestMust(t, e)
	conjunction, e := m.MkAnd(eq, p)
	astTestMust(t, e)
	astTestPin(t, m, conjunction)

	classes, e := NewExprEquivClass(m)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, classes.Close()) })
	rest, e := m.FactorEqualities([]ASTHandle{conjunction}, classes)
	astTestMust(t, e)
	if len(rest) != 1 || rest[0] != p {
		t.Fatal("factor remainder", rest)
	}
	same, e := classes.AreEquivalent(x, y)
	astTestMust(t, e)
	if !same {
		t.Fatal("normalized y + -1*x = 0 did not merge x and y")
	}
}

func TestASTExprEquivClassEmissionAndOwnership(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	a, e := m.MkConst(astTestSymbol(t, m, "equiv-a"), b)
	astTestMust(t, e)
	c, e := m.MkConst(astTestSymbol(t, m, "equiv-c"), b)
	astTestMust(t, e)
	d, e := m.MkConst(astTestSymbol(t, m, "equiv-d"), b)
	astTestMust(t, e)
	for _, h := range []ASTHandle{a, c, d} {
		astTestPin(t, m, h)
	}
	equiv, e := NewExprEquivClass(m)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, equiv.Close()) })
	astTestMust(t, equiv.Merge(a, c))
	astTestMust(t, equiv.Merge(c, d))
	compact, e := m.EquivToExpr(equiv)
	astTestMust(t, e)
	if len(compact) != 2 {
		t.Fatal("representative equalities", len(compact))
	}
	for i := range compact {
		astTestMust(t, m.DelRef(&compact[i]))
	}
	full, dirty, e := m.EquivToExprFull(equiv)
	astTestMust(t, e)
	if !dirty || len(full) != 3 {
		t.Fatal("full equality closure", dirty, len(full))
	}
	for i := range full {
		astTestMust(t, m.DelRef(&full[i]))
	}
	astTestMust(t, equiv.Reset())
	if len(equiv.objects) != 0 || len(equiv.ids) != 0 {
		t.Fatal("reset retained class state")
	}
}
