package ast

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
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
