//go:build g3trace

package ast

import (
	"bytes"
	"testing"

	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestASTUnitResolutionSourceTraceBothOverloads(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	var output bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	trueProof, err := m.MkProofHypothesis(m.external(m.trueApp))
	astTestMust(t, err)
	falseProof, err := m.MkProofHypothesis(m.external(m.falseApp))
	astTestMust(t, err)
	proofs := []ASTHandle{trueProof, falseProof}
	tags.Enable("unit_resolution")
	first, err := m.MkProofUnitResolution(proofs)
	astTestMust(t, err)
	second, err := m.MkProofUnitResolutionFact(proofs, m.external(m.falseApp))
	astTestMust(t, err)
	if first != second {
		t.Fatal("source proof interning", first, second)
	}
	printer, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: budget})
	astTestMust(t, err)
	astTestMust(t, m.DisplayLowLevel(printer, first, true, true))
	printed, err := printer.String()
	astTestMust(t, err)
	astTestMust(t, printer.Close())
	want := "-------- [unit_resolution] mk_unit_resolution src/ast/ast.cpp:2964 ---------\n" +
		"unit_resolution generating fact\n" + printed +
		"------------------------------------------------\n" +
		"-------- [unit_resolution] mk_unit_resolution src/ast/ast.cpp:3015 ---------\n" +
		"unit_resolution using fact\n" + printed +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source unit-resolution trace:\n got %q\nwant %q", got, want)
	}
}
