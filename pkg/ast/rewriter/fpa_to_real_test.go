package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestFPAToRealDispatchDefaultAndHardwareUnspecified(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	rw, err := NewFPARewriter(m)
	astTestMust(t, err)
	defer func() { astTestMust(t, rw.Close()) }()
	sort, err := m.MkFamilySort(rw.family, 3, nil)
	astTestMust(t, err)
	parameter, err := m.NewParameterAST(sort)
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelParameter(&parameter)) }()
	nan, err := m.MkFamilyApp(rw.family, 8, []ast.ASTParameter{parameter}, nil)
	astTestMust(t, err)
	astTestPin(t, m, nan)
	variable, err := m.MkVar(0, sort)
	astTestMust(t, err)
	astTestPin(t, m, variable)
	decl, err := m.MkFamilyFuncDecl(rw.family, ast.FPAOpToReal, nil, []ast.ASTHandle{sort}, ast.ASTHandle{})
	astTestMust(t, err)
	realSort, err := m.GetRange(decl)
	astTestMust(t, err)
	zero, err := m.GetSomeValue(realSort)
	astTestMust(t, err)
	out := astTestPin(t, m, zero)
	status, err := rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{nan}), out)
	astTestMust(t, err)
	if status != RewriteFailed {
		t.Fatal("default unspecified NaN should refuse", status)
	}
	astTestMust(t, rw.SetHighFPUnspecified(true))
	status, err = rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{nan}), out)
	astTestMust(t, err)
	got, err := m.RefGet(out)
	astTestMust(t, err)
	if status != RewriteDone || got != zero {
		t.Fatal("hardware unspecified NaN", status, got)
	}
	status, err = rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{variable}), out)
	astTestMust(t, err)
	if status != RewriteFailed {
		t.Fatal("symbolic fp.to_real", status)
	}
	astTestMust(t, rw.SetHighFPUnspecified(false))
	status, err = rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{nan}), out)
	astTestMust(t, err)
	if status != RewriteFailed {
		t.Fatal("disabled hardware unspecified NaN", status)
	}
}
