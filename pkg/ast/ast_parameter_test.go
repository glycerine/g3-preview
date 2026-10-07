// Licensed under the MIT license.
package ast

import (
	"testing"
)

func TestASTSixParametersAndAttachmentOwnership(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	v, e := m.MkVar(91, b)
	astTestMust(t, e)
	astTestPin(t, m, v)
	before, e := m.GetRefCount(v)
	astTestMust(t, e)
	r, e := m.numerals.NewValue()
	astTestMust(t, e)
	astTestMust(t, m.numerals.SetString(&r, "123456789012345678901234567890/7"))
	var ps [6]ASTParameter
	ps[0], e = m.NewParameterInt(-7)
	astTestMust(t, e)
	ps[1], e = m.NewParameterAST(v)
	astTestMust(t, e)
	ps[2], e = m.NewParameterSymbol(astTestSymbol(t, m, "parameter"))
	astTestMust(t, e)
	ps[3], e = m.NewParameterRational(&r)
	astTestMust(t, e)
	ps[4], e = m.NewParameterDouble(12.5)
	astTestMust(t, e)
	ps[5], e = m.NewParameterExternal(17)
	astTestMust(t, e)
	astTestMust(t, m.numerals.Del(&r))
	intValue, e := m.ParameterInt(ps[0])
	astTestMust(t, e)
	astValue, e := m.ParameterAST(ps[1])
	astTestMust(t, e)
	symbolValue, e := m.ParameterSymbol(ps[2])
	astTestMust(t, e)
	doubleValue, e := m.ParameterDouble(ps[4])
	astTestMust(t, e)
	externalValue, e := m.ParameterExternal(ps[5])
	astTestMust(t, e)
	if intValue != -7 || astValue != v || symbolValue != astTestSymbol(t, m, "parameter") || doubleValue != 12.5 || externalValue != 17 {
		t.Fatal("typed source parameter getters")
	}
	out, e := m.numerals.NewValue()
	astTestMust(t, e)
	matched, e := m.ParameterRationalCopy(ps[3], &out)
	astTestMust(t, e)
	text, e := m.numerals.String(&out)
	astTestMust(t, e)
	if !matched || text != "17636684144620811271604938270" {
		t.Fatal("source rational extraction copies owned value", matched, text)
	}
	matched, e = m.ParameterRationalCopy(ps[0], &out)
	astTestMust(t, e)
	if matched {
		t.Fatal("source kind short-circuit")
	}
	astTestMust(t, m.numerals.Del(&out))
	for i, p := range ps {
		k, e := m.ParameterKind(p)
		astTestMust(t, e)
		if int(k) != i {
			t.Fatal("source parameter discriminant", i, k)
		}
	}
	rc, e := m.GetRefCount(v)
	astTestMust(t, e)
	if rc != before {
		t.Fatal("unattached AST parameter retained its child")
	}
	info, e := m.NewSortInfo(0, 71, ps[:], FiniteSortSize(11), false)
	astTestMust(t, e)
	s, e := m.MkSort(astTestSymbol(t, m, "attached"), info)
	astTestMust(t, e)
	astTestMust(t, m.IncRef(s))
	astTestMust(t, m.DelSortInfo(&info))
	for i := range ps {
		astTestMust(t, m.DelParameter(&ps[i]))
	}
	rc, e = m.GetRefCount(v)
	astTestMust(t, e)
	if rc != before+1 {
		t.Fatal("unique metadata attachment missing retain", rc, before)
	}
	astTestMust(t, m.DecRef(s))
	rc, e = m.GetRefCount(v)
	astTestMust(t, e)
	if rc != before {
		t.Fatal("metadata attachment did not release", rc, before)
	}
}

func TestASTMetadataIgnoredFields(t *testing.T) {
	m := astTestManager(t)
	i1, e := m.NewSortInfo(0, 71, nil, FiniteSortSize(2), false)
	astTestMust(t, e)
	i2, e := m.NewSortInfo(0, 71, nil, FiniteSortSize(900), true)
	astTestMust(t, e)
	name := astTestSymbol(t, m, "same-sort")
	s1, e := m.MkSort(name, i1)
	astTestMust(t, e)
	astTestPin(t, m, s1)
	s2, e := m.MkSort(name, i2)
	astTestMust(t, e)
	if s1 != s2 {
		t.Fatal("sort cardinality/private flag affect source interning")
	}
	astTestMust(t, m.DelSortInfo(&i1))
	astTestMust(t, m.DelSortInfo(&i2))
	f1, e := m.NewFuncDeclInfo(0, ASTOpAnd, nil, ASTFuncFlags{LeftAssociative: true, RightAssociative: true}, false)
	astTestMust(t, e)
	f2, e := m.NewFuncDeclInfo(0, ASTOpAnd, nil, ASTFuncFlags{LeftAssociative: true, RightAssociative: true, Idempotent: true}, true)
	astTestMust(t, e)
	name = astTestSymbol(t, m, "same-decl")
	d1, e := m.MkFuncDecl(name, []ASTHandle{s1, s1}, s1, f1)
	astTestMust(t, e)
	astTestPin(t, m, d1)
	d2, e := m.MkFuncDecl(name, []ASTHandle{s1, s1}, s1, f2)
	astTestMust(t, e)
	if d1 != d2 {
		t.Fatal("idempotence/private fields affect source equality")
	}
	astTestMust(t, m.DelFuncDeclInfo(&f1))
	astTestMust(t, m.DelFuncDeclInfo(&f2))
}
