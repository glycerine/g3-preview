package ast

import (
	"errors"
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTProofCanonicalTrueFalseComplements(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	pt, e := m.MkProofHypothesis(m.external(m.trueApp))
	astTestMust(t, e)
	pf, e := m.MkProofHypothesis(m.external(m.falseApp))
	astTestMust(t, e)
	for _, ps := range [][]ASTHandle{{pt, pf}, {pf, pt}} {
		p, e := m.MkProofUnitResolution(ps)
		astTestMust(t, e)
		proofTestFact(t, m, p, m.external(m.falseApp))
		p, e = m.MkProofUnitResolutionFact(ps, m.external(m.falseApp))
		astTestMust(t, e)
		proofTestFact(t, m, p, m.external(m.falseApp))
	}
}

func TestASTProofFalseDiagnosticsUseCanonicalIdentity(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	s, e := m.MkBoolSort()
	astTestMust(t, e)
	info, e := m.NewFuncDeclInfo(0, ASTOpFalse, nil, ASTFuncFlags{}, false)
	astTestMust(t, e)
	defer m.DelFuncDeclInfo(&info)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "noncanonical-false"), nil, s, info)
	astTestMust(t, e)
	fake, e := m.MkApp(d, nil)
	astTestMust(t, e)
	if fake == m.external(m.falseApp) {
		t.Fatal("independent false-kind node")
	}
	p, e := m.MkProofHypothesis(fake)
	astTestMust(t, e)
	a, b := proofTestBool(t, m, "a"), proofTestBool(t, m, "b")
	_, e = m.MkProofLemma(p, a)
	if ASTSourceDebug {
		if !errors.Is(e, ErrASTDomain) {
			t.Fatal("lemma canonical false diagnostic", e)
		}
	} else {
		astTestMust(t, e)
	}
	pt, e := m.MkProofHypothesis(m.external(m.trueApp))
	astTestMust(t, e)
	pf, e := m.MkProofHypothesis(m.external(m.falseApp))
	astTestMust(t, e)
	_, e = m.MkProofUnitResolutionFact([]ASTHandle{pt, pf}, fake)
	if ASTSourceDebug {
		if !errors.Is(e, ErrASTDomain) {
			t.Fatal("pair canonical false diagnostic", e)
		}
	} else {
		astTestMust(t, e)
	}
	clause := proofTestApp(t, m, ASTOpOr, a, b)
	pc, e := m.MkProofHypothesis(clause)
	astTestMust(t, e)
	pa, e := m.MkProofHypothesis(proofTestApp(t, m, ASTOpNot, a))
	astTestMust(t, e)
	pb, e := m.MkProofHypothesis(proofTestApp(t, m, ASTOpNot, b))
	astTestMust(t, e)
	_, e = m.MkProofUnitResolutionFact([]ASTHandle{pc, pa, pb}, fake)
	if ASTSourceDebug {
		if !errors.Is(e, ErrASTDomain) {
			t.Fatal("all matched canonical false diagnostic", e)
		}
	} else {
		astTestMust(t, e)
	}
}

func TestASTProofBooleanRulesAndUnitResolution(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	a, b := proofTestBool(t, m, "a"), proofTestBool(t, m, "b")
	na := proofTestApp(t, m, ASTOpNot, a)
	pa, e := m.MkProofHypothesis(a)
	astTestMust(t, e)
	pn, e := m.MkProofHypothesis(na)
	astTestMust(t, e)
	p, e := m.MkProofUnitResolution([]ASTHandle{pa, pn})
	astTestMust(t, e)
	proofTestFact(t, m, p, m.external(m.falseApp))
	lemma := proofTestApp(t, m, ASTOpOr, a, na)
	p, e = m.MkProofLemma(p, lemma)
	astTestMust(t, e)
	proofTestFact(t, m, p, lemma)
	p, e = m.MkProofIffTrue(pa)
	astTestMust(t, e)
	proofTestFact(t, m, p, proofTestApp(t, m, ASTOpIff, a, m.external(m.trueApp)))
	p, e = m.MkProofIffFalse(pn)
	astTestMust(t, e)
	proofTestFact(t, m, p, proofTestApp(t, m, ASTOpIff, a, m.external(m.falseApp)))
	clause := proofTestApp(t, m, ASTOpOr, a, b, a)
	pc, e := m.MkProofHypothesis(clause)
	astTestMust(t, e)
	p, e = m.MkProofUnitResolution([]ASTHandle{pc, pn})
	astTestMust(t, e)
	proofTestFact(t, m, p, b)
	p, e = m.MkProofUnitResolutionFact([]ASTHandle{pc, pn}, b)
	astTestMust(t, e)
	proofTestFact(t, m, p, b)
	conj := proofTestApp(t, m, ASTOpAnd, a, b)
	pq, e := m.MkProofHypothesis(conj)
	astTestMust(t, e)
	p, e = m.MkProofAndElim(pq, 1)
	astTestMust(t, e)
	proofTestFact(t, m, p, b)
	nor := proofTestApp(t, m, ASTOpNot, proofTestApp(t, m, ASTOpOr, a, na))
	pq, e = m.MkProofHypothesis(nor)
	astTestMust(t, e)
	p, e = m.MkProofNotOrElim(pq, 0)
	astTestMust(t, e)
	proofTestFact(t, m, p, na)
	p, e = m.MkProofNotOrElim(pq, 1)
	astTestMust(t, e)
	proofTestFact(t, m, p, a)
	p, e = m.MkProofCommutativity(proofTestApp(t, m, ASTOpOr, a, b))
	astTestMust(t, e)
	proofTestFact(t, m, p, proofTestApp(t, m, ASTOpIff, proofTestApp(t, m, ASTOpOr, a, b), proofTestApp(t, m, ASTOpOr, b, a)))
}
func TestASTProofQuantifierRewriteAndNNFFamilies(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	a, b := proofTestBool(t, m, "a"), proofTestBool(t, m, "b")
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	q1, e := m.MkQuantifier(ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{sort}, Names: []usymbol.Symbol{astTestSymbol(t, m, "x")}, Body: a})
	astTestMust(t, e)
	q2, e := m.MkQuantifier(ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{sort}, Names: []usymbol.Symbol{astTestSymbol(t, m, "x")}, Body: b})
	astTestMust(t, e)
	p, e := m.MkProofRewrite(a, b)
	astTestMust(t, e)
	out, e := m.MkProofQuantIntro(q1, q2, p)
	astTestMust(t, e)
	proofTestFact(t, m, out, proofTestApp(t, m, ASTOpIff, q1, q2))
	p, e = m.MkProofOEqRewrite(a, b)
	astTestMust(t, e)
	out, e = m.MkProofOEqQuantIntro(q1, q2, p)
	astTestMust(t, e)
	proofTestFact(t, m, out, proofTestApp(t, m, ASTOpOEq, q1, q2))
	// Source calls validation but ignores its false result for NNF parents.
	wrong, e := m.MkProofHypothesis(a)
	astTestMust(t, e)
	valid, e := m.CheckNNFProofParents([]ASTHandle{wrong})
	astTestMust(t, e)
	if valid {
		t.Fatal("independent validator")
	}
	out, e = m.MkProofNNFPos(a, b, []ASTHandle{wrong})
	astTestMust(t, e)
	proofTestFact(t, m, out, proofTestApp(t, m, ASTOpOEq, a, b))
	out, e = m.MkProofNNFNeg(a, b, []ASTHandle{wrong})
	astTestMust(t, e)
	proofTestFact(t, m, out, proofTestApp(t, m, ASTOpOEq, proofTestApp(t, m, ASTOpNot, a), b))
	out, e = m.MkProofRewriteStar(a, b, []ASTHandle{p})
	astTestMust(t, e)
	proofTestFact(t, m, out, proofTestApp(t, m, ASTOpIff, a, b))
	out, e = m.MkProofPullQuant(a, q1)
	astTestMust(t, e)
	proofTestFact(t, m, out, proofTestApp(t, m, ASTOpIff, a, q1))
	out, e = m.MkProofPushQuant(q1, a)
	astTestMust(t, e)
	proofTestFact(t, m, out, proofTestApp(t, m, ASTOpIff, q1, a))
	out, e = m.MkProofElimUnusedVars(q1, a)
	astTestMust(t, e)
	proofTestFact(t, m, out, proofTestApp(t, m, ASTOpIff, q1, a))
	out, e = m.MkProofDER(q1, a)
	astTestMust(t, e)
	proofTestFact(t, m, out, proofTestApp(t, m, ASTOpIff, q1, a))
	out, e = m.MkProofSkolemization(q1, a)
	astTestMust(t, e)
	proofTestFact(t, m, out, proofTestApp(t, m, ASTOpOEq, q1, a))
	out, e = m.MkProofApplyDefs(a, b, []ASTHandle{p})
	astTestMust(t, e)
	proofTestFact(t, m, out, proofTestApp(t, m, ASTOpOEq, a, b))
}
