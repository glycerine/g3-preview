package ast

import (
	"strings"
	"testing"
)

func TestASTFPAIEEEEqualityDeclarationAndDomainValidation(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	sort, err := m.MkFamilySort(family, 3, nil)
	astTestMust(t, err)
	other, err := m.MkFamilySort(family, 4, nil)
	astTestMust(t, err)
	decl, err := m.MkFamilyFuncDecl(family, astFPAEq, nil, []ASTHandle{sort, sort, other}, ASTHandle{})
	astTestMust(t, err)
	r, err := m.internal(decl)
	astTestMust(t, err)
	name, err := m.Symbols.String(m.declAt(r.slot).name)
	astTestMust(t, err)
	info, err := m.funcInfo(m.declAt(r.slot).info)
	astTestMust(t, err)
	if name != "fp.eq" || m.declAt(r.slot).arity != 2 || m.declAt(r.slot).rangeSort != m.boolSort || !info.flags.Chainable {
		t.Fatal("source binary relation declaration", name, m.declAt(r.slot).arity, info.flags)
	}
	parameter, err := m.NewParameterAST(sort)
	astTestMust(t, err)
	zero, err := m.MkFamilyApp(family, astFPAPlusZero, []ASTParameter{parameter}, nil)
	astTestMust(t, err)
	app, err := m.MkFamilyApp(family, astFPAEq, nil, []ASTHandle{zero, zero})
	astTestMust(t, err)
	resultSort, err := m.GetSort(app)
	astTestMust(t, err)
	if resultSort != m.external(m.boolSort) {
		t.Fatal("fp.eq application did not produce Bool")
	}
	for _, relation := range []struct {
		kind int32
		name string
	}{
		{astFPALt, "fp.lt"}, {astFPAGt, "fp.gt"},
		{astFPALe, "fp.leq"}, {astFPAGe, "fp.geq"},
	} {
		decl, err := m.MkFamilyFuncDecl(family, relation.kind, nil, []ASTHandle{sort, sort}, ASTHandle{})
		astTestMust(t, err)
		r, err := m.internal(decl)
		astTestMust(t, err)
		name, err := m.Symbols.String(m.declAt(r.slot).name)
		astTestMust(t, err)
		info, err := m.funcInfo(m.declAt(r.slot).info)
		astTestMust(t, err)
		if name != relation.name || !info.flags.Chainable || m.declAt(r.slot).arity != 2 {
			t.Fatalf("comparison declaration %d: %q", relation.kind, name)
		}
		app, err := m.MkFamilyApp(family, relation.kind, nil, []ASTHandle{zero, zero})
		astTestMust(t, err)
		resultSort, err := m.GetSort(app)
		astTestMust(t, err)
		if resultSort != m.external(m.boolSort) {
			t.Fatalf("comparison %d did not produce Bool", relation.kind)
		}
	}
	for _, tc := range []struct {
		domain []ASTHandle
		want   string
	}{
		{[]ASTHandle{sort}, "invalid number of arguments"},
		{[]ASTHandle{sort, other}, "sort mismatch"},
		{[]ASTHandle{m.external(m.boolSort), m.external(m.boolSort)}, "sort mismatch"},
	} {
		_, err := m.MkFamilyFuncDecl(family, astFPAEq, nil, tc.domain, ASTHandle{})
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("domain %v: got %v, want %q", tc.domain, err, tc.want)
		}
	}
}
