package model_test

// Source: src/model/model_evaluator.cpp:207-221,422-499,565-569.

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"testing"
)

func TestModelEvaluatorOriginalFunctionArrayValueExpansionAndPrecedence(t *testing.T) {
	m, md, ev, _, _, _ := evaluatorFixture(t)
	fid, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	bs := mustBoolSort(t, m)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	f, e := m.MkFuncDecl(astTestSymbol(t, m, "expand-array-function"), []ast.ASTHandle{bs}, bs, ast.ASTFuncDeclInfo{})
	astTestMust(t, e)
	astTestPin(t, m, f)
	fi, e := model.NewFunctionInterpretation(m, 1)
	astTestMust(t, e)
	astTestMust(t, fi.SetElse(no))
	astTestMust(t, fi.InsertEntry([]ast.ASTHandle{yes}, yes))
	astTestMust(t, md.Core().RegisterFunction(f, fi))
	array, e := m.MkAsArray(f)
	astTestMust(t, e)
	astTestPin(t, m, array)
	sort, e := m.GetSort(array)
	astTestMust(t, e)
	parameter, e := m.NewParameterAST(sort)
	astTestMust(t, e)
	constant, e := m.MkFamilyApp(fid, ast.ASTArrayOpConst, []ast.ASTParameter{parameter}, []ast.ASTHandle{no})
	astTestMust(t, e)
	astTestPin(t, m, constant)
	astTestMust(t, m.DelParameter(&parameter))
	want, e := m.MkFamilyApp(fid, ast.ASTArrayOpStore, nil, []ast.ASTHandle{constant, yes, yes})
	astTestMust(t, e)
	astTestPin(t, m, want)
	out := astTestPin(t, m, ast.ASTHandle{})
	astTestMust(t, ev.Evaluate(array, out))
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if got != want {
		t.Fatal("function-backed value not expanded", got, want)
	}
	// The store prefix overrides an entry with the same index. Expansion follows
	// reverse reconstruction, without an additional simplifying traversal.
	override, e := m.MkFamilyApp(fid, ast.ASTArrayOpStore, nil, []ast.ASTHandle{array, yes, no})
	astTestMust(t, e)
	astTestPin(t, m, override)
	expected, e := m.MkFamilyApp(fid, ast.ASTArrayOpStore, nil, []ast.ASTHandle{want, yes, no})
	astTestMust(t, e)
	astTestPin(t, m, expected)
	astTestMust(t, ev.Evaluate(override, out))
	got, e = m.RefGet(out)
	astTestMust(t, e)
	if got != expected {
		t.Fatal("store prefix precedence", got, expected)
	}
	selected, e := m.MkFamilyApp(fid, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{got, yes})
	astTestMust(t, e)
	astTestPin(t, m, selected)
	astTestMust(t, ev.Evaluate(selected, out))
	got, e = m.RefGet(out)
	astTestMust(t, e)
	if got != no {
		t.Fatal("expanded override semantics")
	}
}

func TestModelEvaluatorOriginalArrayExpansionPruningAndGroundness(t *testing.T) {
	m, md, ev, _, _, _ := evaluatorFixture(t)
	fid, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	bs := mustBoolSort(t, m)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	x := goalTestAtom(t, m, "nonground-value-index")
	f, e := m.MkFuncDecl(astTestSymbol(t, m, "prune-array-function"), []ast.ASTHandle{bs}, bs, ast.ASTFuncDeclInfo{})
	astTestMust(t, e)
	astTestPin(t, m, f)
	fi, e := model.NewFunctionInterpretation(m, 1)
	astTestMust(t, e)
	astTestMust(t, fi.SetElse(no))
	astTestMust(t, fi.InsertEntry([]ast.ASTHandle{yes}, no))
	astTestMust(t, md.Core().RegisterFunction(f, fi))
	array, e := m.MkAsArray(f)
	astTestMust(t, e)
	astTestPin(t, m, array)
	sort, e := m.GetSort(array)
	astTestMust(t, e)
	parameter, e := m.NewParameterAST(sort)
	astTestMust(t, e)
	constant, e := m.MkFamilyApp(fid, ast.ASTArrayOpConst, []ast.ASTParameter{parameter}, []ast.ASTHandle{no})
	astTestMust(t, e)
	astTestPin(t, m, constant)
	astTestMust(t, m.DelParameter(&parameter))
	out := astTestPin(t, m, ast.ASTHandle{})
	check := func(want ast.ASTHandle) {
		t.Helper()
		astTestMust(t, ev.Reset(nil))
		astTestMust(t, ev.Evaluate(array, out))
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if got != want {
			t.Fatal("source expansion", got, want)
		}
	}
	check(constant)
	// A non-value store prefix disables trimming function rows, including
	// an otherwise redundant entry equal to the else case.
	prefix, e := m.MkFamilyApp(fid, ast.ASTArrayOpStore, nil, []ast.ASTHandle{array, x, yes})
	astTestMust(t, e)
	astTestPin(t, m, prefix)
	retained, e := m.MkFamilyApp(fid, ast.ASTArrayOpStore, nil, []ast.ASTHandle{constant, yes, no})
	astTestMust(t, e)
	astTestPin(t, m, retained)
	prefixed, e := m.MkFamilyApp(fid, ast.ASTArrayOpStore, nil, []ast.ASTHandle{retained, x, yes})
	astTestMust(t, e)
	astTestPin(t, m, prefixed)
	astTestMust(t, ev.Reset(nil))
	astTestMust(t, ev.Evaluate(prefix, out))
	prefixGot, e := m.RefGet(out)
	astTestMust(t, e)
	if prefixGot != prefixed {
		t.Fatal("non-value prefix changed trimming")
	}
	// A ground, non-value index in the final entry stops the backwards pruning
	// loop, retaining an earlier else-equal row exactly as the source does.
	astTestMust(t, fi.InsertEntry([]ast.ASTHandle{x}, yes))
	inner, e := m.MkFamilyApp(fid, ast.ASTArrayOpStore, nil, []ast.ASTHandle{constant, x, yes})
	astTestMust(t, e)
	astTestPin(t, m, inner)
	expected, e := m.MkFamilyApp(fid, ast.ASTArrayOpStore, nil, []ast.ASTHandle{inner, yes, no})
	astTestMust(t, e)
	astTestPin(t, m, expected)
	check(expected)
	variable, e := m.MkVar(0, bs)
	astTestMust(t, e)
	astTestPin(t, m, variable)
	astTestMust(t, fi.SetElse(variable))
	check(array)
	astTestMust(t, fi.SetElse(ast.ASTHandle{}))
	check(array)
	unknownF, e := m.MkFuncDecl(astTestSymbol(t, m, "missing-array-function"), []ast.ASTHandle{bs}, bs, ast.ASTFuncDeclInfo{})
	astTestMust(t, e)
	astTestPin(t, m, unknownF)
	unknownArray, e := m.MkAsArray(unknownF)
	astTestMust(t, e)
	astTestPin(t, m, unknownArray)
	astTestMust(t, ev.Evaluate(unknownArray, out))
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if got != unknownArray {
		t.Fatal("missing graph extraction changed value")
	}
}

func TestModelEvaluatorOriginalArrayExpansionAllocationRefusals(t *testing.T) {
	m, md, ev, _, _, _ := evaluatorFixture(t)
	_, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	bs := mustBoolSort(t, m)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	f, e := m.MkFuncDecl(astTestSymbol(t, m, "fault-array-function"), []ast.ASTHandle{bs, bs}, bs, ast.ASTFuncDeclInfo{})
	astTestMust(t, e)
	astTestPin(t, m, f)
	fi, e := model.NewFunctionInterpretation(m, 2)
	astTestMust(t, e)
	astTestMust(t, fi.SetElse(no))
	astTestMust(t, fi.InsertEntry([]ast.ASTHandle{yes, yes}, yes))
	astTestMust(t, fi.InsertEntry([]ast.ASTHandle{yes, no}, yes))
	astTestMust(t, fi.InsertEntry([]ast.ASTHandle{no, yes}, no))
	astTestMust(t, md.Core().RegisterFunction(f, fi))
	array, e := m.MkAsArray(f)
	astTestMust(t, e)
	astTestPin(t, m, array)
	out := astTestPin(t, m, array)
	astTestMust(t, ev.Evaluate(array, out))
	expected, e := m.RefGet(out)
	astTestMust(t, e)
	astTestPin(t, m, expected)
	roots := []ast.ASTHandle{f, array, yes, no}
	for fault := int64(0); fault < 128; fault++ {
		astTestMust(t, ev.Reset(nil))
		astTestMust(t, m.SetRef(out, array))
		counts := make([]uint32, len(roots))
		for i, h := range roots {
			counts[i], e = m.GetRefCount(h)
			astTestMust(t, e)
		}
		m.Budget().FailAfter(fault)
		failure := ev.Evaluate(array, out)
		m.Budget().FailAfter(-1)
		if failure != nil && !errors.Is(failure, arena.ErrInjectedFailure) {
			t.Fatal("unexpected refusal", fault, failure)
		}
		if failure == nil {
			got, e := m.RefGet(out)
			astTestMust(t, e)
			if got != expected {
				t.Fatal("successful reconstruction", fault)
			}
			return
		}
		// Expansion publishes a const/store prefix before later source allocations.
		// Restore that output before comparing real retained reference counts.
		astTestMust(t, m.SetRef(out, array))
		astTestMust(t, ev.Reset(nil))
		for i, h := range roots {
			count, e := m.GetRefCount(h)
			astTestMust(t, e)
			if count != counts[i] {
				t.Fatal("unjoined vector reference leaked", fault, i, count, counts[i])
			}
		}
	}
	t.Fatal("allocation campaign did not reach success")
}
