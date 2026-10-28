package ast

// Source: src/ast/rewriter/bv_rewriter.cpp:305-306;
// src/ast/bv_decl_plugin.cpp:571-630.

import "testing"

func TestASTBitVectorAddBorrowedCommonSuffix(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	var args []ASTHandle
	for i := uint32(0); i < 40; i++ {
		h, e := m.MkVar(i, s)
		astTestMust(t, e)
		astTestPin(t, m, h)
		args = append(args, h)
	}
	original, e := m.MkFamilyApp(fid, 4, nil, args)
	astTestMust(t, e)
	ref, e := m.NewRef(original)
	astTestMust(t, e)
	view, e := m.AppArgumentView(original)
	astTestMust(t, e)
	suffix, e := view.Range(1, 39)
	astTestMust(t, e)
	want, e := m.MkFamilyApp(fid, 4, nil, args[1:])
	astTestMust(t, e)
	astTestPin(t, m, want)
	before := m.Budget().Used()
	got, e := m.MkBitVectorAddArguments(suffix)
	astTestMust(t, e)
	if got != want || m.Budget().Used() != before {
		t.Fatal("borrowed common ADD suffix changed factory identity")
	}
	astTestMust(t, m.DelRef(&ref))
	if _, e = m.MkBitVectorAddArguments(suffix); e != ErrASTHandle {
		t.Fatal("expired common suffix", e)
	}
}
