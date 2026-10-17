package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestFPAClassificationCoreSpecialValuesAndSymbolicRefusal(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	rw, err := NewFPARewriter(m)
	astTestMust(t, err)
	defer func() { astTestMust(t, rw.Close()) }()
	floatSort, err := m.MkFamilySort(rw.family, 3, nil)
	astTestMust(t, err)
	parameter, err := m.NewParameterAST(floatSort)
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelParameter(&parameter)) }()
	values := []struct {
		kind int32
		want map[ast.FPAClassificationKind]bool
	}{
		{8, map[ast.FPAClassificationKind]bool{ast.FPAIsNaN: true}},
		{6, map[ast.FPAClassificationKind]bool{ast.FPAIsInfinite: true, ast.FPAIsPositive: true}},
		{7, map[ast.FPAClassificationKind]bool{ast.FPAIsInfinite: true, ast.FPAIsNegative: true}},
		{9, map[ast.FPAClassificationKind]bool{ast.FPAIsZero: true, ast.FPAIsPositive: true}},
		{10, map[ast.FPAClassificationKind]bool{ast.FPAIsZero: true, ast.FPAIsNegative: true}},
	}
	kinds := []ast.FPAClassificationKind{ast.FPAIsNaN, ast.FPAIsInfinite, ast.FPAIsZero, ast.FPAIsNormal, ast.FPAIsSubnormal, ast.FPAIsNegative, ast.FPAIsPositive}
	trueTerm, err := m.MkTrue()
	astTestMust(t, err)
	falseTerm, err := m.MkFalse()
	astTestMust(t, err)
	out := astTestPin(t, m, trueTerm)
	for _, tc := range values {
		term, err := m.MkFamilyApp(rw.family, tc.kind, []ast.ASTParameter{parameter}, nil)
		astTestMust(t, err)
		astTestPin(t, m, term)
		for _, kind := range kinds {
			decl, err := m.MkFamilyFuncDecl(rw.family, int32(kind), nil, []ast.ASTHandle{floatSort}, ast.ASTHandle{})
			astTestMust(t, err)
			status, err := rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{term}), out)
			astTestMust(t, err)
			got, err := m.RefGet(out)
			astTestMust(t, err)
			want := falseTerm
			if tc.want[kind] {
				want = trueTerm
			}
			if status != RewriteDone || got != want {
				t.Fatalf("value kind %d, relation %d: status=%v result=%v want=%v", tc.kind, kind, status, got, want)
			}
		}
	}
	variable, err := m.MkVar(0, floatSort)
	astTestMust(t, err)
	decl, err := m.MkFamilyFuncDecl(rw.family, int32(ast.FPAIsNaN), nil, []ast.ASTHandle{floatSort}, ast.ASTHandle{})
	astTestMust(t, err)
	astTestMust(t, m.SetRef(out, trueTerm))
	status, err := rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{variable}), out)
	astTestMust(t, err)
	got, err := m.RefGet(out)
	astTestMust(t, err)
	if status != RewriteFailed || got != trueTerm {
		t.Fatal("symbolic classification changed the caller's output", status, got)
	}
}
