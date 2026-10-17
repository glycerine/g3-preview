package rewriter

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// Source: array_rewriter.cpp:35-42. The actual select dispatcher emits the
// declaration and each supplied argument before it chooses a rewrite branch.
func TestArrayRewriterAppCoreSourceTrace(t *testing.T) {
	c, g, p, budget := dynAckSourceFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	fid, err := m.RegisterArrayPlugin()
	astTestMust(t, err)
	rw, err := NewArrayRewriter(m, c, g, p)
	astTestMust(t, err)
	defer func() { astTestMust(t, rw.Close()) }()
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	param, err := m.NewParameterAST(boolean)
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelParameter(&param)) }()
	arraySort, err := m.MkFamilySort(fid, 0, []ast.ASTParameter{param, param, param})
	astTestMust(t, err)
	symbol, err := m.Symbols.Intern("array-trace-base")
	astTestMust(t, err)
	baseDecl, err := m.MkFuncDecl(symbol, nil, arraySort, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	base, err := m.MkApp(baseDecl, nil)
	astTestMust(t, err)
	astTestPin(t, m, base)
	truth, err := m.MkTrue()
	astTestMust(t, err)
	selectApp, err := m.MkFamilyApp(fid, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{base, truth, truth})
	astTestMust(t, err)
	astTestPin(t, m, selectApp)
	selectDecl, err := m.GetDecl(selectApp)
	astTestMust(t, err)
	out := astTestPin(t, m, truth)
	tags.Enable("array_rewriter")
	status, err := rw.MkAppCore(selectDecl, ast.SliceArguments([]ast.ASTHandle{base, truth, truth}), out)
	astTestMust(t, err)
	result, err := m.RefGet(out)
	astTestMust(t, err)
	if status != RewriteFailed || result != truth {
		t.Fatal("source select fallback", status, result, truth)
	}
	if !tags.ShouldTrace("array_rewriter") {
		if output.Len() != 0 {
			t.Fatal("erased source trace", output.String())
		}
		return
	}
	want := "-------- [array_rewriter] mk_app_core src/ast/rewriter/array_rewriter.cpp:38 ---------\n" +
		"(declare-fun select ((Array Bool Bool Bool) Bool Bool) Bool)\n" +
		"array-trace-base\ntrue\ntrue\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source dispatch trace got %q, want %q", got, want)
	}
}
