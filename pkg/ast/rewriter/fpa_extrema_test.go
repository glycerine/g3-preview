package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestFPAExtremaNaNInfinitySignedZeroAndSymbolic(t *testing.T) {
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
	out := astTestPin(t, m, pzero)
	for _, kind := range []int32{ast.FPAOpMin, ast.FPAOpMax} {
		decl, e := m.MkFamilyFuncDecl(rw.family, kind, nil, []ast.ASTHandle{sort, sort}, ast.ASTHandle{})
		astTestMust(t, e)
		wantInfinity := pinf
		if kind == ast.FPAOpMin {
			wantInfinity = ninf
		}
		for _, tc := range []struct {
			left, right ast.ASTHandle
			status      RewriteStatus
			want        ast.ASTHandle
		}{
			{nan, variable, RewriteDone, variable},
			{variable, nan, RewriteDone, variable},
			{nan, nan, RewriteDone, nan},
			{pzero, pzero, RewriteDone, pzero},
			{nzero, nzero, RewriteDone, nzero},
			{pzero, nzero, RewriteFailed, pzero},
			{nzero, pzero, RewriteFailed, pzero},
			{variable, pzero, RewriteFailed, pzero},
			{pzero, variable, RewriteFailed, pzero},
			{pinf, ninf, RewriteDone, wantInfinity},
		} {
			astTestMust(t, m.SetRef(out, pzero))
			status, e := rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{tc.left, tc.right}), out)
			astTestMust(t, e)
			got, e := m.RefGet(out)
			astTestMust(t, e)
			if status != tc.status || got != tc.want {
				t.Fatalf("extremum %d (%v,%v): status=%v got=%v want=%v,%v", kind, tc.left, tc.right, status, got, tc.status, tc.want)
			}
		}
	}
}
