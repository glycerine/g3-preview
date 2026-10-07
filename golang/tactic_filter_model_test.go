// Source: src/tactic/filter_model_converter.cpp:23-57;
// src/tactic/filter_model_converter.h:26-49.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import (
	"fmt"
	"reflect"
	"testing"
)

func TestTacticFilterModelRealSATBindingsUniverseAndReplacement(t *testing.T) {
	m := astTestManager(t)
	s, _, _ := satSolverCheckFixture(t)
	tr, e := newSATGoalTranslator(m, s)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, tr.Close()) })
	b, e := m.MkBoolSort()
	satOK(t, e)
	var atoms, decls []ASTHandle
	for _, name := range []string{"filter-p", "filter-q", "filter-aux"} {
		a, e := m.MkConst(astTestSymbol(t, m, name), b)
		satOK(t, e)
		d, e := m.GetDecl(a)
		satOK(t, e)
		atoms, decls = append(atoms, a), append(decls, d)
	}
	nq, e := m.MkNot(atoms[1])
	satOK(t, e)
	satOK(t, tr.Compile([]ASTHandle{atoms[0], nq, atoms[2]}))
	r, e := s.Check(nil)
	satOK(t, e)
	if r != lTrue {
		t.Fatal(r)
	}
	values, e := s.AppendModel(nil)
	satOK(t, e)
	old, e := buildSATBooleanModel(tr, values)
	satOK(t, e)
	var out tacticModelRef
	t.Cleanup(func() { satOK(t, out.Close()) })
	satOK(t, out.Set(old))
	sort, e := m.MkUninterpretedSort(astTestSymbol(t, m, "filter-universe"), nil)
	satOK(t, e)
	v, e := m.GetSomeValue(sort)
	satOK(t, e)
	satOK(t, old.derived.RegisterUniverse(sort, []ASTHandle{v}))
	var wantOrder []ASTHandle
	for _, d := range old.constantDeclarations.values {
		if d != decls[2] {
			wantOrder = append(wantOrder, d)
		}
	}
	c, e := newTacticFilterModel(m)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, c.Close()) })
	satOK(t, c.Insert(decls[2]))
	satOK(t, tr.Close())
	satOK(t, s.Close())
	satOK(t, c.ApplyModel(&out, 77))
	got, e := out.Get()
	satOK(t, e)
	if got == old || !old.closed {
		t.Fatal("source replacement did not retire old model")
	}
	if !reflect.DeepEqual(got.constantDeclarations.values, wantOrder) {
		t.Fatal("source constant insertion order", got.constantDeclarations.values, wantOrder)
	}
	removed, e := got.ConstantInterpretation(decls[2])
	satOK(t, e)
	if !removed.IsNull() {
		t.Fatal("auxiliary retained in filtered model")
	}
	for i, expected := range []bool{true, false} {
		value, e := got.ConstantInterpretation(decls[i])
		satOK(t, e)
		truth, e := m.MkTrue()
		satOK(t, e)
		falsity, e := m.MkFalse()
		satOK(t, e)
		if (value == truth) != expected || (value == falsity) == expected {
			t.Fatal("filtered model violates original assertion", i)
		}
	}
	u, e := got.derived.Universe(sort)
	satOK(t, e)
	if !reflect.DeepEqual(u, []ASTHandle{v}) {
		t.Fatal("source universe copy", u)
	}
	satOK(t, c.Close())
	if _, e = m.GetKind(decls[2]); e == nil {
		t.Fatal("filtered auxiliary declaration leaked")
	}
	_, e = m.GetKind(v)
	satOK(t, e)
	satOK(t, out.Close())
	if _, e = m.GetKind(v); e == nil {
		t.Fatal("copied universe value leaked")
	}
}

func TestTacticFilterModelEmptyFilterExternalModelAndNoopCallbacks(t *testing.T) {
	m := astTestManager(t)
	old, e := newSolverModel(m)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, old.Close()) })
	d := modelTestDecl(t, m, "unchanged-binding")
	v, e := m.MkTrue()
	satOK(t, e)
	satOK(t, old.core.RegisterConstant(d, v))
	satOK(t, old.core.IncRef())
	var out tacticModelRef
	t.Cleanup(func() { satOK(t, out.Close()) })
	satOK(t, out.Set(&old.core))
	c, e := newTacticFilterModel(m)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, c.Close()) })
	satOK(t, c.ApplyUnindexed(&out))
	got, e := out.Get()
	satOK(t, e)
	if got == &old.core || old.closed || old.core.references != 1 {
		t.Fatal("empty filter must copy; external old model must survive")
	}
	value, e := got.ConstantInterpretation(d)
	satOK(t, e)
	if value != v {
		t.Fatal("empty filter lost binding")
	}
	labels := tacticModelLabels{budget: m.budget}
	t.Cleanup(labels.Close)
	prefix := astTestSymbol(t, m, "filter-label")
	satOK(t, labels.Append(prefix))
	m.budget.FailAfter(0)
	satOK(t, c.ApplyLabels(&labels, 99))
	satOK(t, c.ApplyLabels(nil, 0))
	satOK(t, c.Cancel())
	if !reflect.DeepEqual(labels.values, []Symbol{prefix}) {
		t.Fatal("filter changed labels")
	}
	m.budget.FailAfter(-1)
	satOK(t, old.core.DecRef())
	if !old.closed || got.closed {
		t.Fatal("external model lifetime crossed replacement")
	}
}

func TestTacticFilterModelDeclarationTrailGrowthDuplicatesAndOwnership(t *testing.T) {
	m := astTestManager(t)
	md, e := newSolverModel(m)
	satOK(t, e)
	var out tacticModelRef
	t.Cleanup(func() { satOK(t, out.Close()) })
	satOK(t, out.Set(&md.core))
	c, e := newTacticFilterModel(m)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, c.Close()) })
	b, e := m.MkBoolSort()
	satOK(t, e)
	var decls []ASTHandle
	for i := 0; i < 40; i++ {
		a, e := m.MkConst(astTestSymbol(t, m, fmt.Sprintf("filter-owned-%d", i)), b)
		satOK(t, e)
		d, e := m.GetDecl(a)
		satOK(t, e)
		satOK(t, c.Insert(d))
		satOK(t, c.Insert(d))
		v, e := m.MkTrue()
		satOK(t, e)
		satOK(t, md.core.RegisterConstant(d, v))
		satOK(t, m.IncRef(a))
		satOK(t, m.DecRef(a))
		decls = append(decls, d)
	}
	satOK(t, c.ApplyUnindexed(&out))
	got, e := out.Get()
	satOK(t, e)
	n, e := got.NumConstants()
	satOK(t, e)
	if n != 0 || !md.closed {
		t.Fatal("all-filtered model must be empty and replace its source", n)
	}
	for _, d := range decls {
		n, e := m.GetRefCount(d)
		satOK(t, e)
		if n != 2 {
			t.Fatal("duplicate trail must retain each reference", n)
		}
	}
	copyOwner := *c
	if e = copyOwner.Insert(decls[0]); e != errTacticModelConverterOwner {
		t.Fatal("copied filter accepted", e)
	}
	satOK(t, c.IncRef())
	satOK(t, c.DecRef())
	for _, d := range decls {
		if _, e = m.GetKind(d); e == nil {
			t.Fatal("declaration trail leaked")
		}
	}
	if e = c.Cancel(); e != errTacticModelConverterOwner {
		t.Fatal("closed filter accepted", e)
	}
}
