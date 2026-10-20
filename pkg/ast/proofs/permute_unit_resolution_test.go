package proofs

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestPermuteUnitResolutionIntoTheoryLemma(t *testing.T) {
	m := proofManager(t)
	proofMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	b, err := m.MkBoolSort()
	proofMust(t, err)
	a, err := m.MkConst(usymbol.Numerical(1901), b)
	proofMust(t, err)
	c, err := m.MkConst(usymbol.Numerical(1902), b)
	proofMust(t, err)
	clause, err := m.MkOr(a, c)
	proofMust(t, err)
	custom, err := m.NewParameterInt(-19)
	proofMust(t, err)
	defer func() { proofMust(t, m.DelParameter(&custom)) }()
	theory, err := m.MkProofTheoryLemma(2, clause, nil, []ast.ASTParameter{custom})
	proofMust(t, err)
	theory = proofPin(t, m, theory)
	notA, err := m.MkNot(a)
	proofMust(t, err)
	unit, err := m.MkProofAsserted(notA)
	proofMust(t, err)
	unit = proofPin(t, m, unit)
	root, err := m.MkProofUnitResolution([]ast.ASTHandle{theory, unit})
	proofMust(t, err)
	root = proofPin(t, m, root)
	ref, err := m.NewRef(root)
	proofMust(t, err)
	defer func() { proofMust(t, m.DelRef(&ref)) }()
	proofMust(t, PermuteUnitResolution(m, ref))
	result, err := m.RefGet(ref)
	proofMust(t, err)
	yes, err := m.IsProofKind(result, ast.ASTPrTheoryLemma)
	proofMust(t, err)
	fact, err := m.GetProofFact(result)
	proofMust(t, err)
	if !yes || fact != c {
		t.Fatal("permuted theory lemma", yes, fact, c)
	}
	if n, e := m.GetProofNumParents(result); e != nil || n != 1 {
		t.Fatal("permuted premise count", n, e)
	}
	parent, err := m.GetProofParent(result, 0)
	proofMust(t, err)
	if parent != unit {
		t.Fatal("unit premise was not moved into theory lemma", parent, unit)
	}
	decl, err := m.GetDecl(result)
	proofMust(t, err)
	params, err := m.GetDeclParametersCopy(decl)
	proofMust(t, err)
	defer func() {
		for i := range params {
			proofMust(t, m.DelParameter(&params[i]))
		}
	}()
	if len(params) != 2 {
		t.Fatal("theory parameters", len(params))
	}
	name, err := m.ParameterSymbol(params[0])
	proofMust(t, err)
	wantName, err := m.Symbols.Display(name)
	proofMust(t, err)
	value, err := m.ParameterInt(params[1])
	proofMust(t, err)
	if wantName != "pattern" || value != -19 {
		t.Fatal("preserved theory parameters", wantName, value)
	}
}

func TestPermuteUnitResolutionOrdinaryProofIdentity(t *testing.T) {
	m := proofManager(t)
	proofMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	truth, err := m.MkTrue()
	proofMust(t, err)
	root, err := m.MkProofAsserted(truth)
	proofMust(t, err)
	root = proofPin(t, m, root)
	ref, err := m.NewRef(root)
	proofMust(t, err)
	defer func() { proofMust(t, m.DelRef(&ref)) }()
	proofMust(t, PermuteUnitResolution(m, ref))
	got, err := m.RefGet(ref)
	proofMust(t, err)
	if got != root {
		t.Fatal("ordinary proof changed", got, root)
	}
}
