package model_test

// Source: src/model/model_evaluator.cpp:183-196;
// src/ast/rewriter/array_rewriter.cpp:340-388.

import (
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestModelEvaluatorOriginalSetOperationsAndMembership(t *testing.T) {
	m, _, ev, _, _, _ := evaluatorFixture(t)
	fid, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	bs, e := m.MkBoolSort()
	astTestMust(t, e)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	s, e := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{bs}), bs)
	astTestMust(t, e)
	astTestPin(t, m, s)
	empty, e := m.MkEmptySet(s)
	astTestMust(t, e)
	astTestPin(t, m, empty)
	full, e := m.MkFullSet(s)
	astTestMust(t, e)
	astTestPin(t, m, full)
	store := func(index ast.ASTHandle) ast.ASTHandle {
		h, e := m.MkFamilyApp(fid, ast.ASTArrayOpStore, nil, []ast.ASTHandle{empty, index, yes})
		astTestMust(t, e)
		astTestPin(t, m, h)
		return h
	}
	a := store(no)
	b, e := m.MkFamilyApp(fid, ast.ASTArrayOpStore, nil, []ast.ASTHandle{full, no, no})
	astTestMust(t, e)
	astTestPin(t, m, b)
	op := func(kind int32, args ...ast.ASTHandle) ast.ASTHandle {
		h, e := m.MkFamilyApp(fid, kind, nil, args)
		astTestMust(t, e)
		astTestPin(t, m, h)
		return h
	}
	out := astTestPin(t, m, ast.ASTHandle{})
	for _, row := range []struct {
		term    ast.ASTHandle
		members [2]ast.ASTHandle
	}{
		{op(ast.ASTArrayOpSetUnion, a), [2]ast.ASTHandle{yes, no}},
		{op(ast.ASTArrayOpSetUnion, a, b), [2]ast.ASTHandle{yes, yes}},
		{op(ast.ASTArrayOpSetUnion, empty, full, empty), [2]ast.ASTHandle{yes, yes}},
		{op(ast.ASTArrayOpSetIntersect, a, b), [2]ast.ASTHandle{no, no}},
		{op(ast.ASTArrayOpSetIntersect, full, empty, full), [2]ast.ASTHandle{no, no}},
		{op(ast.ASTArrayOpSetComplement, empty), [2]ast.ASTHandle{yes, yes}},
		{op(ast.ASTArrayOpSetComplement, a), [2]ast.ASTHandle{no, yes}},
		{op(ast.ASTArrayOpSetDifference, a, b), [2]ast.ASTHandle{yes, no}},
	} {
		for i, index := range []ast.ASTHandle{no, yes} {
			selected := op(ast.ASTArrayOpSelect, row.term, index)
			astTestMust(t, ev.Reset(nil))
			astTestMust(t, ev.Evaluate(selected, out))
			got, e := m.RefGet(out)
			astTestMust(t, e)
			if got != row.members[i] {
				t.Fatal("set membership", row.term, i, got, row.members[i])
			}
		}
	}
	for _, row := range []struct{ a, b, want ast.ASTHandle }{{empty, full, yes}, {a, a, yes}, {b, b, yes}, {a, b, no}, {b, a, no}} {
		subset := op(ast.ASTArrayOpSetSubset, row.a, row.b)
		astTestMust(t, ev.Reset(nil))
		astTestMust(t, ev.Evaluate(subset, out))
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if got != row.want {
			t.Fatal("set subset", got, row.want)
		}
	}
	// Source map_core declines differently indexed stores, and select_core has
	// no select-over-map rule. Preserve the actual residual rather than claiming
	// the original evaluator fully decides every concrete set expression.
	mismatch := store(yes)
	union := op(ast.ASTArrayOpSetUnion, a, mismatch)
	or, e := m.MkOrDecl()
	astTestMust(t, e)
	mapped, e := m.MkArrayMap(or, ast.SliceArguments([]ast.ASTHandle{a, mismatch}))
	astTestMust(t, e)
	astTestPin(t, m, mapped)
	want := op(ast.ASTArrayOpSelect, mapped, no)
	astTestMust(t, ev.Reset(nil))
	astTestMust(t, ev.Evaluate(op(ast.ASTArrayOpSelect, union, no), out))
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if got != want {
		t.Fatal("source unresolved select-over-map", got, want)
	}
	and, e := m.MkAndDecl()
	astTestMust(t, e)
	difference, e := m.MkArrayMap(and, ast.SliceArguments([]ast.ASTHandle{full, b}))
	astTestMust(t, e)
	astTestPin(t, m, difference)
	want = op(ast.ASTArrayOpSelect, difference, no)
	astTestMust(t, ev.Reset(nil))
	astTestMust(t, ev.Evaluate(op(ast.ASTArrayOpSelect, op(ast.ASTArrayOpSetDifference, full, a), no), out))
	got, e = m.RefGet(out)
	astTestMust(t, e)
	if got != want {
		t.Fatal("source mixed-shape difference residual", got, want)
	}
	want, e = m.MkEq(difference, empty)
	astTestMust(t, e)
	astTestPin(t, m, want)
	astTestMust(t, ev.Reset(nil))
	astTestMust(t, ev.Evaluate(op(ast.ASTArrayOpSetSubset, full, a), out))
	got, e = m.RefGet(out)
	astTestMust(t, e)
	if got != want {
		t.Fatal("source mixed-shape subset residual", got, want)
	}
}
