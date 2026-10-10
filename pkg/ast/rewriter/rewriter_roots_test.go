package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestRewriterNativeResultAndProofRootsGrowth(t *testing.T) {
	m := astTestManager(t)
	satOK(t, m.ToggleProofMode(ast.ASTProofEnabled))
	p := goalTestAtom(t, m, "native-result-root")
	pr, e := m.MkProofHypothesis(p)
	satOK(t, e)
	astTestPin(t, m, pr)
	c, e := newRewriterCore(m, true)
	satOK(t, e)
	defer func() { satOK(t, c.Close()) }()
	beforeP, e := m.GetRefCount(p)
	satOK(t, e)
	beforeProof, e := m.GetRefCount(pr)
	satOK(t, e)
	used := m.Budget().Used()
	m.Budget().FailAfter(0)
	defer m.Budget().FailAfter(-1)
	proofs := uint32(0)
	for i := 0; i < 4097; i++ {
		satOK(t, appendRewriterRoot(m, &c.results, p))
		h := ast.ASTHandle{}
		if i%3 != 0 {
			h = pr
			proofs++
		}
		satOK(t, appendRewriterRoot(m, &c.proofResults, h))
	}
	count, e := m.GetRefCount(p)
	satOK(t, e)
	if count != beforeP+4097 {
		t.Fatal("result references", count, beforeP)
	}
	count, e = m.GetRefCount(pr)
	satOK(t, e)
	if count != beforeProof+proofs {
		t.Fatal("proof references", count, beforeProof, proofs)
	}
	satOK(t, c.elimReflexProofs(1))
	if len(c.proofResults) != int(proofs)+1 || !c.proofResults[0].IsNull() {
		t.Fatal("proof prefix/compaction", len(c.proofResults), proofs)
	}
	count, e = m.GetRefCount(pr)
	satOK(t, e)
	if count != beforeProof+proofs {
		t.Fatal("compaction changed ownership", count, beforeProof, proofs)
	}
	satOK(t, shrinkRewriterRoots(m, &c.results, 7))
	count, e = m.GetRefCount(p)
	satOK(t, e)
	if count != beforeP+7 {
		t.Fatal("preserved result prefix", count, beforeP)
	}
	satOK(t, shrinkRewriterRoots(m, &c.results, 0))
	satOK(t, shrinkRewriterRoots(m, &c.proofResults, 0))
	count, e = m.GetRefCount(p)
	satOK(t, e)
	if count != beforeP {
		t.Fatal("result release", count, beforeP)
	}
	count, e = m.GetRefCount(pr)
	satOK(t, e)
	if count != beforeProof {
		t.Fatal("proof release", count, beforeProof)
	}
	if m.Budget().Used() != used {
		t.Fatal("native root backing charged arena")
	}
}

func TestRewriterNativeRootOnlyReferenceAliasReplacement(t *testing.T) {
	m := astTestManager(t)
	sort, e := m.MkBoolSort()
	satOK(t, e)
	h, e := m.MkVar(201, sort)
	satOK(t, e)
	var roots []ast.ASTHandle
	satOK(t, appendRewriterRoot(m, &roots, h))
	satOK(t, replaceRewriterRoot(m, &roots, 0, h))
	count, e := m.GetRefCount(h)
	satOK(t, e)
	if count != 1 {
		t.Fatal("alias replacement owns one reference", count)
	}
	satOK(t, shrinkRewriterRoots(m, &roots, 0))
	if _, e = m.GetKind(h); e != ast.ErrASTHandle {
		t.Fatal("last root did not delete AST", e)
	}
	// A repeated reset must not decrement the already-consumed reference.
	satOK(t, shrinkRewriterRoots(m, &roots, 0))
}
