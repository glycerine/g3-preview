package g3

import (
	"errors"
	"testing"
)

func TestASTProofRewriteAndUnaryMatchers(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	a, b := proofTestBool(t, m, "a"), proofTestBool(t, m, "b")
	p, e := m.MkProofRewrite(a, b)
	astTestMust(t, e)
	l, r := b, a
	yes, e := m.MatchProofRewrite(p, &l, &r)
	astTestMust(t, e)
	if !yes || l != a || r != b {
		t.Fatal("source iff extraction", yes, l, r)
	}
	pa, e := m.MkProofAsserted(a)
	astTestMust(t, e)
	l, r = b, a
	yes, e = m.MatchProofRewrite(pa, &l, &r)
	astTestMust(t, e)
	if yes || l != b || r != a {
		t.Fatal("unmatched outputs preserved")
	}
	yes, e = m.MatchProofAsserted(pa, &l)
	astTestMust(t, e)
	if !yes || l != a {
		t.Fatal("asserted unary")
	}
	ph, e := m.MkProofHypothesis(a)
	astTestMust(t, e)
	yes, e = m.MatchProofHypothesis(ph, &l)
	astTestMust(t, e)
	if !yes || l != a {
		t.Fatal("hypothesis unary")
	}
	pf, e := m.MkProofHypothesis(m.external(m.falseApp))
	astTestMust(t, e)
	pl, e := m.MkProofLemma(pf, a)
	astTestMust(t, e)
	l = b
	yes, e = m.MatchProofLemma(pl, &l)
	astTestMust(t, e)
	if yes || l != b {
		t.Fatal("source lemma matcher requires exactly one argument")
	}
	// VERIFY(eq || iff) executes in every build and does not accept OEQ.
	p, e = m.MkProofOEqRewrite(a, b)
	astTestMust(t, e)
	l, r = b, a
	_, e = m.MatchProofRewrite(p, &l, &r)
	if !errors.Is(e, ErrASTDomain) || l != b || r != a {
		t.Fatal("VERIFY failure boundary", e, l, r)
	}
}
