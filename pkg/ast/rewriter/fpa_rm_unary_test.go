package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestFPARMUnarySqrtAndIntegralDispatch(t *testing.T) {
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
	pinf, ninf, pzero, nzero, nan := value(6), value(7), value(9), value(10), value(8)
	variable, err := m.MkVar(0, floatSort)
	astTestMust(t, err)
	astTestPin(t, m, variable)
	variableRM, err := m.MkVar(0, rmSort)
	astTestMust(t, err)
	astTestPin(t, m, variableRM)
	out := astTestPin(t, m, pzero)
	for _, kind := range []int32{ast.FPAOpSqrt, ast.FPAOpRoundToIntegral} {
		decl, e := m.MkFamilyFuncDecl(rw.family, kind, nil, []ast.ASTHandle{rmSort, floatSort}, ast.ASTHandle{})
		astTestMust(t, e)
		for _, tc := range []struct {
			rm, value ast.ASTHandle
			status    RewriteStatus
			want      ast.ASTHandle
		}{
			{rm, pinf, RewriteDone, pinf},
			{rm, pzero, RewriteDone, pzero},
			{rm, nzero, RewriteDone, nzero},
			{rm, nan, RewriteDone, nan},
			{rm, variable, RewriteFailed, pzero},
			{variableRM, pzero, RewriteFailed, pzero},
		} {
			astTestMust(t, m.SetRef(out, pzero))
			status, e := rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{tc.rm, tc.value}), out)
			astTestMust(t, e)
			got, e := m.RefGet(out)
			astTestMust(t, e)
			if status != tc.status || got != tc.want {
				t.Fatalf("RM unary %d (%v,%v): status=%v got=%v want=%v,%v", kind, tc.rm, tc.value, status, got, tc.status, tc.want)
			}
		}
		if kind == ast.FPAOpSqrt {
			astTestMust(t, m.SetRef(out, pzero))
			status, e := rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{rm, ninf}), out)
			astTestMust(t, e)
			got, e := m.RefGet(out)
			astTestMust(t, e)
			if status != RewriteDone || got != nan {
				t.Fatal("negative infinity sqrt", status, got)
			}
		}
	}
}
