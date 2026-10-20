package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func hoistQuantifier(t *testing.T, m *ast.ASTManager, forall bool, name string, body ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	sort, err := m.MkBoolSort()
	astTestMust(t, err)
	q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: forall, Sorts: []ast.ASTHandle{sort}, Names: []usymbol.Symbol{astTestSymbol(t, m, name)}, Body: body})
	astTestMust(t, err)
	return q
}

func TestQuantifierHoisterPolarityCompatibilityAndBooleanRewrite(t *testing.T) {
	m, c, g, p := boolRewriterConfigFixture(t)
	h, err := NewQuantifierHoister(m, c, g, p)
	astTestMust(t, err)
	defer func() { astTestMust(t, h.Close()) }()
	sort, err := m.MkBoolSort()
	astTestMust(t, err)
	v0, err := m.MkVar(0, sort)
	astTestMust(t, err)
	q := hoistQuantifier(t, m, true, "qx", v0)
	truth, err := m.MkTrue()
	astTestMust(t, err)
	root, err := m.MkAndArgs([]ast.ASTHandle{q, truth})
	astTestMust(t, err)
	astTestPin(t, m, root)
	result, err := h.Hoist(root, false, true)
	astTestMust(t, err)
	if len(result.Variables) != 1 || !result.IsForall {
		t.Fatal("forall extraction", len(result.Variables), result.IsForall)
	}
	formula, err := m.RefGet(result.Formula)
	astTestMust(t, err)
	variable, err := m.RefGet(result.Variables[0])
	astTestMust(t, err)
	if formula != variable {
		t.Fatal("Boolean rewrite did not remove true", formula, variable)
	}
	astTestMust(t, result.Close())

	exists := hoistQuantifier(t, m, false, "ex", v0)
	negated, err := m.MkNot(exists)
	astTestMust(t, err)
	astTestPin(t, m, negated)
	result, err = h.Hoist(negated, false, true)
	astTestMust(t, err)
	if len(result.Variables) != 1 || !result.IsForall {
		t.Fatal("negative existential polarity", len(result.Variables), result.IsForall)
	}
	astTestMust(t, result.Close())

	result, err = h.PullExists(q, false, true)
	astTestMust(t, err)
	if len(result.Variables) != 0 {
		t.Fatal("incompatible forall was extracted")
	}
	unchanged, err := m.RefGet(result.Formula)
	astTestMust(t, err)
	if unchanged != q {
		t.Fatal("incompatible quantifier changed")
	}
	astTestMust(t, result.Close())
}

func TestQuantifierHoisterIndexedTopChainAndNestedExtraction(t *testing.T) {
	m, c, g, p := boolRewriterConfigFixture(t)
	h, err := NewQuantifierHoister(m, c, g, p)
	astTestMust(t, err)
	defer func() { astTestMust(t, h.Close()) }()
	sort, err := m.MkBoolSort()
	astTestMust(t, err)
	v0, err := m.MkVar(0, sort)
	astTestMust(t, err)
	inner := hoistQuantifier(t, m, true, "inner", v0)
	outer := hoistQuantifier(t, m, true, "outer", inner)
	astTestPin(t, m, outer)
	indexed, err := h.PullQuantifierIndexed(true, outer, true, true)
	astTestMust(t, err)
	if indexed.Index != 2 || len(indexed.Names) != 2 || len(indexed.Sorts) != 2 || len(indexed.Variables) != 0 {
		t.Fatal("top quantifier chain", indexed.Index, len(indexed.Names), len(indexed.Sorts), len(indexed.Variables))
	}
	astTestMust(t, indexed.Close())

	a, err := m.MkConst(astTestSymbol(t, m, "hoist-a"), sort)
	astTestMust(t, err)
	nested := hoistQuantifier(t, m, true, "nested", v0)
	root, err := m.MkAndArgs([]ast.ASTHandle{a, nested})
	astTestMust(t, err)
	astTestPin(t, m, root)
	indexed, err = h.PullQuantifierIndexed(true, root, false, true)
	astTestMust(t, err)
	if indexed.Index != 1 || len(indexed.Names) != 1 || len(indexed.Sorts) != 1 || len(indexed.Variables) != 1 {
		t.Fatal("nested indexed extraction", indexed.Index, len(indexed.Names), len(indexed.Sorts), len(indexed.Variables))
	}
	formula, err := m.RefGet(indexed.Formula)
	astTestMust(t, err)
	second, err := m.GetArg(formula, 1)
	astTestMust(t, err)
	index, err := m.GetVarIndex(second)
	astTestMust(t, err)
	if index != 0 {
		t.Fatal("fresh constant was not replaced by de Bruijn variable", index)
	}
	astTestMust(t, indexed.Close())
}

func TestQuantifierHoisterImplicationPolarityRewriteFlagAndInaccessibleTerm(t *testing.T) {
	m, c, g, p := boolRewriterConfigFixture(t)
	h, err := NewQuantifierHoister(m, c, g, p)
	astTestMust(t, err)
	defer func() { astTestMust(t, h.Close()) }()
	sort, err := m.MkBoolSort()
	astTestMust(t, err)
	v0, err := m.MkVar(0, sort)
	astTestMust(t, err)
	q := hoistQuantifier(t, m, true, "antecedent", v0)
	a, err := m.MkConst(astTestSymbol(t, m, "hoist-consequent"), sort)
	astTestMust(t, err)
	implication, err := m.MkImplies(q, a)
	astTestMust(t, err)
	astTestPin(t, m, implication)
	result, err := h.Hoist(implication, false, true)
	astTestMust(t, err)
	if len(result.Variables) != 1 || result.IsForall {
		t.Fatal("negative antecedent must dualize forall", len(result.Variables), result.IsForall)
	}
	astTestMust(t, result.Close())

	truth, err := m.MkTrue()
	astTestMust(t, err)
	conjunction, err := m.MkAndArgs([]ast.ASTHandle{q, truth})
	astTestMust(t, err)
	astTestPin(t, m, conjunction)
	result, err = h.Hoist(conjunction, false, false)
	astTestMust(t, err)
	formula, err := m.RefGet(result.Formula)
	astTestMust(t, err)
	if kind, e := m.GetDeclKind(formula); e != nil || kind != ast.ASTOpAnd {
		t.Fatalf("rewrite=false conjunction: kind=%d err=%v", kind, e)
	}
	astTestMust(t, result.Close())

	decl, err := m.MkFuncDecl(astTestSymbol(t, m, "opaque"), []ast.ASTHandle{sort}, sort, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	opaque, err := m.MkApp(decl, []ast.ASTHandle{q})
	astTestMust(t, err)
	astTestPin(t, m, opaque)
	result, err = h.Hoist(opaque, false, true)
	astTestMust(t, err)
	got, err := m.RefGet(result.Formula)
	astTestMust(t, err)
	if got != opaque || len(result.Variables) != 0 {
		t.Fatal("inaccessible quantifier changed", got, len(result.Variables))
	}
	astTestMust(t, result.Close())
}
