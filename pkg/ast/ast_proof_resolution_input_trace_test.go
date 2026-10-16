package ast

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func TestASTUnitResolutionInputSourceTraces(t *testing.T) {
	c, globals, _, budget := dynAckSourceFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	previous, err := uparams.BindProcessGlobalParamsService(globals)
	astTestMust(t, err)
	t.Cleanup(func() { _, e := uparams.BindProcessGlobalParamsService(previous); astTestMust(t, e) })
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	a := proofTestBool(t, m, "resolution-a")
	b := proofTestBool(t, m, "resolution-b")
	hypothesis := func(f ASTHandle) ASTHandle {
		p, e := m.MkProofHypothesis(f)
		astTestMust(t, e)
		astTestPin(t, m, p)
		return p
	}
	pa, pb := hypothesis(a), hypothesis(b)
	_, err = m.MkProofUnitResolution([]ASTHandle{pa, pb})
	if ASTSourceDebug && !errors.Is(err, ErrASTDomain) {
		t.Fatalf("invalid-clause debug guard: %v", err)
	} else if !ASTSourceDebug {
		astTestMust(t, err)
	}
	if output.Len() != 0 {
		t.Fatalf("disabled invalid-clause tag wrote %q", output.String())
	}
	tags.Enable("mk_unit_resolution_bug")
	_, err = m.MkProofUnitResolution([]ASTHandle{pa, pb})
	if ASTSourceDebug && !errors.Is(err, ErrASTDomain) {
		t.Fatalf("invalid-clause trace must precede guard: %v", err)
	} else if !ASTSourceDebug {
		astTestMust(t, err)
	}
	if tags.ShouldTrace("mk_unit_resolution_bug") {
		want := "-------- [mk_unit_resolution_bug] mk_unit_resolution src/ast/ast.cpp:2918 ---------\n" +
			"resolution-a resolution-b\n" +
			"------------------------------------------------\n"
		if got := output.String(); got != want {
			t.Fatalf("invalid-clause trace: got %q, want %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased invalid-clause trace wrote %q", output.String())
	}
	output.Reset()
	negative := proofTestApp(t, m, ASTOpNot, a)
	pneg := hypothesis(negative)
	tags.Enable("unit_bug")
	_, err = m.MkProofUnitResolutionFact([]ASTHandle{pa, pneg}, m.external(m.falseApp))
	astTestMust(t, err)
	if tags.ShouldTrace("unit_bug") {
		want := "-------- [unit_bug] mk_unit_resolution src/ast/ast.cpp:2969 ---------\n" +
			"resolution-a\n(not resolution-a)\n===>\nfalse\n" +
			"------------------------------------------------\n"
		if got := output.String(); got != want {
			t.Fatalf("explicit-fact trace: got %q, want %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased explicit-fact trace wrote %q", output.String())
	}
}

func TestASTUnitResolutionDebugDiagnostics(t *testing.T) {
	c, globals, _, budget := dynAckSourceFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	previous, err := uparams.BindProcessGlobalParamsService(globals)
	astTestMust(t, err)
	t.Cleanup(func() { _, e := uparams.BindProcessGlobalParamsService(previous); astTestMust(t, e) })
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	a, b, d := proofTestBool(t, m, "unit-a"), proofTestBool(t, m, "unit-b"), proofTestBool(t, m, "unit-d")
	hypothesis := func(f ASTHandle) ASTHandle {
		p, e := m.MkProofHypothesis(f)
		astTestMust(t, e)
		astTestPin(t, m, p)
		return p
	}
	clause := hypothesis(proofTestApp(t, m, ASTOpOr, a, b))
	notA := hypothesis(proofTestApp(t, m, ASTOpNot, a))
	notD := hypothesis(proofTestApp(t, m, ASTOpNot, d))
	tags.Enable("mk_unit_resolution_bug")
	_, err = m.MkProofUnitResolution([]ASTHandle{clause, notD})
	if ASTSourceDebug && !errors.Is(err, ErrASTDomain) {
		t.Fatalf("missing complement guard: %v", err)
	} else if !ASTSourceDebug {
		astTestMust(t, err)
	}
	if ASTSourceDebug && tags.ShouldTrace("mk_unit_resolution_bug") {
		want := "-------- [mk_unit_resolution_bug] mk_unit_resolution src/ast/ast.cpp:2942 ---------\n" +
			"(or unit-a unit-b)\nIndex 1 was not found:\n(not unit-d)\n" +
			"------------------------------------------------\n"
		if got := output.String(); got != want {
			t.Fatalf("missing complement trace: got %q, want %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased debug-only trace wrote %q", output.String())
	}
	output.Reset()
	tags.Enable("cunit_bug")
	three := hypothesis(proofTestApp(t, m, ASTOpOr, a, b, d))
	_, err = m.MkProofUnitResolutionFact([]ASTHandle{three, notA}, b)
	if ASTSourceDebug && !errors.Is(err, ErrASTDomain) {
		t.Fatalf("clause-count guard: %v", err)
	} else if !ASTSourceDebug {
		astTestMust(t, err)
	}
	if ASTSourceDebug && tags.ShouldTrace("cunit_bug") {
		want := "-------- [cunit_bug] mk_unit_resolution src/ast/ast.cpp:2987 ---------\n" +
			"(or unit-a unit-b unit-d)\n(not unit-a)\n===>\nunit-b\n" +
			"------------------------------------------------\n"
		if got := output.String(); got != want {
			t.Fatalf("clause-count trace: got %q, want %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased clause-count trace wrote %q", output.String())
	}
	output.Reset()
	tags.Enable("unit_bug1")
	_, err = m.MkProofUnitResolutionFact([]ASTHandle{clause, notA}, d)
	if ASTSourceDebug && !errors.Is(err, ErrASTDomain) {
		t.Fatalf("wrong result literal guard: %v", err)
	} else if !ASTSourceDebug {
		astTestMust(t, err)
	}
	if ASTSourceDebug && tags.ShouldTrace("unit_bug1") {
		want := "-------- [unit_bug1] mk_unit_resolution src/ast/ast.cpp:3006 ---------\n" +
			"unit-d\n\nunit-b\n\n" +
			"------------------------------------------------\n"
		if got := output.String(); got != want {
			t.Fatalf("wrong result literal trace: got %q, want %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased wrong-result trace wrote %q", output.String())
	}
}
