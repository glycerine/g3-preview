package ast

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func TestASTModusPonensInvalidSecondFactSourceTrace(t *testing.T) {
	c, _, _ := paramsTestOwner(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: arena.NewBudget(^uint64(0)), TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	a := proofTestBool(t, m, "modus-ponens-invalid")
	proof, err := m.MkProofHypothesis(a)
	astTestMust(t, err)
	astTestPin(t, m, proof)
	_, err = m.MkProofModusPonens(proof, proof)
	if ASTSourceDebug && !errors.Is(err, ErrASTDomain) {
		t.Fatalf("source debug guard: %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("disabled tag wrote %q", output.String())
	}
	tags.Enable("mk_modus_ponens")
	_, err = m.MkProofModusPonens(proof, proof)
	if ASTSourceDebug && !errors.Is(err, ErrASTDomain) {
		t.Fatalf("trace must precede source debug guard: %v", err)
	}
	if !tags.ShouldTrace("mk_modus_ponens") {
		if output.Len() != 0 {
			t.Fatalf("erased trace wrote %q", output.String())
		}
		return
	}
	const prefix = "-------- [mk_modus_ponens] mk_modus_ponens src/ast/ast.cpp:2641 ---------\n"
	const suffix = "------------------------------------------------\n"
	got := output.String()
	if !strings.HasPrefix(got, prefix) || !strings.HasSuffix(got, suffix) {
		t.Fatalf("source record boundaries: %q", got)
	}
	body := strings.TrimSuffix(strings.TrimPrefix(got, prefix), suffix)
	wantBody := "[hypothesis]: modus-ponens-invalid\n\n[hypothesis]: modus-ponens-invalid\n\n"
	if body != wantBody {
		t.Fatalf("source printed each identical proof once: got %q, want %q", body, wantBody)
	}
}

func TestASTModusPonensMismatchAndNongroundSourceTraces(t *testing.T) {
	c, g, _, budget := dynAckSourceFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mk_modus_ponens")
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	previous, err := uparams.BindProcessGlobalParamsService(g)
	astTestMust(t, err)
	t.Cleanup(func() { _, e := uparams.BindProcessGlobalParamsService(previous); astTestMust(t, e) })
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	a := proofTestBool(t, m, "mp-a")
	b := proofTestBool(t, m, "mp-b")
	left, err := m.MkProofHypothesis(a)
	astTestMust(t, err)
	astTestPin(t, m, left)
	iff := proofTestApp(t, m, ASTOpIff, b, a)
	right, err := m.MkProofHypothesis(iff)
	astTestMust(t, err)
	astTestPin(t, m, right)
	_, err = m.MkProofModusPonens(left, right)
	if ASTSourceDebug && !errors.Is(err, ErrASTDomain) {
		t.Fatalf("source mismatched antecedent guard: %v", err)
	} else if !ASTSourceDebug {
		astTestMust(t, err)
	}
	if tags.ShouldTrace("mk_modus_ponens") {
		got := output.String()
		if !strings.HasPrefix(got, "-------- [mk_modus_ponens] mk_modus_ponens src/ast/ast.cpp:2645 ---------\n") ||
			!strings.Contains(got, "mp-a\n") || !strings.Contains(got, "mp-b") ||
			!strings.HasSuffix(got, "------------------------------------------------\n") {
			t.Fatalf("source antecedent mismatch record: %q", got)
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased trace wrote %q", output.String())
	}
	output.Reset()
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	variable, err := m.MkVar(0, boolSort)
	astTestMust(t, err)
	astTestPin(t, m, variable)
	variableFact, err := m.MkProofHypothesis(variable)
	astTestMust(t, err)
	astTestPin(t, m, variableFact)
	variableIff := proofTestApp(t, m, ASTOpIff, variable, a)
	variableRule, err := m.MkProofHypothesis(variableIff)
	astTestMust(t, err)
	astTestPin(t, m, variableRule)
	_, err = m.MkProofModusPonens(variableFact, variableRule)
	astTestMust(t, err)
	if tags.ShouldTrace("mk_modus_ponens") {
		got := output.String()
		if strings.Count(got, "Non-ground: ") != 2 ||
			!strings.Contains(got, "src/ast/ast.cpp:2648 ---------") ||
			!strings.Contains(got, "src/ast/ast.cpp:2649 ---------") {
			t.Fatalf("source nonground proof records: %q", got)
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased trace wrote %q", output.String())
	}
}
