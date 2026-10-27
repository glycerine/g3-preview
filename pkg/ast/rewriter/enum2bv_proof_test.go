package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestEnum2BVProofModeCapturedUntilCleanup(t *testing.T) {
	m, _, _ := enum2BVFixture(t)
	truth, err := m.MkTrue()
	astTestMust(t, err)
	astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	old, err := m.MkProofAsserted(truth)
	astTestMust(t, err)
	astTestPin(t, m, old)
	astTestMust(t, m.ToggleProofMode(ast.ASTProofDisabled))
	r, err := NewEnum2BVRewriter(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, r.Close()) })
	out, proof := enum2BVRefs(t, m)
	astTestMust(t, m.SetRef(proof, old))
	astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	astTestMust(t, r.Rewrite(truth, out, proof))
	got, err := m.RefGet(proof)
	astTestMust(t, err)
	if got != old {
		t.Fatal("non-proof traversal overwrote caller proof after manager mode changed")
	}
	astTestMust(t, r.Cleanup())
	astTestMust(t, r.Rewrite(truth, out, proof))
	got, err = m.RefGet(proof)
	astTestMust(t, err)
	yes, err := m.IsProofKind(got, ast.ASTPrReflexivity)
	astTestMust(t, err)
	if !yes {
		t.Fatal("cleanup did not reconstruct traversal in current proof mode")
	}
}

func TestEnum2BVProofComposesRewrittenChild(t *testing.T) {
	m, sort, constructors := enum2BVFixture(t)
	astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	r, err := NewEnum2BVRewriter(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, r.Close()) })
	out, proof := enum2BVRefs(t, m)
	x, err := m.MkConst(astTestSymbol(t, m, "proof-light"), sort)
	astTestMust(t, err)
	red, err := m.MkApp(constructors[0], nil)
	astTestMust(t, err)
	eq, err := m.MkEq(x, red)
	astTestMust(t, err)
	root, err := m.MkNot(eq)
	astTestMust(t, err)
	astTestPin(t, m, root)
	astTestMust(t, r.Rewrite(root, out, proof))
	pr, err := m.RefGet(proof)
	astTestMust(t, err)
	yes, err := m.IsProofKind(pr, ast.ASTPrMonotonicity)
	astTestMust(t, err)
	if !yes {
		t.Fatal("parent rewrite must compose child proof by monotonicity")
	}
	n, err := m.GetProofNumParents(pr)
	astTestMust(t, err)
	if n != 1 {
		t.Fatal("child proof count", n)
	}
	child, err := m.GetProofParent(pr, 0)
	astTestMust(t, err)
	yes, err = m.IsProofKind(child, ast.ASTPrRewrite)
	astTestMust(t, err)
	if !yes {
		t.Fatal("missing enum equality rewrite premise")
	}
	fact, err := m.GetProofFact(pr)
	astTestMust(t, err)
	left, err := m.GetArg(fact, 0)
	astTestMust(t, err)
	right, err := m.GetArg(fact, 1)
	astTestMust(t, err)
	result, err := m.RefGet(out)
	astTestMust(t, err)
	if left != root || right != result {
		t.Fatal("proof does not relate original and rewritten expressions")
	}
}

func TestEnum2BVProofComposesQuantifierBody(t *testing.T) {
	m, sort, constructors := enum2BVFixture(t)
	astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	r, err := NewEnum2BVRewriter(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, r.Close()) })
	out, proof := enum2BVRefs(t, m)
	x, err := m.MkConst(astTestSymbol(t, m, "quant-proof-light"), sort)
	astTestMust(t, err)
	red, err := m.MkApp(constructors[0], nil)
	astTestMust(t, err)
	body, err := m.MkEq(x, red)
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	root := enum2BVQuantifier(t, m, boolean, 1, true, body)
	astTestMust(t, r.Rewrite(root, out, proof))
	pr, err := m.RefGet(proof)
	astTestMust(t, err)
	yes, err := m.IsProofKind(pr, ast.ASTPrQuantIntro)
	astTestMust(t, err)
	if !yes {
		t.Fatal("quantifier must introduce rewritten body proof")
	}
	child, err := m.GetProofParent(pr, 0)
	astTestMust(t, err)
	yes, err = m.IsProofKind(child, ast.ASTPrRewrite)
	astTestMust(t, err)
	if !yes {
		t.Fatal("quantifier lost body rewrite premise")
	}
	fact, err := m.GetProofFact(pr)
	astTestMust(t, err)
	left, err := m.GetArg(fact, 0)
	astTestMust(t, err)
	right, err := m.GetArg(fact, 1)
	astTestMust(t, err)
	result, err := m.RefGet(out)
	astTestMust(t, err)
	if left != root || right != result {
		t.Fatal("quantifier proof endpoints")
	}
}
