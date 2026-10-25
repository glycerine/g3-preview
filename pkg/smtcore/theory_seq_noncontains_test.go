package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTSequenceNonContainsQueueAndAssignmentScopes(t *testing.T) {
	m := astTestManager(t)
	sample, err := m.MkString("")
	astTestMust(t, err)
	stringSort, err := m.GetSort(sample)
	astTestMust(t, err)
	left, err := m.MkConst(astTestSymbol(t, m, "queued-contains-left"), stringSort)
	astTestMust(t, err)
	right, err := m.MkConst(astTestSymbol(t, m, "queued-contains-right"), stringSort)
	astTestMust(t, err)
	contains, err := m.MkSequenceContains(left, right)
	astTestMust(t, err)
	family, err := m.SequenceFamilyID()
	astTestMust(t, err)
	theory, err := NewSMTSequenceTheory(family)
	astTestMust(t, err)
	_, _, state := smtInternalizerFixture(t, m, nil)
	astTestMust(t, state.RegisterTheory(theory))
	astTestMust(t, state.PushScope())
	astTestMust(t, state.Internalize(contains, true))
	variable, found := state.BoolVar(contains)
	if !found {
		t.Fatal("contains Boolean variable missing")
	}
	astTestMust(t, theory.Assign(variable, false))
	if theory.nonContains.Len() != 1 {
		t.Fatal("non-contains assignment queue", theory.nonContains.Len())
	}
	if theory.relations.Len() != 1 {
		t.Fatal("relation assignment queue", theory.relations.Len())
	}
	expression, dependency, err := theory.nonContains.At(0)
	astTestMust(t, err)
	if expression != contains || dependency == nil {
		t.Fatal("non-contains entry", expression, dependency)
	}
	assumptions := theory.dependencies.Linearize(dependency)
	if len(assumptions) != 1 || assumptions[0].literal != NewSMTLiteral(variable, true) {
		t.Fatal("non-contains explanation", assumptions)
	}
	astTestMust(t, state.PopScopes(1))
	if theory.nonContains.Len() != 0 {
		t.Fatal("non-contains scope rollback", theory.nonContains.Len())
	}
	if theory.relations.Len() != 0 {
		t.Fatal("relation scope rollback", theory.relations.Len())
	}
}

func TestSMTSequenceNonContainsForcesEmptyBeforeTailExposure(t *testing.T) {
	m := astTestManager(t)
	needle, err := m.MkString("z")
	astTestMust(t, err)
	sort, err := m.GetSort(needle)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "lazy-noncontains-x"), sort)
	astTestMust(t, err)
	family, err := m.SequenceFamilyID()
	astTestMust(t, err)
	theory, err := NewSMTSequenceTheory(family)
	astTestMust(t, err)
	_, _, state := smtInternalizerFixture(t, m, nil)
	astTestMust(t, state.RegisterTheory(theory))
	contains, err := m.MkSequenceContains(x, needle)
	astTestMust(t, err)
	astTestMust(t, state.Internalize(contains, true))
	literal, err := state.Literal(contains)
	astTestMust(t, err)
	astTestMust(t, state.AssignLiteral(literal.Negated(), NewSMTBooleanAxiom()))
	_, err = state.PropagateCore()
	astTestMust(t, err)
	empty, err := m.MkSequenceEmpty(sort)
	astTestMust(t, err)
	emptyEquality, err := m.MkEq(x, empty)
	astTestMust(t, err)
	emptyLiteral, err := state.Literal(emptyEquality)
	astTestMust(t, err)
	data, err := state.BooleanData(emptyLiteral.Var())
	astTestMust(t, err)
	if value, err := state.Assignment(emptyLiteral); err != nil || value != lbool.Undef || !data.PhaseAvailable || !data.Phase || !state.isRelevant(emptyEquality) {
		t.Fatal("source empty phase was not forced before recursive containment", value, err)
	}
	zero, err := theory.integer(0)
	astTestMust(t, err)
	tail, err := theory.relationSkolem("seq.tail", []ast.ASTHandle{x, zero}, sort)
	astTestMust(t, err)
	tailContains, err := m.MkSequenceContains(tail, needle)
	astTestMust(t, err)
	astTestPin(t, m, tailContains)
	if _, found := state.BoolVar(tailContains); found {
		t.Fatal("recursive tail atom exposed before the empty decision")
	}
	astTestMust(t, state.PushScope())
	astTestMust(t, state.AssignLiteral(emptyLiteral.Negated(), NewSMTBooleanAxiom()))
	_, err = state.PropagateCore()
	astTestMust(t, err)
	astTestMust(t, theory.addNegativeContainsStep(contains))
	astTestMust(t, theory.Propagate())
	_, err = state.PropagateCore()
	astTestMust(t, err)
	if _, found := state.BoolVar(tailContains); !found {
		t.Fatal("nonempty branch did not expose the deferred tail")
	}
	astTestMust(t, state.PopScopes(1))
	if _, found := state.BoolVar(tailContains); found {
		t.Fatal("child tail exposure survived scope pop")
	}
}

func TestSMTSequenceNonContainsCanonicalDisjunctionAndScope(t *testing.T) {
	m := astTestManager(t)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	var elements []ast.ASTHandle
	for _, name := range []string{"nc-first", "nc-second", "nc-needle"} {
		element, err := m.MkConst(astTestSymbol(t, m, name), boolean)
		astTestMust(t, err)
		elements = append(elements, element)
	}
	guard, err := m.MkConst(astTestSymbol(t, m, "nc-original-negative-literal"), boolean)
	astTestMust(t, err)
	family, err := m.SequenceFamilyID()
	astTestMust(t, err)
	theory, err := NewSMTSequenceTheory(family)
	astTestMust(t, err)
	_, _, state := smtInternalizerFixture(t, m, nil)
	astTestMust(t, state.RegisterTheory(theory))
	astTestMust(t, state.Internalize(guard, true))
	literal, err := state.Literal(guard)
	astTestMust(t, err)
	astTestMust(t, state.AssignLiteral(literal.Negated(), NewSMTBooleanAxiom()))
	_, err = state.PropagateCore()
	astTestMust(t, err)
	// Seed the canonical Boolean record directly: solve_nc accepts the
	// equality/disjunction forms produced by canonical rewriting too.
	var equalities []ast.ASTHandle
	for _, element := range elements[:2] {
		equality, err := m.MkEq(element, elements[2])
		astTestMust(t, err)
		equalities = append(equalities, equality)
	}
	disjunction, err := m.MkOrArray(equalities)
	astTestMust(t, err)
	dependency := theory.dependencies.Leaf(smtSequenceAssumption{literal: literal.Negated()})
	astTestMust(t, theory.nonContains.Push(disjunction, dependency))
	astTestMust(t, state.PushScope())
	progress, err := theory.checkNegativeContains()
	astTestMust(t, err)
	if !progress || theory.nonContains.Len() != 0 || state.Inconsistent() {
		t.Fatal("canonical disjunction was not split and retired", progress, theory.nonContains.Len())
	}
	for _, element := range elements[:2] {
		equality, err := m.MkEq(element, elements[2])
		astTestMust(t, err)
		eqLiteral, err := state.Literal(equality)
		astTestMust(t, err)
		value, err := state.Assignment(eqLiteral)
		astTestMust(t, err)
		if value != lbool.False {
			t.Fatal("negative disjunct equality was not propagated", value)
		}
	}
	astTestMust(t, state.PopScopes(1))
	if theory.nonContains.Len() != 1 {
		t.Fatal("parent negative-containment record was not restored")
	}
	parent, dependency, err := theory.nonContains.At(0)
	astTestMust(t, err)
	if parent != disjunction || dependency == nil {
		t.Fatal("parent expression or explanation was lost")
	}
	progress, err = theory.checkNegativeContains()
	astTestMust(t, err)
	if !progress || theory.nonContains.Len() != 0 {
		t.Fatal("restored canonical relation could not be reduced again")
	}
}

func TestSMTSequenceNonContainsCanonicalConstantsSkipQueue(t *testing.T) {
	for _, needle := range []string{"b", "z"} {
		t.Run(needle, func(t *testing.T) {
			m := astTestManager(t)
			left, err := m.MkString("abc")
			astTestMust(t, err)
			right, err := m.MkString(needle)
			astTestMust(t, err)
			contains, err := m.MkSequenceContains(left, right)
			astTestMust(t, err)
			family, err := m.SequenceFamilyID()
			astTestMust(t, err)
			theory, err := NewSMTSequenceTheory(family)
			astTestMust(t, err)
			_, _, state := smtInternalizerFixture(t, m, nil)
			astTestMust(t, state.RegisterTheory(theory))
			astTestMust(t, state.Internalize(contains, true))
			literal, err := state.Literal(contains)
			astTestMust(t, err)
			astTestMust(t, state.AssignLiteral(literal.Negated(), NewSMTBooleanAxiom()))
			_, err = state.PropagateCore()
			astTestMust(t, err)
			if theory.nonContains.Len() != 0 || state.Inconsistent() != (needle == "b") {
				t.Fatal("terminal canonical relation exposed a queue entry or lost conflict", theory.nonContains.Len(), state.Inconsistent())
			}
		})
	}
}

func TestSMTSequenceNonContainsCanonicalSolutionEvidenceAndRollback(t *testing.T) {
	m := astTestManager(t)
	needle, err := m.MkString("a")
	astTestMust(t, err)
	sort, err := m.GetSort(needle)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "nc-canonical-solution"), sort)
	astTestMust(t, err)
	contains, err := m.MkSequenceContains(x, needle)
	astTestMust(t, err)
	family, err := m.SequenceFamilyID()
	astTestMust(t, err)
	theory, err := NewSMTSequenceTheory(family)
	astTestMust(t, err)
	_, _, state := smtInternalizerFixture(t, m, nil)
	astTestMust(t, state.RegisterTheory(theory))
	astTestMust(t, state.Internalize(contains, true))
	literal, err := state.Literal(contains)
	astTestMust(t, err)
	astTestMust(t, state.AssignLiteral(literal.Negated(), NewSMTBooleanAxiom()))
	_, err = state.PropagateCore()
	astTestMust(t, err)
	astTestMust(t, state.PushScope())
	value, err := m.MkString("abc")
	astTestMust(t, err)
	equality, err := m.MkEq(x, value)
	astTestMust(t, err)
	astTestMust(t, state.Internalize(equality, true))
	eqLiteral, err := state.Literal(equality)
	astTestMust(t, err)
	astTestMust(t, state.AssignLiteral(eqLiteral, NewSMTBooleanAxiom()))
	_, err = state.PropagateCore()
	astTestMust(t, err)
	_, dependency, err := theory.nonContains.At(0)
	astTestMust(t, err)
	ref, dependency, err := theory.canonizeSequenceExpression(contains, dependency)
	astTestMust(t, err)
	canonical, err := m.RefGet(ref)
	astTestMust(t, err)
	if !m.IsTrue(canonical) {
		t.Fatal("scoped sequence solution was not expanded")
	}
	assumptions := theory.dependencies.Linearize(dependency)
	hasRelation, hasEquality := false, false
	for _, assumption := range assumptions {
		hasRelation = hasRelation || assumption.literal == literal.Negated()
		hasEquality = hasEquality || assumption.literal == eqLiteral || assumption.left != nil && assumption.right != nil
	}
	if !hasRelation || !hasEquality {
		t.Fatal("canonical conflict lost a relation or equality premise", assumptions)
	}
	astTestMust(t, m.DelRef(&ref))
	progress, err := theory.checkNegativeContains()
	astTestMust(t, err)
	if !progress || !state.Inconsistent() {
		t.Fatal("true canonical negative-containment record did not conflict")
	}
	astTestMust(t, state.PopScopes(1))
	if state.Inconsistent() || theory.nonContains.Len() != 1 || !theory.solutions.IsRoot(x) {
		t.Fatal("child canonical conflict or solution survived rollback")
	}
}

func TestSMTSequenceNonContainsUnresolvedQueuePreventsDone(t *testing.T) {
	m := astTestManager(t)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	p, err := m.MkConst(astTestSymbol(t, m, "nc-unresolved-p"), boolean)
	astTestMust(t, err)
	q, err := m.MkConst(astTestSymbol(t, m, "nc-unresolved-q"), boolean)
	astTestMust(t, err)
	conjunction, err := m.MkAnd(p, q)
	astTestMust(t, err)
	family, err := m.SequenceFamilyID()
	astTestMust(t, err)
	theory, err := NewSMTSequenceTheory(family)
	astTestMust(t, err)
	_, _, state := smtInternalizerFixture(t, m, nil)
	astTestMust(t, state.RegisterTheory(theory))
	astTestMust(t, theory.nonContains.Push(conjunction, nil))
	status, err := theory.FinalCheck()
	astTestMust(t, err)
	if status != SMTFinalCheckGiveUp || theory.nonContains.Len() != 1 {
		t.Fatal("unresolved negative-containment record accepted as solved", status, theory.nonContains.Len())
	}
}

func TestSMTSequenceCanonicalConditionalScope(t *testing.T) {
	m := astTestManager(t)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	condition, err := m.MkConst(astTestSymbol(t, m, "nc-conditional"), boolean)
	astTestMust(t, err)
	yes, err := m.MkString("abc")
	astTestMust(t, err)
	no, err := m.MkString("zzz")
	astTestMust(t, err)
	choice, err := m.MkIte(condition, yes, no)
	astTestMust(t, err)
	needle, err := m.MkString("a")
	astTestMust(t, err)
	contains, err := m.MkSequenceContains(choice, needle)
	astTestMust(t, err)
	astTestPin(t, m, contains)
	family, err := m.SequenceFamilyID()
	astTestMust(t, err)
	theory, err := NewSMTSequenceTheory(family)
	astTestMust(t, err)
	_, _, state := smtInternalizerFixture(t, m, nil)
	astTestMust(t, state.RegisterTheory(theory))
	astTestMust(t, state.Internalize(condition, true))
	literal, err := state.Literal(condition)
	astTestMust(t, err)
	for _, truth := range []bool{true, false, true} {
		astTestMust(t, state.PushScope())
		assigned := literal
		if !truth {
			assigned = assigned.Negated()
		}
		astTestMust(t, state.AssignLiteral(assigned, NewSMTBooleanAxiom()))
		ref, dependency, err := theory.canonizeSequenceExpression(contains, nil)
		astTestMust(t, err)
		canonical, err := m.RefGet(ref)
		astTestMust(t, err)
		if m.IsTrue(canonical) != truth || m.IsFalse(canonical) == truth {
			t.Fatal("conditional canonicalization chose the wrong branch", truth)
		}
		assumptions := theory.dependencies.Linearize(dependency)
		if len(assumptions) != 1 || assumptions[0].literal != assigned {
			t.Fatal("conditional canonicalization lost its decision premise", assumptions)
		}
		astTestMust(t, m.DelRef(&ref))
		astTestMust(t, state.PopScopes(1))
		ref, dependency, err = theory.canonizeSequenceExpression(contains, nil)
		astTestMust(t, err)
		canonical, err = m.RefGet(ref)
		astTestMust(t, err)
		if m.IsTrue(canonical) || m.IsFalse(canonical) || dependency != nil {
			t.Fatal("conditional expansion reused a popped decision")
		}
		astTestMust(t, m.DelRef(&ref))
	}
}
