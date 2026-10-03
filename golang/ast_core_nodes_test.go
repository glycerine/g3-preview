// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
package g3

import "testing"

func TestASTEagerNormalAndFormatConstructors(t *testing.T) {
	s, q, b := astTestOwners(t)
	m, e := NewASTManager(ASTManagerConfig{Symbols: s, Numerals: q, Budget: b})
	astTestMust(t, e)
	f, e := m.GetFormatManager()
	astTestMust(t, e)
	if f == m || !f.IsFormatManager() || m.IsFormatManager() {
		t.Fatal("normal manager needs a real nested format owner")
	}
	self, e := f.GetFormatManager()
	astTestMust(t, e)
	if self != f {
		t.Fatal("format self accessor")
	}
	for _, owner := range []*ASTManager{m, f} {
		for i, name := range []string{"basic", "label", "pattern", "model-value", "user-sort", "arith"} {
			id, e := owner.families.GetID(astTestSymbol(t, owner, name))
			astTestMust(t, e)
			if id != int32(i) {
				t.Fatal("source family order", i, id)
			}
		}
		boolSort, e := owner.MkBoolSort()
		astTestMust(t, e)
		sz, e := owner.GetSortSize(boolSort)
		astTestMust(t, e)
		n, e := sz.Size()
		astTestMust(t, e)
		if n != 2 {
			t.Fatal("real Bool cardinality")
		}
		for _, k := range []int32{ASTOpTrue, ASTOpFalse} {
			h, e := owner.MkFamilyApp(0, k, nil, nil)
			astTestMust(t, e)
			rc, e := owner.GetRefCount(h)
			astTestMust(t, e)
			if rc != 1 {
				t.Fatal("source manager app pin", k, rc)
			}
		}
		missing, e := owner.MkFamilySort(5, 0, nil)
		astTestMust(t, e)
		if missing != (ASTHandle{}) {
			t.Fatal("arith family is not an arithmetic plugin")
		}
	}
	hb, e := m.MkBoolSort()
	astTestMust(t, e)
	hf, e := f.MkBoolSort()
	astTestMust(t, e)
	if hb == hf {
		t.Fatal("nested managers must have distinct identities")
	}
	if e = m.IncRef(hf); e == nil {
		t.Fatal("foreign owner accepted")
	}
	astTestMust(t, m.Close())
	standalone, e := NewASTFormatManager(ASTManagerConfig{Symbols: s, Numerals: q, Budget: b})
	astTestMust(t, e)
	if !standalone.IsFormatManager() {
		t.Fatal("format constructor")
	}
	astTestMust(t, standalone.Close())
}

func TestASTAllFiveKindsAndQuantifierIdentity(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	v, e := m.MkVar(0, b)
	astTestMust(t, e)
	astTestPin(t, m, v)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "f"), []ASTHandle{b}, b, ASTFuncDeclInfo{})
	astTestMust(t, e)
	astTestPin(t, m, d)
	a, e := m.MkApp(d, []ASTHandle{v})
	astTestMust(t, e)
	astTestPin(t, m, a)
	spec := ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{b}, Names: []Symbol{astTestSymbol(t, m, "x")}, Body: a, Weight: 1, QID: astTestSymbol(t, m, "q1")}
	q, e := m.MkQuantifier(spec)
	astTestMust(t, e)
	astTestPin(t, m, q)
	spec.QID = astTestSymbol(t, m, "q2")
	spec.SkolemID = astTestSymbol(t, m, "sk")
	q2, e := m.MkQuantifier(spec)
	astTestMust(t, e)
	if q != q2 {
		t.Fatal("source qid/skid omitted from equality")
	}
	for _, tc := range []struct {
		h ASTHandle
		k ASTKind
	}{{b, ASTSort}, {d, ASTFuncDecl}, {a, ASTApp}, {v, ASTVar}, {q, ASTQuantifier}} {
		k, e := m.GetKind(tc.h)
		astTestMust(t, e)
		if k != tc.k {
			t.Fatal(k, tc.k)
		}
	}
	if id, e := m.GetASTID(b); e != nil || id < 1<<31 {
		t.Fatal("sort declaration ID namespace", id, e)
	}
	if id, e := m.GetASTID(v); e != nil || id >= 1<<31 {
		t.Fatal("expression ID namespace", id, e)
	}
}
