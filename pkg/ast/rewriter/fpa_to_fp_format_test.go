package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestFPAToFPFormatSpecialAndSymbolicDispatch(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	rw, err := NewFPARewriter(m)
	astTestMust(t, err)
	defer func() { astTestMust(t, rw.Close()) }()
	rmSort, err := m.MkFamilySort(rw.family, 1, nil)
	astTestMust(t, err)
	inputSort, err := m.MkFamilySort(rw.family, 3, nil)
	astTestMust(t, err)
	eParam, err := m.NewParameterInt(5)
	astTestMust(t, err)
	sParam, err := m.NewParameterInt(11)
	astTestMust(t, err)
	defer func() {
		astTestMust(t, m.DelParameter(&sParam))
		astTestMust(t, m.DelParameter(&eParam))
	}()
	decl, err := m.MkFamilyFuncDecl(rw.family, ast.FPAOpToFP, []ast.ASTParameter{eParam, sParam}, []ast.ASTHandle{rmSort, inputSort}, ast.ASTHandle{})
	astTestMust(t, err)
	astTestPin(t, m, decl)
	rne, err := m.MkFamilyApp(rw.family, 0, nil, nil)
	astTestMust(t, err)
	astTestPin(t, m, rne)
	sortParam, err := m.NewParameterAST(inputSort)
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelParameter(&sortParam)) }()
	pinf, err := m.MkFamilyApp(rw.family, 6, []ast.ASTParameter{sortParam}, nil)
	astTestMust(t, err)
	astTestPin(t, m, pinf)
	out := astTestPin(t, m, pinf)
	status, err := rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{rne, pinf}), out)
	astTestMust(t, err)
	got, err := m.RefGet(out)
	astTestMust(t, err)
	kind, err := m.GetDeclKind(got)
	astTestMust(t, err)
	if status != RewriteDone || kind != 6 || got == pinf {
		t.Fatal("FP format conversion dispatch", status, kind, got)
	}
	variable, err := m.MkVar(0, inputSort)
	astTestMust(t, err)
	status, err = rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{rne, variable}), out)
	astTestMust(t, err)
	if status != RewriteFailed {
		t.Fatal("symbolic FP should refuse", status)
	}
}
