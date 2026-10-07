// Source ast_manager proof bodies; these tests do not replace proof_checker.
package ast

import (
	"errors"
	"testing"
)

func proofTestApp(t *testing.T, m *ASTManager, k int32, args ...ASTHandle) ASTHandle {
	t.Helper()
	h, e := m.MkFamilyApp(0, k, nil, args)
	astTestMust(t, e)
	return h
}
func proofTestBool(t *testing.T, m *ASTManager, name string) ASTHandle {
	t.Helper()
	s, e := m.MkBoolSort()
	astTestMust(t, e)
	h, e := m.MkConst(astTestSymbol(t, m, name), s)
	astTestMust(t, e)
	return h
}
func proofTestFact(t *testing.T, m *ASTManager, p, want ASTHandle) {
	t.Helper()
	h, e := m.GetProofFact(p)
	astTestMust(t, e)
	if h != want {
		t.Fatal("source fact", h, want)
	}
}
func TestASTProofModesReadersAndLegacyHelper(t *testing.T) {
	m := astTestManager(t)
	a := proofTestBool(t, m, "a")
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	p, e := m.MkProofHypothesis(a)
	astTestMust(t, e)
	yes, e := m.IsProof(p)
	astTestMust(t, e)
	if !yes {
		t.Fatal("real Proof sort")
	}
	proofTestFact(t, m, p, a)
	n, e := m.GetProofNumParents(p)
	astTestMust(t, e)
	if n != 0 {
		t.Fatal(n)
	}
	undef, e := m.UndefProof()
	astTestMust(t, e)
	yes, e = m.ProofHasFact(undef)
	astTestMust(t, e)
	if yes {
		t.Fatal("undef has no fact")
	}
	assertion, e := m.MkProofAsserted(a)
	astTestMust(t, e)
	proofTestFact(t, m, assertion, a)
	if _, e = m.MkProofGoal(a); !errors.Is(e, ErrASTDomain) {
		t.Fatal("source fixed goal declaration expects two parents", e)
	}
	// The protected source three-input helper actually forwards only two inputs.
	b := proofTestBool(t, m, "b")
	m.mu.Lock()
	r, e := m.proofThree(0, ASTPrLemma, p, a, b)
	m.mu.Unlock()
	astTestMust(t, e)
	proofTestFact(t, m, m.external(r), a)
	astTestMust(t, m.ToggleProofMode(ASTProofDisabled))
	q, e := m.MkProofAsserted(a)
	astTestMust(t, e)
	if !q.IsNull() {
		t.Fatal("disabled helper")
	}
	q, e = m.MkProofRewrite(a, b)
	astTestMust(t, e)
	if !q.IsNull() {
		t.Fatal("disabled rewrite")
	}
	q, e = m.MkProofHypothesis(a)
	astTestMust(t, e)
	if q != p {
		t.Fatal("hypothesis directly constructs even when disabled")
	}
	q, e = m.MkProofReflexivity(a)
	astTestMust(t, e)
	if q.IsNull() {
		t.Fatal("reflexivity directly constructs even when disabled")
	}
}
func TestASTProofSymmetryTransitivityAndModusPonens(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	a, b, c := proofTestBool(t, m, "a"), proofTestBool(t, m, "b"), proofTestBool(t, m, "c")
	ab, bc := proofTestApp(t, m, ASTOpIff, a, b), proofTestApp(t, m, ASTOpIff, b, c)
	p, e := m.MkProofHypothesis(ab)
	astTestMust(t, e)
	q, e := m.MkProofHypothesis(bc)
	astTestMust(t, e)
	s, e := m.MkProofSymmetry(p)
	astTestMust(t, e)
	proofTestFact(t, m, s, proofTestApp(t, m, ASTOpIff, b, a))
	ss, e := m.MkProofSymmetry(s)
	astTestMust(t, e)
	if ss != p {
		t.Fatal("double symmetry identity")
	}
	r, e := m.MkProofTransitivity(p, q)
	astTestMust(t, e)
	proofTestFact(t, m, r, proofTestApp(t, m, ASTOpIff, a, c))
	rf, e := m.MkProofReflexivity(a)
	astTestMust(t, e)
	r, e = m.MkProofTransitivity(rf, p)
	astTestMust(t, e)
	if r != p {
		t.Fatal("left reflexivity identity")
	}
	pa, e := m.MkProofHypothesis(a)
	astTestMust(t, e)
	mp, e := m.MkProofModusPonens(pa, p)
	astTestMust(t, e)
	proofTestFact(t, m, mp, b)
	r, e = m.MkProofModusPonens(ASTHandle{}, p)
	astTestMust(t, e)
	if !r.IsNull() {
		t.Fatal("null early return")
	}
	r, e = m.MkProofTransitivity(ASTHandle{}, p)
	astTestMust(t, e)
	if r != p {
		t.Fatal("null transitivity identity")
	}
}

func TestASTProofReleaseShortCircuitReaders(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	a, b := proofTestBool(t, m, "a"), proofTestBool(t, m, "b")
	undef, e := m.UndefProof()
	astTestMust(t, e)
	eq := proofTestApp(t, m, ASTOpIff, a, b)
	p, e := m.MkProofHypothesis(eq)
	astTestMust(t, e)
	mp, e := m.MkProofModusPonens(undef, p)
	if ASTSourceDebug {
		if !errors.Is(e, ErrASTDomain) {
			t.Fatal("source has_fact assertion", e)
		}
	} else {
		astTestMust(t, e)
		proofTestFact(t, m, mp, b)
	}
	reflex, e := m.MkProofReflexivity(a)
	astTestMust(t, e)
	out, e := m.MkProofTransitivity(reflex, undef)
	if ASTSourceDebug {
		if !errors.Is(e, ErrASTDomain) {
			t.Fatal("source has_fact assertion before identity", e)
		}
	} else {
		astTestMust(t, e)
		if out != undef {
			t.Fatal("release reflexivity identity before fact reads")
		}
	}
}

func TestASTProofModusPonensReflexivityBeforeReleaseFact(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	a := proofTestBool(t, m, "a")
	p, e := m.MkProofHypothesis(a)
	astTestMust(t, e)
	s, e := m.MkFamilySort(0, 1, nil)
	astTestMust(t, e)
	info, e := m.NewFuncDeclInfo(0, ASTPrReflexivity, nil, ASTFuncFlags{}, false)
	astTestMust(t, e)
	defer m.DelFuncDeclInfo(&info)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "zero-arity-reflexivity"), nil, s, info)
	astTestMust(t, e)
	q, e := m.MkApp(d, nil)
	astTestMust(t, e)
	r, e := m.MkProofModusPonens(p, q)
	if ASTSourceDebug {
		if !errors.Is(e, ErrASTDomain) {
			t.Fatal("source has_fact before identity", e)
		}
	} else {
		astTestMust(t, e)
		if r != p {
			t.Fatal("reflexivity release short circuit", r, p)
		}
	}
}
