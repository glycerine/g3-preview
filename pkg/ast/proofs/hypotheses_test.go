package proofs

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func proofMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func proofManager(t *testing.T) *ast.ASTManager {
	t.Helper()
	b := arena.NewBudget(1 << 28)
	sy, err := symbol.NewManager(symbol.Config{Budget: b})
	proofMust(t, err)
	n, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}},
	}})
	proofMust(t, err)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: sy, Numerals: n, Budget: b})
	proofMust(t, err)
	t.Cleanup(func() {
		proofMust(t, m.Close())
		proofMust(t, n.Close())
		proofMust(t, sy.Close())
		if b.Used() != 0 {
			t.Fatal("proof owner retained bytes", b.Used())
		}
	})
	return m
}

func TestProofHypothesesConcreteUnionAndInheritedCopy(t *testing.T) {
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
	a, middle, c := atom(901), atom(902), atom(903)
	iff := func(x, y ast.ASTHandle) ast.ASTHandle {
		h, e := m.MkFamilyApp(m.BasicFamilyID(), ast.ASTOpIff, nil, []ast.ASTHandle{x, y})
		proofMust(t, e)
		return pin(h)
	}
	ab, bc := iff(a, middle), iff(middle, c)
	p, err := m.MkProofHypothesis(ab)
	proofMust(t, err)
	p = pin(p)
	q, err := m.MkProofHypothesis(bc)
	proofMust(t, err)
	q = pin(q)
	oneParent, err := m.MkProofSymmetry(p)
	proofMust(t, err)
	oneParent = pin(oneParent)
	joined, err := m.MkProofTransitivity(p, q)
	proofMust(t, err)
	joined = pin(joined)
	asserted, err := m.MkProofAsserted(ab)
	proofMust(t, err)
	asserted = pin(asserted)

	h := newHypothesisSets(m)
	for _, proof := range []ast.ASTHandle{p, q, oneParent, joined, asserted} {
		proofMust(t, h.add(proof))
	}
	if len(h.byProof[p]) != 1 || len(h.byProof[q]) != 1 ||
		len(h.byProof[oneParent]) != 1 || len(h.byProof[joined]) != 2 || h.byProof[asserted] != nil {
		t.Fatal("source hypothesis cardinalities", h.byProof)
	}
	if _, found := h.byProof[joined][ab]; !found {
		t.Fatal("first parent hypothesis was lost")
	}
	if _, found := h.byProof[joined][bc]; !found {
		t.Fatal("second parent hypothesis was lost")
	}
	// A one-parent result borrows its parent's set; the first union copies it.
	h.byProof[oneParent][bc] = struct{}{}
	if _, shared := h.byProof[p][bc]; !shared {
		t.Fatal("one-parent result did not inherit its set")
	}
	delete(h.byProof[p], bc)
	h.byProof[joined][a] = struct{}{}
	if _, changed := h.byProof[p][a]; changed {
		t.Fatal("union mutated its inherited parent")
	}
	h.reset()
	if len(h.byProof) != 0 {
		t.Fatal("source reset retained proof metadata")
	}
}
