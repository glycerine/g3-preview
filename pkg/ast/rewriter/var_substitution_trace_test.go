package rewriter

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// Source: var_subst.cpp:26-40. The record observes the rewritten result,
// source binding order, and the borrowed argument before the call returns.
func TestVarSubstitutionResultSourceTrace(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	bindBVTraceGlobals(t, m)
	sort, err := m.MkBoolSort()
	astTestMust(t, err)
	v, err := m.MkVar(0, sort)
	astTestMust(t, err)
	astTestPin(t, m, v)
	truth, err := m.MkTrue()
	astTestMust(t, err)
	out := astTestPin(t, m, v)
	tags.Enable("var_subst_bug")
	s, err := NewVarSubstitution(m, true)
	astTestMust(t, err)
	astTestMust(t, s.Substitute(v, []ast.ASTHandle{truth}, out))
	result, err := m.RefGet(out)
	astTestMust(t, err)
	astTestMust(t, s.Close())
	if result != truth {
		t.Fatal("source substitution result", result, truth)
	}
	if !tags.ShouldTrace("var_subst_bug") {
		if output.Len() != 0 {
			t.Fatal("erased substitution trace", output.String())
		}
		return
	}
	want := "-------- [var_subst_bug] operator() src/ast/rewriter/var_subst.cpp:35 ---------\n" +
		"m_std_order: 1\n(:var 0)\nusing\ntrue\n\n------>\ntrue\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source substitution trace got %q, want %q", got, want)
	}
}
