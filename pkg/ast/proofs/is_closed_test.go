package proofs

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestProofIsClosedLemmaLiteralsAndScopeReset(t *testing.T) {
	m := proofManager(t)
	proofMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	pin := func(h ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		proofMust(t, m.IncRef(h))
		t.Cleanup(func() { proofMust(t, m.DecRef(h)) })
		return h
	}
	boolSort, err := m.MkBoolSort()
	proofMust(t, err)
	a, err := m.MkConst(symbol.Numerical(917), boolSort)
	proofMust(t, err)
	a = pin(a)
	truth, err := m.MkTrue()
	proofMust(t, err)
	falsehood, err := m.MkFalse()
	proofMust(t, err)
	notFalse, err := m.MkNot(falsehood)
	proofMust(t, err)
	notFalse = pin(notFalse)
	orClause, err := m.MkOr(a, notFalse)
	proofMust(t, err)
	orClause = pin(orClause)
	hypothesis, err := m.MkProofHypothesis(falsehood)
	proofMust(t, err)
	hypothesis = pin(hypothesis)
	asserted, err := m.MkProofAsserted(falsehood)
	proofMust(t, err)
	asserted = pin(asserted)
	lemma := func(fact ast.ASTHandle) ast.ASTHandle {
		p, e := m.MkProofLemma(hypothesis, fact)
		proofMust(t, e)
		return pin(p)
	}
	for _, tc := range []struct {
		proof ast.ASTHandle
		want  bool
	}{
		{hypothesis, false},
		{asserted, true},
		{lemma(truth), true},
		{lemma(a), false},
		{lemma(orClause), true},
		{hypothesis, false},
	} {
		got, e := IsClosed(m, tc.proof)
		proofMust(t, e)
		if got != tc.want {
			t.Fatal("source scoped lemma closure", got, tc.want, tc.proof)
		}
	}
}
