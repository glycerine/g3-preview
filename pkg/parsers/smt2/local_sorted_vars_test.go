package smt2

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMT2SortedVariablesReverseIndicesAndShadowRestoration(t *testing.T) {
	m := localLookupManager(t)
	sort, err := m.MkBoolSort()
	if err != nil {
		t.Fatal(err)
	}
	x, y := symbol.Numerical(51), symbol.Numerical(52)
	var env smt2LocalEnv
	var exprs localExprStack
	baseline := m.Stats().LiveReferences
	level, err := env.bindSortedVariables(m, []symbol.Symbol{x}, []ast.ASTHandle{sort}, 0, &exprs)
	if err != nil || level != 1 {
		t.Fatal(level, err)
	}
	outer, _ := env.find(x)
	level, err = env.bindSortedVariables(m, []symbol.Symbol{x, y}, []ast.ASTHandle{sort, sort}, level, &exprs)
	if err != nil || level != 3 {
		t.Fatal(level, err)
	}
	if got := m.Stats().LiveReferences; got != baseline+3 {
		t.Fatalf("expression stack references: got %d, want %d", got, baseline+3)
	}
	for i, name := range []symbol.Symbol{x, y} {
		local, ok := env.find(name)
		if !ok || local.level != 3 {
			t.Fatalf("binding %d missing or at wrong level: %+v, %v", i, local, ok)
		}
		want, err := m.MkVar(uint32(1-i), sort)
		if err != nil || local.term != want {
			t.Fatalf("binding %d index: got %v, want %v, error %v", i, local.term, want, err)
		}
	}
	if err := env.endScope(); err != nil {
		t.Fatal(err)
	}
	if got, ok := env.find(x); !ok || got != outer || env.contains(y) {
		t.Fatal("inner scope failed to restore shadowed binding")
	}
	if err := exprs.shrink(m, 1); err != nil {
		t.Fatal(err)
	}
	if err := env.endScope(); err != nil || env.contains(x) {
		t.Fatal("outer scope failed to remove binding", err)
	}
	if err := exprs.reset(m); err != nil || m.Stats().LiveReferences != baseline {
		t.Fatal("sorted variable references survived stack reset", err)
	}
}

func TestSMT2SortedVariablesRejectMismatchedInputsBeforeScope(t *testing.T) {
	m := localLookupManager(t)
	var env smt2LocalEnv
	var exprs localExprStack
	level, err := env.bindSortedVariables(m, []symbol.Symbol{symbol.Numerical(1)}, nil, 4, &exprs)
	if err != errSortedVarInputs || level != 4 || len(env.limits) != 0 || len(exprs.refs) != 0 {
		t.Fatal("mismatched input mutated parser state", level, err)
	}
}

func TestSMT2SortedVariablesFailedSortRestoresPriorScope(t *testing.T) {
	m := localLookupManager(t)
	sort, err := m.MkBoolSort()
	if err != nil {
		t.Fatal(err)
	}
	x, y := symbol.Numerical(61), symbol.Numerical(62)
	var env smt2LocalEnv
	var exprs localExprStack
	level, err := env.bindSortedVariables(m, []symbol.Symbol{x}, []ast.ASTHandle{sort}, 0, &exprs)
	if err != nil || level != 1 {
		t.Fatal(level, err)
	}
	previous, _ := env.find(x)
	refs := m.Stats().LiveReferences
	level, err = env.bindSortedVariables(m, []symbol.Symbol{x, y}, []ast.ASTHandle{sort, {}}, level, &exprs)
	if err == nil || level != 1 || len(env.limits) != 1 || len(exprs.refs) != 1 || m.Stats().LiveReferences != refs {
		t.Fatal("failed sort left partial binding state", level, err)
	}
	if current, ok := env.find(x); !ok || current != previous || env.contains(y) {
		t.Fatal("failed sort did not restore previous local")
	}
	if err := env.endScope(); err != nil {
		t.Fatal(err)
	}
	if err := exprs.reset(m); err != nil {
		t.Fatal(err)
	}
}
