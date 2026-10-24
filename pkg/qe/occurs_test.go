package qe_test

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/qe"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestQEOccursVariableShiftsThroughBodiesAndIgnoresPatterns(t *testing.T) {
	f := setup(t)
	v0, v1, v2 := f.variable(t, 0, f.boolean), f.variable(t, 1, f.boolean), f.variable(t, 2, f.boolean)
	name, err := f.m.Symbols.Intern("bound")
	must(t, err)
	quantifier := func(body ast.ASTHandle, patterns []ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		h, err := f.m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{f.boolean}, Names: []usymbol.Symbol{name}, Body: body, Patterns: patterns})
		must(t, err)
		return f.pin(t, h)
	}
	decl, err := f.m.MkFuncDecl(name, []ast.ASTHandle{f.boolean, f.boolean}, f.boolean, ast.ASTFuncDeclInfo{})
	must(t, err)
	trigger, err := f.m.MkApp(decl, []ast.ASTHandle{v1, v0})
	must(t, err)
	f.pin(t, trigger)
	pattern, err := f.m.MkPattern([]ast.ASTHandle{trigger})
	must(t, err)
	f.pin(t, pattern)
	local := quantifier(v0, nil)
	outer := quantifier(v1, nil)
	nested := quantifier(quantifier(v2, nil), nil)
	patternOnly := quantifier(v0, []ast.ASTHandle{pattern})
	shared, err := f.m.MkAndArray([]ast.ASTHandle{local, outer, local})
	must(t, err)
	f.pin(t, shared)
	for _, tc := range []struct {
		h     ast.ASTHandle
		index uint32
		want  bool
	}{
		{v0, 0, true}, {v0, 1, false}, {local, 0, false}, {outer, 0, true}, {outer, 1, false}, {nested, 0, true}, {nested, 1, false}, {patternOnly, 0, false}, {shared, 0, true},
	} {
		got, err := qe.OccursVariable(f.m, tc.index, tc.h)
		must(t, err)
		if got != tc.want {
			t.Fatal(tc.index, got, tc.want)
		}
	}
}
