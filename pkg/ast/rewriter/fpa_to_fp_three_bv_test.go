package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestFPAToFPThreeBVDispatch(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	rw, err := NewFPARewriter(m)
	astTestMust(t, err)
	defer func() { astTestMust(t, rw.Close()) }()
	signSort, err := m.BitVectorSort(1)
	astTestMust(t, err)
	expSort, err := m.BitVectorSort(8)
	astTestMust(t, err)
	sigSort, err := m.BitVectorSort(23)
	astTestMust(t, err)
	decl, err := m.MkFamilyFuncDecl(rw.family, ast.FPAOpToFP, nil,
		[]ast.ASTHandle{signSort, expSort, sigSort}, ast.ASTHandle{})
	astTestMust(t, err)
	astTestPin(t, m, decl)
	sign, err := m.BitVectorNumeralUint64(1, 1)
	astTestMust(t, err)
	astTestPin(t, m, sign)
	exponent, err := m.BitVectorNumeralUint64(127, 8)
	astTestMust(t, err)
	astTestPin(t, m, exponent)
	significand, err := m.BitVectorNumeralUint64(0, 23)
	astTestMust(t, err)
	astTestPin(t, m, significand)
	out := astTestPin(t, m, sign)
	status, err := rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{sign, exponent, significand}), out)
	astTestMust(t, err)
	got, err := m.RefGet(out)
	astTestMust(t, err)
	kind, err := m.GetDeclKind(got)
	astTestMust(t, err)
	if status != RewriteDone || kind != 5 {
		t.Fatal("three-BV FP conversion dispatch", status, kind)
	}
	symbolic, err := m.MkVar(0, sigSort)
	astTestMust(t, err)
	status, err = rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{sign, exponent, symbolic}), out)
	astTestMust(t, err)
	if status != RewriteFailed {
		t.Fatal("symbolic significand should refuse", status)
	}
}
