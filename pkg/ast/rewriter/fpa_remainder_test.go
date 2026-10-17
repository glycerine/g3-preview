package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestFPARemainderNumericSpecialAndSymbolicRefusal(t *testing.T) {
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
	value := func(kind int32) ast.ASTHandle {
		h, e := m.MkFamilyApp(rw.family, kind, []ast.ASTParameter{parameter}, nil)
		astTestMust(t, e)
		astTestPin(t, m, h)
		return h
	}
	nan, pinf, pzero, nzero := value(8), value(6), value(9), value(10)
	variable, err := m.MkVar(0, sort)
	astTestMust(t, err)
	astTestPin(t, m, variable)
	decl, err := m.MkFamilyFuncDecl(rw.family, ast.FPAOpRem, nil, []ast.ASTHandle{sort, sort}, ast.ASTHandle{})
	astTestMust(t, err)
	out := astTestPin(t, m, pzero)
	for _, tc := range []struct {
		left, right ast.ASTHandle
		status      RewriteStatus
		want        ast.ASTHandle
	}{
		{pzero, pinf, RewriteDone, pzero},
		{nzero, pinf, RewriteDone, nzero},
		{pinf, pzero, RewriteDone, nan},
		{pzero, pzero, RewriteDone, nan},
		{variable, pzero, RewriteFailed, pzero},
		{pzero, variable, RewriteFailed, pzero},
	} {
		astTestMust(t, m.SetRef(out, pzero))
		status, e := rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{tc.left, tc.right}), out)
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if status != tc.status || got != tc.want {
			t.Fatalf("fp.rem(%v,%v): status=%v got=%v want status=%v value=%v", tc.left, tc.right, status, got, tc.status, tc.want)
		}
	}
}
