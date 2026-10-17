package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestFPARMBinaryNumericAndSubtractionRewrite(t *testing.T) {
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
	floatSort, err := m.MkFamilySort(rw.family, 3, nil)
	astTestMust(t, err)
	rm, err := m.MkFamilyApp(rw.family, 0, nil, nil)
	astTestMust(t, err)
	astTestPin(t, m, rm)
	parameter, err := m.NewParameterAST(floatSort)
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelParameter(&parameter)) }()
	value := func(kind int32) ast.ASTHandle {
		h, e := m.MkFamilyApp(rw.family, kind, []ast.ASTParameter{parameter}, nil)
		astTestMust(t, e)
		astTestPin(t, m, h)
		return h
	}
	nan, pinf, pzero := value(8), value(6), value(9)
	variable, err := m.MkVar(0, floatSort)
	astTestMust(t, err)
	astTestPin(t, m, variable)
	variableRM, err := m.MkVar(0, rmSort)
	astTestMust(t, err)
	astTestPin(t, m, variableRM)
	out := astTestPin(t, m, pzero)
	for _, tc := range []struct {
		kind   int32
		args   []ast.ASTHandle
		status RewriteStatus
		want   ast.ASTHandle
	}{
		{ast.FPAOpAdd, []ast.ASTHandle{rm, pinf, pzero}, RewriteDone, pinf},
		{ast.FPAOpMul, []ast.ASTHandle{rm, pzero, pinf}, RewriteDone, nan},
		{ast.FPAOpDiv, []ast.ASTHandle{rm, pzero, pzero}, RewriteDone, nan},
		{ast.FPAOpAdd, []ast.ASTHandle{variableRM, pinf, pzero}, RewriteFailed, pzero},
		{ast.FPAOpMul, []ast.ASTHandle{rm, variable, pinf}, RewriteFailed, pzero},
	} {
		decl, e := m.MkFamilyFuncDecl(rw.family, tc.kind, nil, []ast.ASTHandle{rmSort, floatSort, floatSort}, ast.ASTHandle{})
		astTestMust(t, e)
		astTestMust(t, m.SetRef(out, pzero))
		status, e := rw.MkAppCore(decl, ast.SliceArguments(tc.args), out)
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if status != tc.status || got != tc.want {
			t.Fatal("rounding binary dispatch", tc.kind, tc.args, status, got, tc.status, tc.want)
		}
	}
	subDecl, err := m.MkFamilyFuncDecl(rw.family, ast.FPAOpSub, nil, []ast.ASTHandle{rmSort, floatSort, floatSort}, ast.ASTHandle{})
	astTestMust(t, err)
	negated, err := m.MkFamilyApp(rw.family, ast.FPAOpNeg, nil, []ast.ASTHandle{variable})
	astTestMust(t, err)
	want, err := m.MkFamilyApp(rw.family, ast.FPAOpAdd, nil, []ast.ASTHandle{variableRM, pinf, negated})
	astTestMust(t, err)
	status, err := rw.MkAppCore(subDecl, ast.SliceArguments([]ast.ASTHandle{variableRM, pinf, variable}), out)
	astTestMust(t, err)
	got, err := m.RefGet(out)
	astTestMust(t, err)
	if status != Rewrite2 || got != want {
		t.Fatal("subtraction must structurally rewrite even with symbolic rounding mode", status, got, want)
	}
}
