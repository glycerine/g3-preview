package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestFPAToFPExponentRationalBothOrdersDispatch(t *testing.T) {
	m := astTestManager(t)
	arith, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	rw, err := NewFPARewriter(m)
	astTestMust(t, err)
	defer func() { astTestMust(t, rw.Close()) }()
	rmSort, err := m.MkFamilySort(rw.family, 1, nil)
	astTestMust(t, err)
	rne, err := m.MkFamilyApp(rw.family, 0, nil, nil)
	astTestMust(t, err)
	astTestPin(t, m, rne)
	realSort, err := m.MkFamilySort(arith, 0, nil)
	astTestMust(t, err)
	intSort, err := m.MkFamilySort(arith, 1, nil)
	astTestMust(t, err)
	realZero, err := m.GetSomeValue(realSort)
	astTestMust(t, err)
	astTestPin(t, m, realZero)
	intZero, err := m.GetSomeValue(intSort)
	astTestMust(t, err)
	astTestPin(t, m, intZero)
	eParam, err := m.NewParameterInt(8)
	astTestMust(t, err)
	sParam, err := m.NewParameterInt(24)
	astTestMust(t, err)
	defer func() {
		astTestMust(t, m.DelParameter(&sParam))
		astTestMust(t, m.DelParameter(&eParam))
	}()
	for _, tc := range []struct {
		sorts []ast.ASTHandle
		args  []ast.ASTHandle
	}{
		{[]ast.ASTHandle{rmSort, realSort, intSort}, []ast.ASTHandle{rne, realZero, intZero}},
		{[]ast.ASTHandle{rmSort, intSort, realSort}, []ast.ASTHandle{rne, intZero, realZero}},
	} {
		decl, e := m.MkFamilyFuncDecl(rw.family, ast.FPAOpToFP,
			[]ast.ASTParameter{eParam, sParam}, tc.sorts, ast.ASTHandle{})
		astTestMust(t, e)
		astTestPin(t, m, decl)
		out := astTestPin(t, m, realZero)
		status, e := rw.MkAppCore(decl, ast.SliceArguments(tc.args), out)
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		kind, e := m.GetDeclKind(got)
		astTestMust(t, e)
		if status != RewriteDone || kind != 9 {
			t.Fatal("three-argument arithmetic zero dispatch", status, kind)
		}
		variable, e := m.MkVar(0, tc.sorts[1])
		astTestMust(t, e)
		args := []ast.ASTHandle{rne, variable, tc.args[2]}
		status, e = rw.MkAppCore(decl, ast.SliceArguments(args), out)
		astTestMust(t, e)
		if status != RewriteFailed {
			t.Fatal("symbolic arithmetic refusal", status)
		}
	}
}
