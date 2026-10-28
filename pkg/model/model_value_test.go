package model_test

// Source: src/model/model.cpp:93-109; src/ast/ast.cpp:2561-2582.

import (
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"testing"
)

func TestModelSomeValuePrefersFirstUniverseAndBorrows(t *testing.T) {
	m := astTestManager(t)
	md, e := model.NewModel(m)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, md.Close()) })
	s, e := m.MkUninterpretedSort(astTestSymbol(t, m, "value-universe"), nil)
	astTestMust(t, e)
	astTestPin(t, m, s)
	defaultValue, e := m.GetSomeValue(s)
	astTestMust(t, e)
	astTestPin(t, m, defaultValue)
	// Distinct model-value indices ensure the universe choice is observable.
	p0, e := m.NewParameterInt(7)
	astTestMust(t, e)
	p1, e := m.NewParameterAST(s)
	astTestMust(t, e)
	chosen, e := m.MkFamilyApp(3, 0, []ast.ASTParameter{p0, p1}, nil)
	astTestMust(t, e)
	astTestMust(t, m.DelParameter(&p0))
	astTestMust(t, m.DelParameter(&p1))
	astTestPin(t, m, chosen)
	astTestMust(t, md.RegisterUniverse(s, []ast.ASTHandle{chosen, defaultValue}))
	before, e := m.GetRefCount(chosen)
	astTestMust(t, e)
	got, e := md.GetSomeValue(s)
	astTestMust(t, e)
	if got != chosen {
		t.Fatal("first universe member", got, chosen)
	}
	after, e := m.GetRefCount(chosen)
	astTestMust(t, e)
	if after != before {
		t.Fatal("borrow acquired a reference", before, after)
	}
	got, e = md.Core().GetSomeValue(s)
	astTestMust(t, e)
	if got != chosen {
		t.Fatal("virtual core dispatch", got)
	}
	// Lexical override must retire when the model call returns.
	got, e = m.GetSomeValue(s)
	astTestMust(t, e)
	if got != defaultValue {
		t.Fatal("callback escaped lexical scope", got)
	}
}

func TestModelSomeValueMissingEmptyAndBooleanFallback(t *testing.T) {
	m := astTestManager(t)
	md, e := model.NewModel(m)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, md.Close()) })
	s, e := m.MkUninterpretedSort(astTestSymbol(t, m, "empty-value-universe"), nil)
	astTestMust(t, e)
	astTestPin(t, m, s)
	want, e := m.GetSomeValue(s)
	astTestMust(t, e)
	astTestPin(t, m, want)
	got, e := md.GetSomeValue(s)
	astTestMust(t, e)
	if got != want {
		t.Fatal("absent universe fallback", got)
	}
	astTestMust(t, md.RegisterUniverse(s, nil))
	got, e = md.GetSomeValue(s)
	astTestMust(t, e)
	if got != want {
		t.Fatal("empty universe fallback", got)
	}
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	got, e = md.GetSomeValue(b)
	astTestMust(t, e)
	if got != no {
		t.Fatal("basic plugin value", got)
	}
	copyOwner := *md
	if _, e = copyOwner.GetSomeValue(s); e == nil {
		t.Fatal("copied model accepted")
	}
	astTestMust(t, md.Close())
	if _, e = md.GetSomeValue(s); e == nil {
		t.Fatal("closed model accepted")
	}
}
