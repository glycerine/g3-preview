package smtcore

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

type smtIntegerDiffFixture struct {
	manager *ast.ASTManager
	family  ast.FamilyID
	state   *SMTInternalizerState
	theory  *SMTIntegerDiffTheory
	sort    ast.ASTHandle
	x, y    ast.ASTHandle
}

func newSMTIntegerDiffFixture(t *testing.T) smtIntegerDiffFixture {
	t.Helper()
	manager := astTestManager(t)
	family, err := manager.RegisterArithmeticPlugin()
	if err != nil {
		t.Fatal("register arithmetic", err)
	}
	sort, err := manager.MkFamilySort(family, 1, nil)
	if err != nil {
		t.Fatal("integer sort", err)
	}
	x, err := manager.MkConst(astTestSymbol(t, manager, "integer-diff-x"), sort)
	if err != nil {
		t.Fatal("x", err)
	}
	y, err := manager.MkConst(astTestSymbol(t, manager, "integer-diff-y"), sort)
	if err != nil {
		t.Fatal("y", err)
	}
	_, _, state := smtInternalizerFixture(t, manager, nil)
	theory, err := NewSMTIntegerDiffTheory(family)
	if err != nil {
		t.Fatal("new theory", err)
	}
	if err = state.RegisterTheory(theory); err != nil {
		t.Fatal("register theory", err)
	}
	return smtIntegerDiffFixture{manager, family, state, theory, sort, x, y}
}

func (f smtIntegerDiffFixture) numeral(t *testing.T, value int64) ast.ASTHandle {
	t.Helper()
	n, err := f.manager.ArithmeticNumeralRat(big.NewRat(value, 1), true)
	astTestMust(t, err)
	return n
}

func (f smtIntegerDiffFixture) difference(t *testing.T, left, right ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	minusOne := f.numeral(t, -1)
	negative, err := f.manager.MkFamilyApp(f.family, smtArithOpMul, nil, []ast.ASTHandle{minusOne, right})
	astTestMust(t, err)
	difference, err := f.manager.MkFamilyApp(f.family, smtArithOpAdd, nil, []ast.ASTHandle{left, negative})
	astTestMust(t, err)
	return difference
}

func (f smtIntegerDiffFixture) bound(t *testing.T, op int32, left, right ast.ASTHandle, value int64) ast.ASTHandle {
	t.Helper()
	difference := f.difference(t, left, right)
	bound, err := f.manager.MkFamilyApp(f.family, op, nil, []ast.ASTHandle{difference, f.numeral(t, value)})
	astTestMust(t, err)
	return bound
}

func TestSMTIntegerDiffTheoryAtomEdgesAndIntegerNegation(t *testing.T) {
	f := newSMTIntegerDiffFixture(t)
	atomExpression := f.bound(t, smtArithOpLE, f.x, f.y, 5)
	astTestMust(t, f.state.Internalize(atomExpression, true))
	variable, found := f.state.BoolVar(atomExpression)
	if !found || len(f.theory.atoms) != 1 || f.theory.atoms[0].variable != variable {
		t.Fatal("difference atom registration", variable, found, f.theory.atoms)
	}
	data, err := f.state.BooleanData(variable)
	astTestMust(t, err)
	if !data.IsTheoryAtom() || data.Theory() != f.family {
		t.Fatal("difference atom theory notification", data)
	}
	atom := f.theory.atoms[0]
	positiveWeight, err := f.theory.graph.EdgeWeight(atom.positive)
	astTestMust(t, err)
	negativeWeight, err := f.theory.graph.EdgeWeight(atom.negative)
	astTestMust(t, err)
	if positiveWeight.Cmp(big.NewRat(5, 1)) != 0 || negativeWeight.Cmp(big.NewRat(-6, 1)) != 0 {
		t.Fatal("integer bound polarity", positiveWeight, negativeWeight)
	}
	positiveSource, err := f.theory.graph.EdgeSource(atom.positive)
	astTestMust(t, err)
	positiveTarget, err := f.theory.graph.EdgeTarget(atom.positive)
	astTestMust(t, err)
	negativeSource, err := f.theory.graph.EdgeSource(atom.negative)
	astTestMust(t, err)
	negativeTarget, err := f.theory.graph.EdgeTarget(atom.negative)
	astTestMust(t, err)
	if positiveSource != negativeTarget || positiveTarget != negativeSource || positiveSource == positiveTarget {
		t.Fatal("opposite edge endpoints", positiveSource, positiveTarget, negativeSource, negativeTarget)
	}
}

func TestSMTIntegerDiffTheoryAssignmentsAndNegativeCycleConflict(t *testing.T) {
	f := newSMTIntegerDiffFixture(t)
	first := f.bound(t, smtArithOpLE, f.x, f.y, 0)
	second := f.bound(t, smtArithOpLE, f.y, f.x, -1)
	astTestMust(t, f.state.Internalize(first, true))
	astTestMust(t, f.state.Internalize(second, true))
	firstVariable, _ := f.state.BoolVar(first)
	secondVariable, _ := f.state.BoolVar(second)
	firstLiteral := NewSMTLiteral(firstVariable, false)
	secondLiteral := NewSMTLiteral(secondVariable, false)
	astTestMust(t, f.state.AssignLiteral(firstLiteral, NewSMTBooleanAxiom()))
	astTestMust(t, f.state.AssignLiteral(secondLiteral, NewSMTBooleanAxiom()))
	astTestMust(t, f.state.PropagateTheoryAssignments())
	if !f.theory.CanPropagate() {
		t.Fatal("assigned bounds were not queued")
	}
	astTestMust(t, f.theory.Propagate())
	conflict, found := f.state.Conflict()
	if !found || !f.state.Inconsistent() || conflict.Kind() != SMTBooleanJustificationObject {
		t.Fatal("negative cycle conflict", found, conflict.Kind())
	}
	justification, ok := conflict.Justification()
	if !ok || justification.FromTheory() != f.family || justification.Name() != "theory-conflict" {
		t.Fatal("negative cycle justification", justification, ok)
	}
	simple := justification.(*SMTTheoryConflictJustification)
	if len(simple.Literals) != 2 {
		t.Fatal("negative cycle antecedents", simple.Literals)
	}
	seen := map[SMTLiteral]bool{}
	for _, literal := range simple.Literals {
		seen[literal] = true
	}
	if !seen[firstLiteral] || !seen[secondLiteral] {
		t.Fatal("negative cycle exact antecedents", simple.Literals)
	}
}

func TestSMTIntegerDiffTheoryUnsupportedTermsAndScopeRollback(t *testing.T) {
	f := newSMTIntegerDiffFixture(t)
	baseline := f.bound(t, smtArithOpGE, f.x, f.y, 2)
	astTestMust(t, f.state.Internalize(baseline, true))
	if status, err := f.theory.FinalCheck(); err != nil || status != SMTFinalCheckDone {
		t.Fatal("supported final check", status, err)
	}
	baselineAtoms, baselineEdges := len(f.theory.atoms), f.theory.graph.NumEdges()
	astTestMust(t, f.state.PushScope())
	sum, err := f.manager.MkFamilyApp(f.family, smtArithOpAdd, nil, []ast.ASTHandle{f.x, f.y})
	astTestMust(t, err)
	unsupported, err := f.manager.MkFamilyApp(f.family, smtArithOpLE, nil, []ast.ASTHandle{sum, f.numeral(t, 7)})
	astTestMust(t, err)
	astTestMust(t, f.state.Internalize(unsupported, true))
	if status, finalErr := f.theory.FinalCheck(); finalErr != nil || status != SMTFinalCheckGiveUp {
		t.Fatal("unsupported final check", status, finalErr)
	}
	if len(f.theory.atoms) != baselineAtoms+1 || f.theory.graph.NumEdges() <= baselineEdges {
		t.Fatal("scoped atom and edges", len(f.theory.atoms), f.theory.graph.NumEdges())
	}
	astTestMust(t, f.state.PopScopes(1))
	if len(f.theory.atoms) != baselineAtoms || f.theory.graph.NumEdges() != baselineEdges {
		t.Fatal("scope rollback", len(f.theory.atoms), f.theory.graph.NumEdges())
	}
	if status, finalErr := f.theory.FinalCheck(); finalErr != nil || status != SMTFinalCheckDone {
		t.Fatal("rolled-back final check", status, finalErr)
	}
}
