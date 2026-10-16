package ast

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func TestASTProofAssertedNonBooleanSourceTraceBeforeDebugGuard(t *testing.T) {
	c, g, _, budget := dynAckSourceFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	previous, err := uparams.BindProcessGlobalParamsService(g)
	astTestMust(t, err)
	t.Cleanup(func() { _, e := uparams.BindProcessGlobalParamsService(previous); astTestMust(t, e) })
	sort, err := m.MkSort(astTestSymbol(t, m, "ProofTraceSort"), ASTSortInfo{})
	astTestMust(t, err)
	value, err := m.MkConst(astTestSymbol(t, m, "proof-trace-value"), sort)
	astTestMust(t, err)
	astTestPin(t, m, value)
	_, err = m.MkProofAsserted(value)
	if ASTSourceDebug && !errors.Is(err, ErrASTDomain) {
		t.Fatalf("expected source debug assertion, got %v", err)
	} else if !ASTSourceDebug {
		astTestMust(t, err)
	}
	if output.Len() != 0 {
		t.Fatalf("disabled tag wrote %q", output.String())
	}
	tags.Enable("mk_asserted_bug")
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	boolean, err := m.MkConst(astTestSymbol(t, m, "proof-trace-boolean"), boolSort)
	astTestMust(t, err)
	_, err = m.MkProofAsserted(boolean)
	astTestMust(t, err)
	if output.Len() != 0 {
		t.Fatalf("Boolean assertion reached conditional TRACE: %q", output.String())
	}
	_, err = m.MkProofAsserted(value)
	if ASTSourceDebug && !errors.Is(err, ErrASTDomain) {
		t.Fatalf("TRACE must precede source debug assertion: %v", err)
	} else if !ASTSourceDebug {
		astTestMust(t, err)
	}
	if !tags.ShouldTrace("mk_asserted_bug") {
		if output.Len() != 0 {
			t.Fatalf("erased trace wrote %q", output.String())
		}
		return
	}
	want := "-------- [mk_asserted_bug] mk_asserted src/ast/ast.cpp:2627 ---------\n" +
		"proof-trace-value\nsort: ProofTraceSort\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source assertion trace: got %q, want %q", got, want)
	}
}
