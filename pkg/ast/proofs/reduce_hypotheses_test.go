package proofs

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestProofReduceHypothesesLemmaAndResolution(t *testing.T) {
	m := proofManager(t)
	proofMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	pin := func(h ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		proofMust(t, m.IncRef(h))
		t.Cleanup(func() { proofMust(t, m.DecRef(h)) })
		return h
	}
	falsehood, err := m.MkFalse()
	proofMust(t, err)
	truth, err := m.MkTrue()
	proofMust(t, err)
	notFalse, err := m.MkNot(falsehood)
	proofMust(t, err)
	notFalse = pin(notFalse)
	b, err := m.MkBoolSort()
	proofMust(t, err)
	a, err := m.MkConst(symbol.Numerical(927), b)
	proofMust(t, err)
	a = pin(a)
	clause, err := m.MkOr(notFalse, a)
	proofMust(t, err)
	clause = pin(clause)
	hypFalse, err := m.MkProofHypothesis(falsehood)
	proofMust(t, err)
	hypFalse = pin(hypFalse)
	lemma := func(fact ast.ASTHandle) ast.ASTHandle {
		p, e := m.MkProofLemma(hypFalse, fact)
		proofMust(t, e)
		return pin(p)
	}
	reduce := func(h ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		ref, e := m.NewRef(h)
		proofMust(t, e)
		defer func() { proofMust(t, m.DelRef(&ref)) }()
		proofMust(t, ReduceHypotheses(m, ref))
		out, e := m.RefGet(ref)
		proofMust(t, e)
		return pin(out)
	}
	matchingLemma := lemma(notFalse)
	if got := reduce(matchingLemma); got != matchingLemma {
		t.Fatal("matching lemma literal remains in the clause", got, matchingLemma)
	}
	if got := reduce(lemma(a)); got != hypFalse {
		t.Fatal("lemma with no matching hypothesis literal reduces to child", got, hypFalse)
	}
	got := reduce(lemma(clause))
	isLemma, err := m.IsProofKind(got, ast.ASTPrLemma)
	proofMust(t, err)
	fact, err := m.GetProofFact(got)
	proofMust(t, err)
	if !isLemma || fact != notFalse {
		t.Fatal("lemma retained only complementary literal", isLemma, fact, notFalse)
	}
	asserted, err := m.MkProofAsserted(falsehood)
	proofMust(t, err)
	asserted = pin(asserted)
	if got := reduce(asserted); got != asserted {
		t.Fatal("ordinary proof without parents changed", got, asserted)
	}
	hypTrue, err := m.MkProofHypothesis(truth)
	proofMust(t, err)
	hypTrue = pin(hypTrue)
	unit, err := m.MkProofUnitResolution([]ast.ASTHandle{hypFalse, hypTrue})
	proofMust(t, err)
	unit = pin(unit)
	ref, err := m.NewRef(unit)
	proofMust(t, err)
	defer func() { proofMust(t, m.DelRef(&ref)) }()
	proofMust(t, ReduceHypotheses(m, ref))
	resolved, err := m.RefGet(ref)
	proofMust(t, err)
	if resolved != hypFalse {
		t.Fatal("source false unit short-circuits to first parent", resolved, hypFalse)
	}
}

func TestProofReduceOrdinaryParentsReachConcreteUnion(t *testing.T) {
	m := proofManager(t)
	proofMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	pin := func(h ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		proofMust(t, m.IncRef(h))
		t.Cleanup(func() { proofMust(t, m.DecRef(h)) })
		return h
	}
	b, err := m.MkBoolSort()
	proofMust(t, err)
	atom := func(id uint32) ast.ASTHandle {
		h, e := m.MkConst(symbol.Numerical(id), b)
		proofMust(t, e)
		return pin(h)
	}
	a, mid, c := atom(931), atom(932), atom(933)
	iff := func(x, y ast.ASTHandle) ast.ASTHandle {
		h, e := m.MkFamilyApp(m.BasicFamilyID(), ast.ASTOpIff, nil, []ast.ASTHandle{x, y})
		proofMust(t, e)
		return pin(h)
	}
	ab, bc := iff(a, mid), iff(mid, c)
	p, err := m.MkProofHypothesis(ab)
	proofMust(t, err)
	p = pin(p)
	q, err := m.MkProofHypothesis(bc)
	proofMust(t, err)
	q = pin(q)
	root, err := m.MkProofTransitivity(p, q)
	proofMust(t, err)
	root = pin(root)
	r := &hypothesisReducer{manager: m, sets: newHypothesisSets(m), cache: make(map[ast.ASTHandle]ast.ASTHandle)}
	result, err := r.elim(root)
	proofMust(t, err)
	if result != root || len(r.sets.byProof[root]) != 2 {
		t.Fatal("ordinary proof must retain both parent hypotheses", result, root, r.sets.byProof[root])
	}
	if _, found := r.sets.byProof[root][ab]; !found {
		t.Fatal("first hypothesis missing after reducer union")
	}
	if _, found := r.sets.byProof[root][bc]; !found {
		t.Fatal("second hypothesis missing after reducer union")
	}
	proofMust(t, r.close())
	ref, err := m.NewRef(root)
	proofMust(t, err)
	defer func() { proofMust(t, m.DelRef(&ref)) }()
	proofMust(t, ReduceHypotheses(m, ref))
	got, err := m.RefGet(ref)
	proofMust(t, err)
	if got != root {
		t.Fatal("unchanged ordinary proof published a different root", got, root)
	}
}

func TestProofReduceUnitResolutionLiteralAndClauseBranches(t *testing.T) {
	m := proofManager(t)
	proofMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	pin := func(h ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		proofMust(t, m.IncRef(h))
		t.Cleanup(func() { proofMust(t, m.DecRef(h)) })
		return h
	}
	b, err := m.MkBoolSort()
	proofMust(t, err)
	atom := func(id uint32) ast.ASTHandle {
		h, e := m.MkConst(symbol.Numerical(id), b)
		proofMust(t, e)
		return pin(h)
	}
	a, other := atom(941), atom(942)
	notA, err := m.MkNot(a)
	proofMust(t, err)
	notA = pin(notA)
	clause, err := m.MkOr(notA, other)
	proofMust(t, err)
	clause = pin(clause)
	first, err := m.MkProofHypothesis(clause)
	proofMust(t, err)
	first = pin(first)
	unit, err := m.MkProofAsserted(a)
	proofMust(t, err)
	unit = pin(unit)
	resolution, err := m.MkProofUnitResolution([]ast.ASTHandle{first, unit})
	proofMust(t, err)
	resolution = pin(resolution)
	ref, err := m.NewRef(resolution)
	proofMust(t, err)
	defer func() { proofMust(t, m.DelRef(&ref)) }()
	proofMust(t, ReduceHypotheses(m, ref))
	result, err := m.RefGet(ref)
	proofMust(t, err)
	fact, err := m.GetProofFact(result)
	proofMust(t, err)
	if fact != other {
		t.Fatal("full-clause resolution lost its unpaired literal", fact, other)
	}

	hypA, err := m.MkProofHypothesis(a)
	proofMust(t, err)
	hypA = pin(hypA)
	hypNotA, err := m.MkProofHypothesis(notA)
	proofMust(t, err)
	hypNotA = pin(hypNotA)
	falseProof, err := m.MkProofUnitResolution([]ast.ASTHandle{hypA, hypNotA})
	proofMust(t, err)
	falseProof = pin(falseProof)
	lemma, err := m.MkProofLemma(falseProof, clause)
	proofMust(t, err)
	lemma = pin(lemma)
	shortResolution, err := m.MkProofUnitResolution([]ast.ASTHandle{lemma, hypA})
	proofMust(t, err)
	shortResolution = pin(shortResolution)
	proofMust(t, m.SetRef(ref, shortResolution))
	proofMust(t, ReduceHypotheses(m, ref))
	result, err = m.RefGet(ref)
	proofMust(t, err)
	fact, err = m.GetProofFact(result)
	proofMust(t, err)
	if !m.IsFalse(fact) {
		t.Fatal("shortened-clause literal branch should resolve to false", fact)
	}
	closedA, err := m.MkProofAsserted(a)
	proofMust(t, err)
	closedA = pin(closedA)
	withUnit, err := m.MkProofUnitResolution([]ast.ASTHandle{lemma, closedA})
	proofMust(t, err)
	withUnit = pin(withUnit)
	r := &hypothesisReducer{manager: m, sets: newHypothesisSets(m), cache: make(map[ast.ASTHandle]ast.ASTHandle), units: make(map[ast.ASTHandle]ast.ASTHandle)}
	replaced, err := r.elim(withUnit)
	proofMust(t, err)
	if r.cache[hypA] != closedA || len(r.units) != 0 || len(r.trail) != 0 {
		t.Fatal("closed unit must replace matching hypothesis and pop its scope", r.cache[hypA], closedA, r.units, r.trail)
	}
	fact, err = m.GetProofFact(replaced)
	proofMust(t, err)
	if !m.IsFalse(fact) {
		t.Fatal("closed-unit replacement changed contradiction", fact)
	}
	proofMust(t, r.close())
}
