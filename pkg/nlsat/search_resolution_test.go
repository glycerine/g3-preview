package nlsat

import (
	"testing"

	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/sat"
)

func TestSearchResolutionAntecedentStagesMarksAndDeduplication(t *testing.T) {
	e, r := searchStateFixture(t)
	q, err := newSearchResolution(r)
	assignmentMust(t, err)
	t.Cleanup(func() { assignmentMust(t, q.close()) })
	x0 := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 0})
	x1 := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 1})
	b0, err := r.store.atoms.mkInequality(AtomGT, []polynomial.Polynomial{x0}, []bool{false})
	assignmentMust(t, err)
	b1, err := r.store.atoms.mkInequality(AtomGT, []polynomial.Polynomial{x1}, []bool{false})
	assignmentMust(t, err)
	evaluatorSetFraction(t, e, 0, 0, 1)
	r.xk = 1
	lowerFalse := sat.MakeLiteral(b0, false)
	assignmentMust(t, q.processAntecedent(lowerFalse))
	assignmentMust(t, q.processAntecedent(lowerFalse))
	if len(q.lemma.result) != 1 || q.lemma.result[0] != lowerFalse || !q.marked(b0) || q.numMarks != 0 {
		t.Fatal("lower-stage arithmetic antecedent", q.lemma.result, q.numMarks)
	}

	currentTrue := sat.MakeLiteral(b1, false)
	assignmentMust(t, r.decide(currentTrue))
	currentFalse := currentTrue.Negated()
	assignmentMust(t, q.processAntecedent(currentFalse))
	assignmentMust(t, q.processAntecedent(currentFalse))
	if len(q.lemma.result) != 1 || !q.marked(b1) || q.numMarks != 1 {
		t.Fatal("current-stage assigned mark", q.lemma.result, q.numMarks)
	}
	previous, err := q.onlyPreviousStages(q.lemma.result)
	assignmentMust(t, err)
	if !previous {
		t.Fatal("lower-stage lemma classification")
	}
	maxLevel, err := q.maxScopeLevel([]sat.Literal{currentFalse, lowerFalse})
	assignmentMust(t, err)
	if maxLevel != r.level {
		t.Fatal("maximum assigned scope level", maxLevel, r.level)
	}
	assignmentMust(t, q.lemma.push(currentFalse))
	assignmentMust(t, q.removeLiteralsFromLevel(&q.lemma, r.level))
	if len(q.lemma.result) != 1 || q.lemma.result[0] != lowerFalse || q.numMarks != 2 {
		t.Fatal("same-level literal removal", q.lemma.result, q.numMarks)
	}
}

func TestSearchResolutionClauseSkipAndNewArithmeticLevel(t *testing.T) {
	e, r := searchStateFixture(t)
	q, err := newSearchResolution(r)
	assignmentMust(t, err)
	t.Cleanup(func() { assignmentMust(t, q.close()) })
	x0 := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 0})
	x1 := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 1})
	b0, err := r.store.atoms.mkInequality(AtomGT, []polynomial.Polynomial{x0}, []bool{false})
	assignmentMust(t, err)
	b1, err := r.store.atoms.mkInequality(AtomGT, []polynomial.Polynomial{x1}, []bool{false})
	assignmentMust(t, err)
	r.xk = 1
	assignmentMust(t, r.newLevel())
	r.store.atoms.bools.levels[b1] = 0
	l0 := sat.MakeLiteral(b0, false)
	l1 := sat.MakeLiteral(b1, false)
	level, err := q.findNewArithmeticLevel([]sat.Literal{l1, l0})
	assignmentMust(t, err)
	if level != 0 {
		t.Fatal("same-stage prior decision level", level)
	}
	level, err = q.findNewArithmeticLevel([]sat.Literal{l0, l1})
	assignmentMust(t, err)
	if level != 0 {
		t.Fatal("no earlier same-stage literal fallback", level)
	}
	boolean, err := q.isBooleanLemma([]sat.Literal{l0, l1})
	assignmentMust(t, err)
	if boolean {
		t.Fatal("arithmetic lemma classified Boolean")
	}
}

func TestSearchResolutionLazyClauseUsesComposedExplanation(t *testing.T) {
	e, r, n, _ := projectionCoefficientFixture(t, true)
	q, err := newSearchResolution(r)
	assignmentMust(t, err)
	t.Cleanup(func() { assignmentMust(t, q.close()) })
	evaluatorSetFraction(t, e, 0, 0, 1)
	evaluatorSetFraction(t, e, 1, 1, 1)
	p := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 2}, evaluatorTerm{1, 1, 0}, evaluatorTerm{-2, 0, 0})
	b, err := r.store.atoms.mkInequality(AtomEQ, []polynomial.Polynomial{p}, []bool{false})
	assignmentMust(t, err)
	conflict := sat.MakeLiteral(b, false)
	j, err := NewLazyJustification(r.store.atoms.ineq.allocator, []sat.Literal{conflict})
	assignmentMust(t, err)
	t.Cleanup(func() { assignmentMust(t, DeleteJustification(r.store.atoms.ineq.allocator, j)) })
	lazy, err := j.Lazy()
	assignmentMust(t, err)
	assignmentMust(t, q.buildLazyClause(lazy, n))
	if len(q.lazy.result) != 2 || q.lazy.result[1] != conflict.Negated() || len(n.output.result) != 0 {
		t.Fatal("lazy explanation clause composition", q.lazy.result, n.output.result)
	}
	root, ok := r.store.atoms.bools.atoms[q.lazy.result[0].Variable()].(*RootAtom)
	if !ok || root.kind != AtomRootLT || !q.lazy.result[0].Sign() {
		t.Fatal("lazy clause projected boundary", q.lazy.result[0], root)
	}
}

func TestSearchResolutionAnalyzesDecisionAndClauseTrail(t *testing.T) {
	_, r := searchStateFixture(t)
	q, err := newSearchResolution(r)
	assignmentMust(t, err)
	t.Cleanup(func() { assignmentMust(t, q.close()) })

	decision, err := r.store.atoms.mkBoolVar()
	assignmentMust(t, err)
	derived, err := r.store.atoms.mkBoolVar()
	assignmentMust(t, err)
	decisionLiteral := sat.MakeLiteral(decision, false)
	derivedLiteral := sat.MakeLiteral(derived, false)
	assignmentMust(t, r.decide(decisionLiteral))
	reason := searchClause(t, r, decisionLiteral.Negated(), derivedLiteral)
	justification, err := NewClauseJustification(reason)
	assignmentMust(t, err)
	assignmentMust(t, r.assign(derivedLiteral, justification))
	conflict := searchClause(t, r, derivedLiteral.Negated())

	found, err := q.analyze(conflict, nil)
	assignmentMust(t, err)
	if !found || len(q.lemma.result) != 1 || q.lemma.result[0] != decisionLiteral.Negated() {
		t.Fatal("clause trail did not resolve to the false decision literal", found, q.lemma.result)
	}
	if q.numMarks != 0 {
		t.Fatal("analysis retained marked trail literals", q.numMarks)
	}
	for b, marked := range q.marks {
		if marked {
			t.Fatal("analysis retained mark", b)
		}
	}

	assignmentMust(t, r.undoUntilLevel(0))
	assignmentMust(t, q.lemma.reset())
	assignmentMust(t, r.decide(decisionLiteral.Negated()))
	direct := searchClause(t, r, decisionLiteral)
	found, err = q.analyze(direct, nil)
	assignmentMust(t, err)
	if !found || len(q.lemma.result) != 1 || q.lemma.result[0] != decisionLiteral {
		t.Fatal("direct conflict decision polarity", found, q.lemma.result)
	}
}

func TestSearchResolutionRejectsUnjoinedAssumptionsBeforeMutation(t *testing.T) {
	_, r := searchStateFixture(t)
	q, err := newSearchResolution(r)
	assignmentMust(t, err)
	t.Cleanup(func() { assignmentMust(t, q.close()) })
	b, err := r.store.atoms.mkBoolVar()
	assignmentMust(t, err)
	literal := sat.MakeLiteral(b, false)
	attachment := &AssumptionAttachment{}
	set, err := r.store.assumptions.mkLeaf(AttachedAssumption(attachment))
	assignmentMust(t, err)
	conflict, err := r.store.mkClause([]sat.Literal{literal}, false, set)
	assignmentMust(t, err)
	if _, err = q.analyze(conflict, nil); err != ErrResolutionAssumptions {
		t.Fatal("assumption-bearing conflict was not rejected explicitly", err)
	}
	if len(q.lemma.result) != 0 || q.numMarks != 0 {
		t.Fatal("rejected conflict mutated analysis scratch", q.lemma.result, q.numMarks)
	}
}
