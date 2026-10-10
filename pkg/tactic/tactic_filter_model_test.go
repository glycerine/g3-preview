// Source: src/tactic/filter_model_converter.cpp:23-57;
// src/tactic/filter_model_converter.h:26-49.
package tactic

import (
	"fmt"
	"github.com/glycerine/g3/pkg/model"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestTacticFilterModelEmptyFilterExternalModelAndNoopCallbacks(t *testing.T) {
	m := astTestManager(t)
	old, e := model.NewModel(m)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, old.Close()) })
	d := modelTestDecl(t, m, "unchanged-binding")
	v, e := m.MkTrue()
	satOK(t, e)
	satOK(t, old.CoreField.RegisterConstant(d, v))
	satOK(t, old.CoreField.IncRef())
	var out ModelRef
	t.Cleanup(func() { satOK(t, out.Close()) })
	satOK(t, out.Set(&old.CoreField))
	c, e := NewFilterModel(m)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, c.Close()) })
	satOK(t, c.ApplyUnindexed(&out))
	got, e := out.Get()
	satOK(t, e)
	if got == &old.CoreField || old.Closed || old.CoreField.References != 1 {
		t.Fatal("empty filter must copy; external old model must survive")
	}
	value, e := got.ConstantInterpretation(d)
	satOK(t, e)
	if value != v {
		t.Fatal("empty filter lost binding")
	}
	labels := ModelLabels{Budget: m.BudgetField}
	t.Cleanup(labels.Close)
	prefix := astTestSymbol(t, m, "filter-label")
	satOK(t, labels.Append(prefix))
	m.BudgetField.FailAfter(0)
	satOK(t, c.ApplyLabels(&labels, 99))
	satOK(t, c.ApplyLabels(nil, 0))
	satOK(t, c.Cancel())
	if !reflect.DeepEqual(labels.Values, []usymbol.Symbol{prefix}) {
		t.Fatal("filter changed labels")
	}
	m.BudgetField.FailAfter(-1)
	satOK(t, old.CoreField.DecRef())
	if !old.Closed || got.Closed {
		t.Fatal("external model lifetime crossed replacement")
	}
}

func TestTacticFilterModelDeclarationTrailGrowthDuplicatesAndOwnership(t *testing.T) {
	m := astTestManager(t)
	md, e := model.NewModel(m)
	satOK(t, e)
	var out ModelRef
	t.Cleanup(func() { satOK(t, out.Close()) })
	satOK(t, out.Set(&md.CoreField))
	c, e := NewFilterModel(m)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, c.Close()) })
	b, e := m.MkBoolSort()
	satOK(t, e)
	var decls []ast.ASTHandle
	for i := 0; i < 40; i++ {
		a, e := m.MkConst(astTestSymbol(t, m, fmt.Sprintf("filter-owned-%d", i)), b)
		satOK(t, e)
		d, e := m.GetDecl(a)
		satOK(t, e)
		satOK(t, c.Insert(d))
		satOK(t, c.Insert(d))
		v, e := m.MkTrue()
		satOK(t, e)
		satOK(t, md.CoreField.RegisterConstant(d, v))
		satOK(t, m.IncRef(a))
		satOK(t, m.DecRef(a))
		decls = append(decls, d)
	}
	satOK(t, c.ApplyUnindexed(&out))
	got, e := out.Get()
	satOK(t, e)
	n, e := got.NumConstants()
	satOK(t, e)
	if n != 0 || !md.Closed {
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
	if e = copyOwner.Insert(decls[0]); e != ErrModelConverterOwner {
		t.Fatal("copied filter accepted", e)
	}
	satOK(t, c.IncRef())
	satOK(t, c.DecRef())
	for _, d := range decls {
		if _, e = m.GetKind(d); e == nil {
			t.Fatal("declaration trail leaked")
		}
	}
	if e = c.Cancel(); e != ErrModelConverterOwner {
		t.Fatal("closed filter accepted", e)
	}
}

func TestTacticFilterModelCopiesUnfilteredFunctionsAndPreservesSource(t *testing.T) {
	m := astTestManager(t)
	old, e := model.NewModel(m)
	satOK(t, e)
	satOK(t, old.Core().IncRef())
	t.Cleanup(func() { satOK(t, old.Core().DecRef()) })
	var out ModelRef
	satOK(t, out.Set(old.Core()))
	t.Cleanup(func() { satOK(t, out.Close()) })
	c, e := NewFilterModel(m)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, c.Close()) })
	b, e := m.MkBoolSort()
	satOK(t, e)
	yes, e := m.MkTrue()
	satOK(t, e)
	no, e := m.MkFalse()
	satOK(t, e)
	var decls [2]ast.ASTHandle
	var graphs [2]*model.FunctionInterpretation
	for i := range decls {
		decls[i], e = m.MkFuncDecl(astTestSymbol(t, m, fmt.Sprintf("filter-f-%d", i)), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
		satOK(t, e)
		graphs[i], e = model.NewFunctionInterpretation(m, 1)
		satOK(t, e)
		satOK(t, graphs[i].InsertEntry([]ast.ASTHandle{yes}, no))
		satOK(t, graphs[i].SetElse(yes))
		satOK(t, old.Core().RegisterFunction(decls[i], graphs[i]))
	}
	marks, e := ast.NewASTFastMark1(m)
	satOK(t, e)
	defer func() { satOK(t, marks.Close()) }()
	for _, d := range decls {
		satOK(t, marks.Mark(d, true))
	}
	satOK(t, c.Insert(decls[0]))
	satOK(t, c.ApplyUnindexed(&out))
	for _, d := range decls {
		marked, e := marks.IsMarked(d)
		satOK(t, e)
		if !marked {
			t.Fatal("filter changed another caller's marks")
		}
	}
	next, e := out.Get()
	satOK(t, e)
	if next.NumFunctionDeclarations() != 1 {
		t.Fatal("filter lost unmarked function")
	}
	removed, e := next.FunctionInterpretation(decls[0])
	satOK(t, e)
	if removed != nil {
		t.Fatal("marked function was copied")
	}
	kept, e := next.FunctionInterpretation(decls[1])
	satOK(t, e)
	if kept == nil || kept == graphs[1] || kept.NumEntries() != 1 {
		t.Fatal("function graph was shared or dropped")
	}
	satOK(t, kept.SetElse(no))
	original, e := graphs[1].Else()
	satOK(t, e)
	if original != yes || old.Closed {
		t.Fatal("filtered model changed external source")
	}
}
