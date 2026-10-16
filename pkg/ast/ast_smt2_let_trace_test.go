package ast

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
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

// The parent publishes an alias before visiting a nested quantifier. Source
// begin_scope keeps that outer alias in m_aliased_exprs, so pp_let reports an
// offset count even though it wraps only the quantifier's local declarations.
func TestASTSMT2LetSourceTraceNestedScopeOffset(t *testing.T) {
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
	shared, err := m.MkAndArgs(args)
	paramsMust(t, err)
	inner, err := m.MkAnd(shared, shared)
	paramsMust(t, err)
	q, err := m.MkQuantifier(ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{bs}, Names: []usymbol.Symbol{astTestSymbol(t, m, "x")}, Body: inner})
	paramsMust(t, err)
	root, err := m.MkAndArgs([]ASTHandle{shared, shared, q})
	paramsMust(t, err)
	tags.Enable("pp_let")
	_, err = m.SMT2PPExprTree(root, nil)
	paramsMust(t, err)
	if !tags.ShouldTrace("pp_let") {
		if output.Len() != 0 {
			t.Fatal("erased nested trace", output.String())
		}
		return
	}
	got := output.String()
	if !strings.Contains(got, "old_sz: 1, sz: 2\n") || !strings.Contains(got, "old_sz: 0, sz: 1\n") {
		t.Fatalf("source nested alias scope counts: %q", got)
	}
}

// A pattern frame is entered with m_use_alias=false in the source. Repeated
// large descendants therefore do not add a let or a pattern-local pp_let
// checkpoint before the quantifier body is visited.
func TestASTSMT2LetPatternDisablesAliasTraversal(t *testing.T) {
	c, globals, local, budget := dynAckSourceFixture(t)
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
	shared, err := m.MkAndArgs(args)
	paramsMust(t, err)
	repeated, err := m.MkAnd(shared, shared)
	paramsMust(t, err)
	pattern, err := m.MkFamilyApp(2, 0, nil, []ASTHandle{repeated})
	paramsMust(t, err)
	body, err := m.MkConst(astTestSymbol(t, m, "body"), bs)
	paramsMust(t, err)
	q, err := m.MkQuantifier(ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{bs}, Names: []usymbol.Symbol{astTestSymbol(t, m, "x")}, Body: body, Patterns: []ASTHandle{pattern}})
	paramsMust(t, err)
	tags.Enable("pp_let")
	format, err := m.SMT2PPExprTree(q, nil)
	paramsMust(t, err)
	printed, err := m.wellSortedTraceFormat(format, c, local, globals)
	paramsMust(t, err)
	if !strings.Contains(printed, ":pattern") || strings.Contains(printed, "(let ") {
		t.Fatalf("pattern descendants gained a source-disabled alias: %q", printed)
	}
	if !tags.ShouldTrace("pp_let") {
		if output.Len() != 0 {
			t.Fatal("erased pattern trace", output.String())
		}
		return
	}
	if got := output.String(); strings.Count(got, "[pp_let]") != 2 || strings.Count(got, "old_sz: 0, sz: 0\n") != 2 {
		t.Fatalf("pattern created an alias or pp_let checkpoint: %q", got)
	}
}
