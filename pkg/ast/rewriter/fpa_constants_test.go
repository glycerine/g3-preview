package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestFPAConstantRewriteReturnsApplication(t *testing.T) {
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
	trueTerm, err := m.MkTrue()
	astTestMust(t, err)
	out := astTestPin(t, m, trueTerm)
	for kind := int32(0); kind <= 10; kind++ {
		if kind == 5 {
			continue // numeral declarations require a live MPF external parameter
		}
		var value ast.ASTHandle
		if kind <= 4 {
			value, err = m.MkFamilyApp(rw.family, kind, nil, nil)
		} else {
			value, err = m.MkFamilyApp(rw.family, kind, []ast.ASTParameter{parameter}, nil)
		}
		astTestMust(t, err)
		astTestPin(t, m, value)
		decl, err := m.GetDecl(value)
		astTestMust(t, err)
		astTestMust(t, m.SetRef(out, trueTerm))
		status, err := rw.MkAppCore(decl, ast.SliceArguments(nil), out)
		astTestMust(t, err)
		got, err := m.RefGet(out)
		astTestMust(t, err)
		if status != RewriteDone || got != value {
			t.Fatalf("FPA constant %d: status=%v got=%v want=%v", kind, status, got, value)
		}
		status, err = rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{trueTerm}), out)
		if status != RewriteFailed || err != ast.ErrASTDomain {
			t.Fatalf("FPA constant %d accepted argument: status=%v err=%v", kind, status, err)
		}
	}
}
