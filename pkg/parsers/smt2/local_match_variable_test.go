package smt2

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMT2MatchVariableSourceBindingLevelAndCaseScope(t *testing.T) {
	m := localLookupManager(t)
	boolSort, err := m.MkBoolSort()
	if err != nil {
		t.Fatal(err)
	}
	x := symbol.Numerical(91)
	var env smt2LocalEnv
	var exprs localExprStack
	baseline := m.Stats().LiveReferences
	env.insert(x, smt2Local{level: 2})
	env.beginScope()
	numBindings := uint32(5)
	if err := env.bindMatchVariable(m, x, boolSort, &numBindings, &exprs); err != nil {
		t.Fatal(err)
	}
	v, err := m.MkVar(0, boolSort)
	if err != nil {
		t.Fatal(err)
	}
	local, ok := env.find(x)
	if !ok || local.term != v || local.level != 5 || numBindings != 6 {
		t.Fatal("source post-increment binding and shadow", local, ok, numBindings)
	}
	if top, err := exprs.top(m); err != nil || top != v || m.Stats().LiveReferences != baseline+1 {
		t.Fatal("match pattern expression ownership", top, err)
	}
	// A bad sort must leave the first case binding and stack untouched.
	if err := env.bindMatchVariable(m, symbol.Numerical(92), ast.ASTHandle{}, &numBindings, &exprs); err == nil {
		t.Fatal("invalid pattern sort accepted")
	}
	if numBindings != 6 || len(exprs.refs) != 1 || env.contains(symbol.Numerical(92)) {
		t.Fatal("failed match variable mutated case state")
	}
	if err := env.endScope(); err != nil {
		t.Fatal(err)
	}
	if restored, ok := env.find(x); !ok || restored.level != 2 {
		t.Fatal("match case did not restore shadowed binding", restored, ok)
	}
	if err := exprs.reset(m); err != nil || m.Stats().LiveReferences != baseline {
		t.Fatal("match case retained expression reference", err)
	}
}
