package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestFPABV2RMSourceNumeralsAndSymbolicRefusal(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	rw, err := NewFPARewriter(m)
	astTestMust(t, err)
	defer func() { astTestMust(t, rw.Close()) }()
	bv3, err := m.BitVectorSort(3)
	astTestMust(t, err)
	rmSort, err := m.MkFamilySort(rw.family, 1, nil)
	astTestMust(t, err)
	decl, err := m.MkFamilyFuncDecl(rw.family, ast.FPAOpBV2RM, nil, []ast.ASTHandle{bv3}, rmSort)
	astTestMust(t, err)
	astTestPin(t, m, decl)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	bv4, err := m.BitVectorSort(4)
	astTestMust(t, err)
	for _, tc := range []struct {
		domain    []ast.ASTHandle
		rangeSort ast.ASTHandle
	}{
		{nil, rmSort},
		{[]ast.ASTHandle{bv4}, rmSort},
		{[]ast.ASTHandle{boolSort}, rmSort},
		{[]ast.ASTHandle{bv3}, boolSort},
	} {
		if _, e := m.MkFamilyFuncDecl(rw.family, ast.FPAOpBV2RM, nil, tc.domain, tc.rangeSort); e == nil {
			t.Fatalf("invalid bv2rm declaration accepted: %v", tc)
		}
	}
	zero, err := m.BitVectorNumeralUnsigned(0, 3)
	astTestMust(t, err)
	out := astTestPin(t, m, zero)
	for n := uint32(0); n < 8; n++ {
		value, e := m.BitVectorNumeralUnsigned(n, 3)
		astTestMust(t, e)
		if n == 0 {
			app, e := m.MkFamilyApp(rw.family, ast.FPAOpBV2RM, nil, []ast.ASTHandle{value})
			astTestMust(t, e)
			appSort, e := m.GetSort(app)
			astTestMust(t, e)
			if appSort != rmSort {
				t.Fatal("implicit bv2rm range", appSort, rmSort)
			}
		}
		status, e := rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{value}), out)
		astTestMust(t, e)
		if status != RewriteDone {
			t.Fatal("numeric bv2rm was not reduced", n, status)
		}
		got, e := m.RefGet(out)
		astTestMust(t, e)
		kind, e := m.GetDeclKind(got)
		astTestMust(t, e)
		want := int32(n)
		if n > 4 {
			want = 4
		}
		if kind != want {
			t.Fatal("rounding-mode encoding", n, kind, want)
		}
	}
	variable, err := m.MkVar(0, bv3)
	astTestMust(t, err)
	before, err := m.RefGet(out)
	astTestMust(t, err)
	status, err := rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{variable}), out)
	astTestMust(t, err)
	if status != RewriteFailed {
		t.Fatal("symbolic bv2rm unexpectedly reduced", status)
	}
	after, err := m.RefGet(out)
	astTestMust(t, err)
	if after != before {
		t.Fatal("failed rewrite changed result reference")
	}
}
