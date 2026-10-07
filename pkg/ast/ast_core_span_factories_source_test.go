// Source: src/ast/ast_util.cpp:173-180; src/ast/ast.cpp:2064-2211.
// Source authors: Leonardo de Moura; Nikolaj Bjorner.
// Licensed under the MIT license; see LICENSE.txt.
package ast

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestASTOrSpanSourceZeroSingletonAndInitializedSpare(t *testing.T) {
	m := astTestManager(t)
	f, e := m.MkFalse()
	astTestMust(t, e)
	got, e := m.MkOrSpan(0, ASTLiteralVectorSpan{})
	astTestMust(t, e)
	if got != f {
		t.Fatal("zero source false", got)
	}
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	x, e := m.MkFreshConst("x", sort)
	astTestMust(t, e)
	astTestPin(t, m, x)
	y, e := m.MkFreshConst("y", sort)
	astTestMust(t, e)
	astTestPin(t, m, y)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: m.BudgetField})
	astTestMust(t, e)
	v, e := NewRuntimeASTLiteralVector(r, m.BudgetField, nil)
	astTestMust(t, e)
	astTestMust(t, v.Push(x))
	astTestMust(t, v.Push(y))
	span, e := v.CPtr()
	astTestMust(t, e)
	astTestMust(t, v.Reset())
	got, e = m.MkOrSpan(1, span)
	astTestMust(t, e)
	if got != x {
		t.Fatal("raw singleton identity", got, x)
	}
	got, e = m.MkOrSpan(2, span)
	astTestMust(t, e)
	want, e := m.MkOr(x, y)
	astTestMust(t, e)
	if got != want {
		t.Fatal("same actual registration engine", got, want)
	}
	astTestPin(t, m, got)
	astTestMust(t, v.Close())
	got, e = m.MkOrSpan(0, span)
	astTestMust(t, e)
	if got != f {
		t.Fatal("zero must leave dead provider unread")
	}
	if _, e = m.MkOrSpan(1, span); e == nil {
		t.Fatal("singleton reaches captured dead backing")
	}
	astTestMust(t, r.Close())
}
