package proofs_test

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/proofs"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func fixture(t *testing.T) *ast.ASTManager {
	t.Helper()
	b := arena.NewBudget(1 << 30)
	s, err := usymbol.NewManager(usymbol.Config{Budget: b})
	must(t, err)
	n, err := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	must(t, err)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: s, Numerals: n, Budget: b, ProofMode: ast.ASTProofEnabled})
	must(t, err)
	t.Cleanup(func() {
		must(t, m.Close())
		must(t, n.Close())
		must(t, s.Close())
		if b.Used() != 0 {
			t.Fatalf("owner graph retained %d bytes", b.Used())
		}
	})
	return m
}

func boolean(t *testing.T, m *ast.ASTManager, name string) ast.ASTHandle {
	t.Helper()
	sort, err := m.MkBoolSort()
	must(t, err)
	sym, err := m.Symbols.Intern(name)
	must(t, err)
	h, err := m.MkConst(sym, sort)
	must(t, err)
	return h
}

func check(t *testing.T, c *proofs.Checker, p ast.ASTHandle, want bool) []ast.ASTHandle {
	t.Helper()
	got, side, err := c.Check(p)
	must(t, err)
	if got != want {
		t.Fatalf("check = %v, want %v", got, want)
	}
	return side
}

func TestCheckerStructuralRulesAndInvalidConclusion(t *testing.T) {
	m := fixture(t)
	c, err := proofs.NewChecker(m)
	must(t, err)
	a, b, d := boolean(t, m, "a"), boolean(t, m, "b"), boolean(t, m, "d")
	ab, err := m.MkIff(a, b)
	must(t, err)
	pa, err := m.MkProofHypothesis(a)
	must(t, err)
	pab, err := m.MkProofHypothesis(ab)
	must(t, err)
	mp, err := m.MkProofModusPonens(pa, pab)
	must(t, err)
	check(t, c, mp, true)

	bad, err := m.MkFamilyApp(m.BasicFamilyID(), ast.ASTPrModusPonens, nil, []ast.ASTHandle{pa, pab, d})
	must(t, err)
	check(t, c, bad, false)

	symm, err := m.MkProofSymmetry(pab)
	must(t, err)
	check(t, c, symm, true)
	refl, err := m.MkProofReflexivity(a)
	must(t, err)
	check(t, c, refl, true)
}

func TestCheckerEliminationResolutionAndRewriteSideConditions(t *testing.T) {
	m := fixture(t)
	c, err := proofs.NewChecker(m)
	must(t, err)
	a, b := boolean(t, m, "a"), boolean(t, m, "b")
	and, err := m.MkAnd(a, b)
	must(t, err)
	pand, err := m.MkProofAsserted(and)
	must(t, err)
	elim, err := m.MkProofAndElim(pand, 1)
	must(t, err)
	check(t, c, elim, true)

	nb, err := m.MkNot(b)
	must(t, err)
	or, err := m.MkOr(a, b)
	must(t, err)
	por, err := m.MkProofAsserted(or)
	must(t, err)
	pnb, err := m.MkProofAsserted(nb)
	must(t, err)
	unit, err := m.MkProofUnitResolution([]ast.ASTHandle{por, pnb})
	must(t, err)
	check(t, c, unit, true)

	rewrite, err := m.MkProofRewrite(a, b)
	must(t, err)
	side := check(t, c, rewrite, true)
	if len(side) != 1 {
		t.Fatalf("rewrite side conditions = %d", len(side))
	}
}

func TestCheckerLemmaClosesHypotheses(t *testing.T) {
	m := fixture(t)
	c, err := proofs.NewChecker(m)
	must(t, err)
	a := boolean(t, m, "a")
	na, err := m.MkNot(a)
	must(t, err)
	pa, err := m.MkProofHypothesis(a)
	must(t, err)
	pna, err := m.MkProofHypothesis(na)
	must(t, err)
	falseProof, err := m.MkProofUnitResolution([]ast.ASTHandle{pa, pna})
	must(t, err)
	tautology, err := m.MkOr(na, a)
	must(t, err)
	lemma, err := m.MkProofLemma(falseProof, tautology)
	must(t, err)
	check(t, c, lemma, true)

	// A single quoted hypothesis closes directly against its negation.
	f, err := m.MkFalse()
	must(t, err)
	pf, err := m.MkFamilyApp(m.BasicFamilyID(), ast.ASTPrUnitResolution, nil, []ast.ASTHandle{pa, pna, f})
	must(t, err)
	closed, err := m.MkProofLemma(pf, na)
	must(t, err)
	check(t, c, closed, false)
}
