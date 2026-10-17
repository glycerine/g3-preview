package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestFPAToIEEEBVDefaultAndHardwareNaNDispatch(t *testing.T) {
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
	pinf, err := m.MkFamilyApp(rw.family, 6, []ast.ASTParameter{parameter}, nil)
	astTestMust(t, err)
	astTestPin(t, m, pinf)
	variable, err := m.MkVar(0, sort)
	astTestMust(t, err)
	astTestPin(t, m, variable)
	decl, err := m.MkFamilyFuncDecl(rw.family, ast.FPAOpToIEEEBV, nil, []ast.ASTHandle{sort}, ast.ASTHandle{})
	astTestMust(t, err)
	zero, err := m.BitVectorNumeralUnsigned(0, 32)
	astTestMust(t, err)
	out := astTestPin(t, m, zero)
	status, err := rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{nan}), out)
	astTestMust(t, err)
	if status != RewriteFailed {
		t.Fatal("default NaN IEEE BV should refuse", status)
	}
	astTestMust(t, rw.SetHighFPUnspecified(true))
	status, err = rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{nan}), out)
	astTestMust(t, err)
	got, err := m.RefGet(out)
	astTestMust(t, err)
	first, err := m.BitVectorNumeralUnsigned(0, 1)
	astTestMust(t, err)
	oneExp, err := m.BitVectorNumeralUnsigned(1, 8)
	astTestMust(t, err)
	negExp, err := m.MkBitVectorNeg(oneExp)
	astTestMust(t, err)
	zeroSig, err := m.BitVectorNumeralUnsigned(0, 22)
	astTestMust(t, err)
	last, err := m.BitVectorNumeralUnsigned(1, 1)
	astTestMust(t, err)
	want, err := m.MkBitVectorConcat([]ast.ASTHandle{first, negExp, zeroSig, last})
	astTestMust(t, err)
	if status != Rewrite1 || got != want {
		t.Fatal("hardware NaN IEEE BV concat", status, got, want)
	}
	wantInf, err := m.BitVectorNumeralUint64(0x7f800000, 32)
	astTestMust(t, err)
	status, err = rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{pinf}), out)
	astTestMust(t, err)
	got, err = m.RefGet(out)
	astTestMust(t, err)
	if status != RewriteDone || got != wantInf {
		t.Fatal("positive infinity IEEE BV", status, got)
	}
	status, err = rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{variable}), out)
	astTestMust(t, err)
	if status != RewriteFailed {
		t.Fatal("symbolic IEEE BV", status)
	}
}
