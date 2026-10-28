package tactic

// Source: src/tactic/replace_proof_converter.cpp; src/tactic/equiv_proof_converter.cpp.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestReplaceProofConverterRewritesAssertedLeaves(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	a := goalTestAtom(t, m, "replace-proof-a")
	b := goalTestAtom(t, m, "replace-proof-b")
	c := goalTestAtom(t, m, "replace-proof-c")
	assertedB, err := m.MkProofAsserted(b)
	astTestMust(t, err)
	bToA, err := m.MkProofRewrite(b, a)
	astTestMust(t, err)
	replacement, err := m.MkProofModusPonens(assertedB, bToA)
	astTestMust(t, err)
	assertedA, err := m.MkProofAsserted(a)
	astTestMust(t, err)
	aToC, err := m.MkProofRewrite(a, c)
	astTestMust(t, err)
	source, err := m.MkProofModusPonens(assertedA, aToC)
	astTestMust(t, err)
	converter, err := NewReplaceProofConverter(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, converter.Close()) })
	astTestMust(t, converter.Insert(replacement))
	out, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.DelRef(&out)) })
	astTestMust(t, converter.Convert(m, []ast.ASTHandle{source}, out))
	result, err := m.RefGet(out)
	astTestMust(t, err)
	fact, err := m.GetProofFact(result)
	astTestMust(t, err)
	if fact != c {
		t.Fatal("replacement proof fact", fact, c)
	}
	firstParent, err := m.GetArg(result, 0)
	astTestMust(t, err)
	if firstParent != replacement {
		t.Fatal("asserted leaf was not replaced", firstParent, replacement)
	}
}

func TestEquivProofConverterBuildsModusPonensReplacement(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	a := goalTestAtom(t, m, "equiv-proof-a")
	b := goalTestAtom(t, m, "equiv-proof-b")
	converter, err := NewEquivProofConverter(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, converter.Close()) })
	astTestMust(t, converter.Insert(a, a))
	astTestMust(t, converter.Insert(a, b))
	assertedB, err := m.MkProofAsserted(b)
	astTestMust(t, err)
	out, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.DelRef(&out)) })
	astTestMust(t, converter.Convert(m, []ast.ASTHandle{assertedB}, out))
	result, err := m.RefGet(out)
	astTestMust(t, err)
	fact, err := m.GetProofFact(result)
	astTestMust(t, err)
	if fact != b {
		t.Fatal("equivalence replacement fact", fact, b)
	}
}
