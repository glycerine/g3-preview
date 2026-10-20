package proofs

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func proofPin(t *testing.T, m *ast.ASTManager, h ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	proofMust(t, m.IncRef(h))
	t.Cleanup(func() { proofMust(t, m.DecRef(h)) })
	return h
}

func proofHyperRead(t *testing.T, m *ast.ASTManager, p ast.ASTHandle) ([]ast.ASTHandle, ast.ASTHandle, [][]ast.ASTHandle) {
	t.Helper()
	conclusion, err := m.NewRef(ast.ASTHandle{})
	proofMust(t, err)
	defer func() { proofMust(t, m.DelRef(&conclusion)) }()
	out, err := ast.NewASTProofHyperOutput(m)
	proofMust(t, err)
	defer func() { proofMust(t, out.Close()) }()
	var premises []ast.ASTHandle
	yes, err := m.MatchProofHyperResolve(p, &premises, conclusion, out)
	proofMust(t, err)
	if !yes {
		t.Fatal("expected hyper-resolution proof", p)
	}
	for _, h := range premises {
		h := h
		defer func() { proofMust(t, m.DecRef(h)) }()
	}
	fact, err := m.RefGet(conclusion)
	proofMust(t, err)
	fact = proofPin(t, m, fact)
	borrowed := make([][]ast.ASTHandle, out.SubstitutionsLen())
	for i := range borrowed {
		values, e := out.Substitution(uint32(i))
		proofMust(t, e)
		borrowed[i] = append([]ast.ASTHandle(nil), values...)
	}
	return append([]ast.ASTHandle(nil), premises...), fact, borrowed
}

func TestPushInstantiationsUpHyperPremiseAndResetSubstitution(t *testing.T) {
	m := proofManager(t)
	proofMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	b, err := m.MkBoolSort()
	proofMust(t, err)
	v, err := m.MkVar(0, b)
	proofMust(t, err)
	q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{b}, Names: []usymbol.Symbol{usymbol.Numerical(1801)}, Body: v})
	proofMust(t, err)
	asserted, err := m.MkProofAsserted(q)
	proofMust(t, err)
	asserted = proofPin(t, m, asserted)
	truth, err := m.MkTrue()
	proofMust(t, err)
	root, err := m.MkProofHyperResolve([]ast.ASTHandle{asserted}, q, nil, [][]ast.ASTHandle{{truth}})
	proofMust(t, err)
	root = proofPin(t, m, root)
	ref, err := m.NewRef(root)
	proofMust(t, err)
	defer func() { proofMust(t, m.DelRef(&ref)) }()
	proofMust(t, PushInstantiationsUp(m, ref))
	result, err := m.RefGet(ref)
	proofMust(t, err)
	parents, conclusion, substitutions := proofHyperRead(t, m, result)
	if conclusion != q || len(parents) != 1 || len(substitutions) != 1 || len(substitutions[0]) != 0 {
		t.Fatal("outer hyper-resolution", parents, conclusion, q, substitutions)
	}
	innerParents, innerConclusion, innerSubstitutions := proofHyperRead(t, m, parents[0])
	if innerConclusion != truth || len(innerParents) != 1 || innerParents[0] != asserted || len(innerSubstitutions) != 1 || len(innerSubstitutions[0]) != 1 || innerSubstitutions[0][0] != truth {
		t.Fatal("pushed leaf instantiation", innerParents, innerConclusion, innerSubstitutions)
	}
}

func TestPushInstantiationsUpThroughQuantifiedModusPonens(t *testing.T) {
	m := proofManager(t)
	proofMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	b, err := m.MkBoolSort()
	proofMust(t, err)
	v, err := m.MkVar(0, b)
	proofMust(t, err)
	nv, err := m.MkNot(v)
	proofMust(t, err)
	name := []usymbol.Symbol{usymbol.Numerical(1811)}
	q1, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{b}, Names: name, Body: v})
	proofMust(t, err)
	q2, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{b}, Names: name, Body: nv})
	proofMust(t, err)
	p0, err := m.MkProofAsserted(q1)
	proofMust(t, err)
	p0 = proofPin(t, m, p0)
	p1, err := m.MkProofRewrite(q1, q2)
	proofMust(t, err)
	p1 = proofPin(t, m, p1)
	mp, err := m.MkProofModusPonens(p0, p1)
	proofMust(t, err)
	mp = proofPin(t, m, mp)
	truth, err := m.MkTrue()
	proofMust(t, err)
	root, err := m.MkProofHyperResolve([]ast.ASTHandle{mp}, q2, nil, [][]ast.ASTHandle{{truth}})
	proofMust(t, err)
	root = proofPin(t, m, root)
	ref, err := m.NewRef(root)
	proofMust(t, err)
	defer func() { proofMust(t, m.DelRef(&ref)) }()
	proofMust(t, PushInstantiationsUp(m, ref))
	result, err := m.RefGet(ref)
	proofMust(t, err)
	parents, _, _ := proofHyperRead(t, m, result)
	if len(parents) != 1 {
		t.Fatal("outer parent count", parents)
	}
	yes, err := m.IsProofKind(parents[0], ast.ASTPrModusPonens)
	proofMust(t, err)
	if !yes {
		t.Fatal("quantified modus ponens was not rebuilt", parents[0])
	}
	fact, err := m.GetProofFact(parents[0])
	proofMust(t, err)
	falsehood, err := m.MkNot(truth)
	proofMust(t, err)
	if fact != falsehood {
		t.Fatal("instantiated modus ponens fact", fact, falsehood)
	}
}
