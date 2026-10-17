package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestFPAToFPUnsignedNumericAndSymbolicDispatch(t *testing.T) {
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
	decl, err := m.MkFamilyFuncDecl(rw.family, ast.FPAOpToFPUnsigned, []ast.ASTParameter{eParam, sParam}, []ast.ASTHandle{rmSort, bvSort}, ast.ASTHandle{})
	astTestMust(t, err)
	astTestPin(t, m, decl)
	rne, err := m.MkFamilyApp(rw.family, 0, nil, nil)
	astTestMust(t, err)
	astTestPin(t, m, rne)
	zero, err := m.BitVectorNumeralUnsigned(0, 32)
	astTestMust(t, err)
	astTestPin(t, m, zero)
	initial, err := m.GetSomeValue(bvSort)
	astTestMust(t, err)
	out := astTestPin(t, m, initial)
	status, err := rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{rne, zero}), out)
	astTestMust(t, err)
	got, err := m.RefGet(out)
	astTestMust(t, err)
	kind, err := m.GetDeclKind(got)
	astTestMust(t, err)
	if status != RewriteDone || kind != 9 {
		t.Fatal("unsigned zero conversion", status, kind)
	}
	symbolic, err := m.MkVar(0, bvSort)
	astTestMust(t, err)
	astTestPin(t, m, symbolic)
	status, err = rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{rne, symbolic}), out)
	astTestMust(t, err)
	if status != RewriteFailed {
		t.Fatal("symbolic BV should refuse", status)
	}
}
