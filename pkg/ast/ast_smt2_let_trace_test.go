package ast

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// Source: src/ast/ast_smt2_pp.cpp:801-844. The empty alias path emits only
// its initial size record; a shared large child reaches all three sites.
func TestASTSMT2LetSourceTraceEmptyAndAliased(t *testing.T) {
	c, _, _, budget := dynAckSourceFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget, TraceTags: tags})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.Close()) }()
	bs, err := m.MkBoolSort()
	paramsMust(t, err)
	args := make([]ASTHandle, 9)
	for i := range args {
		args[i], err = m.MkConst(astTestSymbol(t, m, string(rune('a'+i))), bs)
		paramsMust(t, err)
	}
	large, err := m.MkAndArgs(args)
	paramsMust(t, err)
	shared, err := m.MkAnd(large, large)
	paramsMust(t, err)
	_, err = m.SMT2PPExprTree(shared, nil)
	paramsMust(t, err)
	if output.Len() != 0 {
		t.Fatal("disabled pp_let trace", output.String())
	}
	tags.Enable("pp_let")
	_, err = m.SMT2PPExprTree(args[0], nil)
	paramsMust(t, err)
	_, err = m.SMT2PPExprTree(shared, nil)
	paramsMust(t, err)
	if !tags.ShouldTrace("pp_let") {
		if output.Len() != 0 {
			t.Fatal("erased pp_let trace", output.String())
		}
		return
	}
	record := func(line string, body string) string {
		return "-------- [pp_let] pp_let src/ast/ast_smt2_pp.cpp:" + line + " ---------\n" +
			body + "------------------------------------------------\n"
	}
	want := record("806", "old_sz: 0, sz: 0\n") +
		record("806", "old_sz: 0, sz: 1\n") +
		record("818", "decls.size(): 1\n") +
		record("833", "num_op: 1\n")
	if output.String() != want {
		t.Fatalf("source pp_let records: got %q, want %q", output.String(), want)
	}
}
