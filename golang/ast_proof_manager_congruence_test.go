// Licensed under the MIT license.
package g3

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/golang/arena"
)

func TestASTProofTransitivityArraysAndCongruence(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	a, b, c := proofTestBool(t, m, "a"), proofTestBool(t, m, "b"), proofTestBool(t, m, "c")
	p, e := m.MkProofRewrite(a, b)
	astTestMust(t, e)
	q, e := m.MkProofRewrite(b, c)
	astTestMust(t, e)
	ps := []ASTHandle{p, q}
	r, e := m.MkProofTransitivityArray(ps)
	astTestMust(t, e)
	proofTestFact(t, m, r, proofTestApp(t, m, ASTOpIff, a, c))
	r, e = m.MkProofTransitivityStar(ps, a, c)
	astTestMust(t, e)
	yes, e := m.IsProofKind(r, ASTPrTransitivityStar)
	astTestMust(t, e)
	if !yes {
		t.Fatal("source star rule")
	}
	proofTestFact(t, m, r, proofTestApp(t, m, ASTOpIff, a, c))
	x := proofTestApp(t, m, ASTOpNot, a)
	y := proofTestApp(t, m, ASTOpNot, b)
	r, e = m.MkProofCongruence(x, y, []ASTHandle{p})
	astTestMust(t, e)
	proofTestFact(t, m, r, proofTestApp(t, m, ASTOpIff, x, y))
	r, e = m.MkProofOEqCongruence(x, y, []ASTHandle{p})
	astTestMust(t, e)
	proofTestFact(t, m, r, proofTestApp(t, m, ASTOpOEq, x, y))
	rel := proofTestApp(t, m, ASTOpIff, x, y)
	d, e := m.GetDecl(rel)
	astTestMust(t, e)
	r, e = m.MkProofMonotonicity(d, x, y, []ASTHandle{p})
	astTestMust(t, e)
	proofTestFact(t, m, r, rel)
	// These constructors are direct even when the protected proof helper is disabled.
	astTestMust(t, m.ToggleProofMode(ASTProofDisabled))
	r, e = m.MkProofCongruence(x, y, []ASTHandle{p})
	astTestMust(t, e)
	if r.IsNull() {
		t.Fatal("direct congruence")
	}
}

func TestASTProofTransitivityStarEarlyReturnsAndDiagnostics(t *testing.T) {
	m := astTestManager(t)
	r, e := m.MkProofTransitivityStar(nil, ASTHandle{}, ASTHandle{})
	astTestMust(t, e)
	if !r.IsNull() {
		t.Fatal("empty star")
	}
	u, e := m.UndefProof()
	astTestMust(t, e)
	r, e = m.MkProofTransitivityStar([]ASTHandle{u}, ASTHandle{}, ASTHandle{})
	astTestMust(t, e)
	if r != u {
		t.Fatal("singleton returns before inspecting endpoints or proof")
	}
	r, e = m.MkProofTransitivityArray([]ASTHandle{u})
	astTestMust(t, e)
	if r != u {
		t.Fatal("singleton chain")
	}
	_, e = m.MkProofTransitivityArray(nil)
	want := arena.ErrBounds
	if astSourceDebug {
		want = ErrASTDomain
	}
	if !errors.Is(e, want) {
		t.Fatal("empty chain source assertion/storage guard", e)
	}
	a := proofTestBool(t, m, "a")
	reflex, e := m.MkProofReflexivity(a)
	astTestMust(t, e)
	r, e = m.MkProofTransitivityStar([]ASTHandle{reflex, reflex}, a, a)
	if astSourceDebug {
		if !errors.Is(e, ErrASTDomain) {
			t.Fatal("source nonreflexive assertion", e)
		}
	} else {
		astTestMust(t, e)
		proofTestFact(t, m, r, proofTestApp(t, m, ASTOpIff, a, a))
	}
}
