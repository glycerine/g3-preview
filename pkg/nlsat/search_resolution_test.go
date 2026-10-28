package nlsat

import (
	"testing"

	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/sat"
	"github.com/glycerine/g3/pkg/util/lbool"
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

func TestSearchResolutionJoinsConflictAssumptions(t *testing.T) {
	_, r := searchStateFixture(t)
	q, err := newSearchResolution(r)
	assignmentMust(t, err)
	t.Cleanup(func() { assignmentMust(t, q.close()) })
	b, err := r.store.atoms.mkBoolVar()
	assignmentMust(t, err)
	literal := sat.MakeLiteral(b, false)
	reason := searchClause(t, r, literal)
	justification, err := NewClauseJustification(reason)
	assignmentMust(t, err)
	assignmentMust(t, r.assign(literal, justification))
	attachment := &AssumptionAttachment{}
	set, err := r.store.assumptions.mkLeaf(AttachedAssumption(attachment))
	assignmentMust(t, err)
	conflict, err := r.store.mkClause([]sat.Literal{literal.Negated()}, false, set)
	assignmentMust(t, err)
	found, err := q.analyze(conflict, nil)
	assignmentMust(t, err)
	if found || len(q.lemma.result) != 0 || q.lemmaAssumptions == nil {
		t.Fatal("assumption-bearing empty lemma", found, q.lemma.result, q.lemmaAssumptions)
	}
	var assumptions []Assumption
	assignmentMust(t, r.store.assumptions.linearize(q.lemmaAssumptions, &assumptions))
	if len(assumptions) != 1 || assumptions[0].Attachment() != attachment {
		t.Fatal("conflict assumption was not retained", assumptions)
	}
}

func TestSearchResolutionPublishesBooleanLemmaAndDetectsEmptyLemma(t *testing.T) {
	t.Run("learned", func(t *testing.T) {
		_, r := searchStateFixture(t)
		q, err := newSearchResolution(r)
		assignmentMust(t, err)
		t.Cleanup(func() { assignmentMust(t, q.close()) })
		decision, err := r.store.atoms.mkBoolVar()
		assignmentMust(t, err)
		derived, err := r.store.atoms.mkBoolVar()
		assignmentMust(t, err)
		a := sat.MakeLiteral(decision, false)
		b := sat.MakeLiteral(derived, false)
		assignmentMust(t, r.decide(a))
		reason := searchClause(t, r, a.Negated(), b)
		justification, err := NewClauseJustification(reason)
		assignmentMust(t, err)
		assignmentMust(t, r.assign(b, justification))
		conflict := searchClause(t, r, b.Negated())

		solved, err := q.resolveConflict(conflict, nil)
		assignmentMust(t, err)
		if !solved || len(r.store.learned) != 1 {
			t.Fatal("conflict did not publish one learned clause", solved, len(r.store.learned))
		}
		learned := r.store.learned[0]
		if len(learned.Literals()) != 1 || learned.Literals()[0] != a.Negated() {
			t.Fatal("learned decision-flipping lemma", learned.Literals())
		}
		value, err := r.assignedValue(a.Negated())
		assignmentMust(t, err)
		if value != lbool.True {
			t.Fatal("learned unit was not asserted after backjump", value)
		}
		justification = r.store.atoms.bools.justifications[decision]
		if !justification.IsClause() || justification.clause != learned {
			t.Fatal("learned assertion did not retain its clause reason")
		}
	})

	t.Run("empty", func(t *testing.T) {
		_, r := searchStateFixture(t)
		q, err := newSearchResolution(r)
		assignmentMust(t, err)
		t.Cleanup(func() { assignmentMust(t, q.close()) })
		b, err := r.store.atoms.mkBoolVar()
		assignmentMust(t, err)
		literal := sat.MakeLiteral(b, false)
		reason := searchClause(t, r, literal)
		ok, err := r.processClause(reason, false)
		assignmentMust(t, err)
		if !ok {
			t.Fatal("unit reason unexpectedly conflicted")
		}
		conflict := searchClause(t, r, literal.Negated())
		solved, err := q.resolveConflict(conflict, nil)
		assignmentMust(t, err)
		if solved || len(q.lemma.result) != 0 || len(r.store.learned) != 0 {
			t.Fatal("empty lemma did not report unsatisfiable", solved, q.lemma.result, len(r.store.learned))
		}
	})
}

func TestSearchResolutionPublishesJoinedReasonAssumptions(t *testing.T) {
	_, r := searchStateFixture(t)
	q, err := newSearchResolution(r)
	assignmentMust(t, err)
	t.Cleanup(func() { assignmentMust(t, q.close()) })
	a, err := r.store.atoms.mkBoolVar()
	assignmentMust(t, err)
	b, err := r.store.atoms.mkBoolVar()
	assignmentMust(t, err)
	al := sat.MakeLiteral(a, false)
	bl := sat.MakeLiteral(b, false)
	assignmentMust(t, r.decide(al))
	attachments := [2]*AssumptionAttachment{{}, {}}
	sets := [2]*AssumptionSet{}
	for i := range sets {
		sets[i], err = r.store.assumptions.mkLeaf(AttachedAssumption(attachments[i]))
		assignmentMust(t, err)
	}
	reason, err := r.store.mkClause([]sat.Literal{al.Negated(), bl}, false, sets[0])
	assignmentMust(t, err)
	justification, err := NewClauseJustification(reason)
	assignmentMust(t, err)
	assignmentMust(t, r.assign(bl, justification))
	conflict, err := r.store.mkClause([]sat.Literal{bl.Negated()}, false, sets[1])
	assignmentMust(t, err)

	solved, err := q.resolveConflict(conflict, nil)
	assignmentMust(t, err)
	if !solved || len(r.store.learned) != 1 || r.store.learned[0].Assumptions() == nil {
		t.Fatal("joined learned assumptions were not published", solved, len(r.store.learned))
	}
	var assumptions []Assumption
	assignmentMust(t, r.store.assumptions.linearize(r.store.learned[0].Assumptions(), &assumptions))
	if len(assumptions) != 2 {
		t.Fatal("learned assumption count", assumptions)
	}
	seen := map[*AssumptionAttachment]bool{}
	for _, assumption := range assumptions {
		seen[assumption.Attachment()] = true
	}
	if !seen[attachments[0]] || !seen[attachments[1]] {
		t.Fatal("learned clause lost a conflict or reason dependency", assumptions)
	}
}
