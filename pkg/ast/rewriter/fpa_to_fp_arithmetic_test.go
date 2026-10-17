package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestFPAToFPArithmeticRealIntDispatch(t *testing.T) {
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
	eParam, err := m.NewParameterInt(8)
	astTestMust(t, err)
	sParam, err := m.NewParameterInt(24)
	astTestMust(t, err)
	defer func() {
		astTestMust(t, m.DelParameter(&sParam))
		astTestMust(t, m.DelParameter(&eParam))
	}()
	for _, sortKind := range []int32{0, 1} {
		sort, e := m.MkFamilySort(arith, sortKind, nil)
		astTestMust(t, e)
		decl, e := m.MkFamilyFuncDecl(rw.family, ast.FPAOpToFP, []ast.ASTParameter{eParam, sParam}, []ast.ASTHandle{rmSort, sort}, ast.ASTHandle{})
		astTestMust(t, e)
		astTestPin(t, m, decl)
		zero, e := m.GetSomeValue(sort)
		astTestMust(t, e)
		astTestPin(t, m, zero)
		out := astTestPin(t, m, zero)
		status, e := rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{rne, zero}), out)
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		kind, e := m.GetDeclKind(got)
		astTestMust(t, e)
		if status != RewriteDone || kind != 9 {
			t.Fatal("arithmetic zero dispatch", sortKind, status, kind)
		}
		variable, e := m.MkVar(0, sort)
		astTestMust(t, e)
		status, e = rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{rne, variable}), out)
		astTestMust(t, e)
		if status != RewriteFailed {
			t.Fatal("symbolic arithmetic refusal", sortKind, status)
		}
	}
}
