package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestFPAFMANumericSpecialAndSymbolicDispatch(t *testing.T) {
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
	decl, err := m.MkFamilyFuncDecl(rw.family, ast.FPAOpFMA, nil, []ast.ASTHandle{rmSort, floatSort, floatSort, floatSort}, ast.ASTHandle{})
	astTestMust(t, err)
	out := astTestPin(t, m, pzero)
	for _, tc := range []struct {
		args   []ast.ASTHandle
		status RewriteStatus
		want   ast.ASTHandle
	}{
		{[]ast.ASTHandle{rm, pzero, pinf, pzero}, RewriteDone, nan},
		{[]ast.ASTHandle{rm, pinf, pzero, pzero}, RewriteDone, nan},
		{[]ast.ASTHandle{rm, variable, pinf, pzero}, RewriteFailed, pzero},
		{[]ast.ASTHandle{variableRM, pzero, pinf, pzero}, RewriteFailed, pzero},
	} {
		astTestMust(t, m.SetRef(out, pzero))
		status, e := rw.MkAppCore(decl, ast.SliceArguments(tc.args), out)
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if status != tc.status || got != tc.want {
			t.Fatal("fma dispatch", tc.args, status, got, tc.status, tc.want)
		}
	}
}
