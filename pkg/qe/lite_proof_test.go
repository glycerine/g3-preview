package qe_test

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/qe"
)

func TestLiteNoProofTraversalPreservesCallerProofReference(t *testing.T) {
	f := setupProofs(t, true)
	x := f.variable(t, 0, f.integer)
	q := exists(t, f, []ast.ASTHandle{f.integer}, f.equality(t, x, f.numeral(t, "3")))
	proof, err := f.m.MkProofReflexivity(q)
	must(t, err)
	f.pin(t, proof)
	refs := f.refs(t, q, proof)
	l, err := qe.NewLite(f.m, nil, true, f.config)
	must(t, err)
	t.Cleanup(func() { must(t, l.Close()) })
	must(t, l.Rewrite(refs[0], refs[1]))
	if !f.m.IsTrue(f.get(t, refs[0])) {
		t.Fatal("existential definition not eliminated")
	}
	if f.get(t, refs[1]) != proof {
		t.Fatal("source no-proof traversal modified caller proof")
	}
}
