package ast

import (
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestStaticFeaturesBooleanTraversalClausesSharingAndDepth(t *testing.T) {
	m := astTestManager(t)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	a, err := m.MkConst(astTestSymbol(t, m, "sf-a"), boolSort)
	astTestMust(t, err)
	astTestPin(t, m, a)
	b, err := m.MkConst(astTestSymbol(t, m, "sf-b"), boolSort)
	astTestMust(t, err)
	astTestPin(t, m, b)
	shared, err := m.MkAnd(a, b)
	astTestMust(t, err)
	astTestPin(t, m, shared)
	notA, err := m.MkNot(a)
	astTestMust(t, err)
	astTestPin(t, m, notA)
	clause, err := m.MkOr(shared, notA)
	astTestMust(t, err)
	astTestPin(t, m, clause)
	f, err := NewStaticFeatures(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, f.Close()) })
	astTestMust(t, f.Collect(clause, shared))
	if f.NumRoots != 1 || f.NumClauses != 1 || f.NumBinClauses != 1 || f.SumClauseSize != 2 {
		t.Fatal("root clause accounting", f.NumRoots, f.NumClauses, f.NumBinClauses, f.SumClauseSize)
	}
	if f.NumSharing < 2 || f.MaxDepth != 3 || f.NumFormulaTrees != 1 || f.NumOrAndTrees != 1 {
		t.Fatal("tree accounting", f.NumSharing, f.MaxDepth, f.NumFormulaTrees, f.NumOrAndTrees)
	}
	if f.CNF || f.NumANDs != 1 || f.NumBoolExprs < 3 {
		t.Fatal("gate accounting", f.CNF, f.NumANDs, f.NumBoolExprs)
	}
}

func TestStaticFeaturesTermITEIsNotAFormulaTree(t *testing.T) {
	m := astTestManager(t)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	condition, err := m.MkConst(astTestSymbol(t, m, "sf-ite-condition"), boolSort)
	astTestMust(t, err)
	astTestPin(t, m, condition)
	termSort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "sf-ite-sort"), nil)
	astTestMust(t, err)
	left, err := m.MkConst(astTestSymbol(t, m, "sf-ite-left"), termSort)
	astTestMust(t, err)
	astTestPin(t, m, left)
	right, err := m.MkConst(astTestSymbol(t, m, "sf-ite-right"), termSort)
	astTestMust(t, err)
	astTestPin(t, m, right)
	ite, err := m.MkIte(condition, left, right)
	astTestMust(t, err)
	astTestPin(t, m, ite)
	f, err := NewStaticFeatures(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, f.Close()) })
	astTestMust(t, f.Collect(ite))
	if f.NumITETerms != 1 || f.NumITEFormulas != 0 || f.NumITETrees != 1 {
		t.Fatal("term ITE accounting", f.NumITETerms, f.NumITEFormulas, f.NumITETrees)
	}
	if f.NumFormulaTrees != 0 || f.MaxFormulaDepth != 0 {
		t.Fatal("term ITE counted as formula tree", f.NumFormulaTrees, f.MaxFormulaDepth)
	}
}

func TestStaticFeaturesTheoryFamiliesSortsEqualitiesAndAliens(t *testing.T) {
	m := astTestManager(t)
	f, err := NewStaticFeatures(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, f.Close()) })
	for i, name := range []string{"arith", "bv", "array", "fpa", "seq"} {
		sym, e := m.Symbols.Intern(name)
		astTestMust(t, e)
		fid, e := m.families.GetID(sym)
		astTestMust(t, e)
		if fid != FamilyID(5+i) {
			t.Fatal("static feature utility family construction order", name, fid)
		}
	}
	arith, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	integerSort := m.external(m.arithmeticPlugin.fields[1])
	x, err := m.MkConst(astTestSymbol(t, m, "sf-int-x"), integerSort)
	astTestMust(t, err)
	astTestPin(t, m, x)
	one, err := m.numerals.NewValue()
	astTestMust(t, err)
	defer func() { astTestMust(t, m.numerals.Del(&one)) }()
	astTestMust(t, m.numerals.SetInt64(&one, 1))
	oneExpr, err := m.ArithmeticNumeral(&one, true)
	astTestMust(t, err)
	astTestPin(t, m, oneExpr)
	eq, err := m.MkEq(x, oneExpr)
	astTestMust(t, err)
	astTestPin(t, m, eq)
	userSort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "sf-theory-result"), nil)
	astTestMust(t, err)
	decl, err := m.MkFuncDecl(usymbol.Numerical(8001), []ASTHandle{integerSort}, userSort, ASTFuncDeclInfo{})
	astTestMust(t, err)
	astTestPin(t, m, decl)
	alien, err := m.MkApp(decl, []ASTHandle{oneExpr})
	astTestMust(t, err)
	astTestPin(t, m, alien)
	astTestMust(t, f.Collect(eq, alien))
	if !f.HasInt || f.NumMarkedTheories != 1 || f.NumEqualities != 1 {
		t.Fatal("sort and equality accounting", f.HasInt, f.NumMarkedTheories, f.NumEqualities)
	}
	if int(arith) >= len(f.NumTheoryEqualities) || f.NumTheoryEqualities[arith] != 1 {
		t.Fatal("per-theory equality accounting", arith, f.NumTheoryEqualities)
	}
	if int(arith) >= len(f.NumTheoryConstants) || f.NumTheoryConstants[arith] != 1 {
		t.Fatal("uninterpreted arithmetic constant accounting", arith, f.NumTheoryConstants)
	}
	if f.NumAliens != 1 || int(arith) >= len(f.NumAliensPerFamily) || f.NumAliensPerFamily[arith] != 1 {
		t.Fatal("alien accounting", f.NumAliens, f.NumAliensPerFamily)
	}
	if int(arith) >= len(f.NumTheoryTerms) || f.NumTheoryTerms[arith] != 1 {
		t.Fatal("arithmetic numeral theory term", arith, f.NumTheoryTerms)
	}
}
