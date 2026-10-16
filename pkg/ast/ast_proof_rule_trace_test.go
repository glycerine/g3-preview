package ast

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func TestASTLemmaAndAndElimSourceTracesBeforeGuards(t *testing.T) {
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
	a := proofTestBool(t, m, "lemma-a")
	b := proofTestBool(t, m, "lemma-b")
	hypothesis, err := m.MkProofHypothesis(a)
	astTestMust(t, err)
	astTestPin(t, m, hypothesis)
	_, err = m.MkProofLemma(hypothesis, b)
	if ASTSourceDebug && !errors.Is(err, ErrASTDomain) {
		t.Fatalf("lemma debug guard: %v", err)
	} else if !ASTSourceDebug {
		astTestMust(t, err)
	}
	if output.Len() != 0 {
		t.Fatalf("disabled lemma tag wrote %q", output.String())
	}
	tags.Enable("mk_lemma")
	_, err = m.MkProofLemma(hypothesis, b)
	if ASTSourceDebug && !errors.Is(err, ErrASTDomain) {
		t.Fatalf("lemma trace must precede guard: %v", err)
	} else if !ASTSourceDebug {
		astTestMust(t, err)
	}
	if tags.ShouldTrace("mk_lemma") {
		want := "-------- [mk_lemma] mk_lemma src/ast/ast.cpp:3026 ---------\n" +
			"[hypothesis]: lemma-a\n\n" +
			"------------------------------------------------\n"
		if got := output.String(); got != want {
			t.Fatalf("lemma source trace: got %q, want %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased lemma trace wrote %q", output.String())
	}
	output.Reset()
	falseProof, err := m.MkProofHypothesis(m.external(m.falseApp))
	astTestMust(t, err)
	astTestPin(t, m, falseProof)
	_, err = m.MkProofLemma(falseProof, b)
	astTestMust(t, err)
	if output.Len() != 0 {
		t.Fatalf("valid lemma wrote conditional trace %q", output.String())
	}
	and := proofTestApp(t, m, ASTOpAnd, a, b)
	andProof, err := m.MkProofHypothesis(and)
	astTestMust(t, err)
	astTestPin(t, m, andProof)
	_, err = m.MkProofAndElim(andProof, 2)
	if !errors.Is(err, arena.ErrBounds) {
		t.Fatalf("and-elim bounds guard: %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("disabled and-elim tag wrote %q", output.String())
	}
	tags.Enable("mk_and_elim")
	_, err = m.MkProofAndElim(andProof, 1)
	astTestMust(t, err)
	if output.Len() != 0 {
		t.Fatalf("valid and-elim wrote conditional trace %q", output.String())
	}
	_, err = m.MkProofAndElim(andProof, 2)
	if !errors.Is(err, arena.ErrBounds) {
		t.Fatalf("and-elim trace must precede bounds guard: %v", err)
	}
	if tags.ShouldTrace("mk_and_elim") {
		want := "-------- [mk_and_elim] mk_and_elim src/ast/ast.cpp:3102 ---------\n" +
			"i: 2\n(and lemma-a lemma-b)\n" +
			"------------------------------------------------\n"
		if got := output.String(); got != want {
			t.Fatalf("and-elim source trace: got %q, want %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased and-elim trace wrote %q", output.String())
	}
}
