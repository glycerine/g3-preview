package ast

import (
	"bytes"
	"errors"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	utext "github.com/glycerine/g3/pkg/util/text"
)

// Source: src/ast/ast_smt2_pp.cpp:705-713. The alias record precedes the
// let-construction records and contains the actual shared child, not its root.
func TestASTSMT2AliasSourceTraceBeforeLetRegistration(t *testing.T) {
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
		t.Fatalf("disabled smt2_pp trace %q", output.String())
	}
	id, err := m.GetASTID(large)
	paramsMust(t, err)
	low, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: budget})
	paramsMust(t, err)
	paramsMust(t, m.DisplayLowLevel(low, large, true, true))
	childText, err := low.String()
	paramsMust(t, err)
	paramsMust(t, low.Close())
	tags.Enable("smt2_pp")
	tags.Enable("pp_let")
	_, err = m.SMT2PPExprTree(shared, nil)
	paramsMust(t, err)
	if !tags.ShouldTrace("smt2_pp") {
		if output.Len() != 0 {
			t.Fatalf("erased alias trace %q", output.String())
		}
		return
	}
	want := "-------- [smt2_pp] store_result src/ast/ast_smt2_pp.cpp:708 ---------\n" +
		"a: a!1 depth: 2, weight: 10, lvl: 0 t: #" + strconv.FormatUint(uint64(id), 10) + "\n" +
		childText + ", is-shared: 1\n" +
		"------------------------------------------------\n" +
		"-------- [pp_let] pp_let src/ast/ast_smt2_pp.cpp:806 ---------\n" +
		"old_sz: 0, sz: 1\n" +
		"------------------------------------------------\n" +
		"-------- [pp_let] pp_let src/ast/ast_smt2_pp.cpp:818 ---------\n" +
		"decls.size(): 1\n" +
		"------------------------------------------------\n" +
		"-------- [pp_let] pp_let src/ast/ast_smt2_pp.cpp:833 ---------\n" +
		"num_op: 1\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("alias and let source order:\ngot %q\nwant %q", got, want)
	}
}

// The source inserts the alias scalars before evaluating mk_ll_pp. A failed
// low-level formatter therefore leaves that prefix and skips the TRACE footer.
func TestASTSMT2AliasSourceTraceFormatterRefusalKeepsPrefix(t *testing.T) {
	c, _, _, budget := dynAckSourceFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget, TraceTags: tags})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.Close()) }()
	bs, err := m.MkBoolSort()
	paramsMust(t, err)
	child, err := m.MkConst(astTestSymbol(t, m, "prefix-child"), bs)
	paramsMust(t, err)
	id, err := m.GetASTID(child)
	paramsMust(t, err)
	tags.Enable("smt2_pp")
	budget.FailAfter(0)
	err = m.traceSMT2Alias(child, "a!7", 3, 12, 2, true)
	budget.FailAfter(-1)
	if !tags.ShouldTrace("smt2_pp") {
		if err != nil || output.Len() != 0 {
			t.Fatalf("erased trace: %v %q", err, output.String())
		}
		return
	}
	if !errors.Is(err, arena.ErrInjectedFailure) {
		t.Fatalf("low-level formatter refusal: %v", err)
	}
	want := "-------- [smt2_pp] store_result src/ast/ast_smt2_pp.cpp:708 ---------\n" +
		"a: a!7 depth: 3, weight: 12, lvl: 2 t: #" + strconv.FormatUint(uint64(id), 10) + "\n"
	if got := output.String(); got != want {
		t.Fatalf("source prefix after formatter refusal: got %q, want %q", got, want)
	}
}
