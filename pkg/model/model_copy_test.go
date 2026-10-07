// Source: src/model/model.cpp:44-75.
package model_test

import (
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"testing"
)

func TestModelCopyConstantsFunctionsUniversesAndIndependentGraphs(t *testing.T) {
	m := astTestManager(t)
	original, e := model.NewModel(m)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, original.Close()) })
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	constant := modelTestDecl(t, m, "copy-constant")
	astTestMust(t, original.Core().RegisterConstant(constant, yes))
	d := modelTestFunctionDecl(t, m, "copy-function")
	fi := modelTestFunction(t, m, 1)
	astTestMust(t, fi.InsertEntry([]ast.ASTHandle{yes}, no))
	astTestMust(t, fi.SetElse(yes))
	astTestMust(t, original.Core().RegisterFunction(d, fi))
	s, e := m.MkUninterpretedSort(astTestSymbol(t, m, "copy-universe"), nil)
	astTestMust(t, e)
	value, e := m.GetSomeValue(s)
	astTestMust(t, e)
	astTestMust(t, original.RegisterUniverse(s, []ast.ASTHandle{value}))
	copied, e := original.Copy()
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, copied.Close()) })
	if copied == original {
		t.Fatal("copy returned source owner")
	}
	cf, e := copied.Core().FunctionInterpretation(d)
	astTestMust(t, e)
	if cf == nil || cf == fi || cf.NumEntries() != 1 {
		t.Fatal("graph not independently copied")
	}
	astTestMust(t, cf.SetElse(no))
	fallback, e := fi.Else()
	astTestMust(t, e)
	if fallback != yes {
		t.Fatal("copy changed source graph")
	}
	n, e := copied.Core().NumDeclarations()
	astTestMust(t, e)
	if n != 2 {
		t.Fatal("copy declaration count", n)
	}
	first, e := copied.Core().Declaration(0)
	astTestMust(t, e)
	if first != constant {
		t.Fatal("source copy order: constants before functions", first)
	}
	astTestMust(t, original.Close())
	fallback, e = cf.Else()
	astTestMust(t, e)
	if fallback != no {
		t.Fatal("source close destroyed copied graph")
	}
	v, e := copied.Core().ConstantInterpretation(constant)
	astTestMust(t, e)
	if v != yes {
		t.Fatal("constant copy lifetime")
	}
	u, e := copied.Universe(s)
	astTestMust(t, e)
	if len(u) != 1 || u[0] != value {
		t.Fatal("universe copy lifetime")
	}
}

func TestModelCopyEveryReservationFailureRetainsCleanupOwner(t *testing.T) {
	m := astTestManager(t)
	source, e := model.NewModel(m)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, source.Close()) })
	d := modelTestFunctionDecl(t, m, "refused-copy-function")
	yes, e := m.MkTrue()
	astTestMust(t, e)
	fi := modelTestFunction(t, m, 1)
	astTestMust(t, fi.InsertEntry([]ast.ASTHandle{yes}, yes))
	astTestMust(t, fi.SetElse(yes))
	astTestMust(t, source.Core().RegisterFunction(d, fi))
	constant := modelTestDecl(t, m, "refused-copy-constant")
	astTestMust(t, source.Core().RegisterConstant(constant, yes))
	sort, e := m.MkUninterpretedSort(astTestSymbol(t, m, "refused-copy-universe"), nil)
	astTestMust(t, e)
	value, e := m.GetSomeValue(sort)
	astTestMust(t, e)
	astTestMust(t, source.RegisterUniverse(sort, []ast.ASTHandle{value}))
	warm, e := source.Copy()
	astTestMust(t, e)
	astTestMust(t, warm.Close())
	baseline := m.Budget().Used()
	refs, e := m.GetRefCount(d)
	astTestMust(t, e)
	for fault := int64(0); fault < 256; fault++ {
		m.Budget().FailAfter(fault)
		copied, e := source.Copy()
		m.Budget().FailAfter(-1)
		if copied != nil {
			astTestMust(t, copied.Close())
		}
		after, er := m.GetRefCount(d)
		astTestMust(t, er)
		if m.Budget().Used() != baseline || after != refs {
			t.Fatal("copy failure leak", fault, m.Budget().Used(), baseline, after, refs)
		}
		if e == nil {
			return
		}
	}
	t.Fatal("copy never completed")
}

func TestModelFunctionCopyReplacementAndSelfCopy(t *testing.T) {
	m := astTestManager(t)
	source, e := model.NewModel(m)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, source.Close()) })
	target, e := model.NewModel(m)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, target.Close()) })
	d := modelTestFunctionDecl(t, m, "copy-replace-function")
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	sf := modelTestFunction(t, m, 1)
	astTestMust(t, sf.SetElse(yes))
	astTestMust(t, source.Core().RegisterFunction(d, sf))
	old := modelTestFunction(t, m, 1)
	astTestMust(t, old.SetElse(no))
	astTestMust(t, target.Core().RegisterFunction(d, old))
	astTestMust(t, target.CopyFunctionInterpretations(source))
	if _, e = old.Else(); e == nil {
		t.Fatal("replaced target graph survived")
	}
	copied, e := target.Core().FunctionInterpretation(d)
	astTestMust(t, e)
	if copied == sf {
		t.Fatal("source graph pointer transferred")
	}
	fallback, e := copied.Else()
	astTestMust(t, e)
	if fallback != yes {
		t.Fatal("wrong copy fallback")
	}
	astTestMust(t, source.CopyFunctionInterpretations(source))
	if _, e = sf.Else(); e == nil {
		t.Fatal("self-copy did not retire old graph")
	}
	replacement, e := source.Core().FunctionInterpretation(d)
	astTestMust(t, e)
	fallback, e = replacement.Else()
	astTestMust(t, e)
	if fallback != yes || source.Core().NumFunctionDeclarations() != 1 {
		t.Fatal("self-copy publication changed declarations")
	}
}
