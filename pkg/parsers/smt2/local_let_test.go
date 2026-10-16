package smt2

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMT2LetLocalsParallelBindingsAndScopeRestoration(t *testing.T) {
	m := localLookupManager(t)
	trueTerm, err := m.MkTrue()
	if err != nil {
		t.Fatal(err)
	}
	falseTerm, err := m.MkFalse()
	if err != nil {
		t.Fatal(err)
	}
	x, y := symbol.Numerical(71), symbol.Numerical(72)
	var env smt2LocalEnv
	env.insert(x, smt2Local{term: trueTerm, level: 2})
	var exprs localExprStack
	baseline := m.Stats().LiveReferences
	for _, term := range []ast.ASTHandle{falseTerm, trueTerm, falseTerm} {
		ref, err := m.NewRef(term)
		if err != nil {
			t.Fatal(err)
		}
		exprs.pushOwned(ref)
	}
	if got, _ := env.find(x); got.term != trueTerm || env.contains(y) {
		t.Fatal("let initializer phase changed visible names")
	}
	if err := env.bindLetLocals(m, []symbol.Symbol{x, y, x}, &exprs, 0, 2); err != nil {
		t.Fatal(err)
	}
	if got, _ := env.find(x); got.term != falseTerm || got.level != 2 {
		t.Fatal("last duplicate let declaration did not win", got)
	}
	if got, _ := env.find(y); got.term != trueTerm || got.level != 2 {
		t.Fatal("let declaration order or binding level changed", got)
	}
	if got := m.Stats().LiveReferences; got != baseline+3 {
		t.Fatalf("local table acquired a reference: got %d, want %d", got, baseline+3)
	}
	result, err := m.NewRef(trueTerm)
	if err != nil {
		t.Fatal(err)
	}
	exprs.pushOwned(result)
	if err := env.finishLetLocalExprs(m, &exprs, 0); err != nil {
		t.Fatal(err)
	}
	if got, _ := env.find(x); got.term != trueTerm || env.contains(y) {
		t.Fatal("let scope did not restore prior bindings")
	}
	if got, err := exprs.top(m); err != nil || got != trueTerm || len(exprs.refs) != 1 || m.Stats().LiveReferences != baseline+1 {
		t.Fatal("let body result did not survive declaration cleanup", got, err)
	}
	if err := exprs.reset(m); err != nil || m.Stats().LiveReferences != baseline {
		t.Fatal("let expression stack retained an initializer", err)
	}
}

func TestSMT2LetLocalsRejectMalformedAndStaleStack(t *testing.T) {
	m := localLookupManager(t)
	var env smt2LocalEnv
	var exprs localExprStack
	x := symbol.Numerical(73)
	if err := env.bindLetLocals(m, []symbol.Symbol{x}, &exprs, 0, 0); err != errLetDeclarations {
		t.Fatal("missing initializer", err)
	}
	exprs.refs = append(exprs.refs, ast.ASTRef{})
	if err := env.bindLetLocals(m, []symbol.Symbol{x}, &exprs, 0, 0); err == nil {
		t.Fatal("stale initializer accepted")
	}
	if len(env.limits) != 0 || env.contains(x) {
		t.Fatal("refused let declaration mutated scope")
	}
}
