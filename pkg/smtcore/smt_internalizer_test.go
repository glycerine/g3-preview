package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func smtInternalizerFixture(t *testing.T, m *ast.ASTManager, registry *SMTQuantifierRegistry) (*SMTCheckerContext, *SMTRelevancyPropagator, *SMTInternalizerState) {
	t.Helper()
	context, err := NewSMTCheckerContext(m)
	astTestMust(t, err)
	relevancy, err := NewSMTRelevancyPropagator(context, true)
	astTestMust(t, err)
	state, err := NewSMTInternalizerState(m, context, registry)
	astTestMust(t, err)
	astTestMust(t, state.SetRelevancy(relevancy))
	t.Cleanup(func() { astTestMust(t, state.Close()) })
	return context, relevancy, state
}

func TestSMTInternalizerBooleanGateClausesAndRootAssertions(t *testing.T) {
	m := astTestManager(t)
	a := smtRelevancyBool(t, m, "internalize-gate-a")
	b := smtRelevancyBool(t, m, "internalize-gate-b")
	conjunction, err := m.MkAnd(a, b)
	astTestMust(t, err)
	_, _, state := smtInternalizerFixture(t, m, nil)
	astTestMust(t, state.Internalize(conjunction, true))
	if _, found := state.BoolVar(a); !found {
		t.Fatal("first child was not internalized")
	}
	if _, found := state.BoolVar(b); !found {
		t.Fatal("second child was not internalized")
	}
	if _, found := state.BoolVar(conjunction); !found {
		t.Fatal("gate variable was not internalized")
	}
	if len(state.Clauses()) != 3 {
		t.Fatal("AND definition clauses", state.Clauses())
	}
	if len(state.context.enodes) != 0 {
		t.Fatal("gate-context formula unexpectedly created enodes")
	}

	c := smtRelevancyBool(t, m, "internalize-root-c")
	d := smtRelevancyBool(t, m, "internalize-root-d")
	disjunction, err := m.MkOr(c, d)
	astTestMust(t, err)
	before := len(state.Clauses())
	astTestMust(t, state.Assert(disjunction, ast.ASTHandle{}, 9))
	clauses := state.Clauses()
	if len(clauses) != before+1 || !clauses[len(clauses)-1].Root || len(clauses[len(clauses)-1].Literals) != 2 {
		t.Fatal("root OR clause", clauses)
	}
	if !state.context.relevant[disjunction] || state.Statistics().MaxGeneration != 9 {
		t.Fatal("root relevance or generation")
	}
}

func TestSMTInternalizerTermITEEqualityClausesAndRelevancy(t *testing.T) {
	m := astTestManager(t)
	condition := smtRelevancyBool(t, m, "internalize-term-condition")
	sort, err := m.MkSort(astTestSymbol(t, m, "InternalizeTermSort"), ast.ASTSortInfo{})
	astTestMust(t, err)
	thenTerm, _ := smtEnodeConstant(t, m, 2240, sort, 0)
	elseTerm, _ := smtEnodeConstant(t, m, 2241, sort, 0)
	term, err := m.MkIte(condition, thenTerm, elseTerm)
	astTestMust(t, err)
	context, relevancy, state := smtInternalizerFixture(t, m, nil)
	astTestMust(t, state.Internalize(term, false))
	if context.enodes[term] == nil || context.enodes[thenTerm] == nil || context.enodes[elseTerm] == nil || len(state.Clauses()) != 2 {
		t.Fatal("term ITE enodes or clauses", len(context.enodes), state.Clauses())
	}
	thenEquality, err := m.MkEq(term, thenTerm)
	astTestMust(t, err)
	elseEquality, err := m.MkEq(term, elseTerm)
	astTestMust(t, err)
	for _, equality := range []ast.ASTHandle{thenEquality, elseEquality} {
		variable, found := state.BoolVar(equality)
		if !found {
			t.Fatal("ITE equality variable")
		}
		data, err := state.BooleanData(variable)
		astTestMust(t, err)
		if !data.IsEquality() || !data.IsENode() {
			t.Fatal("equality metadata", *data)
		}
	}
	astTestMust(t, relevancy.MarkRelevant(term))
	astTestMust(t, relevancy.Propagate())
	conditionVar, _ := state.BoolVar(condition)
	astTestMust(t, state.SetAssignment(conditionVar, lbool.False))
	astTestMust(t, relevancy.AssignmentEvent(condition, false))
	astTestMust(t, relevancy.Propagate())
	if !relevancy.IsRelevant(elseEquality) || relevancy.IsRelevant(thenEquality) {
		t.Fatal("term ITE selected equality relevance")
	}
}

func TestSMTInternalizerQuantifierRegistrationAndDefaultAssertion(t *testing.T) {
	m := astTestManager(t)
	registry, _ := smtQuickRegistry(t, m)
	_, _, state := smtInternalizerFixture(t, m, registry)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	variable, err := m.MkVar(0, boolean)
	astTestMust(t, err)
	quantifier, err := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true, Sorts: []ast.ASTHandle{boolean}, Names: []symbol.Symbol{astTestSymbol(t, m, "internalize-q-x")}, Body: variable,
	})
	astTestMust(t, err)
	astTestMust(t, state.SetCachedGeneration(quantifier, 12))
	astTestMust(t, state.Internalize(quantifier, true))
	if generation, err := registry.Generation(quantifier); err != nil || generation != 12 {
		t.Fatal("quantifier generation", generation, err)
	}
	atom := smtRelevancyBool(t, m, "internalize-asserted-atom")
	astTestMust(t, state.Assert(atom, ast.ASTHandle{}, 4))
	atomVar, found := state.BoolVar(atom)
	if !found {
		t.Fatal("asserted atom variable")
	}
	if value, err := state.Assignment(NewSMTLiteral(atomVar, false)); err != nil || value != lbool.True {
		t.Fatal("default assertion assignment", value, err)
	}
	if !state.context.relevant[atom] {
		t.Fatal("default assertion relevance")
	}
}

func TestSMTInternalizerConsumesMaterializedQuantifierInstance(t *testing.T) {
	m := astTestManager(t)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	variable, err := m.MkVar(0, boolean)
	astTestMust(t, err)
	quantifier, err := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true, Sorts: []ast.ASTHandle{boolean}, Names: []symbol.Symbol{astTestSymbol(t, m, "internalize-instance-x")}, Body: variable,
	})
	astTestMust(t, err)
	_, binding := smtEnodeConstant(t, m, 2250, boolean, 2)
	checkerContext, err := NewSMTCheckerContext(m)
	astTestMust(t, err)
	builder := smtQIInstanceBuilder(t, m, checkerContext)
	prepared, err := builder.Prepare(SMTQIQueueEntry{Fingerprint: &SMTFingerprint{data: quantifier, args: []*SMTEnode{binding}}, Cost: 6, Generation: 2, Instantiated: true})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, prepared.Close()) })
	registry, _ := smtQuickRegistry(t, m)
	_, _, state := smtInternalizerFixture(t, m, registry)
	instance, err := builder.MaterializeAndInternalize(prepared, smtQuickTheoryRewriter(t, m), &SMTQuantifierStat{}, 0, state)
	astTestMust(t, err)
	if instance == nil {
		t.Fatal("missing materialized instance")
	}
	t.Cleanup(func() { astTestMust(t, instance.Close()) })
	if len(state.Clauses()) != 1 || !state.Clauses()[0].Root {
		t.Fatal("internalized quantifier lemma", state.Clauses())
	}
	if state.Statistics().MaxGeneration != instance.Generation() {
		t.Fatal("instance generation", state.Statistics().MaxGeneration, instance.Generation())
	}
}

func TestSMTInternalizerDeepAssertionUsesTopologicalPrepass(t *testing.T) {
	m := astTestManager(t)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	info, err := m.NewFuncDeclInfo(ast.NullFamilyID, -1, nil, ast.ASTFuncFlags{}, false)
	astTestMust(t, err)
	wrapper, err := m.MkFuncDeclWithInfo(astTestSymbol(t, m, "internalize-deep-wrapper"), []ast.ASTHandle{boolean}, boolean, info)
	astTestMust(t, err)
	astTestMust(t, m.DelFuncDeclInfo(&info))
	expression := smtRelevancyBool(t, m, "internalize-deep-leaf")
	for depth := uint32(0); depth <= smtDeepExpressionThreshold; depth++ {
		expression, err = m.MkApp(wrapper, []ast.ASTHandle{expression})
		astTestMust(t, err)
	}
	actualDepth, err := m.GetDepth(expression)
	astTestMust(t, err)
	if actualDepth <= smtDeepExpressionThreshold {
		t.Fatal("fixture is not deep", actualDepth)
	}
	_, _, state := smtInternalizerFixture(t, m, nil)
	astTestMust(t, state.Assert(expression, ast.ASTHandle{}, 3))
	if len(state.boolVar) != int(smtDeepExpressionThreshold)+2 {
		t.Fatal("deep assertion internalization count", len(state.boolVar))
	}
	rootVariable, found := state.BoolVar(expression)
	if !found {
		t.Fatal("deep assertion root was not internalized")
	}
	if value, err := state.Assignment(NewSMTLiteral(rootVariable, false)); err != nil || value != lbool.True {
		t.Fatal("deep assertion root assignment", value, err)
	}
}
