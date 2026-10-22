package interp

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestInterpolationOptionsErrorsAndLinearParents(t *testing.T) {
	_, manager := interpProofManagerFixture(t)
	base, err := NewBase(manager, nil, nil, nil)
	interpMust(t, err)
	t.Cleanup(func() { interpMust(t, base.Close()) })
	options := NewOptions()
	options.Set("weak", "1")
	interpMust(t, options.Apply(base))
	if weak, e := base.WeakMode(); e != nil || !weak {
		t.Fatal("weak option", weak, e)
	}
	if !IsLinearParents(nil) || !IsLinearParents([]int{1, 2, ScopeTop}) || IsLinearParents([]int{2, 2, ScopeTop}) {
		t.Fatal("linear parent recognition")
	}
	if (InterpolationBadTreeError{}).Error() != "iz3_bad_tree" || (IncompletenessError{}).Error() != "iz3_incompleteness" || (InterpolationFailure{Message: "failed"}).Error() != "failed" || (InterpolationError{}).Error() != "theory not supported by interpolation or bad proof" {
		t.Fatal("interpolation error contract")
	}
	if err = options.Apply(nil); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("nil base guard", err)
	}
}

func TestInterpolationProofAssumptionsDeduplicateDAG(t *testing.T) {
	m, manager := interpProofManagerFixture(t)
	boolSort, err := manager.BoolType()
	interpMust(t, err)
	p, err := manager.MakeVariable("assumption-p", boolSort)
	interpMust(t, err)
	p = interpPin(t, m, p)
	q, err := manager.MakeVariable("assumption-q", boolSort)
	interpMust(t, err)
	q = interpPin(t, m, q)
	pp, err := m.MkProofAsserted(p)
	interpMust(t, err)
	pp = interpPin(t, m, pp)
	qp, err := m.MkProofAsserted(q)
	interpMust(t, err)
	qp = interpPin(t, m, qp)
	falsehood, err := manager.Make(OperatorFalse)
	interpMust(t, err)
	arithmetic, err := m.ArithmeticFamilyID()
	interpMust(t, err)
	root := interpTheoryLemma(t, m, arithmetic, "farkas", falsehood, []ast.ASTHandle{pp, qp, pp}, "1", "1", "1")
	assumptions, err := ProofAssumptions(manager, root)
	interpMust(t, err)
	if len(assumptions) != 2 || assumptions[0] != p || assumptions[1] != q {
		t.Fatal("proof assumptions", assumptions)
	}
}

func TestInterpolationCollectAndAssertConjuncts(t *testing.T) {
	m, manager := interpProofManagerFixture(t)
	boolSort, err := manager.BoolType()
	interpMust(t, err)
	makeVar := func(name string) ast.ASTHandle {
		h, e := manager.MakeVariable(name, boolSort)
		interpMust(t, e)
		return interpPin(t, m, h)
	}
	p, q, r, s := makeVar("collect-p"), makeVar("collect-q"), makeVar("collect-r"), makeVar("collect-s")
	qr, err := manager.Make(OperatorAnd, q, r)
	interpMust(t, err)
	marked, err := manager.Make(OperatorInterp, qr)
	interpMust(t, err)
	tree, err := manager.Make(OperatorAnd, p, marked, s)
	interpMust(t, err)
	tree = interpPin(t, m, tree)
	constraints, err := CollectConjuncts(manager, tree)
	interpMust(t, err)
	if len(constraints) != 3 || constraints[0] != p || constraints[1] != qr || constraints[2] != s {
		t.Fatal("collected conjuncts", constraints)
	}
	target, err := newInterpolationScriptSolver(m, lbool.Undef)
	interpMust(t, err)
	t.Cleanup(func() { interpMust(t, target.Close()) })
	asserted, err := AssertConjuncts(manager, target, tree)
	interpMust(t, err)
	if len(asserted) != 3 || len(target.assertions) != 3 {
		t.Fatal("asserted conjuncts", asserted, target.assertions)
	}
}

func TestInterpolationFrameReducerTreeAndRestoration(t *testing.T) {
	m, manager := interpProofManagerFixture(t)
	boolSort, err := manager.BoolType()
	interpMust(t, err)
	parts := make([][]ast.ASTHandle, 4)
	for i, name := range []string{"frame-a", "frame-b", "frame-c", "frame-d"} {
		formula, e := manager.MakeVariable(name, boolSort)
		interpMust(t, e)
		parts[i] = []ast.ASTHandle{interpPin(t, m, formula)}
	}
	proof, err := m.MkProofAsserted(parts[0][0])
	interpMust(t, err)
	proof = interpPin(t, m, proof)
	reducer, err := NewFrameReducer(manager)
	interpMust(t, err)
	reduced, parents, err := reducer.Reduce(parts, []int{2, 2, 3, ScopeTop}, proof)
	interpMust(t, err)
	if len(reduced) != 2 || reduced[0][0] != parts[0][0] || reduced[1][0] != parts[3][0] {
		t.Fatal("reduced frames", reduced)
	}
	if len(parents) != 2 || parents[0] != ScopeTop || parents[1] != ScopeTop {
		t.Fatal("reduced parents", parents)
	}
	fixed, err := reducer.FixInterpolants([]ast.ASTHandle{parts[1][0]})
	interpMust(t, err)
	defer func() {
		for i := range fixed {
			interpMust(t, m.DelRef(&fixed[i]))
		}
	}()
	if len(fixed) != 3 {
		t.Fatal("fixed interpolant count", len(fixed))
	}
	got0, err := m.RefGet(fixed[0])
	interpMust(t, err)
	got1, err := m.RefGet(fixed[1])
	interpMust(t, err)
	got2, err := m.RefGet(fixed[2])
	interpMust(t, err)
	if got0 != parts[1][0] || got2 != parts[1][0] {
		t.Fatal("restored interpolants", got0, got2)
	}
	if truth, e := manager.IsTrue(got1); e != nil || !truth {
		t.Fatal("removed sibling interpolant", truth, e)
	}
	if _, err = reducer.FixInterpolants(nil); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("fixed interpolant length guard", err)
	}
}

func TestInterpolationFrameReducerSequenceAndGuards(t *testing.T) {
	m, manager := interpProofManagerFixture(t)
	truth, err := manager.Make(OperatorTrue)
	interpMust(t, err)
	truth = interpPin(t, m, truth)
	falsehood, err := manager.Make(OperatorFalse)
	interpMust(t, err)
	falsehood = interpPin(t, m, falsehood)
	proof, err := m.MkProofAsserted(truth)
	interpMust(t, err)
	proof = interpPin(t, m, proof)
	reducer, err := NewFrameReducer(manager)
	interpMust(t, err)
	parts := [][]ast.ASTHandle{{truth}, {falsehood}}
	reduced, parents, err := reducer.Reduce(parts, nil, proof)
	interpMust(t, err)
	if len(reduced) != 2 || parents != nil {
		t.Fatal("sequence reduction", reduced, parents)
	}
	if _, _, err = reducer.Reduce(nil, nil, proof); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("empty frame guard", err)
	}
	if _, _, err = reducer.Reduce(parts, []int{ScopeTop}, proof); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("parent length guard", err)
	}
}

func TestFullTranslationProofAnalysisAndResolution(t *testing.T) {
	m, manager := interpRewritingManagerFixture(t)
	boolean, err := manager.BoolType()
	interpMust(t, err)
	p, err := manager.MakeVariable("translation-a", boolean)
	interpMust(t, err)
	p = interpPin(t, m, p)
	q, err := manager.MakeVariable("translation-b", boolean)
	interpMust(t, err)
	q = interpPin(t, m, q)
	base, err := NewBase(manager, []ast.ASTHandle{p, q}, nil, nil)
	interpMust(t, err)
	t.Cleanup(func() { interpMust(t, base.Close()) })
	translation, err := NewFullTranslation(base)
	interpMust(t, err)
	t.Cleanup(func() { interpMust(t, translation.Close()) })

	falsehood, err := manager.Make(OperatorFalse)
	interpMust(t, err)
	truth, err := manager.Make(OperatorTrue)
	interpMust(t, err)
	hypothesis, err := m.MkProofHypothesis(falsehood)
	interpMust(t, err)
	lemma, err := m.MkProofLemma(hypothesis, truth)
	interpMust(t, err)
	hypotheses, err := translation.ProofHypotheses(lemma)
	interpMust(t, err)
	if len(hypotheses) != 0 {
		t.Fatal("discharged hypotheses", hypotheses)
	}

	mixed, err := manager.Make(OperatorAnd, p, q)
	interpMust(t, err)
	equivalence, err := manager.Make(OperatorIff, mixed, p)
	interpMust(t, err)
	equivalenceProof, err := m.MkProofAsserted(equivalence)
	interpMust(t, err)
	interpMust(t, translation.FindEquivalences(equivalenceProof))
	local, proof, found, err := translation.EquivalentLocal(mixed)
	if err != nil || !found || local != p || proof != equivalenceProof {
		t.Fatal("local equivalence", local, proof, found, err)
	}

	clause, err := manager.Make(OperatorOr, p, q)
	interpMust(t, err)
	clauseProof, err := m.MkProofAsserted(clause)
	interpMust(t, err)
	notP, err := manager.MkNot(p)
	interpMust(t, err)
	unitProof, err := m.MkProofAsserted(notP)
	interpMust(t, err)
	resolution, err := m.MkProofUnitResolution([]ast.ASTHandle{clauseProof, unitProof})
	interpMust(t, err)
	literals, err := translation.UnitResolutionLiterals(resolution, 1)
	interpMust(t, err)
	if len(literals) != 1 || literals[0] != q {
		t.Fatal("unit resolution literals", literals)
	}
	empty, err := translation.ClauseLiterals(falsehood)
	if err != nil || len(empty) != 0 {
		t.Fatal("false clause", empty, err)
	}
	assertedP, err := m.MkProofAsserted(p)
	interpMust(t, err)
	if locality, e := translation.Locality(assertedP); e != nil || locality != 0 {
		t.Fatal("asserted locality", locality, e)
	}
	assertedQ, err := m.MkProofAsserted(q)
	interpMust(t, err)
	if locality, e := translation.Locality(assertedQ); e != nil || locality != 1 {
		t.Fatal("second asserted locality", locality, e)
	}
	interpMust(t, translation.ScanSkolems(assertedP))
	if quantified, e := translation.Quantify(p, DownwardScopeRange(0)); e != nil || quantified != p {
		t.Fatal("identity quantification", quantified, e)
	}
	mixedHypothesis, err := m.MkProofHypothesis(mixed)
	interpMust(t, err)
	destination, err := NewInterpolationProof(base)
	interpMust(t, err)
	t.Cleanup(func() { interpMust(t, destination.Close()) })
	node, err := translation.Translate(mixedHypothesis, destination)
	interpMust(t, err)
	interpolant, err := destination.Interpolate(node, DownwardScopeRange(0), false)
	interpMust(t, err)
	interpMust(t, m.DelRef(&interpolant))
}
