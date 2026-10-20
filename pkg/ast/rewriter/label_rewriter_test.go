package rewriter

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestLabelRewriterRemovesNestedLabelsAndPreservesNonlabels(t *testing.T) {
	m := astTestManager(t)
	rw, err := NewLabelRewriter(m)
	astTestMust(t, err)
	defer func() { astTestMust(t, rw.Close()) }()
	b, err := m.MkBoolSort()
	astTestMust(t, err)
	a, err := m.MkConst(astTestSymbol(t, m, "label-a"), b)
	astTestMust(t, err)
	inner, err := m.MkLabel(true, []usymbol.Symbol{astTestSymbol(t, m, "inner")}, a)
	astTestMust(t, err)
	outer, err := m.MkLabel(false, []usymbol.Symbol{astTestSymbol(t, m, "outer")}, inner)
	astTestMust(t, err)
	root, err := m.MkNot(outer)
	astTestMust(t, err)
	astTestPin(t, m, root)
	formula := astTestPin(t, m, root)
	proof := astTestPin(t, m, ast.ASTHandle{})
	astTestMust(t, rw.RemoveLabels(formula, proof))
	got, err := m.RefGet(formula)
	astTestMust(t, err)
	want, err := m.MkNot(a)
	astTestMust(t, err)
	if got != want {
		t.Fatal("nested label removal", got, want)
	}
	gotProof, err := m.RefGet(proof)
	astTestMust(t, err)
	if !gotProof.IsNull() {
		t.Fatal("null proof changed", gotProof)
	}
	astTestMust(t, rw.RemoveLabels(formula, proof))
	again, err := m.RefGet(formula)
	astTestMust(t, err)
	if again != got {
		t.Fatal("label-free formula changed", got, again)
	}
}

func TestLabelRewriterAdvancesExistingProof(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	rw, err := NewLabelRewriter(m)
	astTestMust(t, err)
	defer func() { astTestMust(t, rw.Close()) }()
	b, err := m.MkBoolSort()
	astTestMust(t, err)
	a, err := m.MkConst(astTestSymbol(t, m, "label-proof-a"), b)
	astTestMust(t, err)
	labeled, err := m.MkLabel(true, []usymbol.Symbol{astTestSymbol(t, m, "proof-label")}, a)
	astTestMust(t, err)
	astTestPin(t, m, labeled)
	prior, err := m.MkProofAsserted(labeled)
	astTestMust(t, err)
	formula := astTestPin(t, m, labeled)
	proof := astTestPin(t, m, prior)
	astTestMust(t, rw.RemoveLabels(formula, proof))
	gotFormula, err := m.RefGet(formula)
	astTestMust(t, err)
	if gotFormula != a {
		t.Fatal("proof path formula", gotFormula, a)
	}
	gotProof, err := m.RefGet(proof)
	astTestMust(t, err)
	if gotProof == prior || gotProof.IsNull() {
		t.Fatal("proof was not advanced", prior, gotProof)
	}
	fact, err := m.GetProofFact(gotProof)
	astTestMust(t, err)
	if fact != a {
		t.Fatal("advanced proof fact", fact, a)
	}
}

func TestLabelRewriterDebugArityAndOwnerGuards(t *testing.T) {
	m := astTestManager(t)
	rw, err := NewLabelRewriter(m)
	astTestMust(t, err)
	out := astTestPin(t, m, ast.ASTHandle{})
	proof := astTestPin(t, m, ast.ASTHandle{})
	b, err := m.MkBoolSort()
	astTestMust(t, err)
	a, err := m.MkConst(astTestSymbol(t, m, "label-guard-a"), b)
	astTestMust(t, err)
	decl, err := m.GetDecl(func() ast.ASTHandle {
		h, e := m.MkLabel(true, []usymbol.Symbol{astTestSymbol(t, m, "guard-label")}, a)
		astTestMust(t, e)
		return h
	}())
	astTestMust(t, err)
	if ast.ASTSourceDebug {
		if _, err = rw.config.ReduceApp(decl, ast.SliceArguments(nil), out, proof); !errors.Is(err, ast.ErrASTDomain) {
			t.Fatal("debug arity assertion", err)
		}
	}
	copyOwner := *rw
	if err = copyOwner.RemoveLabels(out, proof); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("copied owner accepted", err)
	}
	astTestMust(t, rw.Close())
	if err = rw.RemoveLabels(out, proof); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("closed owner accepted", err)
	}
	astTestMust(t, rw.Close())
}
