package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestFPAToFPIEEEBVNumericAndSymbolicDispatch(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	rw, err := NewFPARewriter(m)
	astTestMust(t, err)
	defer func() { astTestMust(t, rw.Close()) }()
	bvSort, err := m.BitVectorSort(32)
	astTestMust(t, err)
	eParam, err := m.NewParameterInt(8)
	astTestMust(t, err)
	sParam, err := m.NewParameterInt(24)
	astTestMust(t, err)
	defer func() {
		astTestMust(t, m.DelParameter(&sParam))
		astTestMust(t, m.DelParameter(&eParam))
	}()
	decl, err := m.MkFamilyFuncDecl(rw.family, ast.FPAOpToFP, []ast.ASTParameter{eParam, sParam}, []ast.ASTHandle{bvSort}, ast.ASTHandle{})
	astTestMust(t, err)
	astTestPin(t, m, decl)
	bits, err := m.BitVectorNumeralUint64(0x7f800000, 32)
	astTestMust(t, err)
	astTestPin(t, m, bits)
	out := astTestPin(t, m, bits)
	status, err := rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{bits}), out)
	astTestMust(t, err)
	got, err := m.RefGet(out)
	astTestMust(t, err)
	kind, err := m.GetDeclKind(got)
	astTestMust(t, err)
	if status != RewriteDone || kind != 6 {
		t.Fatal("one-BV FP conversion dispatch", status, kind)
	}
	variable, err := m.MkVar(0, bvSort)
	astTestMust(t, err)
	status, err = rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{variable}), out)
	astTestMust(t, err)
	if status != RewriteFailed {
		t.Fatal("symbolic BV should refuse", status)
	}
}
