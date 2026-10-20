package rewriter

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func safeReplaceArg(t *testing.T, m *ast.ASTManager, h ast.ASTHandle, i uint32) ast.ASTHandle {
	t.Helper()
	result, err := m.GetArg(h, i)
	astTestMust(t, err)
	return result
}

func TestExprSafeReplaceApplicationsDuplicatesAndReset(t *testing.T) {
	m := astTestManager(t)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	a, err := m.MkConst(astTestSymbol(t, m, "safe-a"), boolSort)
	astTestMust(t, err)
	b, err := m.MkConst(astTestSymbol(t, m, "safe-b"), boolSort)
	astTestMust(t, err)
	c, err := m.MkConst(astTestSymbol(t, m, "safe-c"), boolSort)
	astTestMust(t, err)
	root, err := m.MkFamilyApp(0, ast.ASTOpAnd, nil, []ast.ASTHandle{a, a})
	astTestMust(t, err)
	astTestPin(t, m, root)
	replace, err := NewExprSafeReplace(m)
	astTestMust(t, err)
	astTestMust(t, replace.Insert(a, b))
	astTestMust(t, replace.Insert(a, c))
	out, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	astTestMust(t, replace.Apply(root, out))
	got, err := m.RefGet(out)
	astTestMust(t, err)
	if safeReplaceArg(t, m, got, 0) != c || safeReplaceArg(t, m, got, 1) != c {
		t.Fatal("last duplicate substitution did not win")
	}
	astTestMust(t, replace.Reset())
	empty, err := replace.Empty()
	astTestMust(t, err)
	if !empty {
		t.Fatal("reset substitution")
	}
	astTestMust(t, m.DelRef(&out))
	astTestMust(t, replace.Close())
	if _, err = replace.Empty(); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatalf("closed guard: %v", err)
	}
}

func TestExprSafeReplaceShiftsThroughQuantifierAnnotations(t *testing.T) {
	m := astTestManager(t)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	v0, err := m.MkVar(0, boolSort)
	astTestMust(t, err)
	v1, err := m.MkVar(1, boolSort)
	astTestMust(t, err)
	v2, err := m.MkVar(2, boolSort)
	astTestMust(t, err)
	body, err := m.MkFamilyApp(0, ast.ASTOpAnd, nil, []ast.ASTHandle{v1, v0})
	astTestMust(t, err)
	triggerDecl, err := m.MkFuncDecl(astTestSymbol(t, m, "safe-trigger"), []ast.ASTHandle{boolSort}, boolSort, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	trigger, err := m.MkApp(triggerDecl, []ast.ASTHandle{v1})
	astTestMust(t, err)
	pattern, err := m.MkFamilyApp(2, 0, nil, []ast.ASTHandle{trigger})
	astTestMust(t, err)
	q, err := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall:   true,
		Sorts:    []ast.ASTHandle{boolSort},
		Names:    []usymbol.Symbol{astTestSymbol(t, m, "x")},
		Body:     body,
		Patterns: []ast.ASTHandle{pattern},
	})
	astTestMust(t, err)
	astTestPin(t, m, q)
	replace, err := NewExprSafeReplace(m)
	astTestMust(t, err)
	astTestMust(t, replace.Insert(v0, v2))
	out, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	astTestMust(t, replace.Apply(q, out))
	got, err := m.RefGet(out)
	astTestMust(t, err)
	gotBody, err := m.GetQuantifierBody(got)
	astTestMust(t, err)
	replaced := safeReplaceArg(t, m, gotBody, 0)
	index, err := m.GetVarIndex(replaced)
	astTestMust(t, err)
	if index != 3 || safeReplaceArg(t, m, gotBody, 1) != v0 {
		t.Fatal("binder-safe body replacement", index)
	}
	gotPattern, err := m.GetPattern(got, 0)
	astTestMust(t, err)
	patternTrigger := safeReplaceArg(t, m, gotPattern, 0)
	patternVar := safeReplaceArg(t, m, patternTrigger, 0)
	index, err = m.GetVarIndex(patternVar)
	astTestMust(t, err)
	if index != 3 {
		t.Fatal("binder-safe pattern replacement", index)
	}
	qNoPattern, err := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall:     true,
		Sorts:      []ast.ASTHandle{boolSort},
		Names:      []usymbol.Symbol{astTestSymbol(t, m, "y")},
		Body:       body,
		NoPatterns: []ast.ASTHandle{trigger},
	})
	astTestMust(t, err)
	astTestPin(t, m, qNoPattern)
	noPatternOut, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	astTestMust(t, replace.Apply(qNoPattern, noPatternOut))
	gotNoPatternQ, err := m.RefGet(noPatternOut)
	astTestMust(t, err)
	gotNoPattern, err := m.GetNoPattern(gotNoPatternQ, 0)
	astTestMust(t, err)
	noPatternVar := safeReplaceArg(t, m, gotNoPattern, 0)
	index, err = m.GetVarIndex(noPatternVar)
	astTestMust(t, err)
	if index != 3 {
		t.Fatal("binder-safe no-pattern replacement", index)
	}
	astTestMust(t, m.DelRef(&noPatternOut))
	astTestMust(t, m.DelRef(&out))
	astTestMust(t, replace.Close())
}

func TestExprSafeReplaceApplySubstitutionAndSortGuard(t *testing.T) {
	m := astTestManager(t)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	a, err := m.MkConst(astTestSymbol(t, m, "apply-a"), boolSort)
	astTestMust(t, err)
	b, err := m.MkConst(astTestSymbol(t, m, "apply-b"), boolSort)
	astTestMust(t, err)
	astTestPin(t, m, a)
	astTestPin(t, m, b)
	value, err := m.NewRef(a)
	astTestMust(t, err)
	replace, err := NewExprSafeReplace(m)
	astTestMust(t, err)
	astTestMust(t, replace.ApplySubstitution(a, b, value))
	if got, e := m.RefGet(value); e != nil || got != b {
		t.Fatalf("one-shot substitution: %v %v", got, e)
	}
	empty, err := replace.Empty()
	astTestMust(t, err)
	if !empty {
		t.Fatal("one-shot substitution retained mapping")
	}
	arith, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	intSort, err := m.MkFamilySort(arith, 1, nil)
	astTestMust(t, err)
	i, err := m.MkConst(astTestSymbol(t, m, "apply-i"), intSort)
	astTestMust(t, err)
	if err = replace.Insert(a, i); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatalf("sort guard: %v", err)
	}
	astTestMust(t, m.DelRef(&value))
	astTestMust(t, replace.Close())
}
