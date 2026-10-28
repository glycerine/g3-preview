package ast

// Source: src/ast/has_free_vars.cpp:20-94; src/ast/has_free_vars.h:22-24.

import (
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTHasFreeVarsBinderDepthSharedNodesAndAnnotations(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	v0, e := m.MkVar(0, b)
	astTestMust(t, e)
	astTestPin(t, m, v0)
	v1, e := m.MkVar(1, b)
	astTestMust(t, e)
	astTestPin(t, m, v1)
	free, e := m.HasFreeVars(v0)
	astTestMust(t, e)
	if !free {
		t.Fatal("top-level variable must be free")
	}
	name := astTestSymbol(t, m, "x")
	bound, e := m.MkQuantifier(ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{b}, Names: []usymbol.Symbol{name}, Body: v0})
	astTestMust(t, e)
	astTestPin(t, m, bound)
	free, e = m.HasFreeVars(bound)
	astTestMust(t, e)
	if free {
		t.Fatal("bound de Bruijn variable reported free")
	}
	open, e := m.MkQuantifier(ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{b}, Names: []usymbol.Symbol{name}, Body: v1})
	astTestMust(t, e)
	astTestPin(t, m, open)
	free, e = m.HasFreeVars(open)
	astTestMust(t, e)
	if !free {
		t.Fatal("variable beyond binder depth not reported free")
	}
	// The same v0 is closed under the quantifier and free as the sibling. The
	// source cache therefore has to key by both expression and delta.
	pair, e := m.MkAndArgs([]ASTHandle{bound, v0})
	astTestMust(t, e)
	astTestPin(t, m, pair)
	free, e = m.HasFreeVars(pair)
	astTestMust(t, e)
	if !free {
		t.Fatal("shared node cache ignored binder delta")
	}
	truth, e := m.MkTrue()
	astTestMust(t, e)
	patternDecl, e := m.MkFuncDecl(astTestSymbol(t, m, "pattern-free"), []ASTHandle{b}, b, ASTFuncDeclInfo{})
	astTestMust(t, e)
	patternApp, e := m.MkApp(patternDecl, []ASTHandle{v1})
	astTestMust(t, e)
	pattern, e := m.MkFamilyApp(2, 0, nil, []ASTHandle{patternApp})
	astTestMust(t, e)
	astTestPin(t, m, pattern)
	annotated, e := m.MkQuantifier(ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{b}, Names: []usymbol.Symbol{name}, Body: truth, Patterns: []ASTHandle{pattern}})
	astTestMust(t, e)
	astTestPin(t, m, annotated)
	free, e = m.HasFreeVars(annotated)
	astTestMust(t, e)
	if free {
		t.Fatal("source traversal must ignore quantifier annotations")
	}
}

func TestASTHasFreeVarsDeepIterativeTraversalAndGuards(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	freeVar, e := m.MkVar(5000, b)
	astTestMust(t, e)
	astTestPin(t, m, freeVar)
	decl, e := m.MkFuncDecl(astTestSymbol(t, m, "free-var-chain"), []ASTHandle{b}, b, ASTFuncDeclInfo{})
	astTestMust(t, e)
	astTestPin(t, m, decl)
	root := freeVar
	for i := 0; i < 4096; i++ {
		root, e = m.MkApp(decl, []ASTHandle{root})
		astTestMust(t, e)
		astTestPin(t, m, root)
	}
	free, e := m.HasFreeVars(root)
	astTestMust(t, e)
	if !free {
		t.Fatal("deep iterative traversal lost free variable")
	}
	truth, e := m.MkTrue()
	astTestMust(t, e)
	free, e = m.HasFreeVars(truth)
	astTestMust(t, e)
	if free {
		t.Fatal("ground constant reported free")
	}
	if _, e = m.HasFreeVars(b); e == nil {
		t.Fatal("non-expression handle accepted")
	}
}
