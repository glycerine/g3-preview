package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestFPAUnaryNegAbsSpecialNumericAndDoubleNeg(t *testing.T) {
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
	nan, pinf, ninf, pzero, nzero := value(8), value(6), value(7), value(9), value(10)
	variable, err := m.MkVar(0, sort)
	astTestMust(t, err)
	astTestPin(t, m, variable)
	inner, err := m.MkFamilyApp(rw.family, ast.FPAOpNeg, nil, []ast.ASTHandle{variable})
	astTestMust(t, err)
	astTestPin(t, m, inner)
	out := astTestPin(t, m, pinf)
	for _, tc := range []struct {
		kind      int32
		arg, want ast.ASTHandle
		status    RewriteStatus
	}{
		{ast.FPAOpNeg, nan, nan, RewriteDone},
		{ast.FPAOpNeg, pinf, ninf, RewriteDone},
		{ast.FPAOpNeg, ninf, pinf, RewriteDone},
		{ast.FPAOpNeg, pzero, nzero, RewriteDone},
		{ast.FPAOpNeg, nzero, pzero, RewriteDone},
		{ast.FPAOpNeg, inner, variable, RewriteDone},
		{ast.FPAOpAbs, nan, nan, RewriteDone},
		{ast.FPAOpAbs, ninf, pinf, RewriteDone},
		{ast.FPAOpAbs, nzero, pzero, RewriteDone},
		{ast.FPAOpAbs, variable, pinf, RewriteFailed},
	} {
		astTestMust(t, m.SetRef(out, pinf))
		decl, err := m.MkFamilyFuncDecl(rw.family, tc.kind, nil, []ast.ASTHandle{sort}, ast.ASTHandle{})
		astTestMust(t, err)
		status, err := rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{tc.arg}), out)
		astTestMust(t, err)
		got, err := m.RefGet(out)
		astTestMust(t, err)
		if status != tc.status || got != tc.want {
			t.Fatalf("unary %d: status=%v result=%v want status=%v result=%v", tc.kind, status, got, tc.status, tc.want)
		}
	}
}
