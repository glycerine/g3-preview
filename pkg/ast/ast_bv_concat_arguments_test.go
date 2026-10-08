// Source: src/ast/bv_decl_plugin.h:404; src/ast/ast.cpp:1568-1638.
package ast

import "testing"

func TestASTBitVectorConcatBorrowedSuffix(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	var args []ASTHandle
	for i := uint32(0); i < 40; i++ {
		s, e := m.BitVectorSort(i + 1)
		astTestMust(t, e)
		h, e := m.MkVar(i, s)
		astTestMust(t, e)
		astTestPin(t, m, h)
		args = append(args, h)
	}
	original, e := m.MkBitVectorConcat(args)
	astTestMust(t, e)
	ref, e := m.NewRef(original)
	astTestMust(t, e)
	view, e := m.AppArgumentView(original)
	astTestMust(t, e)
	suffix, e := view.Range(1, 39)
	astTestMust(t, e)
	want, e := m.MkBitVectorConcat(args[1:])
	astTestMust(t, e)
	astTestPin(t, m, want)
	before := m.Budget().Used()
	got, e := m.MkBitVectorConcatArguments(suffix)
	astTestMust(t, e)
	if got != want || m.Budget().Used() != before {
		t.Fatal("borrowed suffix differs from original factory")
	}
	width, e := m.BitVectorSize(got)
	astTestMust(t, e)
	if width != 819 {
		t.Fatal("concat domain width", width)
	}
	astTestMust(t, m.DelRef(&ref))
	// A borrowed source range cannot survive deletion of its application.
	if _, e = m.MkBitVectorConcatArguments(suffix); e != ErrASTHandle {
		t.Fatal("expired source range", e)
	}
}
