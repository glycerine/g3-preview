package ast

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func TestASTTransitivitySourceTracesBeforeDebugGuards(t *testing.T) {
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
	a := proofTestBool(t, m, "trans-a")
	b := proofTestBool(t, m, "trans-b")
	d := proofTestBool(t, m, "trans-d")
	proof := func(op int32, x, y ASTHandle) ASTHandle {
		fact := proofTestApp(t, m, op, x, y)
		p, e := m.MkProofHypothesis(fact)
		astTestMust(t, e)
		astTestPin(t, m, p)
		return p
	}
	left := proof(ASTOpIff, a, b)
	differentDecl := proof(ASTOpImplies, b, d)
	_, err = m.MkProofTransitivity(left, differentDecl)
	if ASTSourceDebug && !errors.Is(err, ErrASTDomain) {
		t.Fatalf("source declaration guard: %v", err)
	} else if !ASTSourceDebug {
		astTestMust(t, err)
	}
	if output.Len() != 0 {
		t.Fatalf("disabled tag wrote %q", output.String())
	}
	tags.Enable("mk_transitivity")
	_, err = m.MkProofTransitivity(left, differentDecl)
	if ASTSourceDebug && !errors.Is(err, ErrASTDomain) {
		t.Fatalf("declaration trace must precede guard: %v", err)
	} else if !ASTSourceDebug {
		astTestMust(t, err)
	}
	if tags.ShouldTrace("mk_transitivity") {
		got := output.String()
		want := "-------- [mk_transitivity] mk_transitivity src/ast/ast.cpp:2717 ---------\n" +
			"(= trans-a trans-b)\n\n(=> trans-b trans-d)\n" +
			"(declare-fun = (Bool Bool) Bool)\n(declare-fun => (Bool Bool) Bool)\n" +
			"------------------------------------------------\n"
		if got != want {
			t.Fatalf("source declaration trace: got %q, want %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased declaration trace wrote %q", output.String())
	}
	output.Reset()
	differentMiddle := proof(ASTOpIff, d, a)
	_, err = m.MkProofTransitivity(left, differentMiddle)
	if ASTSourceDebug && !errors.Is(err, ErrASTDomain) {
		t.Fatalf("middle trace must precede guard: %v", err)
	} else if !ASTSourceDebug {
		astTestMust(t, err)
	}
	if tags.ShouldTrace("mk_transitivity") {
		got := output.String()
		want := "-------- [mk_transitivity] mk_transitivity src/ast/ast.cpp:2726 ---------\n" +
			"(= trans-a trans-b)\n\n(= trans-d trans-a)\n" +
			"(hypothesis (iff trans-a trans-b))\n\n(hypothesis (iff trans-d trans-a))\n\n" +
			"------------------------------------------------\n"
		if got != want {
			t.Fatalf("source middle trace: got %q, want %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased middle trace wrote %q", output.String())
	}
}
