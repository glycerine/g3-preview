//go:build g3trace

package rewriter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestInstantiateQuantifierSourceTraceSites(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	bindBVTraceGlobals(t, m)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	v, e := m.MkVar(0, b)
	astTestMust(t, e)
	q, e := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{b}, Names: []usymbol.Symbol{usymbol.Numerical(0)}, Body: v})
	astTestMust(t, e)
	truth, e := m.MkTrue()
	astTestMust(t, e)
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, m.DelRef(&out)) })
	tags.Enable("var_subst")
	tags.Enable("instantiate_bug")
	astTestMust(t, InstantiateQuantifier(m, q, []ast.ASTHandle{truth}, out))
	got := output.String()
	for _, want := range []string{
		"[var_subst] instantiate src/ast/rewriter/var_subst.cpp:164",
		"[instantiate_bug] instantiate src/ast/rewriter/var_subst.cpp:168",
		"\nusing\ntrue\n\n----->\ntrue\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing trace %q in:\n%s", want, got)
		}
	}
}
