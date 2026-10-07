// Source: src/ast/bv_decl_plugin.cpp:571-630; src/ast/bv_decl_plugin.h:406.
package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestASTBitVectorBorrowedORArguments(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	sort, e := m.BitVectorSort(8)
	astTestMust(t, e)
	var args []ASTHandle
	for i := uint32(0); i < 40; i++ {
		h, e := m.MkVar(i, sort)
		astTestMust(t, e)
		astTestPin(t, m, h)
		args = append(args, h)
	}
	input, e := m.MkFamilyApp(fid, 4, nil, args)
	astTestMust(t, e)
	astTestPin(t, m, input)
	view, e := m.AppArgumentView(input)
	astTestMust(t, e)
	expected, e := m.MkBitVectorOr(args)
	astTestMust(t, e)
	astTestPin(t, m, expected)
	before := m.Stats()
	// The source app factory allocates its candidate even for a hash-cons hit.
	m.Budget().FailAfter(0)
	_, e = m.MkBitVectorOrArguments(view)
	m.Budget().FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || m.Stats() != before {
		t.Fatal("candidate allocation refusal", e, m.Stats(), before)
	}
	got, e := m.MkBitVectorOrArguments(view)
	astTestMust(t, e)
	if got != expected || m.Stats() != before {
		t.Fatal("borrowed OR changed identity/stats")
	}
	v, e := m.NewRefVectorFrom(args)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRefVector(&v)) }()
	part, e := m.RefVectorArgumentsRange(v, 3, 20)
	astTestMust(t, e)
	got, e = m.MkBitVectorOrArguments(part)
	astTestMust(t, e)
	astTestPin(t, m, got)
	count, e := m.GetNumArgs(got)
	astTestMust(t, e)
	if count != 20 {
		t.Fatal("borrowed range arity", count)
	}
	for i := uint32(0); i < count; i++ {
		h, e := m.GetArg(got, i)
		astTestMust(t, e)
		if h != args[i+3] {
			t.Fatal("borrowed range order", i)
		}
	}
}

func TestASTBitVectorBorrowedORGuardsAndLifetime(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s8, e := m.BitVectorSort(8)
	astTestMust(t, e)
	s16, e := m.BitVectorSort(16)
	astTestMust(t, e)
	x, e := m.MkVar(0, s8)
	astTestMust(t, e)
	astTestPin(t, m, x)
	y, e := m.MkVar(1, s16)
	astTestMust(t, e)
	astTestPin(t, m, y)
	if _, e = m.MkBitVectorOrArguments(SliceArguments(nil)); e == nil {
		t.Fatal("empty operands accepted")
	}
	if _, e = m.MkBitVectorOrArguments(SliceArguments([]ASTHandle{x, x, y})); e == nil {
		t.Fatal("mismatched variadic domain accepted")
	}
	input, e := m.MkFamilyApp(fid, 4, nil, []ASTHandle{x, x})
	astTestMust(t, e)
	r, e := m.NewRef(input)
	astTestMust(t, e)
	view, e := m.AppArgumentView(input)
	astTestMust(t, e)
	got, e := m.MkBitVectorOrArguments(view)
	astTestMust(t, e)
	astTestPin(t, m, got)
	astTestMust(t, m.DelRef(&r))
	if _, e = view.At(0); e == nil {
		t.Fatal("argument view retained the dead input")
	}
	if _, e = m.MkBitVectorOrArguments(view); e == nil {
		t.Fatal("stale argument view accepted")
	}
	for i := uint32(0); i < 2; i++ {
		h, e := m.GetArg(got, i)
		astTestMust(t, e)
		if h != x {
			t.Fatal("output lost its operands", i)
		}
	}
}
