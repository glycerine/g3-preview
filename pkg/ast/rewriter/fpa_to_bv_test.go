package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestFPAToBVDispatchFiniteUnspecifiedAndSymbolic(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	rw, err := NewFPARewriter(m)
	astTestMust(t, err)
	defer func() { astTestMust(t, rw.Close()) }()
	fpSort, err := m.MkFamilySort(rw.family, 3, nil)
	astTestMust(t, err)
	rmSort, err := m.MkFamilySort(rw.family, 1, nil)
	astTestMust(t, err)
	rne, err := m.MkFamilyApp(rw.family, 0, nil, nil)
	astTestMust(t, err)
	sortParam, err := m.NewParameterAST(fpSort)
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelParameter(&sortParam)) }()
	plusZero, err := m.MkFamilyApp(rw.family, 9, []ast.ASTParameter{sortParam}, nil)
	astTestMust(t, err)
	nan, err := m.MkFamilyApp(rw.family, 8, []ast.ASTParameter{sortParam}, nil)
	astTestMust(t, err)
	variable, err := m.MkVar(0, fpSort)
	astTestMust(t, err)
	width, err := m.NewParameterInt(4)
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelParameter(&width)) }()
	zero, err := m.BitVectorNumeralUnsigned(0, 4)
	astTestMust(t, err)
	out := astTestPin(t, m, zero)
	for _, kind := range []int32{ast.FPAOpToUBV, ast.FPAOpToSBV} {
		decl, e := m.MkFamilyFuncDecl(rw.family, kind, []ast.ASTParameter{width}, []ast.ASTHandle{rmSort, fpSort}, ast.ASTHandle{})
		astTestMust(t, e)
		for _, tc := range []struct {
			value ast.ASTHandle
			high  bool
			want  RewriteStatus
		}{
			{plusZero, false, RewriteDone},
			{nan, false, RewriteFailed},
			{nan, true, RewriteDone},
			{variable, true, RewriteFailed},
		} {
			astTestMust(t, rw.SetHighFPUnspecified(tc.high))
			status, e := rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{rne, tc.value}), out)
			astTestMust(t, e)
			if status != tc.want {
				t.Fatal("conversion dispatch", kind, tc, status)
			}
			if status == RewriteDone {
				got, e := m.RefGet(out)
				astTestMust(t, e)
				if got != zero {
					t.Fatal("conversion bits", kind, tc, got)
				}
			}
		}
	}
}
