package ast

import (
	"testing"
)

func TestASTAllEagerPluginBranches(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	a, e := m.MkConst(astTestSymbol(t, m, "a"), b)
	astTestMust(t, e)
	astTestPin(t, m, a)
	for _, k := range []int32{ASTOpTrue, ASTOpFalse, ASTOpAnd, ASTOpOr, ASTOpIff, ASTOpXor, ASTOpNot, ASTOpInterp, ASTOpImplies, ASTOpEq, ASTOpOEq, ASTOpIte, ASTOpDistinct} {
		args := []ASTHandle{a, a}
		if k == ASTOpTrue || k == ASTOpFalse {
			args = nil
		}
		if k == ASTOpNot || k == ASTOpInterp {
			args = args[:1]
		}
		if k == ASTOpIte {
			args = append(args, a)
		}
		h, e := m.MkFamilyApp(0, k, nil, args)
		astTestMust(t, e)
		if h == (ASTHandle{}) {
			t.Fatal("missing basic branch", k)
		}
		astTestPin(t, m, h)
	}
	pos, e := m.NewParameterInt(1)
	astTestMust(t, e)
	tag, e := m.NewParameterSymbol(astTestSymbol(t, m, "tag"))
	astTestMust(t, e)
	label, e := m.MkFamilyApp(1, 0, []ASTParameter{pos, tag}, []ASTHandle{a})
	astTestMust(t, e)
	astTestPin(t, m, label)
	lit, e := m.MkFamilyApp(1, 1, []ASTParameter{tag}, nil)
	astTestMust(t, e)
	astTestPin(t, m, lit)
	pattern, e := m.MkFamilyApp(2, 0, nil, []ASTHandle{a})
	astTestMust(t, e)
	astTestPin(t, m, pattern)
	idx, e := m.NewParameterInt(3)
	astTestMust(t, e)
	sortParam, e := m.NewParameterAST(b)
	astTestMust(t, e)
	value, e := m.MkFamilyApp(3, 0, []ASTParameter{idx, sortParam}, nil)
	astTestMust(t, e)
	astTestPin(t, m, value)
	yes, e := m.IsUniqueValue(value)
	astTestMust(t, e)
	if !yes {
		t.Fatal("model value is not unique")
	}
	if yes, e = m.IsModelValue(value); e != nil || !yes {
		t.Fatal("model value classification", yes, e)
	}
	if yes, e = m.IsModelValue(a); e != nil || yes {
		t.Fatal("ordinary constant classified as model value", yes, e)
	}
	us, e := m.MkUninterpretedSort(astTestSymbol(t, m, "User"), nil)
	astTestMust(t, e)
	astTestPin(t, m, us)
	fid, e := m.GetFamilyID(us)
	astTestMust(t, e)
	if fid != 4 {
		t.Fatal("actual user-sort plugin missing")
	}
	for _, p := range []*ASTParameter{&pos, &tag, &idx, &sortParam} {
		astTestMust(t, m.DelParameter(p))
	}
}

func TestASTBuiltinValueAndDefaultPluginPolicies(t *testing.T) {
	m := astTestManager(t)
	truth, e := m.MkFamilyApp(0, ASTOpTrue, nil, nil)
	astTestMust(t, e)
	falsehood, e := m.MkFamilyApp(0, ASTOpFalse, nil, nil)
	astTestMust(t, e)
	for _, h := range []ASTHandle{truth, falsehood} {
		unique, e := m.IsUniqueValue(h)
		astTestMust(t, e)
		if !unique {
			t.Fatal("source basic truth value unique")
		}
		value, e := m.IsValue(h)
		astTestMust(t, e)
		if !value {
			t.Fatal("basic value")
		}
	}
	equal, e := m.AreEqual(truth, truth)
	astTestMust(t, e)
	if !equal {
		t.Fatal("default identity equality")
	}
	distinct, e := m.AreDistinct(truth, falsehood)
	astTestMust(t, e)
	if !distinct {
		t.Fatal("default unique-value distinctness")
	}
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	some, e := m.GetSomeValue(b)
	astTestMust(t, e)
	if some != falsehood {
		t.Fatal("basic some-value")
	}
	user, e := m.MkUninterpretedSort(astTestSymbol(t, m, "ValueSort"), nil)
	astTestMust(t, e)
	astTestPin(t, m, user)
	some, e = m.GetSomeValue(user)
	astTestMust(t, e)
	if some.IsNull() {
		t.Fatal("manager must fall back to actual model value")
	}
	astTestPin(t, m, some)
	interpreted, e := m.IsFullyInterpreted(user)
	astTestMust(t, e)
	if interpreted {
		t.Fatal("manager treats user sorts as uninterpreted")
	}
}
